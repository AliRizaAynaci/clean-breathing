package db

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"nasa-app/internal/config"
	"nasa-app/internal/models"
)

var gormDB *gorm.DB

func ConnectPostgres(cfg *config.Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		cfg.PostgresHost, cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresDB, cfg.PostgresPort,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Migration
	log.Println("Running migrations...")
	if err := db.AutoMigrate(&models.User{}); err != nil {
		return nil, err
	}

	gormDB = db
	return db, nil
}

func GetDB() (*gorm.DB, error) {
	if gormDB == nil {
		return nil, fmt.Errorf("DB not initialized")
	}
	return gormDB, nil
}
