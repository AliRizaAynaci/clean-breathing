package user

import (
	"errors"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("user not found")

type Service struct{ repo Repository }

func NewService(r Repository) *Service { return &Service{r} }

// FindOrCreate checks by GoogleID; inserts a new row if not found
func (s *Service) FindOrCreate(gID, email, name, pic string) (*User, error) {
	u, err := s.repo.FindByGoogleID(gID)
	switch {
	case err == nil:
		// ✅ Sadece name güncellemesi, picture kaldırıldı
		needUpdate := false
		if (u.Name == "" || u.Name == "Google User") && name != "" {
			u.Name, needUpdate = name, true
		}
		if needUpdate {
			_ = s.repo.UpdateName(u.ID, u.Name)
		}
		return u, nil

	case errors.Is(err, gorm.ErrRecordNotFound):
		// ✅ Picture field kaldırıldı
		u = &User{GoogleID: gID, Email: email, Name: name}
		if err := s.repo.Create(u); err != nil {
			return nil, err
		}
		return u, nil

	default:
		return nil, err
	}
}

func (s *Service) GetByID(id uint) (*User, error) {
	return s.repo.FindByID(id)
}
