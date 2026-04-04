package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

type CheckInHandler struct {
	DB *gorm.DB
}

func (h *CheckInHandler) Create(c *gin.Context) {
	userID := c.GetUint("userID")

	var input struct {
		ChurchID uint   `json:"church_id" binding:"required"`
		Notes    string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify church exists
	var church models.Church
	if err := h.DB.First(&church, input.ChurchID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "church not found"})
		return
	}

	// Prevent duplicate check-ins within 2 hours
	var recent models.CheckIn
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	if err := h.DB.Where("user_id = ? AND church_id = ? AND created_at > ?",
		userID, input.ChurchID, twoHoursAgo).First(&recent).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "you already checked in recently"})
		return
	}

	checkin := models.CheckIn{
		UserID:   userID,
		ChurchID: input.ChurchID,
		Notes:    input.Notes,
	}

	if err := h.DB.Create(&checkin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check in"})
		return
	}

	h.DB.Preload("Church").Preload("User").First(&checkin, checkin.ID)
	c.JSON(http.StatusCreated, checkin)
}

func (h *CheckInHandler) MyCheckIns(c *gin.Context) {
	userID := c.GetUint("userID")
	var checkins []models.CheckIn
	if err := h.DB.Where("user_id = ?", userID).
		Preload("Church").
		Order("created_at DESC").
		Limit(50).
		Find(&checkins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch check-ins"})
		return
	}
	c.JSON(http.StatusOK, checkins)
}

func (h *CheckInHandler) ChurchCheckIns(c *gin.Context) {
	churchID := c.Param("id")
	var checkins []models.CheckIn
	if err := h.DB.Where("church_id = ?", churchID).
		Preload("User").
		Order("created_at DESC").
		Limit(50).
		Find(&checkins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch check-ins"})
		return
	}
	c.JSON(http.StatusOK, checkins)
}

// UserStats returns check-in stats for a user (loyalty tracking)
func (h *CheckInHandler) UserStats(c *gin.Context) {
	userID := c.GetUint("userID")

	var totalCheckins int64
	h.DB.Model(&models.CheckIn{}).Where("user_id = ?", userID).Count(&totalCheckins)

	// Most visited churches
	type ChurchVisit struct {
		ChurchID   uint   `json:"church_id"`
		ChurchName string `json:"church_name"`
		Visits     int    `json:"visits"`
	}
	var topChurches []ChurchVisit
	h.DB.Model(&models.CheckIn{}).
		Select("check_ins.church_id, churches.name as church_name, count(*) as visits").
		Joins("JOIN churches ON churches.id = check_ins.church_id").
		Where("check_ins.user_id = ?", userID).
		Group("check_ins.church_id, churches.name").
		Order("visits DESC").
		Limit(5).
		Find(&topChurches)

	// Check-ins in last 30 days
	var recentCount int64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	h.DB.Model(&models.CheckIn{}).Where("user_id = ? AND created_at > ?", userID, thirtyDaysAgo).Count(&recentCount)

	c.JSON(http.StatusOK, gin.H{
		"total_checkins":  totalCheckins,
		"recent_30_days":  recentCount,
		"top_churches":    topChurches,
	})
}

// ChurchLoyalUsers returns users most loyal to a specific church
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
	h.DB.Model(&models.CheckIn{}).
		Select("check_ins.user_id, users.name as user_name, count(*) as visits").
		Joins("JOIN users ON users.id = check_ins.user_id").
		Where("check_ins.church_id = ?", churchID).
		Group("check_ins.user_id, users.name").
		Order("visits DESC").
		Limit(20).
		Find(&loyalUsers)

	c.JSON(http.StatusOK, loyalUsers)
}
