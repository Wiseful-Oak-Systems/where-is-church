package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func TestAdminUserManagement(t *testing.T) {
	t.Run("Admin can list all registered users", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		adminToken := app.CreateAdmin("Admin", "admin@test.com", "pass123")
		app.CreateUser("User1", "u1@test.com", "pass123", "Catholic")
		app.CreateUser("User2", "u2@test.com", "pass123", "Orthodox")

		resp := app.Request("GET", "/api/admin/users", nil, adminToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		if len(results) < 3 { // admin + 2 users
			t.Errorf("expected at least 3 users, got %d", len(results))
		}
	})

	t.Run("Admin can promote a loyal user to moderator role", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		adminToken := app.CreateAdmin("Admin", "admin@test.com", "pass123")
		app.CreateUser("Loyal User", "loyal@test.com", "pass123", "Catholic")

		// Get user ID
		resp := app.Request("GET", "/api/admin/users", nil, adminToken)
		users := testutil.ParseJSONArray(resp)
		var loyalUserID float64
		for _, u := range users {
			if u["email"] == "loyal@test.com" {
				loyalUserID = u["id"].(float64)
				break
			}
		}

		resp = app.Request("PUT", fmt.Sprintf("/api/admin/users/%d/role", int(loyalUserID)), map[string]any{
			"role": "moderator",
		}, adminToken)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("Admin can promote a user to admin role", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		adminToken := app.CreateAdmin("Super Admin", "super@test.com", "pass123")
		app.CreateUser("New Admin", "newadmin@test.com", "pass123", "Catholic")

		resp := app.Request("GET", "/api/admin/users", nil, adminToken)
		users := testutil.ParseJSONArray(resp)
		var targetID float64
		for _, u := range users {
			if u["email"] == "newadmin@test.com" {
				targetID = u["id"].(float64)
				break
			}
		}

		resp = app.Request("PUT", fmt.Sprintf("/api/admin/users/%d/role", int(targetID)), map[string]any{
			"role": "admin",
		}, adminToken)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("Regular user cannot access admin user management", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Regular", "reg@test.com", "pass123", "Catholic")

		resp := app.Request("GET", "/api/admin/users", nil, token)
		if resp.Code != http.StatusForbidden {
			t.Errorf("regular user should be forbidden, got %d", resp.Code)
		}
	})

	t.Run("Moderator cannot access admin user management", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		modToken := app.CreateModerator("Mod", "mod@test.com", "pass123")

		resp := app.Request("GET", "/api/admin/users", nil, modToken)
		if resp.Code != http.StatusForbidden {
			t.Errorf("moderator should be forbidden from admin routes, got %d", resp.Code)
		}
	})
}

func TestAdminChurchManagement(t *testing.T) {
	t.Run("Admin can verify a user-submitted church", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		adminToken := app.CreateAdmin("Admin", "admin@test.com", "pass123")
		churchID := app.SeedChurch("Unverified Church", "Catholic", "Addr", -23.0, -46.0)

		// Set as unverified
		app.DB.Exec("UPDATE churches SET verified = false WHERE id = ?", churchID)

		resp := app.Request("PUT", fmt.Sprintf("/api/admin/churches/%d/verify", churchID), nil, adminToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("Admin can delete a church entry", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		adminToken := app.CreateAdmin("Admin", "admindel@test.com", "pass123")
		churchID := app.SeedChurch("To Delete", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("DELETE", fmt.Sprintf("/api/admin/churches/%d", churchID), nil, adminToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}

		// Verify it's gone
		userToken := app.CreateUser("Checker", "checker@test.com", "pass123", "Catholic")
		resp = app.Request("GET", fmt.Sprintf("/api/churches/%d", churchID), nil, userToken)
		if resp.Code != http.StatusNotFound {
			t.Errorf("deleted church should return 404, got %d", resp.Code)
		}
	})

	t.Run("Regular user cannot verify or delete churches via admin routes", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Regular", "regadmin@test.com", "pass123", "Catholic")
		churchID := app.SeedChurch("Protected", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("PUT", fmt.Sprintf("/api/admin/churches/%d/verify", churchID), nil, token)
		if resp.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", resp.Code)
		}

		resp = app.Request("DELETE", fmt.Sprintf("/api/admin/churches/%d", churchID), nil, token)
		if resp.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", resp.Code)
		}
	})
}
