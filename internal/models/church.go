package models

import "time"

type Church struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	Name         string         `gorm:"not null" json:"name"`
	Denomination string         `gorm:"not null;default:'Catholic'" json:"denomination"`
	Address      string         `gorm:"not null" json:"address"`
	Latitude     float64        `gorm:"not null;index" json:"latitude"`
	Longitude    float64        `gorm:"not null;index" json:"longitude"`
	Phone        string         `json:"phone,omitempty"`
	Website      string         `json:"website,omitempty"`
	Description  string         `json:"description,omitempty"`
	Verified     bool           `gorm:"default:false" json:"verified"`
	CreatedByID  uint           `json:"created_by_id"`
	CreatedBy    *User          `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
	Schedules    []MassSchedule `gorm:"foreignKey:ChurchID" json:"schedules,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type MassSchedule struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ChurchID  uint      `gorm:"not null;index" json:"church_id"`
	DayOfWeek int       `gorm:"not null" json:"day_of_week"` // 0=Sunday, 6=Saturday
	StartTime string    `gorm:"not null" json:"start_time"`  // "HH:MM" format
	Language  string    `gorm:"default:'English'" json:"language"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

var DayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
