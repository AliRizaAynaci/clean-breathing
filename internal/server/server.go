package server

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
	"nasa-app/internal/handlers"
)

func NewFiberApp(db *gorm.DB) *fiber.App {
	app := fiber.New()

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		sqlDB, err := db.DB()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "db error"})
		}
		if err := sqlDB.Ping(); err != nil {
			return c.Status(500).JSON(fiber.Map{"status": "db down"})
		}
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// User routes
	app.Get("/users", handlers.GetUsers(db))
	app.Post("/users", handlers.CreateUser(db))

	// Oauth2
	app.Get("/auth/google/login", handlers.GoogleLogin)
	app.Get("/auth/google/callback", handlers.GoogleCallback)

	return app
}
