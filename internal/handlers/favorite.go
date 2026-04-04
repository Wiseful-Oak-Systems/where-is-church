package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

type FavoriteHandler struct {
	DB *gorm.DB
}

func (h *FavoriteHandler) Add(c *gin.Context) {
	userID := c.GetUint("userID")
	churchID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid church id"})
		return
	}

	ctx := c.Request.Context()

	// Verify church exists
	var church models.Church
	if err := h.DB.WithContext(ctx).First(&church, churchID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "church not found"})
		return
	}

	fav := models.Favorite{
		UserID:   userID,
		ChurchID: uint(churchID),
	}

	if err := h.DB.WithContext(ctx).Create(&fav).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "church already in favorites"})
		return
	}

	c.JSON(http.StatusCreated, fav)
}

func (h *FavoriteHandler) Remove(c *gin.Context) {
	userID := c.GetUint("userID")
	churchID := c.Param("id")
	ctx := c.Request.Context()

	result := h.DB.WithContext(ctx).Where("user_id = ? AND church_id = ?", userID, churchID).Delete(&models.Favorite{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove favorite"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "favorite not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "removed from favorites"})
}

func (h *FavoriteHandler) List(c *gin.Context) {
	userID := c.GetUint("userID")
	ctx := c.Request.Context()

	var favorites []models.Favorite
	if err := h.DB.WithContext(ctx).Where("user_id = ?", userID).
		Preload("Church").Preload("Church.Schedules").
		Order("created_at DESC").
		Find(&favorites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list favorites"})
		return
	}
	c.JSON(http.StatusOK, favorites)
}

func (h *FavoriteHandler) Check(c *gin.Context) {
	userID := c.GetUint("userID")
	churchID := c.Param("id")
	ctx := c.Request.Context()

	var count int64
	h.DB.WithContext(ctx).Model(&models.Favorite{}).Where("user_id = ? AND church_id = ?", userID, churchID).Count(&count)
	c.JSON(http.StatusOK, gin.H{"is_favorite": count > 0})
}
