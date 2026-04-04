package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleModerator Role = "moderator"
	RoleAdmin     Role = "admin"
)

// MaxBcryptPasswordLen is the maximum password length bcrypt supports without silent truncation.
const MaxBcryptPasswordLen = 72

// MinPasswordLen is the minimum password length enforced at registration.
const MinPasswordLen = 8

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	Password     string    `gorm:"column:password_hash;not null" json:"-"`
	Name         string    `gorm:"not null" json:"name"`
	Denomination string    `gorm:"not null;default:'Catholic'" json:"denomination"`
	Role         Role      `gorm:"not null;default:'user'" json:"role"`
	Latitude     *float64  `json:"latitude,omitempty"`
	Longitude    *float64  `json:"longitude,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) SetPassword(password string) error {
	if len([]byte(password)) > MaxBcryptPasswordLen {
		password = string([]byte(password)[:MaxBcryptPasswordLen])
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	if len([]byte(password)) > MaxBcryptPasswordLen {
		password = string([]byte(password)[:MaxBcryptPasswordLen])
	}
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) == nil
}

// Favorite represents a user's bookmarked church.
type Favorite struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_church" json:"user_id"`
	ChurchID  uint      `gorm:"not null;uniqueIndex:idx_user_church" json:"church_id"`
	Church    *Church   `gorm:"foreignKey:ChurchID" json:"church,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
