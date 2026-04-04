package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

type AdminHandler struct {
	DB *gorm.DB
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	ctx := c.Request.Context()
	limit := parseLimit(c.Query("limit"))
	offset := parseOffset(c.Query("offset"))

	var users []models.User
	if err := h.DB.WithContext(ctx).Order("created_at DESC").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list users"})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *AdminHandler) SetRole(c *gin.Context) {
	userID := c.Param("id")

	var input struct {
		Role models.Role `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Role != models.RoleUser && input.Role != models.RoleModerator && input.Role != models.RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role: must be 'user', 'moderator', or 'admin'"})
		return
	}

	if err := h.DB.WithContext(c.Request.Context()).Model(&models.User{}).Where("id = ?", userID).Update("role", input.Role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

func (h *AdminHandler) VerifyChurch(c *gin.Context) {
	churchID := c.Param("id")
	now := time.Now()

	if err := h.DB.WithContext(c.Request.Context()).Model(&models.Church{}).Where("id = ?", churchID).
		Updates(map[string]any{"verified": true, "last_verified": now}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify church"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "church verified"})
}

func (h *AdminHandler) DeleteChurch(c *gin.Context) {
	churchID := c.Param("id")

	if err := h.DB.WithContext(c.Request.Context()).Delete(&models.Church{}, churchID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete church"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "church deleted"})
}
