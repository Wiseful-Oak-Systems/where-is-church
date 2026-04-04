package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

type OwnershipHandler struct {
	DB *gorm.DB
}

// ClaimChurch godoc
// @Summary      Claim ownership of a church
// @Description  Submit a claim to be recognized as the owner/administrator of a church location.
//
//	The claim must include evidence (e.g. parish role, contact info) and will be
//	reviewed by a Community Manager or Admin.
//
// @Tags         ownership
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int     true  "Church ID"
// @Param        body  body  object  true  "Claim data (evidence)"
// @Success      201   {object}  models.ChurchOwnership
// @Failure      409   {object}  map[string]string  "Already claimed"
// @Router       /churches/{id}/claim [post]
func (h *OwnershipHandler) ClaimChurch(c *gin.Context) {
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

	// Check for existing pending/approved claim by this user
	var existing models.ChurchOwnership
	if err := h.DB.WithContext(ctx).Where("user_id = ? AND church_id = ? AND status IN ?",
		userID, churchID, []models.ClaimStatus{models.ClaimPending, models.ClaimApproved}).
		First(&existing).Error; err == nil {
		if existing.Status == models.ClaimApproved {
			c.JSON(http.StatusConflict, gin.H{"error": "you already own this church"})
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "you already have a pending claim for this church"})
		}
		return
	}

	var input struct {
		Evidence string `json:"evidence" binding:"required,max=2000"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	claim := models.ChurchOwnership{
		UserID:   userID,
		ChurchID: uint(churchID),
		Status:   models.ClaimPending,
		Evidence: input.Evidence,
	}

	if err := h.DB.WithContext(ctx).Create(&claim).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit claim"})
		return
	}

	c.JSON(http.StatusCreated, claim)
}

// ListClaims godoc
// @Summary      List ownership claims
// @Description  Returns all church ownership claims, filterable by status. Community Manager and Admin only.
// @Tags         ownership
// @Security     BearerAuth
// @Param        status  query  string  false  "Filter by status (pending, approved, rejected)"
// @Success      200     {array}  models.ChurchOwnership
// @Router       /admin/claims [get]
func (h *OwnershipHandler) ListClaims(c *gin.Context) {
	ctx := c.Request.Context()
	limit := parseLimit(c.Query("limit"))
	offset := parseOffset(c.Query("offset"))

	query := h.DB.WithContext(ctx).
		Preload("User").Preload("Church").Preload("ReviewedBy").
		Order("claimed_at DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	var claims []models.ChurchOwnership
	if err := query.Limit(limit).Offset(offset).Find(&claims).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list claims"})
		return
	}
	c.JSON(http.StatusOK, claims)
}

// ReviewClaim godoc
// @Summary      Approve or reject an ownership claim
// @Description  Review a pending church ownership claim. On approval, the user is promoted to church_owner
//
//	role (if not already higher) and gains direct edit access to that church.
//
// @Tags         ownership
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  int     true  "Claim ID"
// @Param        body  body  object  true  "Review data (status: approved|rejected, review_note)"
// @Success      200   {object}  models.ChurchOwnership
// @Router       /admin/claims/{id} [put]
func (h *OwnershipHandler) ReviewClaim(c *gin.Context) {
	claimID := c.Param("id")
	reviewerID := c.GetUint("userID")

	var input struct {
		Status     models.ClaimStatus `json:"status" binding:"required"`
		ReviewNote string             `json:"review_note" binding:"max=2000"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Status != models.ClaimApproved && input.Status != models.ClaimRejected {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'approved' or 'rejected'"})
		return
	}

	ctx := c.Request.Context()

	var claim models.ChurchOwnership
	if err := h.DB.WithContext(ctx).First(&claim, claimID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "claim not found"})
		return
	}

	if claim.Status != models.ClaimPending {
		c.JSON(http.StatusConflict, gin.H{"error": "claim has already been reviewed"})
		return
	}

	now := time.Now()
	claim.Status = input.Status
	claim.ReviewedByID = &reviewerID
	claim.ReviewNote = input.ReviewNote
	claim.ReviewedAt = &now

	if err := h.DB.WithContext(ctx).Save(&claim).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update claim"})
		return
	}

	// On approval, promote user to church_owner if their current role is lower
	if input.Status == models.ClaimApproved {
		var user models.User
		if err := h.DB.WithContext(ctx).First(&user, claim.UserID).Error; err == nil {
			if !models.HasAtLeastRole(user.Role, models.RoleChurchOwner) {
				h.DB.WithContext(ctx).Model(&user).Update("role", models.RoleChurchOwner)
			}
		}
		// Also verify the church if not already
		h.DB.WithContext(ctx).Model(&models.Church{}).Where("id = ?", claim.ChurchID).
			Updates(map[string]any{"verified": true, "last_verified": now})
	}

	h.DB.WithContext(ctx).Preload("User").Preload("Church").Preload("ReviewedBy").First(&claim, claim.ID)
	c.JSON(http.StatusOK, claim)
}

// MyClaims godoc
// @Summary      List my ownership claims
// @Description  Returns all ownership claims submitted by the current user.
// @Tags         ownership
// @Security     BearerAuth
// @Success      200  {array}  models.ChurchOwnership
// @Router       /my-churches/claims [get]
func (h *OwnershipHandler) MyClaims(c *gin.Context) {
	userID := c.GetUint("userID")
	ctx := c.Request.Context()

	var claims []models.ChurchOwnership
	if err := h.DB.WithContext(ctx).Where("user_id = ?", userID).
		Preload("Church").
		Order("claimed_at DESC").
		Find(&claims).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch claims"})
		return
	}
	c.JSON(http.StatusOK, claims)
}

// MyChurches godoc
// @Summary      List churches I own
// @Description  Returns all churches where the current user has approved ownership.
// @Tags         ownership
// @Security     BearerAuth
// @Success      200  {array}  models.Church
// @Router       /my-churches [get]
func (h *OwnershipHandler) MyChurches(c *gin.Context) {
	userID := c.GetUint("userID")
	ctx := c.Request.Context()

	var ownerships []models.ChurchOwnership
	if err := h.DB.WithContext(ctx).Where("user_id = ? AND status = ?", userID, models.ClaimApproved).
		Preload("Church").Preload("Church.Schedules").
		Find(&ownerships).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch owned churches"})
		return
	}

	churches := make([]models.Church, 0, len(ownerships))
	for _, o := range ownerships {
		if o.Church != nil {
			churches = append(churches, *o.Church)
		}
	}
	c.JSON(http.StatusOK, churches)
}

// IsChurchOwner checks if a user has approved ownership of a specific church.
func IsChurchOwner(db *gorm.DB, userID, churchID uint) bool {
	var count int64
	db.Model(&models.ChurchOwnership{}).
		Where("user_id = ? AND church_id = ? AND status = ?", userID, churchID, models.ClaimApproved).
		Count(&count)
	return count > 0
}
