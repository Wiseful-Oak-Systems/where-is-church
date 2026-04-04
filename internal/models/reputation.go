package models

import "time"

// UserReputation tracks contribution metrics that feed into trust scoring.
// The reputation score determines eligibility for moderator promotion and
// auto-approval of contributions.
type UserReputation struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	User             *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CheckInCount     int       `gorm:"default:0" json:"checkin_count"`
	SuggestionCount  int       `gorm:"default:0" json:"suggestion_count"`
	ApprovedCount    int       `gorm:"default:0" json:"approved_count"`    // suggestions that were approved
	RejectedCount    int       `gorm:"default:0" json:"rejected_count"`    // suggestions that were rejected
	ReportAccuracy   float64   `gorm:"default:0" json:"report_accuracy"`   // approved / (approved + rejected)
	ConsecutiveDays  int       `gorm:"default:0" json:"consecutive_days"`  // streak of daily check-ins
	TrustScore       int       `gorm:"default:0;index" json:"trust_score"` // computed score (0-1000)
	Level            int       `gorm:"default:1" json:"level"`             // 1=New, 2=Active, 3=Trusted, 4=Expert, 5=Guardian
	LastCheckInDate  *time.Time `json:"last_checkin_date,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// TrustLevel names for display.
var TrustLevelNames = map[int]string{
	1: "New Member",
	2: "Active Contributor",
	3: "Trusted Member",
	4: "Expert",
	5: "Guardian",
}

// TrustThresholds define the trust score needed for each level.
var TrustThresholds = map[int]int{
	1: 0,
	2: 50,
	3: 200,
	4: 500,
	5: 1000,
}
