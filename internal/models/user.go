package models

import "time"

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:100"`
	Email     string `gorm:"uniqueIndex;size:150;not null"`
	GoogleID  string `gorm:"uniqueIndex;size:100"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
