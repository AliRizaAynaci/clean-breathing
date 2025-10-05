package airquality

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	airQualityBaseURL = "https://air-quality-api.open-meteo.com/v1/air-quality"
	weatherBaseURL    = "https://api.open-meteo.com/v1/forecast"
)

// Service retrieves AQI data from Open-Meteo APIs.
type Service struct {
	client             *http.Client
	airQualityURL      string
	weatherForecastURL string
}

// NewService constructs a Service using the provided HTTP client. If client is nil,
// a client with a 10 second timeout is created. baseURL is optional and falls back to
// the Open-Meteo endpoint when empty (legacy support - now ignored).
func NewService(client *http.Client, baseURL string) *Service {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Service{
		client:             client,
		airQualityURL:      airQualityBaseURL,
		weatherForecastURL: weatherBaseURL,
	}
}

// Metrics represents the latest pollutant measurements (µg/m³).
type Metrics struct {
	Temperature       float64
	Humidity          float64
	PM25              float64
	PM10              float64
	NO2               float64
	SO2               float64
	CO                float64
	PopulationDensity float64
}

// GetMetrics fetches the most recent pollutant values for the given coordinates.
// Makes two API calls: one for air quality data and one for weather data.
func (s *Service) GetMetrics(latitude, longitude float64) (Metrics, error) {
	// Fetch air quality data (pollutants)
	airQualityURL := fmt.Sprintf("%s?latitude=%f&longitude=%f&hourly=carbon_monoxide,sulphur_dioxide,nitrogen_dioxide,pm10,pm2_5&timezone=UTC", s.airQualityURL, latitude, longitude)

	req, err := http.NewRequest(http.MethodGet, airQualityURL, nil)
	if err != nil {
		return Metrics{}, fmt.Errorf("create air quality request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return Metrics{}, fmt.Errorf("air quality request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return Metrics{}, fmt.Errorf("air quality request failed: status %d, url: %s, body read error: %v", resp.StatusCode, airQualityURL, readErr)
		}
		return Metrics{}, fmt.Errorf("air quality request failed: status %d, url: %s, response: %s", resp.StatusCode, airQualityURL, string(bodyBytes))
	}

	var airQualityPayload struct {
		Hourly struct {
			PM25            []float64 `json:"pm2_5"`
			PM10            []float64 `json:"pm10"`
			NitrogenDioxide []float64 `json:"nitrogen_dioxide"`
			SulphurDioxide  []float64 `json:"sulphur_dioxide"`
			CarbonMonoxide  []float64 `json:"carbon_monoxide"`
		} `json:"hourly"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&airQualityPayload); err != nil {
		return Metrics{}, fmt.Errorf("decode air quality response: %w", err)
	}

	// Fetch weather data (temperature and humidity)
	weatherURL := fmt.Sprintf("%s?latitude=%f&longitude=%f&hourly=temperature_2m,relative_humidity_2m&timezone=UTC", s.weatherForecastURL, latitude, longitude)

	req, err = http.NewRequest(http.MethodGet, weatherURL, nil)
	if err != nil {
		return Metrics{}, fmt.Errorf("create weather request: %w", err)
	}

	resp, err = s.client.Do(req)
	if err != nil {
		return Metrics{}, fmt.Errorf("weather request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return Metrics{}, fmt.Errorf("weather request failed: status %d, url: %s, body read error: %v", resp.StatusCode, weatherURL, readErr)
		}
		return Metrics{}, fmt.Errorf("weather request failed: status %d, url: %s, response: %s", resp.StatusCode, weatherURL, string(bodyBytes))
	}

	var weatherPayload struct {
		Hourly struct {
			Temperature []float64 `json:"temperature_2m"`
			Humidity    []float64 `json:"relative_humidity_2m"`
		} `json:"hourly"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&weatherPayload); err != nil {
		return Metrics{}, fmt.Errorf("decode weather response: %w", err)
	}

	// Combine data into Metrics
	metrics := Metrics{}
	var ok bool

	if metrics.Temperature, ok = latestFloat(weatherPayload.Hourly.Temperature); !ok {
		return Metrics{}, errors.New("weather response missing temperature data")
	}
	if metrics.Humidity, ok = latestFloat(weatherPayload.Hourly.Humidity); !ok {
		return Metrics{}, errors.New("weather response missing humidity data")
	}
	if metrics.PM25, ok = latestFloat(airQualityPayload.Hourly.PM25); !ok {
		return Metrics{}, errors.New("air quality response missing PM2.5 data")
	}
	if metrics.PM10, ok = latestFloat(airQualityPayload.Hourly.PM10); !ok {
		return Metrics{}, errors.New("air quality response missing PM10 data")
	}
	if metrics.NO2, ok = latestFloat(airQualityPayload.Hourly.NitrogenDioxide); !ok {
		return Metrics{}, errors.New("air quality response missing NO2 data")
	}
	if metrics.SO2, ok = latestFloat(airQualityPayload.Hourly.SulphurDioxide); !ok {
		return Metrics{}, errors.New("air quality response missing SO2 data")
	}
	if metrics.CO, ok = latestFloat(airQualityPayload.Hourly.CarbonMonoxide); !ok {
		return Metrics{}, errors.New("air quality response missing CO data")
	}
	// Open-Meteo API does not provide population density, use fixed default value.
	metrics.PopulationDensity = 497

	return metrics, nil
}

// FeatureVector returns the ordered feature slice expected by the ML model.
func (m Metrics) FeatureVector() []float64 {
	return []float64{
		m.Temperature,
		m.Humidity,
		m.PM25,
		m.PM10,
		m.NO2,
		m.SO2,
		m.CO,
		m.PopulationDensity,
	}
}

func latestFloat(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	return values[len(values)-1], true
}
