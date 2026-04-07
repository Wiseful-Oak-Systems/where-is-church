package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

type TagHandler struct {
	DB *gorm.DB
}

// ListTags godoc
// @Summary      List all available tags
// @Description  Returns all approved tags, optionally filtered by category.
// @Tags         tags
// @Produce      json
// @Param        category  query  string  false  "Filter by category"
// @Success      200  {array}  models.Tag
// @Router       /tags [get]
func (h *TagHandler) ListTags(c *gin.Context) {
	ctx := c.Request.Context()
	query := h.DB.WithContext(ctx).Where("approved = ?", true).Order("category ASC, label_en ASC")

	if cat := c.Query("category"); cat != "" {
		query = query.Where("category = ?", cat)
	}

	var tags []models.Tag
	if err := query.Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tags"})
		return
	}
	c.JSON(http.StatusOK, tags)
}

// AddTagToChurch godoc
// @Summary      Tag a church with a characteristic
// @Description  Add a tag to a church (e.g., "children-friendly", "latin_mass").
//
//	If the tag already exists on this church, increments the confirmation count.
//
// @Tags         tags
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path  int  true  "Church ID"
// @Param        body    body  object  true  "tag_id"
// @Success      201  {object}  models.ChurchTag
// @Router       /churches/{id}/tags [post]
func (h *TagHandler) AddTagToChurch(c *gin.Context) {
	userID := c.GetUint("userID")
	churchID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid church id"})
		return
	}

	var input struct {
		TagID uint `json:"tag_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Verify church and tag exist
	if err := h.DB.WithContext(ctx).First(&models.Church{}, churchID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "church not found"})
		return
	}
	if err := h.DB.WithContext(ctx).Where("id = ? AND approved = ?", input.TagID, true).First(&models.Tag{}).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}

	// Check if already tagged by this user — if so, just confirm
	var existing models.ChurchTag
	if err := h.DB.WithContext(ctx).Where("church_id = ? AND tag_id = ?", churchID, input.TagID).First(&existing).Error; err == nil {
		existing.Confirmed++
		h.DB.WithContext(ctx).Save(&existing)
		c.JSON(http.StatusOK, existing)
		return
	}

	ct := models.ChurchTag{
		ChurchID:  uint(churchID),
		TagID:     input.TagID,
		AddedByID: userID,
		Confirmed: 1,
	}
	if err := h.DB.WithContext(ctx).Create(&ct).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add tag"})
		return
	}

	h.DB.WithContext(ctx).Preload("Tag").First(&ct, ct.ID)
	c.JSON(http.StatusCreated, ct)
}

// GetChurchTags godoc
// @Summary      Get all tags for a church
// @Tags         tags
// @Param        id  path  int  true  "Church ID"
// @Success      200  {array}  models.ChurchTag
// @Router       /churches/{id}/tags [get]
func (h *TagHandler) GetChurchTags(c *gin.Context) {
	churchID := c.Param("id")
	ctx := c.Request.Context()

	var tags []models.ChurchTag
	if err := h.DB.WithContext(ctx).Where("church_id = ?", churchID).
		Preload("Tag").Order("confirmed DESC").Find(&tags).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get tags"})
		return
	}
	c.JSON(http.StatusOK, tags)
}

// SearchByTag godoc
// @Summary      Find churches by tag
// @Description  Returns all churches that have a specific tag.
// @Tags         tags
// @Param        tag_id  query  int  true  "Tag ID"
// @Success      200  {array}  models.Church
// @Router       /churches/by-tag [get]
func (h *TagHandler) SearchByTag(c *gin.Context) {
	tagID := c.Query("tag_id")
	if tagID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tag_id is required"})
		return
	}

	ctx := c.Request.Context()
	limit := parseLimit(c.Query("limit"))

	var churchIDs []uint
	h.DB.WithContext(ctx).Model(&models.ChurchTag{}).
		Where("tag_id = ?", tagID).
		Pluck("church_id", &churchIDs)

	if len(churchIDs) == 0 {
		c.JSON(http.StatusOK, []models.Church{})
		return
	}

	var churches []models.Church
	if err := h.DB.WithContext(ctx).Where("id IN ?", churchIDs).
		Preload("Schedules").Limit(limit).Find(&churches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}
	c.JSON(http.StatusOK, churches)
}
