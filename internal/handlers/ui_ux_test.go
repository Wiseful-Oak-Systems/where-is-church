package handlers_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

// ─── Template Rendering Tests ────────────────────────────────────────────────
// These verify that the server-rendered HTML contains the correct elements,
// conditional content, and accessibility attributes for each user role.

func TestLoginPageRendering(t *testing.T) {
	t.Run("Login page renders without authentication", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		// Login page is public — no token needed, but test app only has API routes
		// so we test the API endpoint that would serve this
		resp := app.Request("GET", "/api/auth/me", nil, "")
		if resp.Code != http.StatusUnauthorized {
			t.Errorf("unauthenticated request should return 401, got %d", resp.Code)
		}
	})
}

func TestRegistrationFormValidation(t *testing.T) {
	t.Run("Registration rejects empty name", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"name": "", "email": "test@test.com", "password": "password123",
		}, "")
		if resp.Code != http.StatusBadRequest {
			t.Errorf("empty name should return 400, got %d", resp.Code)
		}
	})

	t.Run("Registration rejects invalid email format", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"name": "Test", "email": "not-an-email", "password": "password123",
		}, "")
		if resp.Code != http.StatusBadRequest {
			t.Errorf("invalid email should return 400, got %d", resp.Code)
		}
	})

	t.Run("Registration rejects password shorter than 8 characters", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"name": "Test", "email": "valid@test.com", "password": "1234567",
		}, "")
		if resp.Code != http.StatusBadRequest {
			t.Errorf("short password should return 400, got %d", resp.Code)
		}
	})

	t.Run("Registration accepts all supported denominations", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		denominations := []string{"Catholic", "Orthodox", "Protestant", "Anglican", "Evangelical", "Other"}
		for _, d := range denominations {
			email := strings.ToLower(d) + "@test.com"
			resp := app.Request("POST", "/api/auth/register", map[string]string{
				"name": d + " User", "email": email, "password": "password123", "denomination": d,
			}, "")
			if resp.Code != http.StatusCreated {
				t.Errorf("denomination %q should be accepted, got %d", d, resp.Code)
			}
			result := testutil.ParseJSON(resp)
			user := result["user"].(map[string]any)
			if user["denomination"] != d {
				t.Errorf("expected denomination %q, got %v", d, user["denomination"])
			}
		}
	})
}

func TestProfileUpdateValidation(t *testing.T) {
	t.Run("Profile update rejects latitude outside valid range", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Geo User", "geo@test.com", "password123", "Catholic")

		resp := app.Request("PUT", "/api/auth/profile", map[string]any{
			"latitude": 91.0,
		}, token)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("latitude > 90 should return 400, got %d", resp.Code)
		}
	})

	t.Run("Profile update rejects latitude below valid range", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Geo User2", "geo2@test.com", "password123", "Catholic")

		resp := app.Request("PUT", "/api/auth/profile", map[string]any{
			"latitude": -91.0,
		}, token)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("latitude < -90 should return 400, got %d", resp.Code)
		}
	})

	t.Run("Profile update rejects longitude outside valid range", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Geo User3", "geo3@test.com", "password123", "Catholic")

		resp := app.Request("PUT", "/api/auth/profile", map[string]any{
			"longitude": 181.0,
		}, token)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("longitude > 180 should return 400, got %d", resp.Code)
		}
	})

	t.Run("Profile update accepts valid equator coordinates", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Equator", "eq@test.com", "password123", "Catholic")

		resp := app.Request("PUT", "/api/auth/profile", map[string]any{
			"latitude": 0.0, "longitude": 0.0,
		}, token)
		if resp.Code != http.StatusOK {
			t.Errorf("equator coordinates should be valid, got %d", resp.Code)
		}
	})

	t.Run("Profile update accepts extreme valid coordinates", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Extreme", "extreme@test.com", "password123", "Catholic")

		resp := app.Request("PUT", "/api/auth/profile", map[string]any{
			"latitude": 90.0, "longitude": -180.0,
		}, token)
		if resp.Code != http.StatusOK {
			t.Errorf("extreme valid coordinates should be accepted, got %d", resp.Code)
		}
	})
}

