package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func TestChurchCreation(t *testing.T) {
	t.Run("Authenticated user can register a new church with location data", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Creator", "creator@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/churches", map[string]any{
			"name": "Nossa Senhora Aparecida", "denomination": "Catholic",
			"address": "Rua da Fé, 100", "latitude": -23.55, "longitude": -46.63,
		}, token)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["name"] != "Nossa Senhora Aparecida" {
			t.Errorf("expected church name, got %v", result["name"])
		}
	})

	t.Run("Church created by regular user is not automatically verified", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Regular", "regular@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/churches", map[string]any{
			"name": "Test Church", "address": "Address",
			"latitude": -23.0, "longitude": -46.0,
		}, token)

		result := testutil.ParseJSON(resp)
		if result["verified"] == true {
			t.Error("churches created by regular users should not be auto-verified")
		}
	})

	t.Run("Church created by moderator is automatically verified", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateModerator("Mod", "mod@test.com", "password123")

		resp := app.Request("POST", "/api/churches", map[string]any{
			"name": "Mod Church", "address": "Mod Address",
			"latitude": -23.0, "longitude": -46.0,
		}, token)

		result := testutil.ParseJSON(resp)
		if result["verified"] != true {
			t.Error("churches created by moderators should be auto-verified")
		}
	})

	t.Run("Church created by admin is automatically verified", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateAdmin("Admin", "admin@test.com", "password123")

		resp := app.Request("POST", "/api/churches", map[string]any{
			"name": "Admin Church", "address": "Admin Address",
			"latitude": -23.0, "longitude": -46.0,
		}, token)

		result := testutil.ParseJSON(resp)
		if result["verified"] != true {
			t.Error("churches created by admins should be auto-verified")
		}
	})

	t.Run("Church defaults denomination to Catholic when not specified", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("User", "u@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/churches", map[string]any{
			"name": "Default Denom", "address": "Addr",
			"latitude": -23.0, "longitude": -46.0,
		}, token)

		result := testutil.ParseJSON(resp)
		if result["denomination"] != "Catholic" {
			t.Errorf("expected 'Catholic', got %v", result["denomination"])
		}
	})

	t.Run("Church creation requires name, address, and coordinates", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("User", "req@test.com", "password123", "Catholic")

		// Missing name
		resp := app.Request("POST", "/api/churches", map[string]any{
			"address": "Addr", "latitude": -23.0, "longitude": -46.0,
		}, token)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("missing name should return 400, got %d", resp.Code)
		}
	})
}

func TestChurchListing(t *testing.T) {
	t.Run("User can list all churches", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("User", "list@test.com", "password123", "Catholic")
		app.SeedChurch("Church A", "Catholic", "Addr A", -23.0, -46.0)
		app.SeedChurch("Church B", "Orthodox", "Addr B", -23.1, -46.1)

		resp := app.Request("GET", "/api/churches", nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		if len(results) < 2 {
			t.Errorf("expected at least 2 churches, got %d", len(results))
		}
	})

	t.Run("User can filter churches by denomination", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("User", "filter@test.com", "password123", "Catholic")
		app.SeedChurch("Catholic Church", "Catholic", "Addr", -23.0, -46.0)
		app.SeedChurch("Orthodox Church", "Orthodox", "Addr", -23.1, -46.1)

		resp := app.Request("GET", "/api/churches?denomination=Catholic", nil, token)
		results := testutil.ParseJSONArray(resp)
		for _, c := range results {
			if c["denomination"] != "Catholic" {
				t.Errorf("expected only Catholic churches, got %v", c["denomination"])
			}
		}
	})
}

func TestChurchDetail(t *testing.T) {
	t.Run("User can view detailed information about a specific church", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("User", "detail@test.com", "password123", "Catholic")
		id := app.SeedChurch("São Bento", "Catholic", "Largo de São Bento", -23.534, -46.634)

		resp := app.Request("GET", fmt.Sprintf("/api/churches/%d", id), nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		result := testutil.ParseJSON(resp)
		if result["name"] != "São Bento" {
			t.Errorf("expected 'São Bento', got %v", result["name"])
		}
	})

	t.Run("Requesting a non-existent church returns 404", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("User", "404@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/churches/99999", nil, token)
		if resp.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.Code)
		}
	})
}

func TestChurchEditing(t *testing.T) {
	t.Run("Moderator can edit church details", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		modToken := app.CreateModerator("Mod", "mod@test.com", "password123")
		id := app.SeedChurch("Old Name", "Catholic", "Old Addr", -23.0, -46.0)

		resp := app.Request("PUT", fmt.Sprintf("/api/churches/%d", id), map[string]any{
			"name": "New Name", "address": "New Address",
		}, modToken)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["name"] != "New Name" {
			t.Errorf("expected 'New Name', got %v", result["name"])
		}
	})

	t.Run("Regular user cannot edit church details", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Regular", "norole@test.com", "password123", "Catholic")
		id := app.SeedChurch("Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("PUT", fmt.Sprintf("/api/churches/%d", id), map[string]any{
			"name": "Hacked Name",
		}, token)

		if resp.Code != http.StatusForbidden {
			t.Errorf("regular user should be forbidden, got %d", resp.Code)
		}
	})
}

func TestMassScheduleManagement(t *testing.T) {
	t.Run("Moderator can add a mass schedule to a church", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		modToken := app.CreateModerator("Mod", "modsch@test.com", "password123")
		churchID := app.SeedChurch("Schedule Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/schedules", churchID), map[string]any{
			"day_of_week": 0, "start_time": "10:00",
			"language": "Portuguese", "notes": "Main Sunday Mass",
		}, modToken)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["start_time"] != "10:00" {
			t.Errorf("expected start_time '10:00', got %v", result["start_time"])
		}
	})

	t.Run("Regular user cannot add mass schedules", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Regular", "regsch@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/schedules", churchID), map[string]any{
			"day_of_week": 0, "start_time": "10:00",
		}, token)

		if resp.Code != http.StatusForbidden {
			t.Errorf("regular user should not add schedules, got %d", resp.Code)
		}
	})

	t.Run("Church detail includes its mass schedule", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		modToken := app.CreateModerator("Mod", "modsee@test.com", "password123")
		churchID := app.SeedChurch("Full Church", "Catholic", "Addr", -23.0, -46.0)

		// Add two schedules
		app.Request("POST", fmt.Sprintf("/api/churches/%d/schedules", churchID), map[string]any{
			"day_of_week": 0, "start_time": "08:00", "language": "English",
		}, modToken)
		app.Request("POST", fmt.Sprintf("/api/churches/%d/schedules", churchID), map[string]any{
			"day_of_week": 0, "start_time": "10:00", "language": "Portuguese",
		}, modToken)

		resp := app.Request("GET", fmt.Sprintf("/api/churches/%d", churchID), nil, modToken)
		result := testutil.ParseJSON(resp)
		schedules, ok := result["schedules"].([]any)
		if !ok || len(schedules) < 2 {
			t.Errorf("expected at least 2 schedules, got %v", result["schedules"])
		}
	})
}
