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

// ListUsers godoc
// @Summary      List all users
// @Description  Returns all registered users. Admin and Community Manager only.
// @Tags         admin
// @Security     BearerAuth
// @Success      200  {array}  models.User
// @Router       /admin/users [get]
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

// SetRole godoc
// @Summary      Change a user's role
// @Description  Assign a role to a user. Admins can set any role. Community Managers can set user/moderator/church_owner.
// @Tags         admin
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Router       /admin/users/{id}/role [put]
func (h *AdminHandler) SetRole(c *gin.Context) {
	userID := c.Param("id")
	callerRole := models.Role(c.GetString("userRole"))

	var input struct {
		Role models.Role `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !models.ValidRoles[input.Role] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role: must be user, moderator, church_owner, community_manager, or admin"})
		return
	}

	// Community managers can only assign up to church_owner level
	if callerRole == models.RoleCommunityManager {
		if input.Role == models.RoleCommunityManager || input.Role == models.RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "community managers cannot assign community_manager or admin roles"})
			return
		}
	}

	if err := h.DB.WithContext(c.Request.Context()).Model(&models.User{}).Where("id = ?", userID).Update("role", input.Role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "role updated"})
}

// ListModerators godoc
// @Summary      List all moderators and their activity
// @Description  Returns users with moderator role and their check-in counts. Community Manager and Admin only.
// @Tags         admin
// @Security     BearerAuth
// @Success      200  {array}  object
// @Router       /admin/moderators [get]
func (h *AdminHandler) ListModerators(c *gin.Context) {
	ctx := c.Request.Context()

	var moderators []models.User
	if err := h.DB.WithContext(ctx).Where("role IN ?", []models.Role{
		models.RoleModerator, models.RoleChurchOwner, models.RoleCommunityManager,
	}).Order("role DESC, created_at ASC").Find(&moderators).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list moderators"})
		return
	}

	c.JSON(http.StatusOK, moderators)
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
