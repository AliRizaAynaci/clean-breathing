package notification

import (
	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) UpsertNotification(n *Notification) error {
	var existing Notification
	err := r.DB.Where("user_id = ?", n.UserID).First(&existing).Error
	if err == nil {
		n.ID = existing.ID
		return r.DB.Save(n).Error
	} else if err == gorm.ErrRecordNotFound {
		return r.DB.Create(n).Error
	}
	return err
}

func (r *Repository) GetAllNotifications() ([]Notification, error) {
	var notifications []Notification
	err := r.DB.Find(&notifications).Error
	return notifications, err
}
