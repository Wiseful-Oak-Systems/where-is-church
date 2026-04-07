package models

import "time"

// AuditAction categorizes what happened in an audit log entry.
type AuditAction string

const (
	AuditChurchCreated   AuditAction = "church.created"
	AuditChurchUpdated   AuditAction = "church.updated"
	AuditChurchVerified  AuditAction = "church.verified"
	AuditChurchDeleted   AuditAction = "church.deleted"
	AuditChurchMerged    AuditAction = "church.merged"
	AuditScheduleAdded   AuditAction = "schedule.added"
	AuditScheduleDeleted AuditAction = "schedule.deleted"
	AuditRoleChanged     AuditAction = "user.role_changed"
	AuditClaimApproved   AuditAction = "claim.approved"
	AuditClaimRejected   AuditAction = "claim.rejected"
	AuditSuggApproved    AuditAction = "suggestion.approved"
	AuditSuggRejected    AuditAction = "suggestion.rejected"
	AuditBulkVerify      AuditAction = "bulk.verify"
	AuditBulkReject      AuditAction = "bulk.reject"
)

// AuditLog tracks every administrative action for accountability and debugging.
type AuditLog struct {
	ID         uint        `gorm:"primaryKey" json:"id"`
	ActorID    uint        `gorm:"not null;index" json:"actor_id"`
	Actor      *User       `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
	Action     AuditAction `gorm:"not null;index" json:"action"`
	EntityType string      `gorm:"not null;index" json:"entity_type"` // "church", "user", "suggestion", "claim"
	EntityID   uint        `gorm:"not null;index" json:"entity_id"`
	Details    string      `gorm:"type:text" json:"details,omitempty"` // JSON-encoded change details
	CreatedAt  time.Time   `gorm:"index" json:"created_at"`
}
