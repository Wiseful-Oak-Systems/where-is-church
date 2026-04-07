package models

import "time"

// Tag represents a characteristic or feature of a church.
// Tags are user-suggestible and admin-approved, enabling rich filtering
// (e.g., "children-friendly", "Latin Mass", "wheelchair accessible").
type Tag struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Key      string `gorm:"uniqueIndex;not null" json:"key"`       // snake_case identifier
	LabelEN  string `gorm:"not null" json:"label_en"`              // English display name
	LabelPT  string `gorm:"not null" json:"label_pt"`              // Portuguese display name
	Category string `gorm:"not null;index" json:"category"`        // grouping category
	Icon     string `json:"icon,omitempty"`                         // optional emoji/icon
	Approved bool   `gorm:"not null;default:true" json:"approved"`  // admin-approved for public use
}

// ChurchTag is the many-to-many join between churches and tags.
type ChurchTag struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ChurchID   uint      `gorm:"not null;uniqueIndex:idx_church_tag" json:"church_id"`
	TagID      uint      `gorm:"not null;uniqueIndex:idx_church_tag" json:"tag_id"`
	Tag        *Tag      `gorm:"foreignKey:TagID" json:"tag,omitempty"`
	AddedByID  uint      `gorm:"not null" json:"added_by_id"`
	Confirmed  int       `gorm:"default:1" json:"confirmed"` // number of users who confirmed this tag
	CreatedAt  time.Time `json:"created_at"`
}
