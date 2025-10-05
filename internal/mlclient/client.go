package mlclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"nasa-app/internal/airquality"
)

const defaultPredictPath = "/predict"

// Client talks to the external ML prediction service.
type Client struct {
	baseURL    string
	predictURL string
	httpClient *http.Client
}

// New creates a client for the ML service.
// baseURL should contain host (e.g. http://localhost:8000)
// predictPath is optional and defaults to /predict.
func New(baseURL, predictPath string, httpClient *http.Client) (*Client, error) {
	baseURL = strings.TrimSuffix(baseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("ml service base URL is required")
	}

	if predictPath == "" {
		predictPath = defaultPredictPath
	} else if !strings.HasPrefix(predictPath, "/") {
		predictPath = "/" + predictPath
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return &Client{
		baseURL:    baseURL,
		predictURL: baseURL + predictPath,
		httpClient: httpClient,
	}, nil
}

// PredictionRequest is the payload sent to the ML service.
type PredictionRequest struct {
	Latitude  float64            `json:"latitude"`
	Longitude float64            `json:"longitude"`
	Metrics   airquality.Metrics `json:"metrics"`
}

// PredictionResponse represents the ML model result.
type PredictionResponse struct {
	PredictedAQI float64        `json:"predicted_aqi"`
	RiskLevel    string         `json:"risk_level,omitempty"`
	Meta         map[string]any `json:"meta,omitempty"`
}

// Predict sends pollutant metrics to the ML service and returns the prediction.
func (c *Client) Predict(req PredictionRequest) (PredictionResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return PredictionResponse{}, fmt.Errorf("marshal ml request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.predictURL, bytes.NewReader(payload))
	if err != nil {
		return PredictionResponse{}, fmt.Errorf("create ml request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return PredictionResponse{}, fmt.Errorf("ml request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return PredictionResponse{}, fmt.Errorf("ml service error: status %d", resp.StatusCode)
	}

	var prediction PredictionResponse
	if err := json.NewDecoder(resp.Body).Decode(&prediction); err != nil {
		return PredictionResponse{}, fmt.Errorf("decode ml response: %w", err)
	}

	return prediction, nil
}
