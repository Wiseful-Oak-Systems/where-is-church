package handlers

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

type ChurchHandler struct {
	DB *gorm.DB
}

// SearchNearby finds churches within a radius (km) of a given lat/lng.
// Uses the Haversine formula in SQL for distance calculation.
func (h *ChurchHandler) SearchNearby(c *gin.Context) {
	lat, err := strconv.ParseFloat(c.Query("lat"), 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat is required"})
		return
	}
	lng, err := strconv.ParseFloat(c.Query("lng"), 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lng is required"})
		return
	}

	radiusKm := 10.0 // default 10km
	if r := c.Query("radius"); r != "" {
		if parsed, err := strconv.ParseFloat(r, 64); err == nil && parsed > 0 {
			radiusKm = math.Min(parsed, 100) // cap at 100km
		}
	}

	denomination := c.Query("denomination")
	if denomination == "" {
		if d, exists := c.Get("userDenomination"); exists {
			denomination = d.(string)
		}
	}

	// Haversine formula for distance in km
	haversine := `(6371 * acos(
		cos(radians(?)) * cos(radians(latitude)) *
		cos(radians(longitude) - radians(?)) +
		sin(radians(?)) * sin(radians(latitude))
	))`

	query := h.DB.Model(&models.Church{}).
		Select("*, "+haversine+" AS distance", lat, lng, lat).
		Where(haversine+" <= ?", lat, lng, lat, radiusKm).
		Order("distance ASC")

	if denomination != "" && denomination != "All" {
		query = query.Where("denomination = ?", denomination)
	}

	var churches []struct {
		models.Church
		Distance float64 `json:"distance"`
	}

	if err := query.Preload("Schedules").Find(&churches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	c.JSON(http.StatusOK, churches)
}

func (h *ChurchHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	var church models.Church
	if err := h.DB.Preload("Schedules").Preload("CreatedBy").First(&church, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "church not found"})
		return
	}
	c.JSON(http.StatusOK, church)
}

func (h *ChurchHandler) Create(c *gin.Context) {
	var input struct {
		Name         string  `json:"name" binding:"required"`
		Denomination string  `json:"denomination"`
		Address      string  `json:"address" binding:"required"`
		Latitude     float64 `json:"latitude" binding:"required"`
		Longitude    float64 `json:"longitude" binding:"required"`
		Phone        string  `json:"phone"`
		Website      string  `json:"website"`
		Description  string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("userID")
	if input.Denomination == "" {
		input.Denomination = "Catholic"
	}

	church := models.Church{
		Name:         input.Name,
		Denomination: input.Denomination,
		Address:      input.Address,
		Latitude:     input.Latitude,
		Longitude:    input.Longitude,
		Phone:        input.Phone,
		Website:      input.Website,
		Description:  input.Description,
		CreatedByID:  userID,
	}

	// Admins/moderators create verified churches
	role := c.GetString("userRole")
	if role == string(models.RoleAdmin) || role == string(models.RoleModerator) {
		church.Verified = true
	}

	if err := h.DB.Create(&church).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create church"})
		return
	}

	c.JSON(http.StatusCreated, church)
}

func (h *ChurchHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var church models.Church
	if err := h.DB.First(&church, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "church not found"})
		return
	}

	var input map[string]any
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Prevent updating sensitive fields
	delete(input, "id")
	delete(input, "created_by_id")
	delete(input, "created_at")

	if err := h.DB.Model(&church).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update church"})
		return
	}

	h.DB.Preload("Schedules").First(&church, id)
	c.JSON(http.StatusOK, church)
}

// AddSchedule adds a mass schedule entry to a church
func (h *ChurchHandler) AddSchedule(c *gin.Context) {
	churchID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid church id"})
		return
	}

	// Verify church exists
	var church models.Church
	if err := h.DB.First(&church, churchID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "church not found"})
		return
	}

	var input struct {
		DayOfWeek int    `json:"day_of_week" binding:"min=0,max=6"`
		StartTime string `json:"start_time" binding:"required"`
		Language  string `json:"language"`
		Notes     string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Language == "" {
		input.Language = "English"
	}

	schedule := models.MassSchedule{
		ChurchID:  uint(churchID),
		DayOfWeek: input.DayOfWeek,
		StartTime: input.StartTime,
		Language:  input.Language,
		Notes:     input.Notes,
	}

	if err := h.DB.Create(&schedule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add schedule"})
		return
	}

	c.JSON(http.StatusCreated, schedule)
}

func (h *ChurchHandler) DeleteSchedule(c *gin.Context) {
	scheduleID := c.Param("scheduleId")
	if err := h.DB.Delete(&models.MassSchedule{}, scheduleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete schedule"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "schedule deleted"})
}

func (h *ChurchHandler) List(c *gin.Context) {
	var churches []models.Church
	query := h.DB.Preload("Schedules").Order("name ASC")

	if denom := c.Query("denomination"); denom != "" && denom != "All" {
		query = query.Where("denomination = ?", denom)
	}

	if err := query.Limit(100).Find(&churches).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list churches"})
		return
	}
	c.JSON(http.StatusOK, churches)
}
