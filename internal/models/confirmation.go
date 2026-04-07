package models

import "time"

// ChurchConfirmation records a user's independent confirmation that a church's
// data (location, schedules, existence) is accurate. When a church accumulates
// ConfirmationsNeeded (3) independent confirmations, its DataQuality graduates
// from "unverified" to "community_confirmed".
//
// This implements the iNaturalist "Research Grade" pattern for crowdsourced
// data quality, adapted for church listings.
type ChurchConfirmation struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_confirm_user_church" json:"user_id"`
	ChurchID  uint      `gorm:"not null;uniqueIndex:idx_confirm_user_church" json:"church_id"`
	Church    *Church   `gorm:"foreignKey:ChurchID" json:"church,omitempty"`
	Comment   string    `json:"comment,omitempty"` // e.g. "Mass times confirmed accurate as of today"
	CreatedAt time.Time `json:"created_at"`
}
