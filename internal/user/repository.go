package user

type Repository interface {
	Create(u *User) error
	FindByID(id uint) (*User, error)
	FindByEmail(e string) (*User, error)
	FindByGoogleID(gid string) (*User, error)
	UpdateName(id uint, name string) error // ✅ Method ismi değiştirildi, pic parametresi kaldırıldı
}
