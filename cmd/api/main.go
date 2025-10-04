package main

import (
	"log"

	"nasa-app/internal/config"
	database "nasa-app/internal/db"
	"nasa-app/internal/handlers"
	"nasa-app/internal/models"
	"nasa-app/internal/server"
)

func main() {
	// Config
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// DB connection
	dsn := config.BuildDSN()
	log.Printf("Connecting to database...")

	db, err := database.Connect(dsn)
	if err != nil {
		log.Fatal("database connection error: ", err)
	}

	// Migrations
	if err := database.Migrate(db, &models.User{}); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	// Google OAuth
	if err := handlers.InitGoogleOAuth(); err != nil {
		log.Fatal("Failed to initialize Google OAuth: ", err)
	}
	log.Println("Google OAuth initialized ✓")

	// Server
	app := server.NewFiberApp(db)
	log.Printf("Server running on :%s 🚀", cfg.AppPort)

	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
