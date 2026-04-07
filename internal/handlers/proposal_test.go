package handlers_test

import (
	"net/http"
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func TestStructuredChurchProposal(t *testing.T) {
	t.Run("User can suggest a new church with structured data (name, address, coordinates)", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Proposer", "proposer@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type":    "new_church",
			"content": "New parish I discovered in my neighborhood",
			"proposal": map[string]any{
				"name":         "Paróquia São José",
				"denomination": "Catholic",
				"address":      "Rua Augusta, 500 - Consolação, São Paulo - SP",
				"latitude":     -23.5558,
				"longitude":    -46.6621,
				"phone":        "(11) 3256-0000",
			},
		}, token)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}

		// Verify the proposal was stored
		var proposal models.ChurchProposal
		err := app.DB.Where("name = ?", "Paróquia São José").First(&proposal).Error
		if err != nil {
			t.Fatal("proposal should be saved in the database")
		}
		if proposal.Latitude != -23.5558 {
			t.Errorf("expected lat -23.5558, got %v", proposal.Latitude)
		}
	})

	t.Run("Suggestion without proposal field still works (backwards compatible)", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Simple", "simple@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type":    "new_church",
			"content": "There's a church at Rua X. I don't know the exact coordinates.",
		}, token)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.Code)
		}
	})

	t.Run("Approving a new_church suggestion with proposal auto-creates the church", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("Submitter", "submit@test.com", "password123", "Catholic")
		modToken := app.CreateModerator("Moderator", "mod@test.com", "password123")

		// Submit structured suggestion
		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type":    "new_church",
			"content": "New church in my area",
			"proposal": map[string]any{
				"name":         "Igreja Nova Esperança",
				"denomination": "Evangelical",
				"address":      "Av. Brasil, 1000 - Centro",
				"latitude":     -22.9068,
				"longitude":    -43.1729,
			},
		}, userToken)
		sugg := testutil.ParseJSON(resp)["suggestion"].(map[string]any)
		suggID := int(sugg["id"].(float64))

		// Moderator approves
		resp = app.Request("PUT", "/api/suggestions/"+itoa(uint(suggID)), map[string]any{
			"status":      "approved",
			"review_note": "Verified — church exists at this location.",
		}, modToken)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		result := testutil.ParseJSON(resp)
		if result["church_created"] == nil {
			t.Fatal("approving a proposal should auto-create the church")
		}
		church := result["church_created"].(map[string]any)
		if church["name"] != "Igreja Nova Esperança" {
			t.Errorf("expected 'Igreja Nova Esperança', got %v", church["name"])
		}
		if church["verified"] != true {
			t.Error("auto-created church should be verified")
		}

		// Verify church exists in DB
		var dbChurch models.Church
		err := app.DB.Where("name = ?", "Igreja Nova Esperança").First(&dbChurch).Error
		if err != nil {
			t.Fatal("church should exist in database after approval")
		}
	})

	t.Run("Rejecting a suggestion with proposal does NOT create a church", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("Rejected", "rejected@test.com", "password123", "Catholic")
		modToken := app.CreateModerator("ModReject", "modrej@test.com", "password123")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type":    "new_church",
			"content": "This is a fake church",
			"proposal": map[string]any{
				"name":      "Fake Church",
				"address":   "Nowhere",
				"latitude":  0.0,
				"longitude": 0.0,
			},
		}, userToken)
		sugg := testutil.ParseJSON(resp)["suggestion"].(map[string]any)
		suggID := int(sugg["id"].(float64))

		resp = app.Request("PUT", "/api/suggestions/"+itoa(uint(suggID)), map[string]any{
			"status":      "rejected",
			"review_note": "This location does not exist.",
		}, modToken)

		result := testutil.ParseJSON(resp)
		if result["church_created"] != nil {
			t.Error("rejected suggestion should NOT create a church")
		}

		var count int64
		app.DB.Model(&models.Church{}).Where("name = ?", "Fake Church").Count(&count)
		if count > 0 {
			t.Error("rejected proposal's church should not be in database")
		}
	})
}
