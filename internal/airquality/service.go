package airquality

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const defaultBaseURL = "https://air-quality-api.open-meteo.com/v1/air-quality"

// Service retrieves AQI data from Open-Meteo air-quality API.
type Service struct {
	client  *http.Client
	baseURL string
}

// NewService constructs a Service using the provided HTTP client. If client is nil,
// a client with a 10 second timeout is created. baseURL is optional and falls back to
// the Open-Meteo endpoint when empty.
func NewService(client *http.Client, baseURL string) *Service {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Service{client: client, baseURL: baseURL}
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
func (s *Service) GetMetrics(latitude, longitude float64) (Metrics, error) {
	url := fmt.Sprintf("%s?latitude=%f&longitude=%f&hourly=carbon_monoxide,sulphur_dioxide,nitrogen_dioxide,pm10,pm2_5,temperature_2m,relative_humidity_2m,population_density&timezone=UTC", s.baseURL, latitude, longitude)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Metrics{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return Metrics{}, fmt.Errorf("air-quality request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return Metrics{}, fmt.Errorf("air-quality request failed: status %d", resp.StatusCode)
	}

	var payload struct {
		Hourly struct {
			CarbonMonoxide  []float64 `json:"carbon_monoxide"`
			SulphurDioxide  []float64 `json:"sulphur_dioxide"`
			NitrogenDioxide []float64 `json:"nitrogen_dioxide"`
			PM10            []float64 `json:"pm10"`
			PM25            []float64 `json:"pm2_5"`
			Temperature     []float64 `json:"temperature_2m"`
			Humidity        []float64 `json:"relative_humidity_2m"`
			Population      []float64 `json:"population_density"`
		} `json:"hourly"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Metrics{}, fmt.Errorf("decode air-quality response: %w", err)
	}

	metrics := Metrics{}
	var ok bool

	if metrics.Temperature, ok = latestFloat(payload.Hourly.Temperature); !ok {
		return Metrics{}, errors.New("air-quality response missing temperature data")
	}
	if metrics.Humidity, ok = latestFloat(payload.Hourly.Humidity); !ok {
		return Metrics{}, errors.New("air-quality response missing humidity data")
	}
	if metrics.PM25, ok = latestFloat(payload.Hourly.PM25); !ok {
		return Metrics{}, errors.New("air-quality response missing PM2.5 data")
	}
	if metrics.PM10, ok = latestFloat(payload.Hourly.PM10); !ok {
		return Metrics{}, errors.New("air-quality response missing PM10 data")
	}
	if metrics.NO2, ok = latestFloat(payload.Hourly.NitrogenDioxide); !ok {
		return Metrics{}, errors.New("air-quality response missing NO2 data")
	}
	if metrics.SO2, ok = latestFloat(payload.Hourly.SulphurDioxide); !ok {
		return Metrics{}, errors.New("air-quality response missing SO2 data")
	}
	if metrics.CO, ok = latestFloat(payload.Hourly.CarbonMonoxide); !ok {
		return Metrics{}, errors.New("air-quality response missing CO data")
	}
	if metrics.PopulationDensity, ok = latestFloat(payload.Hourly.Population); !ok {
		return Metrics{}, errors.New("air-quality response missing population density data")
	}

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
