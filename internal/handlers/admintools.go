package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

type AdminToolsHandler struct {
	DB *gorm.DB
}

// ─── Audit Log ───────────────────────────────────────────────────────────────

// LogAction records an administrative action in the audit trail.
func (h *AdminToolsHandler) LogAction(actorID uint, action models.AuditAction, entityType string, entityID uint, details string) {
	h.DB.Create(&models.AuditLog{
		ActorID:    actorID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		Details:    details,
	})
}

// GetAuditLog godoc
// @Summary      View audit trail
// @Description  Returns all administrative actions with actor, action type, and timestamps.
//
//	Filterable by action type, entity type, and actor.
//
// @Tags         admin-tools
// @Security     BearerAuth
// @Param        action       query  string  false  "Filter by action (e.g. church.verified)"
// @Param        entity_type  query  string  false  "Filter by entity type (church, user, suggestion)"
// @Param        actor_id     query  int     false  "Filter by actor user ID"
// @Success      200  {array}  models.AuditLog
// @Router       /admin/audit-log [get]
func (h *AdminToolsHandler) GetAuditLog(c *gin.Context) {
	ctx := c.Request.Context()
	limit := parseLimit(c.Query("limit"))
	offset := parseOffset(c.Query("offset"))

	query := h.DB.WithContext(ctx).Preload("Actor").Order("created_at DESC")

	if action := c.Query("action"); action != "" {
		query = query.Where("action = ?", action)
	}
	if entityType := c.Query("entity_type"); entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}
	if actorID := c.Query("actor_id"); actorID != "" {
		query = query.Where("actor_id = ?", actorID)
	}

	var logs []models.AuditLog
	if err := query.Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit log"})
		return
	}
	c.JSON(http.StatusOK, logs)
}

// ─── Bulk Operations ─────────────────────────────────────────────────────────

// BulkVerifyChurches godoc
// @Summary      Bulk verify multiple churches
// @Description  Verify multiple churches at once. Logs each action in the audit trail.
// @Tags         admin-tools
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  object  true  "Church IDs to verify"
// @Success      200   {object}  map[string]any
// @Router       /admin/bulk/verify-churches [post]
func (h *AdminToolsHandler) BulkVerifyChurches(c *gin.Context) {
	actorID := c.GetUint("userID")
	ctx := c.Request.Context()

	var input struct {
		ChurchIDs []uint `json:"church_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(input.ChurchIDs) == 0 || len(input.ChurchIDs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide 1-100 church IDs"})
		return
	}

	now := time.Now()
	result := h.DB.WithContext(ctx).Model(&models.Church{}).
		Where("id IN ? AND verified = ?", input.ChurchIDs, false).
		Updates(map[string]any{"verified": true, "last_verified": now})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "bulk verify failed"})
		return
	}

	details, _ := json.Marshal(input.ChurchIDs)
	h.LogAction(actorID, models.AuditBulkVerify, "church", 0, string(details))

	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("verified %d churches", result.RowsAffected),
		"verified": result.RowsAffected,
	})
}

// BulkReviewSuggestions godoc
// @Summary      Bulk approve or reject suggestions
// @Description  Review multiple suggestions at once with the same status and note.
// @Tags         admin-tools
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Router       /admin/bulk/review-suggestions [post]
func (h *AdminToolsHandler) BulkReviewSuggestions(c *gin.Context) {
	actorID := c.GetUint("userID")
	ctx := c.Request.Context()

	var input struct {
		SuggestionIDs []uint                  `json:"suggestion_ids" binding:"required"`
		Status        models.SuggestionStatus `json:"status" binding:"required"`
		ReviewNote    string                  `json:"review_note" binding:"max=2000"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Status != models.SuggestionApproved && input.Status != models.SuggestionRejected {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'approved' or 'rejected'"})
		return
	}

	if len(input.SuggestionIDs) == 0 || len(input.SuggestionIDs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide 1-100 suggestion IDs"})
		return
	}

	result := h.DB.WithContext(ctx).Model(&models.Suggestion{}).
		Where("id IN ? AND status = ?", input.SuggestionIDs, models.SuggestionPending).
		Updates(map[string]any{
			"status":         input.Status,
			"reviewed_by_id": actorID,
			"review_note":    input.ReviewNote,
		})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "bulk review failed"})
		return
	}

	action := models.AuditBulkReject
	if input.Status == models.SuggestionApproved {
		action = models.AuditSuggApproved
	}
	details, _ := json.Marshal(map[string]any{"ids": input.SuggestionIDs, "status": input.Status})
	h.LogAction(actorID, action, "suggestion", 0, string(details))

	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("reviewed %d suggestions as %s", result.RowsAffected, input.Status),
		"reviewed": result.RowsAffected,
	})
}

