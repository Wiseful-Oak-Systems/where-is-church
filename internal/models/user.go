package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Role defines user permission levels in ascending order of privilege.
//
// Hierarchy: User < Moderator < ChurchOwner < CommunityManager < Admin
//
//   - User: search, check-in, suggest, favorite
//   - Moderator: + edit any church, manage schedules, review suggestions
//   - ChurchOwner: + direct management of owned/claimed churches
//   - CommunityManager: + oversee moderators, approve ownership claims, community health
//   - Admin: + manage all users, delete churches, full system control
type Role string

const (
	RoleUser             Role = "user"
	RoleModerator        Role = "moderator"
	RoleChurchOwner      Role = "church_owner"
	RoleCommunityManager Role = "community_manager"
	RoleAdmin            Role = "admin"
)

// ValidRoles is the set of all assignable roles.
var ValidRoles = map[Role]bool{
	RoleUser:             true,
	RoleModerator:        true,
	RoleChurchOwner:      true,
	RoleCommunityManager: true,
	RoleAdmin:            true,
}

// MaxBcryptPasswordLen is the maximum password length bcrypt supports without silent truncation.
const MaxBcryptPasswordLen = 72

// MinPasswordLen is the minimum password length enforced at registration.
const MinPasswordLen = 8

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	Password     string    `gorm:"column:password_hash;not null" json:"-"`
	Name         string    `gorm:"not null" json:"name"`
	Denomination string    `gorm:"not null;default:'Catholic'" json:"denomination"`
	Role         Role      `gorm:"not null;default:'user'" json:"role"`
	Latitude     *float64  `json:"latitude,omitempty"`
	Longitude    *float64  `json:"longitude,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) SetPassword(password string) error {
	if len([]byte(password)) > MaxBcryptPasswordLen {
		password = string([]byte(password)[:MaxBcryptPasswordLen])
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	if len([]byte(password)) > MaxBcryptPasswordLen {
		password = string([]byte(password)[:MaxBcryptPasswordLen])
	}
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password)) == nil
}

// HasAtLeastRole checks if the user has at least the given role level.
func HasAtLeastRole(userRole Role, required Role) bool {
	levels := map[Role]int{
		RoleUser:             0,
		RoleModerator:        1,
		RoleChurchOwner:      2,
		RoleCommunityManager: 3,
		RoleAdmin:            4,
	}
	return levels[userRole] >= levels[required]
}

// Favorite represents a user's bookmarked church.
type Favorite struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;uniqueIndex:idx_user_church" json:"user_id"`
	ChurchID  uint      `gorm:"not null;uniqueIndex:idx_user_church" json:"church_id"`
	Church    *Church   `gorm:"foreignKey:ChurchID" json:"church,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// ClaimStatus tracks the lifecycle of a church ownership claim.
type ClaimStatus string

const (
	ClaimPending  ClaimStatus = "pending"
	ClaimApproved ClaimStatus = "approved"
	ClaimRejected ClaimStatus = "rejected"
)

// ChurchOwnership represents a claim of ownership over a church location.
// Church owners/administrators can claim their church and, once approved,
// manage its details directly without going through the suggestion workflow.
type ChurchOwnership struct {
	ID           uint        `gorm:"primaryKey" json:"id"`
	UserID       uint        `gorm:"not null;index" json:"user_id"`
	User         *User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ChurchID     uint        `gorm:"not null;index" json:"church_id"`
	Church       *Church     `gorm:"foreignKey:ChurchID" json:"church,omitempty"`
	Status       ClaimStatus `gorm:"not null;default:'pending'" json:"status"`
	Evidence     string      `gorm:"type:text" json:"evidence"`
	ReviewedByID *uint       `json:"reviewed_by_id,omitempty"`
	ReviewedBy   *User       `gorm:"foreignKey:ReviewedByID" json:"reviewed_by,omitempty"`
	ReviewNote   string      `json:"review_note,omitempty"`
	ClaimedAt    time.Time   `gorm:"autoCreateTime" json:"claimed_at"`
	ReviewedAt   *time.Time  `json:"reviewed_at,omitempty"`
}
