package models_test

import (
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/models"
)

func TestSuggestionModel(t *testing.T) {
	t.Run("Suggestion types cover all expected user contribution categories", func(t *testing.T) {
		types := map[models.SuggestionType]string{
			models.SuggestNewChurch:  "new_church",
			models.SuggestEditChurch: "edit_church",
			models.SuggestSchedule:   "schedule",
			models.SuggestGeneral:    "general",
		}
		for constant, expected := range types {
			if string(constant) != expected {
				t.Errorf("expected %q, got %q", expected, constant)
			}
		}
	})

	t.Run("Suggestion statuses represent the full review lifecycle", func(t *testing.T) {
		statuses := map[models.SuggestionStatus]string{
			models.SuggestionPending:  "pending",
			models.SuggestionApproved: "approved",
			models.SuggestionRejected: "rejected",
		}
		for constant, expected := range statuses {
			if string(constant) != expected {
				t.Errorf("expected %q, got %q", expected, constant)
			}
		}
	})

	t.Run("Suggestion can optionally be linked to a specific church", func(t *testing.T) {
		churchID := uint(42)
		suggestion := models.Suggestion{
			ChurchID: &churchID,
			Type:     models.SuggestEditChurch,
			Content:  "Wrong address",
		}
		if suggestion.ChurchID == nil || *suggestion.ChurchID != 42 {
			t.Error("suggestion should optionally reference a church")
		}
	})

	t.Run("Suggestion without church ID is valid for new church submissions", func(t *testing.T) {
		suggestion := models.Suggestion{
			Type:    models.SuggestNewChurch,
			Content: "There is a new church at Rua X",
		}
		if suggestion.ChurchID != nil {
			t.Error("new church suggestion should not require a church ID")
		}
	})

	t.Run("Suggestion tracks who reviewed it and their notes", func(t *testing.T) {
		reviewerID := uint(10)
		suggestion := models.Suggestion{
			ReviewedByID: &reviewerID,
			ReviewNote:   "Verified in person, approved.",
		}
		if suggestion.ReviewedByID == nil {
			t.Error("reviewer ID should be tracked")
		}
		if suggestion.ReviewNote == "" {
			t.Error("review note should be stored")
		}
	})
}