// ─── Data Quality ────────────────────────────────────────────────────────────

// DataHealthDashboard godoc
// @Summary      Data quality health dashboard
// @Description  Returns counts and metrics about data quality: stale churches,
//
//	unverified count, pending suggestions, pending claims, etc.
//
// @Tags         admin-tools
// @Security     BearerAuth
// @Success      200  {object}  map[string]any
// @Router       /admin/data-health [get]
func (h *AdminToolsHandler) DataHealthDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	var totalChurches, verifiedChurches, unverifiedChurches int64
	h.DB.WithContext(ctx).Model(&models.Church{}).Count(&totalChurches)
	h.DB.WithContext(ctx).Model(&models.Church{}).Where("verified = ?", true).Count(&verifiedChurches)
	unverifiedChurches = totalChurches - verifiedChurches

	// Churches not verified in the last 90 days
	staleDate := time.Now().AddDate(0, 0, -90)
	var staleChurches int64
	h.DB.WithContext(ctx).Model(&models.Church{}).
		Where("verified = ? AND (last_verified IS NULL OR last_verified < ?)", true, staleDate).
		Count(&staleChurches)

	// Churches with no schedules
	var noScheduleChurches int64
	h.DB.WithContext(ctx).Model(&models.Church{}).
		Where("id NOT IN (?)",
			h.DB.Model(&models.MassSchedule{}).Select("DISTINCT church_id"),
		).Count(&noScheduleChurches)

	var pendingSuggestions, pendingClaims int64
	h.DB.WithContext(ctx).Model(&models.Suggestion{}).Where("status = ?", models.SuggestionPending).Count(&pendingSuggestions)
	h.DB.WithContext(ctx).Model(&models.ChurchOwnership{}).Where("status = ?", models.ClaimPending).Count(&pendingClaims)

	var totalUsers, totalCheckins int64
	h.DB.WithContext(ctx).Model(&models.User{}).Count(&totalUsers)
	h.DB.WithContext(ctx).Model(&models.CheckIn{}).Count(&totalCheckins)

	// Recent activity (last 7 days)
	weekAgo := time.Now().AddDate(0, 0, -7)
	var recentCheckins, recentSuggestions, newUsers int64
	h.DB.WithContext(ctx).Model(&models.CheckIn{}).Where("created_at > ?", weekAgo).Count(&recentCheckins)
	h.DB.WithContext(ctx).Model(&models.Suggestion{}).Where("created_at > ?", weekAgo).Count(&recentSuggestions)
	h.DB.WithContext(ctx).Model(&models.User{}).Where("created_at > ?", weekAgo).Count(&newUsers)

	c.JSON(http.StatusOK, gin.H{
		"churches": gin.H{
			"total":       totalChurches,
			"verified":    verifiedChurches,
			"unverified":  unverifiedChurches,
			"stale":       staleChurches,
			"no_schedule": noScheduleChurches,
		},
		"moderation": gin.H{
			"pending_suggestions": pendingSuggestions,
			"pending_claims":      pendingClaims,
		},
		"community": gin.H{
			"total_users":    totalUsers,
			"total_checkins": totalCheckins,
		},
		"activity_7d": gin.H{
			"checkins":    recentCheckins,
			"suggestions": recentSuggestions,
			"new_users":   newUsers,
		},
	})
}

