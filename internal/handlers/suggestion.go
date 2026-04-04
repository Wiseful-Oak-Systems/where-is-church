package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

type SuggestionHandler struct {
	DB *gorm.DB
}

func (h *SuggestionHandler) Create(c *gin.Context) {
	userID := c.GetUint("userID")

	var input struct {
		ChurchID *uint                `json:"church_id"`
		Type     models.SuggestionType `json:"type" binding:"required"`
		Content  string               `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	suggestion := models.Suggestion{
		UserID:   userID,
		ChurchID: input.ChurchID,
		Type:     input.Type,
		Content:  input.Content,
		Status:   models.SuggestionPending,
	}

	if err := h.DB.Create(&suggestion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create suggestion"})
		return
	}

	c.JSON(http.StatusCreated, suggestion)
}

func (h *SuggestionHandler) List(c *gin.Context) {
	var suggestions []models.Suggestion
	query := h.DB.Preload("User").Preload("Church").Order("created_at DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Limit(100).Find(&suggestions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list suggestions"})
		return
	}

	c.JSON(http.StatusOK, suggestions)
}

func (h *SuggestionHandler) Review(c *gin.Context) {
	id := c.Param("id")
	reviewerID := c.GetUint("userID")

	var input struct {
		Status     models.SuggestionStatus `json:"status" binding:"required"`
		ReviewNote string                  `json:"review_note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var suggestion models.Suggestion
	if err := h.DB.First(&suggestion, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "suggestion not found"})
		return
	}

	suggestion.Status = input.Status
	suggestion.ReviewedByID = &reviewerID
	suggestion.ReviewNote = input.ReviewNote

	if err := h.DB.Save(&suggestion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update suggestion"})
		return
	}

	c.JSON(http.StatusOK, suggestion)
}

func (h *SuggestionHandler) MySuggestions(c *gin.Context) {
	userID := c.GetUint("userID")
	var suggestions []models.Suggestion
	if err := h.DB.Where("user_id = ?", userID).
		Preload("Church").
		Order("created_at DESC").
		Find(&suggestions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch suggestions"})
		return
	}
	c.JSON(http.StatusOK, suggestions)
}
