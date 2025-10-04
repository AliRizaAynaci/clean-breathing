package db

import (
	"log"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var gormDB *gorm.DB

// Connect opens a DB connection and returns *gorm.DB
func Connect(dsn string) (*gorm.DB, error) {
	// SSL mode kontrolü - Heroku için otomatik ekle
	if !strings.Contains(dsn, "sslmode=") {
		if strings.Contains(dsn, "?") {
			dsn += "&sslmode=require"
		} else {
			dsn += "?sslmode=require"
		}
		log.Println("🔒 SSL mode automatically enabled")
	}

	log.Println("📡 Connecting to database...")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Printf("❌ Database connection failed: %v", err)
		return nil, err
	}

	// Connection test
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("❌ Failed to get database instance: %v", err)
		return nil, err
	}

	if err := sqlDB.Ping(); err != nil {
		log.Printf("❌ Database ping failed: %v", err)
		return nil, err
	}

	// Global DB'yi set et - ÖNEMLİ!
	gormDB = db

	log.Println("✅ Database connected successfully!")
	return db, nil
}

// Migrate runs AutoMigrate for the given models
func Migrate(db *gorm.DB, models ...interface{}) error {
	// run only if MIGRATE_ON_START=true (env-toggled)
	if os.Getenv("MIGRATE_ON_START") != "true" {
		log.Println("⏭️  Skipping migrations (MIGRATE_ON_START != true)")
		return nil
	}

	log.Println("🔄 Running migrations...")

	err := db.AutoMigrate(models...)
	if err != nil {
		log.Printf("❌ Migration failed: %v", err)
		return err
	}

	log.Println("✅ Migrations completed successfully!")
	return nil
}

// GetDB returns the global DB instance
func GetDB() *gorm.DB {
	return gormDB
}
