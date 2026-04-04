package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

type CommunityHandler struct {
	DB *gorm.DB
}

// ConfirmChurch godoc
// @Summary      Confirm a church's data is accurate
// @Description  Submit an independent confirmation that a church listing is correct.
//
//	When a church accumulates 3+ confirmations, its data quality graduates
//	from "unverified" to "community_confirmed" (iNaturalist Research Grade model).
//
// @Tags         community
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "Church ID"
// @Success      201  {object}  map[string]any
// @Failure      409  {object}  map[string]string  "Already confirmed"
// @Router       /churches/{id}/confirm [post]
func (h *CommunityHandler) ConfirmChurch(c *gin.Context) {
	userID := c.GetUint("userID")
	churchID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid church id"})
		return
	}

	ctx := c.Request.Context()

	var church models.Church
	if err := h.DB.WithContext(ctx).First(&church, churchID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "church not found"})
		return
	}

	// Check for existing confirmation by this user
	var existing models.ChurchConfirmation
	if err := h.DB.WithContext(ctx).Where("user_id = ? AND church_id = ?", userID, churchID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "you have already confirmed this church"})
		return
	}

	var input struct {
		Comment string `json:"comment" binding:"max=500"`
	}
	_ = c.ShouldBindJSON(&input) // optional

	confirmation := models.ChurchConfirmation{
		UserID:   userID,
		ChurchID: uint(churchID),
		Comment:  input.Comment,
	}
	if err := h.DB.WithContext(ctx).Create(&confirmation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit confirmation"})
		return
	}

	// Count total confirmations and potentially upgrade quality
	var confirmCount int64
	h.DB.WithContext(ctx).Model(&models.ChurchConfirmation{}).Where("church_id = ?", churchID).Count(&confirmCount)

	updates := map[string]any{"confirmations": confirmCount}
	if confirmCount >= int64(models.ConfirmationsNeeded) && church.DataQuality == models.QualityUnverified {
		updates["data_quality"] = models.QualityCommunityConfirmed
	}
	h.DB.WithContext(ctx).Model(&church).Updates(updates)

	// Update reputation
	_, _ = RecalculateUserReputation(h.DB, userID)

	c.JSON(http.StatusCreated, gin.H{
		"message":        "Thank you for confirming this church!",
		"confirmations":  confirmCount,
		"data_quality":   church.DataQuality,
		"quality_changed": confirmCount == int64(models.ConfirmationsNeeded),
	})
}

// ContributionImpact godoc
// @Summary      View your contribution impact
// @Description  Shows how many people your contributions have helped find churches.
//
//	"Your corrections have helped X people find Mass" — the most powerful
//	contributor retention mechanic (Waze model).
//
// @Tags         community
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]any
// @Router       /impact [get]
func (h *CommunityHandler) ContributionImpact(c *gin.Context) {
	userID := c.GetUint("userID")
	ctx := c.Request.Context()

	// Count approved suggestions
	var approvedSuggestions int64
	h.DB.WithContext(ctx).Model(&models.Suggestion{}).
		Where("user_id = ? AND status = ?", userID, models.SuggestionApproved).
		Count(&approvedSuggestions)

	// Count churches the user contributed to (created or has approved suggestions for)
	var churchesHelped int64
	h.DB.WithContext(ctx).Model(&models.Suggestion{}).
		Where("user_id = ? AND status = ? AND church_id IS NOT NULL", userID, models.SuggestionApproved).
		Distinct("church_id").
		Count(&churchesHelped)

	// Count how many check-ins happened at churches the user helped
	var peopleHelped int64
	h.DB.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT ci.user_id) FROM check_ins ci
		WHERE ci.church_id IN (
			SELECT DISTINCT s.church_id FROM suggestions s
			WHERE s.user_id = ? AND s.status = 'approved' AND s.church_id IS NOT NULL
			UNION
			SELECT id FROM churches WHERE created_by_id = ?
		) AND ci.user_id != ?
	`, userID, userID, userID).Scan(&peopleHelped)

	// Count confirmations made
	var confirmationsMade int64
	h.DB.WithContext(ctx).Model(&models.ChurchConfirmation{}).
		Where("user_id = ?", userID).
		Count(&confirmationsMade)

	// Count churches the user created
	var churchesCreated int64
	h.DB.WithContext(ctx).Model(&models.Church{}).
		Where("created_by_id = ?", userID).
		Count(&churchesCreated)

	c.JSON(http.StatusOK, gin.H{
		"approved_suggestions": approvedSuggestions,
		"churches_helped":      churchesHelped,
		"churches_created":     churchesCreated,
		"confirmations_made":   confirmationsMade,
		"people_helped":        peopleHelped,
		"impact_message":       impactMessage(peopleHelped, approvedSuggestions),
	})
}

func impactMessage(people, suggestions int64) string {
	if people == 0 && suggestions == 0 {
		return "Start contributing to help others find churches near them!"
	}
	if people > 0 {
		return "Your contributions have helped " + strconv.FormatInt(people, 10) + " people find Mass. Thank you!"
	}
	return "Your " + strconv.FormatInt(suggestions, 10) + " approved suggestions are making church data better for everyone."
}