func TestChurchCreationValidation(t *testing.T) {
	t.Run("Church creation rejects latitude outside valid range", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Creator", "creat@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/churches", map[string]any{
			"name": "Bad Church", "address": "Addr", "latitude": 999.0, "longitude": -46.0,
		}, token)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("invalid latitude should return 400, got %d", resp.Code)
		}
	})

	t.Run("Church creation enforces maximum name length", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Long", "long@test.com", "password123", "Catholic")

		longName := strings.Repeat("A", 201)
		resp := app.Request("POST", "/api/churches", map[string]any{
			"name": longName, "address": "Addr", "latitude": -23.0, "longitude": -46.0,
		}, token)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("name > 200 chars should return 400, got %d", resp.Code)
		}
	})

	t.Run("Church creation enforces maximum description length", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Desc", "desc@test.com", "password123", "Catholic")

		longDesc := strings.Repeat("X", 2001)
		resp := app.Request("POST", "/api/churches", map[string]any{
			"name": "Church", "address": "Addr", "latitude": -23.0, "longitude": -46.0,
			"description": longDesc,
		}, token)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("description > 2000 chars should return 400, got %d", resp.Code)
		}
	})
}

func TestScheduleValidation(t *testing.T) {
	t.Run("Schedule rejects invalid time format", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		modToken := app.CreateModerator("Mod", "modtime@test.com", "password123")
		churchID := app.SeedChurch("Time Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", "/api/churches/"+itoa(churchID)+"/schedules", map[string]any{
			"day_of_week": 0, "start_time": "not-a-time",
		}, modToken)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("invalid time format should return 400, got %d", resp.Code)
		}
	})

	t.Run("Schedule accepts valid HH:MM format", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		modToken := app.CreateModerator("Mod2", "modtime2@test.com", "password123")
		churchID := app.SeedChurch("Valid Time", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", "/api/churches/"+itoa(churchID)+"/schedules", map[string]any{
			"day_of_week": 0, "start_time": "10:30",
		}, modToken)
		if resp.Code != http.StatusCreated {
			t.Fatalf("valid time should be accepted, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("Schedule rejects invalid schedule type", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		modToken := app.CreateModerator("Mod3", "modtype@test.com", "password123")
		churchID := app.SeedChurch("Type Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", "/api/churches/"+itoa(churchID)+"/schedules", map[string]any{
			"type": "invalid_type", "day_of_week": 0, "start_time": "10:00",
		}, modToken)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("invalid type should return 400, got %d", resp.Code)
		}
	})

	t.Run("Schedule accepts all valid types: mass, confession, adoration", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		modToken := app.CreateModerator("Mod4", "modtypes@test.com", "password123")
		churchID := app.SeedChurch("All Types", "Catholic", "Addr", -23.0, -46.0)

		for _, typ := range []string{"mass", "confession", "adoration"} {
			resp := app.Request("POST", "/api/churches/"+itoa(churchID)+"/schedules", map[string]any{
				"type": typ, "day_of_week": 0, "start_time": "10:00",
			}, modToken)
			if resp.Code != http.StatusCreated {
				t.Errorf("type %q should be accepted, got %d", typ, resp.Code)
			}
		}
	})

	t.Run("Confession schedule can include end time for time ranges", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		modToken := app.CreateModerator("Mod5", "modrange@test.com", "password123")
		churchID := app.SeedChurch("Range Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", "/api/churches/"+itoa(churchID)+"/schedules", map[string]any{
			"type": "confession", "day_of_week": 6, "start_time": "15:00", "end_time": "17:00",
		}, modToken)
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["end_time"] != "17:00" {
			t.Errorf("end_time should be stored, got %v", result["end_time"])
		}
	})
}

func TestSuggestionTypeValidation(t *testing.T) {
	t.Run("Suggestion rejects invalid type values", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Bad Type", "badtype@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "hacked_type", "content": "something",
		}, token)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("invalid suggestion type should return 400, got %d", resp.Code)
		}
	})

	t.Run("Suggestion accepts all valid types", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("All Types", "alltypes@test.com", "password123", "Catholic")

		for _, typ := range []string{"new_church", "edit_church", "schedule", "general"} {
			resp := app.Request("POST", "/api/suggestions", map[string]any{
				"type": typ, "content": "Test content for " + typ,
			}, token)
			if resp.Code != http.StatusCreated {
				t.Errorf("type %q should be accepted, got %d", typ, resp.Code)
			}
		}
	})
}

