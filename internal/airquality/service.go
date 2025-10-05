package airquality

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
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

// Metrics represents the latest pollutant measurements (µg/m³) and AQI.
type Metrics struct {
	AQI  int
	CO   float64
	SO2  float64
	NO2  float64
	O3   float64
	PM10 float64
	PM25 float64
}

// GetMetrics fetches the most recent pollutant values for the given coordinates.
func (s *Service) GetMetrics(latitude, longitude float64) (Metrics, error) {
	url := fmt.Sprintf("%s?latitude=%f&longitude=%f&hourly=european_aqi,carbon_monoxide,sulphur_dioxide,nitrogen_dioxide,ozone,pm10,pm2_5&timezone=UTC", s.baseURL, latitude, longitude)

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
			EuropeanAQI     []float64 `json:"european_aqi"`
			CarbonMonoxide  []float64 `json:"carbon_monoxide"`
			SulphurDioxide  []float64 `json:"sulphur_dioxide"`
			NitrogenDioxide []float64 `json:"nitrogen_dioxide"`
			Ozone           []float64 `json:"ozone"`
			PM10            []float64 `json:"pm10"`
			PM25            []float64 `json:"pm2_5"`
		} `json:"hourly"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Metrics{}, fmt.Errorf("decode air-quality response: %w", err)
	}

	metrics := Metrics{}
	var ok bool
	metrics.AQI, ok = latestInt(payload.Hourly.EuropeanAQI)
	if !ok {
		return Metrics{}, errors.New("air-quality response missing AQI data")
	}

	if metrics.CO, ok = latestFloat(payload.Hourly.CarbonMonoxide); !ok {
		return Metrics{}, errors.New("air-quality response missing CO data")
	}
	if metrics.SO2, ok = latestFloat(payload.Hourly.SulphurDioxide); !ok {
		return Metrics{}, errors.New("air-quality response missing SO2 data")
	}
	if metrics.NO2, ok = latestFloat(payload.Hourly.NitrogenDioxide); !ok {
		return Metrics{}, errors.New("air-quality response missing NO2 data")
	}
	if metrics.O3, ok = latestFloat(payload.Hourly.Ozone); !ok {
		return Metrics{}, errors.New("air-quality response missing O3 data")
	}
	if metrics.PM10, ok = latestFloat(payload.Hourly.PM10); !ok {
		return Metrics{}, errors.New("air-quality response missing PM10 data")
	}
	if metrics.PM25, ok = latestFloat(payload.Hourly.PM25); !ok {
		return Metrics{}, errors.New("air-quality response missing PM2.5 data")
	}

	return metrics, nil
}

// GetAQI is retained for backward compatibility.
func (s *Service) GetAQI(latitude, longitude float64) (int, error) {
	m, err := s.GetMetrics(latitude, longitude)
	if err != nil {
		return 0, err
	}
	return m.AQI, nil
}

func latestInt(values []float64) (int, bool) {
	if len(values) == 0 {
		return 0, false
	}
	return int(math.Round(values[len(values)-1])), true
}

func latestFloat(values []float64) (float64, bool) {
	if len(values) == 0 {
		return 0, false
	}
	return values[len(values)-1], true
}
