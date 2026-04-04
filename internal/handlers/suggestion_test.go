package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func TestSuggestionSubmission(t *testing.T) {
	t.Run("User can suggest a new church to be added to the system", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Suggest User", "suggest@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type":    "new_church",
			"content": "There is a beautiful old church at Rua Augusta, 500 that is not listed yet.",
		}, token)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["type"] != "new_church" {
			t.Errorf("expected type 'new_church', got %v", result["type"])
		}
		if result["status"] != "pending" {
			t.Errorf("new suggestion should have 'pending' status, got %v", result["status"])
		}
	})

	t.Run("User can suggest a correction to an existing church", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Correct User", "correct@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Wrong Info Church", "Catholic", "Old Addr", -23.0, -46.0)

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type":      "edit_church",
			"church_id": churchID,
			"content":   "The address has changed to Rua Nova, 200.",
		}, token)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.Code)
		}
	})

	t.Run("User can suggest a mass schedule update", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Schedule User", "sched@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Schedule Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type":      "schedule",
			"church_id": churchID,
			"content":   "They added a new Saturday evening mass at 18:00.",
		}, token)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.Code)
		}
	})

	t.Run("User can submit general feedback about the platform", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Feedback User", "feedback@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type":    "general",
			"content": "Would love to see a feature to share check-ins on social media!",
		}, token)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.Code)
		}
	})

	t.Run("Suggestion requires type and content", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Empty User", "empty@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general",
		}, token)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("missing content should return 400, got %d", resp.Code)
		}
	})
}

func TestUserSuggestionHistory(t *testing.T) {
	t.Run("User can view their own submitted suggestions and their statuses", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("History User", "histsugg@test.com", "password123", "Catholic")

		app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general", "content": "First suggestion",
		}, token)
		app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general", "content": "Second suggestion",
		}, token)

		resp := app.Request("GET", "/api/suggestions/mine", nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		if len(results) != 2 {
			t.Errorf("expected 2 suggestions, got %d", len(results))
		}
	})
}

func TestSuggestionReview(t *testing.T) {
	t.Run("Moderator can review and approve a user suggestion", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("Submitter", "sub@test.com", "password123", "Catholic")
		modToken := app.CreateModerator("Reviewer", "reviewer@test.com", "password123")

		// User submits suggestion
		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "new_church", "content": "New church near the park",
		}, userToken)
		sugg := testutil.ParseJSON(resp)
		suggID := int(sugg["id"].(float64))

		// Moderator approves
		resp = app.Request("PUT", fmt.Sprintf("/api/suggestions/%d", suggID), map[string]any{
			"status":      "approved",
			"review_note": "Verified, will add the church.",
		}, modToken)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["status"] != "approved" {
			t.Errorf("expected status 'approved', got %v", result["status"])
		}
		if result["review_note"] != "Verified, will add the church." {
			t.Error("review note should be stored")
		}
	})

	t.Run("Moderator can reject a suggestion with a reason", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("SubReject", "reject@test.com", "password123", "Catholic")
		modToken := app.CreateModerator("ModReject", "modreject@test.com", "password123")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general", "content": "Something invalid",
		}, userToken)
		sugg := testutil.ParseJSON(resp)
		suggID := int(sugg["id"].(float64))

		resp = app.Request("PUT", fmt.Sprintf("/api/suggestions/%d", suggID), map[string]any{
			"status":      "rejected",
			"review_note": "This does not meet our guidelines.",
		}, modToken)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		result := testutil.ParseJSON(resp)
		if result["status"] != "rejected" {
			t.Errorf("expected 'rejected', got %v", result["status"])
		}
	})

	t.Run("Regular user cannot review suggestions", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("NoReview", "noreview@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general", "content": "Some suggestion",
		}, userToken)
		sugg := testutil.ParseJSON(resp)
		suggID := int(sugg["id"].(float64))

		resp = app.Request("PUT", fmt.Sprintf("/api/suggestions/%d", suggID), map[string]any{
			"status": "approved",
		}, userToken)

		if resp.Code != http.StatusForbidden {
			t.Errorf("regular user should not review suggestions, got %d", resp.Code)
		}
	})

	t.Run("Moderator can list all suggestions filtered by status", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("Lister", "lister@test.com", "password123", "Catholic")
		modToken := app.CreateModerator("ModList", "modlist@test.com", "password123")

		app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general", "content": "Pending one",
		}, userToken)

		resp := app.Request("GET", "/api/suggestions?status=pending", nil, modToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		for _, s := range results {
			if s["status"] != "pending" {
				t.Errorf("expected only pending suggestions, got %v", s["status"])
			}
		}
	})
}