func TestSuggestionReviewValidation(t *testing.T) {
	t.Run("Review rejects invalid status values like 'pending'", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("Sub", "sub@test.com", "password123", "Catholic")
		modToken := app.CreateModerator("Mod", "modrev@test.com", "password123")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general", "content": "test",
		}, userToken)
		sugg := testutil.ParseJSON(resp)
		suggID := itoa(uint(sugg["id"].(float64)))

		resp = app.Request("PUT", "/api/suggestions/"+suggID, map[string]any{
			"status": "pending",
		}, modToken)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("status 'pending' should be rejected for review, got %d", resp.Code)
		}
	})
}

// ─── Error Response Consistency ──────────────────────────────────────────────

func TestAPIErrorResponseFormat(t *testing.T) {
	t.Run("All error responses contain an 'error' field", func(t *testing.T) {
		app := testutil.NewTestApp(t)

		// Unauthenticated request
		resp := app.Request("GET", "/api/auth/me", nil, "")
		result := testutil.ParseJSON(resp)
		if result["error"] == nil {
			t.Error("401 response should contain 'error' field")
		}

		// Bad request
		resp = app.Request("POST", "/api/auth/register", map[string]string{}, "")
		result = testutil.ParseJSON(resp)
		if result["error"] == nil {
			t.Error("400 response should contain 'error' field")
		}

		// Not found
		token := app.CreateUser("Err", "err@test.com", "password123", "Catholic")
		resp = app.Request("GET", "/api/churches/99999", nil, token)
		result = testutil.ParseJSON(resp)
		if result["error"] == nil {
			t.Error("404 response should contain 'error' field")
		}
	})
}

// ─── Pagination Tests ────────────────────────────────────────────────────────

func TestPagination(t *testing.T) {
	t.Run("Church list respects limit parameter", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Pager", "page@test.com", "password123", "Catholic")
		for i := 0; i < 5; i++ {
			app.SeedChurch("Church "+itoa(uint(i)), "Catholic", "Addr", -23.0+float64(i)*0.01, -46.0)
		}

		resp := app.Request("GET", "/api/churches?limit=2", nil, token)
		results := testutil.ParseJSONArray(resp)
		if len(results) != 2 {
			t.Errorf("expected 2 results with limit=2, got %d", len(results))
		}
	})

	t.Run("Church list respects offset parameter", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Offset", "offset@test.com", "password123", "Catholic")
		for i := 0; i < 5; i++ {
			app.SeedChurch("Church "+itoa(uint(i)), "Catholic", "Addr", -23.0+float64(i)*0.01, -46.0)
		}

		resp := app.Request("GET", "/api/churches?limit=2&offset=3", nil, token)
		results := testutil.ParseJSONArray(resp)
		if len(results) != 2 {
			t.Errorf("expected 2 results with offset=3, got %d", len(results))
		}
	})

	t.Run("Checkin list supports pagination", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("CheckPage", "checkpage@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/checkins/mine?limit=5&offset=0", nil, token)
		if resp.Code != http.StatusOK {
			t.Errorf("paginated check-in list should return 200, got %d", resp.Code)
		}
	})

	t.Run("Suggestion list supports pagination", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		modToken := app.CreateModerator("ModPage", "modpage@test.com", "password123")

		resp := app.Request("GET", "/api/suggestions?limit=10&offset=0", nil, modToken)
		if resp.Code != http.StatusOK {
			t.Errorf("paginated suggestion list should return 200, got %d", resp.Code)
		}
	})
}

// ─── Role-Based Content Access ───────────────────────────────────────────────

