package notification

type Notification struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"uniqueIndex"`
	Latitude  float64
	Longitude float64
	Threshold int
	Email     string
}
