package models

import "time"

// AttachableType identifies what entity an attachment belongs to.
type AttachableType string

const (
	AttachSuggestion AttachableType = "suggestion"
	AttachClaim      AttachableType = "claim"
	AttachChurch     AttachableType = "church"
)

// Attachment represents an uploaded file linked to a suggestion, claim, or church.
type Attachment struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	UploadedByID   uint           `gorm:"not null;index" json:"uploaded_by_id"`
	UploadedBy     *User          `gorm:"foreignKey:UploadedByID" json:"uploaded_by,omitempty"`
	AttachableType AttachableType `gorm:"not null;index" json:"attachable_type"`
	AttachableID   uint           `gorm:"not null;index" json:"attachable_id"`
	Filename       string         `gorm:"not null" json:"filename"`
	StorageKey     string         `gorm:"not null" json:"storage_key"`
	ContentType    string         `gorm:"not null" json:"content_type"`
	SizeBytes      int64          `gorm:"not null" json:"size_bytes"`
	URL            string         `gorm:"-" json:"url"`
	CreatedAt      time.Time      `json:"created_at"`
}
