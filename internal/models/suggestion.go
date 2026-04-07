package models

import "time"

type SuggestionStatus string

const (
	SuggestionPending  SuggestionStatus = "pending"
	SuggestionApproved SuggestionStatus = "approved"
	SuggestionRejected SuggestionStatus = "rejected"
)

type SuggestionType string

const (
	SuggestNewChurch    SuggestionType = "new_church"
	SuggestEditChurch   SuggestionType = "edit_church"
	SuggestSchedule     SuggestionType = "schedule"
	SuggestGeneral      SuggestionType = "general"
)

type Suggestion struct {
	ID           uint             `gorm:"primaryKey" json:"id"`
	UserID       uint             `gorm:"not null;index" json:"user_id"`
	User         *User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ChurchID     *uint            `gorm:"index" json:"church_id,omitempty"`
	Church       *Church          `gorm:"foreignKey:ChurchID" json:"church,omitempty"`
	Type         SuggestionType   `gorm:"not null" json:"type"`
	Content      string           `gorm:"not null;type:text" json:"content"`
	Status       SuggestionStatus `gorm:"not null;default:'pending'" json:"status"`
	ReviewedByID *uint            `json:"reviewed_by_id,omitempty"`
	ReviewedBy   *User            `gorm:"foreignKey:ReviewedByID" json:"reviewed_by,omitempty"`
	ReviewNote   string           `json:"review_note,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}
