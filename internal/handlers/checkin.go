package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

const DuplicateCheckInWindow = 2 * time.Hour

type CheckInHandler struct {
	DB *gorm.DB
}

// Create godoc
// @Summary      Check in at a church
// @Description  Record attendance at a church (Foursquare-style). Duplicate check-ins within 2 hours are prevented.
// @Tags         checkins
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      object  true  "Check-in data (church_id, notes)"
// @Success      201   {object}  models.CheckIn
// @Failure      404   {object}  map[string]string  "Church not found"
// @Failure      409   {object}  map[string]string  "Already checked in recently"
// @Router       /checkins [post]
func (h *CheckInHandler) Create(c *gin.Context) {
	userID := c.GetUint("userID")

	var input struct {
		ChurchID uint   `json:"church_id" binding:"required"`
		Notes    string `json:"notes" binding:"max=500"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	var church models.Church
	if err := h.DB.WithContext(ctx).First(&church, input.ChurchID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "church not found"})
		return
	}

	var recent models.CheckIn
	windowStart := time.Now().Add(-DuplicateCheckInWindow)
	if err := h.DB.WithContext(ctx).Where("user_id = ? AND church_id = ? AND created_at > ?",
		userID, input.ChurchID, windowStart).First(&recent).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "you already checked in recently"})
		return
	}

	checkin := models.CheckIn{
		UserID:   userID,
		ChurchID: input.ChurchID,
		Notes:    input.Notes,
	}

	if err := h.DB.WithContext(ctx).Create(&checkin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check in"})
		return
	}

	if err := h.DB.WithContext(ctx).Preload("Church").Preload("User").First(&checkin, checkin.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "check-in created but failed to load details"})
		return
	}
	// Update reputation after check-in
	_, _ = RecalculateUserReputation(h.DB, userID)

	c.JSON(http.StatusCreated, checkin)
}

func (h *CheckInHandler) MyCheckIns(c *gin.Context) {
	userID := c.GetUint("userID")
	ctx := c.Request.Context()
	limit := parseLimit(c.Query("limit"))
	offset := parseOffset(c.Query("offset"))

	var checkins []models.CheckIn
	if err := h.DB.WithContext(ctx).Where("user_id = ?", userID).
		Preload("Church").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&checkins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch check-ins"})
		return
	}
	c.JSON(http.StatusOK, checkins)
}

func (h *CheckInHandler) ChurchCheckIns(c *gin.Context) {
	churchID := c.Param("id")
	ctx := c.Request.Context()
	limit := parseLimit(c.Query("limit"))
	offset := parseOffset(c.Query("offset"))

	var checkins []models.CheckIn
	if err := h.DB.WithContext(ctx).Where("church_id = ?", churchID).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&checkins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch check-ins"})
		return
	}
	c.JSON(http.StatusOK, checkins)
}

// UserStats godoc
// @Summary      Get user attendance statistics
// @Description  Returns total check-ins, last 30 days count, and top 5 most visited churches
// @Tags         checkins
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]any
// @Router       /checkins/stats [get]
func (h *CheckInHandler) UserStats(c *gin.Context) {
	userID := c.GetUint("userID")
	ctx := c.Request.Context()

	var totalCheckins int64
	if err := h.DB.WithContext(ctx).Model(&models.CheckIn{}).Where("user_id = ?", userID).Count(&totalCheckins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count check-ins"})
		return
	}

	type ChurchVisit struct {
		ChurchID   uint   `json:"church_id"`
		ChurchName string `json:"church_name"`
		Visits     int    `json:"visits"`
	}
	var topChurches []ChurchVisit
	if err := h.DB.WithContext(ctx).Model(&models.CheckIn{}).
		Select("check_ins.church_id, churches.name as church_name, count(*) as visits").
		Joins("JOIN churches ON churches.id = check_ins.church_id").
		Where("check_ins.user_id = ?", userID).
		Group("check_ins.church_id, churches.name").
		Order("visits DESC").
		Limit(5).
		Find(&topChurches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch top churches"})
		return
	}

	var recentCount int64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err := h.DB.WithContext(ctx).Model(&models.CheckIn{}).Where("user_id = ? AND created_at > ?", userID, thirtyDaysAgo).Count(&recentCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count recent check-ins"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_checkins": totalCheckins,
		"recent_30_days": recentCount,
		"top_churches":   topChurches,
	})
}

func (h *CheckInHandler) ChurchLoyalUsers(c *gin.Context) {
	churchID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid church id"})
		return
	}

	type LoyalUser struct {
		UserID   uint   `json:"user_id"`
		UserName string `json:"user_name"`
		Visits   int    `json:"visits"`
	}
	var loyalUsers []LoyalUser
	if err := h.DB.WithContext(c.Request.Context()).Model(&models.CheckIn{}).
		Select("check_ins.user_id, users.name as user_name, count(*) as visits").
		Joins("JOIN users ON users.id = check_ins.user_id").
		Where("check_ins.church_id = ?", churchID).
		Group("check_ins.user_id, users.name").
		Order("visits DESC").
		Limit(20).
		Find(&loyalUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch loyal users"})
		return
	}

	c.JSON(http.StatusOK, loyalUsers)
}
