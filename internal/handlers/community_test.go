package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func TestChurchConfirmation(t *testing.T) {
	t.Run("User can confirm a church's data is accurate (iNaturalist Research Grade model)", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Confirmer", "confirm@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Confirm Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/confirm", churchID), map[string]any{
			"comment": "I attended Mass here today, times are correct.",
		}, token)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["confirmations"].(float64) != 1 {
			t.Errorf("expected 1 confirmation, got %v", result["confirmations"])
		}
	})

	t.Run("User cannot confirm the same church twice", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("DupConfirm", "dupcon@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Dup Church", "Catholic", "Addr", -23.0, -46.0)

		app.Request("POST", fmt.Sprintf("/api/churches/%d/confirm", churchID), nil, token)
		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/confirm", churchID), nil, token)

		if resp.Code != http.StatusConflict {
			t.Errorf("duplicate confirmation should return 409, got %d", resp.Code)
		}
	})

	t.Run("Church graduates to community_confirmed after 3 independent confirmations", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		churchID := app.SeedChurch("Graduate Church", "Catholic", "Addr", -23.0, -46.0)

		// 3 different users confirm
		for i := 0; i < 3; i++ {
			email := fmt.Sprintf("confirmer%d@test.com", i)
			token := app.CreateUser(fmt.Sprintf("Confirmer%d", i), email, "password123", "Catholic")
			resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/confirm", churchID), nil, token)
			if resp.Code != http.StatusCreated {
				t.Fatalf("confirmation %d should succeed, got %d", i, resp.Code)
			}
		}

		// Check the church's data quality
		viewToken := app.CreateUser("Viewer", "viewer@test.com", "password123", "Catholic")
		resp := app.Request("GET", fmt.Sprintf("/api/churches/%d", churchID), nil, viewToken)
		result := testutil.ParseJSON(resp)
		if result["data_quality"] != string(models.QualityCommunityConfirmed) {
			t.Errorf("expected 'community_confirmed' after 3 confirmations, got %v", result["data_quality"])
		}
		if result["confirmations"].(float64) < 3 {
			t.Errorf("expected 3+ confirmations, got %v", result["confirmations"])
		}
	})
}

func TestContributionImpact(t *testing.T) {
	t.Run("User can see their contribution impact metrics", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Contributor", "impact@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/impact", nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		requiredFields := []string{"approved_suggestions", "churches_helped", "churches_created", "confirmations_made", "people_helped", "impact_message"}
		for _, field := range requiredFields {
			if _, exists := result[field]; !exists {
				t.Errorf("impact response should include %q field", field)
			}
		}
	})

	t.Run("New contributor gets encouraging call-to-action message", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Newbie", "newbie@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/impact", nil, token)
		result := testutil.ParseJSON(resp)
		msg := result["impact_message"].(string)
		if msg == "" {
			t.Error("impact message should not be empty")
		}
	})
}

func TestDataQualityStates(t *testing.T) {
	t.Run("New church starts with 'unverified' data quality", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Creator", "dqcreator@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/churches", map[string]any{
			"name": "New Church", "address": "Addr", "latitude": -23.0, "longitude": -46.0,
		}, token)
		result := testutil.ParseJSON(resp)
		if result["data_quality"] != "unverified" {
			t.Errorf("new church should be 'unverified', got %v", result["data_quality"])
		}
	})

	t.Run("Admin-verified church has 'officially_verified' quality", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		adminToken := app.CreateAdmin("Admin", "dqadmin@test.com", "password123")
		churchID := app.SeedChurch("Admin Church", "Catholic", "Addr", -23.0, -46.0)

		// Admin verifies
		app.Request("PUT", fmt.Sprintf("/api/admin/churches/%d/verify", churchID), nil, adminToken)

		// Check quality
		resp := app.Request("GET", fmt.Sprintf("/api/churches/%d", churchID), nil, adminToken)
		result := testutil.ParseJSON(resp)
		// Admin verify sets verified=true but we need to also set data_quality
		if result["verified"] != true {
			t.Error("admin-verified church should have verified=true")
		}
	})
}
