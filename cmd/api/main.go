package main

import (
	"fmt"
	"log"

	"nasa-app/internal/config"
	"nasa-app/internal/db"
	"nasa-app/internal/handlers"
	"nasa-app/internal/server"
)

func main() {
	// Config
	cfg, err := config.LoadConfig(".env")
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// DB connection
	conn, err := db.ConnectPostgres(cfg)
	if err != nil {
		log.Fatal("db connection error: ", err)
	}

	// OAuth Config - ÖNEMLİ: Bu satırı ekleyin!
	if err := handlers.InitGoogleOAuth(); err != nil {
		log.Fatal("Failed to initialize Google OAuth: ", err)
	}

	oauthCfg, _ := config.LoadOauthConfig(".env")
	fmt.Println("Oauth Config ClientID:", oauthCfg.ClientID[:20]+"...") // İlk 20 karakteri göster

	// Server
	app := server.NewFiberApp(conn)
	log.Printf("Server running on :%s 🚀", cfg.AppPort)

	if err := app.Listen(":" + cfg.AppPort); err != nil {
		log.Fatal(err)
	}
}