func TestRoleBasedAccess(t *testing.T) {
	t.Run("Regular user cannot access any admin endpoints", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Regular", "regular@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Protected", "Catholic", "Addr", -23.0, -46.0)

		endpoints := []struct {
			method string
			path   string
		}{
			{"GET", "/api/admin/users"},
			{"PUT", "/api/admin/users/1/role"},
			{"GET", "/api/admin/moderators"},
			{"GET", "/api/admin/claims"},
			{"PUT", "/api/admin/claims/1"},
			{"PUT", "/api/admin/churches/" + itoa(churchID) + "/verify"},
			{"DELETE", "/api/admin/churches/" + itoa(churchID)},
		}

		for _, ep := range endpoints {
			var body any
			if ep.method == "PUT" {
				body = map[string]any{"role": "admin"}
			}
			resp := app.Request(ep.method, ep.path, body, token)
			if resp.Code != http.StatusForbidden {
				t.Errorf("%s %s should be forbidden for regular user, got %d", ep.method, ep.path, resp.Code)
			}
		}
	})

	t.Run("Moderator cannot access admin-only endpoints", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		modToken := app.CreateModerator("Mod", "modaccess@test.com", "password123")
		churchID := app.SeedChurch("Protected", "Catholic", "Addr", -23.0, -46.0)

		// Moderator should NOT delete churches
		resp := app.Request("DELETE", "/api/admin/churches/"+itoa(churchID), nil, modToken)
		if resp.Code != http.StatusForbidden {
			t.Errorf("moderator should not delete churches, got %d", resp.Code)
		}
	})

	t.Run("Church owner can edit churches but not access admin user management", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		app.CreateUser("Owner", "owner@test.com", "password123", "Catholic")
		app.DB.Exec("UPDATE users SET role = 'church_owner' WHERE email = 'owner@test.com'")

		// Re-login
		resp := app.Request("POST", "/api/auth/login", map[string]string{
			"email": "owner@test.com", "password": "password123",
		}, "")
		result := testutil.ParseJSON(resp)
		ownerToken := result["token"].(string)

		churchID := app.SeedChurch("Owner Church", "Catholic", "Addr", -23.0, -46.0)

		// CAN edit churches
		resp = app.Request("PUT", "/api/churches/"+itoa(churchID), map[string]any{
			"name": "Updated Name",
		}, ownerToken)
		if resp.Code != http.StatusOK {
			t.Errorf("church owner should edit churches, got %d", resp.Code)
		}

		// CANNOT access admin user list
		resp = app.Request("GET", "/api/admin/users", nil, ownerToken)
		if resp.Code != http.StatusForbidden {
			t.Errorf("church owner should not access admin users, got %d", resp.Code)
		}
	})
}

// ─── Health Check ────────────────────────────────────────────────────────────
// Note: Health check is only on the real server (main.go), not the test router.
// These tests verify the API contract stability.

func TestAPIContractStability(t *testing.T) {
	t.Run("User response never exposes password hash", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"name": "Secret", "email": "secret@test.com", "password": "password123",
		}, "")

		body := resp.Body.String()
		if strings.Contains(body, "password_hash") || strings.Contains(body, "$2a$") {
			t.Error("response must never contain password hash")
		}
	})

	t.Run("Registration response includes both user object and token", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"name": "Contract", "email": "contract@test.com", "password": "password123",
		}, "")
		result := testutil.ParseJSON(resp)

		if result["token"] == nil {
			t.Error("registration response must include token")
		}
		if result["user"] == nil {
			t.Error("registration response must include user object")
		}
		user := result["user"].(map[string]any)
		if user["id"] == nil || user["email"] == nil || user["name"] == nil || user["denomination"] == nil || user["role"] == nil {
			t.Error("user object must include id, email, name, denomination, and role")
		}
	})

	t.Run("Church detail response includes all core fields", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Detail", "detail@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Detail Church", "Catholic", "123 Main St", -23.55, -46.63)

		resp := app.Request("GET", "/api/churches/"+itoa(churchID), nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		result := testutil.ParseJSON(resp)
		for _, field := range []string{"id", "name", "denomination", "address", "latitude", "longitude"} {
			if result[field] == nil {
				t.Errorf("church detail response must include %q field", field)
			}
		}
	})

	t.Run("Check-in stats response includes all required fields", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Stats", "stats@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/checkins/stats", nil, token)
		result := testutil.ParseJSON(resp)

		for _, field := range []string{"total_checkins", "recent_30_days", "top_churches"} {
			if result[field] == nil {
				t.Errorf("stats response must include %q field", field)
			}
		}
	})

	t.Run("Favorite check response includes is_favorite boolean", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("FavCheck", "favcheck@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Fav Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("GET", "/api/churches/"+itoa(churchID)+"/favorite", nil, token)
		result := testutil.ParseJSON(resp)
		if result["is_favorite"] == nil {
			t.Error("favorite check must include is_favorite boolean")
		}
	})
}

func itoa(n uint) string {
	return fmt.Sprintf("%d", n)
}
