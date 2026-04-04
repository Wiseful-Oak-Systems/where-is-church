package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

// TrustScoreWeights define how much each metric contributes to the score.
const (
	WeightCheckIn          = 2  // per check-in
	WeightApprovedSugg     = 10 // per approved suggestion
	WeightRejectedSugg     = -5 // per rejected suggestion
	WeightConsecutiveDay   = 3  // per day in streak
	WeightAccountAgeDays   = 1  // per day since registration (capped at 365)
	MaxAccountAgeBonus     = 365
	MaxRejectionStreak     = 3                // consecutive rejections before suspension
	SuspensionDuration     = 48 * time.Hour
	NewUserSuggestionLimit = 3                // max suggestions per 24h for trust < 100
	AutoApproveMinorEdit   = 500              // trust score for auto-approving minor edits
	AutoApproveSchedule    = 750              // trust score for auto-approving schedule changes
)

type ReputationHandler struct {
	DB *gorm.DB
}

// RecalculateUserReputation computes a user's trust score and level from their activity.
func RecalculateUserReputation(db *gorm.DB, userID uint) (*models.UserReputation, error) {
	var rep models.UserReputation
	db.Where("user_id = ?", userID).FirstOrCreate(&rep, models.UserReputation{UserID: userID})

	// Count check-ins
	var checkinCount int64
	db.Model(&models.CheckIn{}).Where("user_id = ?", userID).Count(&checkinCount)
	rep.CheckInCount = int(checkinCount)

	// Count suggestions
	var suggCount int64
	db.Model(&models.Suggestion{}).Where("user_id = ?", userID).Count(&suggCount)
	rep.SuggestionCount = int(suggCount)

	var approvedCount, rejectedCount int64
	db.Model(&models.Suggestion{}).Where("user_id = ? AND status = ?", userID, models.SuggestionApproved).Count(&approvedCount)
	db.Model(&models.Suggestion{}).Where("user_id = ? AND status = ?", userID, models.SuggestionRejected).Count(&rejectedCount)
	rep.ApprovedCount = int(approvedCount)
	rep.RejectedCount = int(rejectedCount)

	if approvedCount+rejectedCount > 0 {
		rep.ReportAccuracy = float64(approvedCount) / float64(approvedCount+rejectedCount)
	}

	// Calculate consecutive check-in days
	var lastCheckin models.CheckIn
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").First(&lastCheckin).Error; err == nil {
		rep.LastCheckInDate = &lastCheckin.CreatedAt

		// Count consecutive days (simplified: check if last 7 days have check-ins)
		streak := 0
		for i := 0; i < 365; i++ {
			day := time.Now().AddDate(0, 0, -i)
			dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
			dayEnd := dayStart.Add(24 * time.Hour)
			var count int64
			db.Model(&models.CheckIn{}).Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, dayStart, dayEnd).Count(&count)
			if count > 0 {
				streak++
			} else {
				break
			}
		}
		rep.ConsecutiveDays = streak
	}

	// Compute trust score
	var user models.User
	db.First(&user, userID)
	accountAge := int(time.Since(user.CreatedAt).Hours() / 24)
	if accountAge > MaxAccountAgeBonus {
		accountAge = MaxAccountAgeBonus
	}

	rep.TrustScore = rep.CheckInCount*WeightCheckIn +
		rep.ApprovedCount*WeightApprovedSugg +
		rep.RejectedCount*WeightRejectedSugg +
		rep.ConsecutiveDays*WeightConsecutiveDay +
		accountAge*WeightAccountAgeDays

	if rep.TrustScore < 0 {
		rep.TrustScore = 0
	}

	// Determine level
	rep.Level = 1
	for level := 5; level >= 1; level-- {
		if rep.TrustScore >= models.TrustThresholds[level] {
			rep.Level = level
			break
		}
	}

	db.Save(&rep)

	return &rep, nil
}

