package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

var validSuggestionTypes = map[models.SuggestionType]bool{
	models.SuggestNewChurch:  true,
	models.SuggestEditChurch: true,
	models.SuggestSchedule:   true,
	models.SuggestGeneral:    true,
}

var validSuggestionStatuses = map[models.SuggestionStatus]bool{
	models.SuggestionApproved: true,
	models.SuggestionRejected: true,
}

type SuggestionHandler struct {
	DB *gorm.DB
}

// Create godoc
// @Summary      Submit a suggestion
// @Description  Submit a new church, edit correction, schedule update, or general feedback
// @Tags         suggestions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      object  true  "Suggestion data (type: new_church|edit_church|schedule|general, content, church_id)"
// @Success      201   {object}  models.Suggestion
// @Failure      400   {object}  map[string]string
// @Router       /suggestions [post]
func (h *SuggestionHandler) Create(c *gin.Context) {
	userID := c.GetUint("userID")

	var input struct {
		ChurchID *uint                 `json:"church_id"`
		Type     models.SuggestionType `json:"type" binding:"required"`
		Content  string                `json:"content" binding:"required,max=5000"`
		// Structured proposal fields for new_church suggestions
		Proposal *struct {
			Name         string  `json:"name"`
			Denomination string  `json:"denomination"`
			Address      string  `json:"address"`
			Latitude     float64 `json:"latitude"`
			Longitude    float64 `json:"longitude"`
			Phone        string  `json:"phone"`
			Website      string  `json:"website"`
			Description  string  `json:"description"`
		} `json:"proposal,omitempty"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !validSuggestionTypes[input.Type] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be one of: new_church, edit_church, schedule, general"})
		return
	}

	// Rate limiting: check rejection streak and daily limit
	allowed, reason := CheckSubmissionAllowed(h.DB, userID)
	if !allowed {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": reason})
		return
	}

	// Determine if suggestion qualifies for auto-approval
	status := models.SuggestionPending
	autoApproved := false
	if ShouldAutoApprove(h.DB, userID, input.Type) {
		status = models.SuggestionApproved
		autoApproved = true
	}

	suggestion := models.Suggestion{
		UserID:   userID,
		ChurchID: input.ChurchID,
		Type:     input.Type,
		Content:  input.Content,
		Status:   status,
	}

	ctx := c.Request.Context()
	if err := h.DB.WithContext(ctx).Create(&suggestion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create suggestion"})
		return
	}

	// Save structured proposal for new_church suggestions
	if input.Type == models.SuggestNewChurch && input.Proposal != nil && input.Proposal.Name != "" {
		proposal := models.ChurchProposal{
			SuggestionID: suggestion.ID,
			Name:         input.Proposal.Name,
			Denomination: input.Proposal.Denomination,
			Address:      input.Proposal.Address,
			Latitude:     input.Proposal.Latitude,
			Longitude:    input.Proposal.Longitude,
			Phone:        input.Proposal.Phone,
			Website:      input.Proposal.Website,
			Description:  input.Proposal.Description,
		}
		if proposal.Denomination == "" {
			proposal.Denomination = "Catholic"
		}
		if err := h.DB.WithContext(ctx).Create(&proposal).Error; err != nil {
			log.Printf("failed to save proposal for suggestion %d: %v", suggestion.ID, err)
		}
	}

	// If auto-approved and has a proposal, create the church immediately
	if autoApproved && input.Type == models.SuggestNewChurch && input.Proposal != nil && input.Proposal.Name != "" {
		church := models.Church{
			Name:         input.Proposal.Name,
			Denomination: input.Proposal.Denomination,
			Address:      input.Proposal.Address,
			Latitude:     input.Proposal.Latitude,
			Longitude:    input.Proposal.Longitude,
			Phone:        input.Proposal.Phone,
			Website:      input.Proposal.Website,
			Description:  input.Proposal.Description,
			CreatedByID:  &userID,
			DataQuality:  models.QualityUnverified,
		}
		if church.Denomination == "" {
			church.Denomination = "Catholic"
		}
		if err := h.DB.WithContext(ctx).Create(&church).Error; err != nil {
			log.Printf("failed to auto-create church from proposal: %v", err)
		}
	}

	// Update reputation after new suggestion
	_, _ = RecalculateUserReputation(h.DB, userID)

	response := gin.H{"suggestion": suggestion}
	if autoApproved {
		response["auto_approved"] = true
		response["message"] = "Your suggestion was auto-approved based on your trust score. Thank you for your contributions!"
	}
	c.JSON(http.StatusCreated, response)
}

func (h *SuggestionHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	var suggestions []models.Suggestion
	query := h.DB.WithContext(ctx).Preload("User").Preload("Church").Order("created_at DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	limit := parseLimit(c.Query("limit"))
	offset := parseOffset(c.Query("offset"))

	if err := query.Limit(limit).Offset(offset).Find(&suggestions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list suggestions"})
		return
	}

	c.JSON(http.StatusOK, suggestions)
}

// Review godoc
// @Summary      Review a suggestion
// @Description  Approve or reject a user suggestion. Requires moderator or admin role.
// @Tags         suggestions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int     true  "Suggestion ID"
// @Param        body  body  object  true  "Review data (status: approved|rejected, review_note)"
// @Success      200   {object}  models.Suggestion
// @Failure      403   {object}  map[string]string
// @Router       /suggestions/{id} [put]
func (h *SuggestionHandler) Review(c *gin.Context) {
	id := c.Param("id")
	reviewerID := c.GetUint("userID")

	var input struct {
		Status     models.SuggestionStatus `json:"status" binding:"required"`
		ReviewNote string                  `json:"review_note" binding:"max=2000"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !validSuggestionStatuses[input.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'approved' or 'rejected'"})
		return
	}

	ctx := c.Request.Context()
	var suggestion models.Suggestion
	if err := h.DB.WithContext(ctx).First(&suggestion, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "suggestion not found"})
		return
	}

	suggestion.Status = input.Status
	suggestion.ReviewedByID = &reviewerID
	suggestion.ReviewNote = input.ReviewNote

	if err := h.DB.WithContext(ctx).Save(&suggestion).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update suggestion"})
		return
	}

	// On approval of a new_church suggestion with a proposal, auto-create the church
	var createdChurch *models.Church
	if input.Status == models.SuggestionApproved && suggestion.Type == models.SuggestNewChurch {
		var proposal models.ChurchProposal
		if err := h.DB.WithContext(ctx).Where("suggestion_id = ?", suggestion.ID).First(&proposal).Error; err == nil {
			church := models.Church{
				Name:         proposal.Name,
				Denomination: proposal.Denomination,
				Address:      proposal.Address,
				Latitude:     proposal.Latitude,
				Longitude:    proposal.Longitude,
				Phone:        proposal.Phone,
				Website:      proposal.Website,
				Description:  proposal.Description,
				CreatedByID:  &suggestion.UserID,
				Verified:     true,
				DataQuality:  models.QualityOfficiallyVerified,
			}
			if err := h.DB.WithContext(ctx).Create(&church).Error; err == nil {
				createdChurch = &church
			}
		}
	}

	response := gin.H{"suggestion": suggestion}
	if createdChurch != nil {
		response["church_created"] = createdChurch
		response["message"] = "Suggestion approved and church created automatically!"
	}
	c.JSON(http.StatusOK, response)
}

func (h *SuggestionHandler) MySuggestions(c *gin.Context) {
	userID := c.GetUint("userID")
	limit := parseLimit(c.Query("limit"))
	offset := parseOffset(c.Query("offset"))

	var suggestions []models.Suggestion
	if err := h.DB.WithContext(c.Request.Context()).Where("user_id = ?", userID).
		Preload("Church").
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&suggestions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch suggestions"})
		return
	}
	c.JSON(http.StatusOK, suggestions)
}
