package airquality

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// MLPredictor is a function type for ML predictions
type MLPredictor func(latitude, longitude float64, metrics Metrics) (string, error)

type Handler struct {
	Service     *Service
	MLPredictor MLPredictor
}

type GetAirQualityRequest struct {
	Latitude  float64 `json:"latitude" query:"latitude"`
	Longitude float64 `json:"longitude" query:"longitude"`
}

type AirQualityResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Metrics   Metrics `json:"metrics"`
	RiskLevel string  `json:"risk_level"`
	Timestamp string  `json:"timestamp"`
}

func NewHandler(service *Service, mlPredictor MLPredictor) *Handler {
	return &Handler{
		Service:     service,
		MLPredictor: mlPredictor,
	}
}

// GetAirQuality fetches current air quality data and ML prediction for a location
func (h *Handler) GetAirQuality(c *fiber.Ctx) error {
	var req GetAirQualityRequest

	// Parse query parameters
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid query parameters",
		})
	}

	// Validate coordinates
	if req.Latitude == 0 || req.Longitude == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Latitude and longitude are required",
		})
	}

	// Fetch air quality metrics
	metrics, err := h.Service.GetMetrics(req.Latitude, req.Longitude)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch air quality data",
		})
	}

	// Get ML prediction
	riskLevel := "unknown"
	if h.MLPredictor != nil {
		predictedRisk, err := h.MLPredictor(req.Latitude, req.Longitude, metrics)
		if err == nil && predictedRisk != "" {
			riskLevel = predictedRisk
		}
	}

	response := AirQualityResponse{
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Metrics:   metrics,
		RiskLevel: riskLevel,
		Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05Z07:00"),
	}

	return c.JSON(response)
}
