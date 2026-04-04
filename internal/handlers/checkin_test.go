package handlers_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func TestCheckIn(t *testing.T) {
	t.Run("User can check in at a church like Foursquare, marking they attended", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Visitor", "visitor@test.com", "pass123", "Catholic")
		churchID := app.SeedChurch("Parish Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", "/api/checkins", map[string]any{
			"church_id": churchID, "notes": "Beautiful homily today!",
		}, token)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["notes"] != "Beautiful homily today!" {
			t.Errorf("check-in notes should be stored")
		}
	})

	t.Run("User cannot check in at the same church twice within 2 hours", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Double", "double@test.com", "pass123", "Catholic")
		churchID := app.SeedChurch("Dup Church", "Catholic", "Addr", -23.0, -46.0)

		// First check-in
		resp := app.Request("POST", "/api/checkins", map[string]any{
			"church_id": churchID,
		}, token)
		if resp.Code != http.StatusCreated {
			t.Fatalf("first check-in should succeed, got %d", resp.Code)
		}

		// Second check-in immediately
		resp = app.Request("POST", "/api/checkins", map[string]any{
			"church_id": churchID,
		}, token)
		if resp.Code != http.StatusConflict {
			t.Errorf("duplicate check-in within 2 hours should return 409, got %d", resp.Code)
		}
	})

	t.Run("User can check in at a different church without restriction", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Multi", "multi@test.com", "pass123", "Catholic")
		church1 := app.SeedChurch("Church 1", "Catholic", "Addr 1", -23.0, -46.0)
		church2 := app.SeedChurch("Church 2", "Catholic", "Addr 2", -23.1, -46.1)

		app.Request("POST", "/api/checkins", map[string]any{"church_id": church1}, token)
		resp := app.Request("POST", "/api/checkins", map[string]any{"church_id": church2}, token)

		if resp.Code != http.StatusCreated {
			t.Errorf("checking in at different church should succeed, got %d", resp.Code)
		}
	})

	t.Run("Check-in at non-existent church returns 404", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Ghost", "ghost@test.com", "pass123", "Catholic")

		resp := app.Request("POST", "/api/checkins", map[string]any{
			"church_id": 99999,
		}, token)

		if resp.Code != http.StatusNotFound {
			t.Errorf("expected 404 for non-existent church, got %d", resp.Code)
		}
	})
}

func TestCheckInHistory(t *testing.T) {
	t.Run("User can view their personal check-in history", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("History", "history@test.com", "pass123", "Catholic")
		church1 := app.SeedChurch("History Church 1", "Catholic", "Addr", -23.0, -46.0)
		church2 := app.SeedChurch("History Church 2", "Catholic", "Addr", -23.1, -46.1)

		app.Request("POST", "/api/checkins", map[string]any{"church_id": church1}, token)
		app.Request("POST", "/api/checkins", map[string]any{"church_id": church2}, token)

		resp := app.Request("GET", "/api/checkins/mine", nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		if len(results) != 2 {
			t.Errorf("expected 2 check-ins, got %d", len(results))
		}
	})

	t.Run("Anyone can see recent visitors at a specific church", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token1 := app.CreateUser("Visitor1", "v1@test.com", "pass123", "Catholic")
		token2 := app.CreateUser("Visitor2", "v2@test.com", "pass123", "Catholic")
		churchID := app.SeedChurch("Popular Church", "Catholic", "Addr", -23.0, -46.0)

		app.Request("POST", "/api/checkins", map[string]any{"church_id": churchID}, token1)
		app.Request("POST", "/api/checkins", map[string]any{"church_id": churchID}, token2)

		resp := app.Request("GET", fmt.Sprintf("/api/churches/%d/checkins", churchID), nil, token1)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		if len(results) != 2 {
			t.Errorf("expected 2 visitors, got %d", len(results))
		}
	})
}

func TestLoyaltyTracking(t *testing.T) {
	t.Run("System tracks user attendance stats including total and recent check-ins", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Loyal", "loyal@test.com", "pass123", "Catholic")
		churchID := app.SeedChurch("Loyalty Church", "Catholic", "Addr", -23.0, -46.0)

		app.Request("POST", "/api/checkins", map[string]any{"church_id": churchID}, token)

		resp := app.Request("GET", "/api/checkins/stats", nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		result := testutil.ParseJSON(resp)
		total := result["total_checkins"].(float64)
		if total < 1 {
			t.Errorf("expected at least 1 total check-in, got %v", total)
		}
		if result["recent_30_days"] == nil {
			t.Error("stats should include recent_30_days count")
		}
		if result["top_churches"] == nil {
			t.Error("stats should include top_churches list")
		}
	})

	t.Run("System identifies the most loyal users of a specific church", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token1 := app.CreateUser("Frequent", "freq@test.com", "pass123", "Catholic")
		churchID := app.SeedChurch("Loyal Parish", "Catholic", "Addr", -23.0, -46.0)

		// Simulate multiple check-ins by inserting directly
		for i := 0; i < 5; i++ {
			app.DB.Create(&models.CheckIn{
				UserID: 1, ChurchID: churchID,
				CreatedAt: time.Now().Add(-time.Duration(i*24) * time.Hour),
			})
		}

		resp := app.Request("GET", fmt.Sprintf("/api/churches/%d/loyal-users", churchID), nil, token1)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		if len(results) == 0 {
			t.Error("expected at least one loyal user")
		}
	})
}
