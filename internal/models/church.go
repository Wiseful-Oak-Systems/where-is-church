package models

import "time"

// DataQuality represents the three-state data quality lifecycle.
// Inspired by iNaturalist's Research Grade system.
type DataQuality string

const (
	QualityUnverified         DataQuality = "unverified"          // User-submitted, no confirmations
	QualityCommunityConfirmed DataQuality = "community_confirmed" // 3+ independent user confirmations
	QualityOfficiallyVerified DataQuality = "officially_verified" // Verified by admin, CM, or church owner
)

// ConfirmationsNeeded is the number of independent user confirmations
// required to graduate from Unverified to Community Confirmed.
const ConfirmationsNeeded = 3

type Church struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"not null" json:"name"`
	Denomination   string         `gorm:"not null;default:'Catholic'" json:"denomination"`
	Address        string         `gorm:"not null" json:"address"`
	Latitude       float64        `gorm:"not null;index" json:"latitude"`
	Longitude      float64        `gorm:"not null;index" json:"longitude"`
	Phone          string         `json:"phone,omitempty"`
	Website        string         `json:"website,omitempty"`
	Description    string         `json:"description,omitempty"`
	Verified       bool           `gorm:"default:false" json:"verified"`
	DataQuality    DataQuality    `gorm:"not null;default:'unverified'" json:"data_quality"`
	Confirmations  int            `gorm:"default:0" json:"confirmations"`
	LastVerified   *time.Time     `json:"last_verified,omitempty"`
	CreatedByID  *uint          `json:"created_by_id,omitempty"`
	CreatedBy    *User          `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
	Schedules    []MassSchedule `gorm:"foreignKey:ChurchID" json:"schedules,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ScheduleType distinguishes mass, confession, and adoration schedules.
type ScheduleType string

const (
	ScheduleMass       ScheduleType = "mass"
	ScheduleConfession ScheduleType = "confession"
	ScheduleAdoration  ScheduleType = "adoration"
)

type MassSchedule struct {
	ID        uint         `gorm:"primaryKey" json:"id"`
	ChurchID  uint         `gorm:"not null;index" json:"church_id"`
	Type      ScheduleType `gorm:"not null;default:'mass'" json:"type"`
	DayOfWeek int          `gorm:"not null" json:"day_of_week"` // 0=Sunday, 6=Saturday
	StartTime string       `gorm:"not null" json:"start_time"`  // "HH:MM" format
	EndTime   string       `json:"end_time,omitempty"`          // for confession/adoration ranges
	Language  string       `gorm:"default:'English'" json:"language"`
	Notes     string       `json:"notes,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

var DayNames = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// ValidScheduleTypes for input validation.
var ValidScheduleTypes = map[ScheduleType]bool{
	ScheduleMass:       true,
	ScheduleConfession: true,
	ScheduleAdoration:  true,
}
