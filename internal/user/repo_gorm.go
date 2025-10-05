package user

import "gorm.io/gorm"

type gormRepo struct{ db *gorm.DB }

func NewGormRepo(db *gorm.DB) Repository { return &gormRepo{db} }

func (r *gormRepo) Create(u *User) error                     { return r.db.Create(u).Error }
func (r *gormRepo) FindByID(id uint) (*User, error)          { return r.find("id = ?", id) }
func (r *gormRepo) FindByEmail(e string) (*User, error)      { return r.find("email = ?", e) }
func (r *gormRepo) FindByGoogleID(gid string) (*User, error) { return r.find("google_id = ?", gid) }

// ✅ Method ismi ve implementasyon düzeltildi
func (r *gormRepo) UpdateName(id uint, name string) error {
	return r.db.Model(&User{}).
		Where("id = ?", id).
		Update("name", name).Error
}

func (r *gormRepo) find(q string, arg interface{}) (*User, error) {
	var u User
	if err := r.db.Where(q, arg).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}
