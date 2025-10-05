package auth

import (
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

// GET /logout – deletes cookie, returns 204
func Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    "",
		Domain:   os.Getenv("DOMAIN"),
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HTTPOnly: true,
		SameSite: "None",
		Secure:   true,
		MaxAge:   -1,
	})
	return c.SendStatus(fiber.StatusNoContent)
}
