package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func TestReputationScore(t *testing.T) {
	t.Run("New user starts with zero trust score and level 1 (New Member)", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Newbie", "newbie@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/reputation", nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["level"].(float64) != 1 {
			t.Errorf("new user should be level 1, got %v", result["level"])
		}
		if result["level_name"] != "New Member" {
			t.Errorf("expected 'New Member', got %v", result["level_name"])
		}
	})

	t.Run("Reputation response includes next level info for gamification", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Gamer", "gamer@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/reputation", nil, token)
		result := testutil.ParseJSON(resp)
		nextLevel := result["next_level"].(map[string]any)
		if nextLevel["level"] == nil || nextLevel["name"] == nil || nextLevel["points_needed"] == nil {
			t.Error("next_level should include level, name, and points_needed")
		}
	})

	t.Run("Check-ins increase the user's trust score", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Active", "active@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Active Church", "Catholic", "Addr", -23.0, -46.0)

		// Directly create check-ins to avoid 2h dedup
		for i := 0; i < 10; i++ {
			app.DB.Create(&models.CheckIn{
				UserID: 1, ChurchID: churchID,
				CreatedAt: time.Now().Add(-time.Duration(i*3) * time.Hour),
			})
		}

		resp := app.Request("GET", "/api/reputation", nil, token)
		result := testutil.ParseJSON(resp)
		score := result["trust_score"].(float64)
		if score <= 0 {
			t.Errorf("user with check-ins should have positive trust score, got %v", score)
		}
		if result["checkin_count"].(float64) < 10 {
			t.Errorf("expected at least 10 check-ins counted, got %v", result["checkin_count"])
		}
	})
}

func TestLeaderboard(t *testing.T) {
	t.Run("Leaderboard shows top contributors by trust score", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Leader", "leader@test.com", "password123", "Catholic")

		// Create a reputation entry
		app.DB.Create(&models.UserReputation{
			UserID: 1, TrustScore: 500, Level: 4, CheckInCount: 50, ApprovedCount: 10,
		})

		resp := app.Request("GET", "/api/leaderboard", nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		if len(results) == 0 {
			t.Error("leaderboard should have at least 1 entry")
		}
	})
}

func TestRateLimiting(t *testing.T) {
	t.Run("New users are limited to 3 suggestions per day", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Spammer", "spam@test.com", "password123", "Catholic")

		// Submit 3 suggestions (should succeed)
		for i := 0; i < 3; i++ {
			resp := app.Request("POST", "/api/suggestions", map[string]any{
				"type": "general", "content": "suggestion " + itoa(uint(i)),
			}, token)
			if resp.Code != http.StatusCreated {
				t.Fatalf("suggestion %d should succeed, got %d: %s", i, resp.Code, resp.Body.String())
			}
		}

		// 4th should be rate limited
		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general", "content": "one too many",
		}, token)
		if resp.Code != http.StatusTooManyRequests {
			t.Errorf("4th suggestion from new user should be rate limited (429), got %d", resp.Code)
		}
	})
}