// FindDuplicateChurches godoc
// @Summary      Find potential duplicate churches
// @Description  Returns pairs of churches that are within 100 meters of each other,
//
//	which likely represent duplicates that should be merged.
//
// @Tags         admin-tools
// @Security     BearerAuth
// @Success      200  {array}  object
// @Router       /admin/duplicates [get]
func (h *AdminToolsHandler) FindDuplicateChurches(c *gin.Context) {
	ctx := c.Request.Context()

	type DuplicatePair struct {
		Church1ID   uint    `json:"church1_id"`
		Church1Name string  `json:"church1_name"`
		Church2ID   uint    `json:"church2_id"`
		Church2Name string  `json:"church2_name"`
		DistanceM   float64 `json:"distance_meters"`
	}

	var pairs []DuplicatePair
	// Find churches within ~100 meters of each other (0.001 degrees ~ 111 meters)
	err := h.DB.WithContext(ctx).Raw(`
		SELECT c1.id as church1_id, c1.name as church1_name,
		       c2.id as church2_id, c2.name as church2_name,
		       (6371000 * acos(
		           cos(radians(c1.latitude)) * cos(radians(c2.latitude)) *
		           cos(radians(c2.longitude) - radians(c1.longitude)) +
		           sin(radians(c1.latitude)) * sin(radians(c2.latitude))
		       )) as distance_m
		FROM churches c1
		JOIN churches c2 ON c1.id < c2.id
		WHERE abs(c1.latitude - c2.latitude) < 0.002
		  AND abs(c1.longitude - c2.longitude) < 0.002
		HAVING distance_m < 100
		ORDER BY distance_m ASC
		LIMIT 50
	`).Scan(&pairs).Error

	if err != nil {
		// Fallback for SQLite (no trig functions) — use simple coordinate proximity
		err = h.DB.WithContext(ctx).Raw(`
			SELECT c1.id as church1_id, c1.name as church1_name,
			       c2.id as church2_id, c2.name as church2_name,
			       0 as distance_m
			FROM churches c1
			JOIN churches c2 ON c1.id < c2.id
			WHERE abs(c1.latitude - c2.latitude) < 0.001
			  AND abs(c1.longitude - c2.longitude) < 0.001
			LIMIT 50
		`).Scan(&pairs).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find duplicates"})
			return
		}
	}

	c.JSON(http.StatusOK, pairs)
}

// MergeChurches godoc
// @Summary      Merge two duplicate churches
// @Description  Merges church B into church A: moves all check-ins, schedules, suggestions,
//
//	favorites, and ownership claims from B to A, then deletes B.
//
// @Tags         admin-tools
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Router       /admin/merge-churches [post]
func (h *AdminToolsHandler) MergeChurches(c *gin.Context) {
	actorID := c.GetUint("userID")
	ctx := c.Request.Context()

	var input struct {
		KeepID   uint `json:"keep_id" binding:"required"`
		RemoveID uint `json:"remove_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.KeepID == input.RemoveID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot merge a church with itself"})
		return
	}

	// Verify both exist
	var keepChurch, removeChurch models.Church
	if err := h.DB.WithContext(ctx).First(&keepChurch, input.KeepID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "keep_id church not found"})
		return
	}
	if err := h.DB.WithContext(ctx).First(&removeChurch, input.RemoveID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "remove_id church not found"})
		return
	}

	// Move all related records from removeID to keepID
	tx := h.DB.WithContext(ctx).Begin()

	tx.Model(&models.CheckIn{}).Where("church_id = ?", input.RemoveID).Update("church_id", input.KeepID)
	tx.Model(&models.MassSchedule{}).Where("church_id = ?", input.RemoveID).Update("church_id", input.KeepID)
	tx.Model(&models.Suggestion{}).Where("church_id = ?", input.RemoveID).Update("church_id", input.KeepID)
	tx.Model(&models.Favorite{}).Where("church_id = ?", input.RemoveID).Update("church_id", input.KeepID)
	tx.Model(&models.ChurchOwnership{}).Where("church_id = ?", input.RemoveID).Update("church_id", input.KeepID)
	tx.Delete(&models.Church{}, input.RemoveID)

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "merge failed"})
		return
	}

	details, _ := json.Marshal(map[string]any{
		"keep": map[string]any{"id": keepChurch.ID, "name": keepChurch.Name},
		"removed": map[string]any{"id": removeChurch.ID, "name": removeChurch.Name},
	})
	h.LogAction(actorID, models.AuditChurchMerged, "church", input.KeepID, string(details))

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("merged '%s' into '%s'", removeChurch.Name, keepChurch.Name),
		"kept":    keepChurch,
	})
}

// StaleChurches godoc
// @Summary      List churches with stale data
// @Description  Returns churches that haven't been verified in over 90 days,
//
//	ordered by staleness. Helps prioritize data maintenance.
//
// @Tags         admin-tools
// @Security     BearerAuth
// @Success      200  {array}  models.Church
// @Router       /admin/stale-churches [get]
func (h *AdminToolsHandler) StaleChurches(c *gin.Context) {
	ctx := c.Request.Context()
	limit := parseLimit(c.Query("limit"))
	offset := parseOffset(c.Query("offset"))

	staleDate := time.Now().AddDate(0, 0, -90)
	var churches []models.Church
	if err := h.DB.WithContext(ctx).
		Where("last_verified IS NULL OR last_verified < ?", staleDate).
		Order("last_verified ASC NULLS FIRST").
		Limit(limit).Offset(offset).
		Find(&churches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch stale churches"})
		return
	}
	c.JSON(http.StatusOK, churches)
}
