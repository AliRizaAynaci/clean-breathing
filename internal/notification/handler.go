package notification

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gorilla/sessions"
)

type Handler struct {
	Repo  *Repository
	Store *sessions.CookieStore
}

type subscribeRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Threshold int     `json:"threshold"`
	Email     string  `json:"email"`
}

func NewHandler(repo *Repository, store *sessions.CookieStore) *Handler {
	return &Handler{Repo: repo, Store: store}
}

func (h *Handler) Subscribe(c *fiber.Ctx) error {
	// CORS ayarları Fiber'da globalde yapıldı

	// Auth kontrolü: user_id Fiber context'ten alınır
	uid := c.Locals("user_id")
	userID, ok := uid.(uint)
	if !ok || userID == 0 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}

	var req subscribeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	n := &Notification{
		UserID:    userID,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Threshold: req.Threshold,
		Email:     req.Email,
	}

	if err := h.Repo.UpsertNotification(n); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Database error"})
	}

	return c.JSON(fiber.Map{"message": "Subscription updated"})
}
