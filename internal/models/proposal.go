package models

// ChurchProposal holds structured data for a new church suggestion.
// When a suggestion of type "new_church" is approved, this data is used
// to auto-create the church — no manual re-entry by moderators needed.
type ChurchProposal struct {
	ID           uint    `gorm:"primaryKey" json:"id"`
	SuggestionID uint    `gorm:"uniqueIndex;not null" json:"suggestion_id"`
	Name         string  `gorm:"not null" json:"name"`
	Denomination string  `gorm:"not null;default:'Catholic'" json:"denomination"`
	Address      string  `gorm:"not null" json:"address"`
	Latitude     float64 `gorm:"not null" json:"latitude"`
	Longitude    float64 `gorm:"not null" json:"longitude"`
	Phone        string  `json:"phone,omitempty"`
	Website      string  `json:"website,omitempty"`
	Description  string  `json:"description,omitempty"`
}