// GetMyReputation godoc
// @Summary      Get my reputation and trust score
// @Description  Returns the current user's reputation metrics, trust score, and level.
// @Tags         reputation
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  models.UserReputation
// @Router       /reputation [get]
func (h *ReputationHandler) GetMyReputation(c *gin.Context) {
	userID := c.GetUint("userID")
	rep, err := RecalculateUserReputation(h.DB, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute reputation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trust_score":     rep.TrustScore,
		"level":           rep.Level,
		"level_name":      models.TrustLevelNames[rep.Level],
		"checkin_count":   rep.CheckInCount,
		"suggestion_count": rep.SuggestionCount,
		"approved_count":  rep.ApprovedCount,
		"rejected_count":  rep.RejectedCount,
		"report_accuracy": rep.ReportAccuracy,
		"consecutive_days": rep.ConsecutiveDays,
		"next_level":      nextLevelInfo(rep),
	})
}

func nextLevelInfo(rep *models.UserReputation) map[string]any {
	nextLevel := rep.Level + 1
	if nextLevel > 5 {
		return map[string]any{"level": 5, "name": "Max level reached", "points_needed": 0}
	}
	needed := models.TrustThresholds[nextLevel] - rep.TrustScore
	if needed < 0 {
		needed = 0
	}
	return map[string]any{
		"level":         nextLevel,
		"name":          models.TrustLevelNames[nextLevel],
		"points_needed": needed,
	}
}

// CheckSubmissionAllowed verifies a user hasn't exceeded rate limits or been suspended.
func CheckSubmissionAllowed(db *gorm.DB, userID uint) (bool, string) {
	var rep models.UserReputation
	db.Where("user_id = ?", userID).First(&rep)

	// Check rejection streak suspension
	if rep.RejectedCount > 0 {
		var recentRejections int64
		db.Model(&models.Suggestion{}).
			Where("user_id = ? AND status = ? AND updated_at > ?", userID, models.SuggestionRejected, time.Now().Add(-SuspensionDuration)).
			Count(&recentRejections)

		if recentRejections >= int64(MaxRejectionStreak) {
			return false, "Your suggestions have been rejected multiple times recently. Please wait 48 hours before submitting again."
		}
	}

	// Check daily rate limit for low-trust users
	if rep.TrustScore < 100 {
		var todaySuggestions int64
		dayStart := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)
		db.Model(&models.Suggestion{}).
			Where("user_id = ? AND created_at >= ?", userID, dayStart).
			Count(&todaySuggestions)

		if todaySuggestions >= int64(NewUserSuggestionLimit) {
			return false, "New users can submit up to 3 suggestions per day. Keep contributing to increase your limit!"
		}
	}

	return true, ""
}

// ShouldAutoApprove checks if a suggestion qualifies for auto-approval based on user trust.
func ShouldAutoApprove(db *gorm.DB, userID uint, suggType models.SuggestionType) bool {
	var rep models.UserReputation
	if err := db.Where("user_id = ?", userID).First(&rep).Error; err != nil {
		return false
	}

	switch suggType {
	case models.SuggestEditChurch:
		return rep.TrustScore >= AutoApproveMinorEdit
	case models.SuggestSchedule:
		return rep.TrustScore >= AutoApproveSchedule
	default:
		return false // new_church and general always need review
	}
}

// Leaderboard godoc
// @Summary      Community leaderboard
// @Description  Returns the top contributors by trust score.
// @Tags         reputation
// @Produce      json
// @Security     BearerAuth
// @Success      200  {array}  object
// @Router       /leaderboard [get]
func (h *ReputationHandler) Leaderboard(c *gin.Context) {
	ctx := c.Request.Context()
	limit := parseLimit(c.Query("limit"))

	type LeaderEntry struct {
		UserID         uint   `json:"user_id"`
		UserName       string `json:"user_name"`
		TrustScore     int    `json:"trust_score"`
		Level          int    `json:"level"`
		LevelName      string `json:"level_name"`
		CheckInCount   int    `json:"checkin_count"`
		ApprovedCount  int    `json:"approved_count"`
	}

	var entries []LeaderEntry
	h.DB.WithContext(ctx).Model(&models.UserReputation{}).
		Select("user_reputations.user_id, users.name as user_name, user_reputations.trust_score, user_reputations.level, user_reputations.check_in_count as checkin_count, user_reputations.approved_count").
		Joins("JOIN users ON users.id = user_reputations.user_id").
		Where("user_reputations.trust_score > 0").
		Order("user_reputations.trust_score DESC").
		Limit(limit).
		Find(&entries)

	for i := range entries {
		entries[i].LevelName = models.TrustLevelNames[entries[i].Level]
	}

	c.JSON(http.StatusOK, entries)
}
