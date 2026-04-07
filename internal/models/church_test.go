package models_test

import (
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/models"
)

func TestChurchModel(t *testing.T) {
	t.Run("Church stores all essential information including name, address, and coordinates", func(t *testing.T) {
		church := models.Church{
			Name:         "Catedral da Sé",
			Denomination: "Catholic",
			Address:      "Praça da Sé, São Paulo",
			Latitude:     -23.5505,
			Longitude:    -46.6333,
		}
		if church.Name != "Catedral da Sé" {
			t.Error("church name should be stored")
		}
		if church.Latitude == 0 || church.Longitude == 0 {
			t.Error("church coordinates should be stored")
		}
	})

	t.Run("Church supports optional contact information like phone and website", func(t *testing.T) {
		church := models.Church{
			Name:    "Test Church",
			Phone:   "+55 11 9999-0000",
			Website: "https://testchurch.com",
		}
		if church.Phone == "" || church.Website == "" {
			t.Error("optional contact fields should be stored")
		}
	})

	t.Run("Church has a verification status to distinguish curated from user-submitted entries", func(t *testing.T) {
		church := models.Church{Name: "Unverified Church"}
		if church.Verified != false {
			t.Error("churches should be unverified by default")
		}
	})

	t.Run("Church supports multiple denominations beyond Catholic", func(t *testing.T) {
		denominations := []string{"Catholic", "Orthodox", "Protestant", "Anglican", "Evangelical"}
		for _, d := range denominations {
			church := models.Church{Denomination: d}
			if church.Denomination != d {
				t.Errorf("church should support denomination %q", d)
			}
		}
	})
}

func TestMassScheduleModel(t *testing.T) {
	t.Run("Mass schedule stores day of week, time, and language", func(t *testing.T) {
		schedule := models.MassSchedule{
			ChurchID:  1,
			DayOfWeek: 0, // Sunday
			StartTime: "10:00",
			Language:  "Portuguese",
		}
		if schedule.DayOfWeek != 0 {
			t.Error("day of week should be stored (0=Sunday)")
		}
		if schedule.StartTime != "10:00" {
			t.Error("start time should be stored")
		}
		if schedule.Language != "Portuguese" {
			t.Error("language should be stored")
		}
	})

	t.Run("Day names array provides human-readable day labels starting from Sunday", func(t *testing.T) {
		if len(models.DayNames) != 7 {
			t.Fatalf("expected 7 day names, got %d", len(models.DayNames))
		}
		if models.DayNames[0] != "Sunday" {
			t.Errorf("expected index 0 to be Sunday, got %q", models.DayNames[0])
		}
		if models.DayNames[6] != "Saturday" {
			t.Errorf("expected index 6 to be Saturday, got %q", models.DayNames[6])
		}
	})

	t.Run("Mass schedule supports optional notes for special occasions", func(t *testing.T) {
		schedule := models.MassSchedule{
			Notes: "Latin Mass - Extraordinary Form",
		}
		if schedule.Notes == "" {
			t.Error("notes should be stored")
		}
	})
}
