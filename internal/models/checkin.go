package models

import "time"

type CheckIn struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	User      *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ChurchID  uint      `gorm:"not null;index" json:"church_id"`
	Church    *Church   `gorm:"foreignKey:ChurchID" json:"church,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
