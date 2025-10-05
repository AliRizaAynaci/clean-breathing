package auth

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// GET /logout – deletes cookie, returns 204
func Logout(c *fiber.Ctx) error {
	settings := resolveSessionCookieSettings()
	cookie := &fiber.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		SameSite: settings.SameSite,
		Secure:   settings.Secure,
		MaxAge:   -1,
	}
	if settings.Domain != "" {
		cookie.Domain = settings.Domain
	}
	c.Cookie(cookie)
	return c.SendStatus(fiber.StatusNoContent)
}
