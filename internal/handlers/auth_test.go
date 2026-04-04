package handlers_test

import (
	"net/http"
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func TestUserRegistration(t *testing.T) {
	t.Run("New user can register with name, email, password, and denomination", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"name": "Maria Silva", "email": "maria@example.com",
			"password": "secret123", "denomination": "Catholic",
		}, "")

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["token"] == nil || result["token"] == "" {
			t.Error("registration should return a JWT token")
		}
		user := result["user"].(map[string]any)
		if user["name"] != "Maria Silva" {
			t.Errorf("expected name 'Maria Silva', got %v", user["name"])
		}
		if user["denomination"] != "Catholic" {
			t.Errorf("expected denomination 'Catholic', got %v", user["denomination"])
		}
	})

	t.Run("Registration defaults denomination to Catholic when not specified", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"name": "João", "email": "joao@example.com", "password": "secret123",
		}, "")

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.Code)
		}
		result := testutil.ParseJSON(resp)
		user := result["user"].(map[string]any)
		if user["denomination"] != "Catholic" {
			t.Errorf("denomination should default to 'Catholic', got %v", user["denomination"])
		}
	})

	t.Run("Registration with an already used email is rejected", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		app.CreateUser("First User", "duplicate@example.com", "pass123", "Catholic")

		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"name": "Second User", "email": "duplicate@example.com", "password": "pass456",
		}, "")

		if resp.Code != http.StatusConflict {
			t.Errorf("expected 409 conflict, got %d", resp.Code)
		}
	})

	t.Run("Registration requires name, email, and password", func(t *testing.T) {
		app := testutil.NewTestApp(t)

		// Missing name
		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"email": "x@test.com", "password": "123456",
		}, "")
		if resp.Code != http.StatusBadRequest {
			t.Errorf("missing name should return 400, got %d", resp.Code)
		}

		// Missing email
		resp = app.Request("POST", "/api/auth/register", map[string]string{
			"name": "Test", "password": "123456",
		}, "")
		if resp.Code != http.StatusBadRequest {
			t.Errorf("missing email should return 400, got %d", resp.Code)
		}

		// Missing password
		resp = app.Request("POST", "/api/auth/register", map[string]string{
			"name": "Test", "email": "x@test.com",
		}, "")
		if resp.Code != http.StatusBadRequest {
			t.Errorf("missing password should return 400, got %d", resp.Code)
		}
	})

	t.Run("Password must have at least 6 characters", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"name": "Test", "email": "short@test.com", "password": "12345",
		}, "")
		if resp.Code != http.StatusBadRequest {
			t.Errorf("short password should return 400, got %d", resp.Code)
		}
	})

	t.Run("New user gets the default 'user' role", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"name": "Normal User", "email": "normal@test.com", "password": "pass123",
		}, "")
		result := testutil.ParseJSON(resp)
		user := result["user"].(map[string]any)
		if user["role"] != "user" {
			t.Errorf("new user should get 'user' role, got %v", user["role"])
		}
	})

	t.Run("User can register with non-Catholic denomination", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		resp := app.Request("POST", "/api/auth/register", map[string]string{
			"name": "Orthodox User", "email": "orthodox@test.com",
			"password": "pass123", "denomination": "Orthodox",
		}, "")
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.Code)
		}
		result := testutil.ParseJSON(resp)
		user := result["user"].(map[string]any)
		if user["denomination"] != "Orthodox" {
			t.Errorf("expected 'Orthodox', got %v", user["denomination"])
		}
	})
}

func TestUserLogin(t *testing.T) {
	t.Run("Registered user can log in with correct credentials", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		app.CreateUser("Login User", "login@test.com", "mypassword", "Catholic")

		resp := app.Request("POST", "/api/auth/login", map[string]string{
			"email": "login@test.com", "password": "mypassword",
		}, "")

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["token"] == nil || result["token"] == "" {
			t.Error("login should return a JWT token")
		}
	})

	t.Run("Login with wrong password is rejected", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		app.CreateUser("User", "wrong@test.com", "correctpass", "Catholic")

		resp := app.Request("POST", "/api/auth/login", map[string]string{
			"email": "wrong@test.com", "password": "wrongpass",
		}, "")

		if resp.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.Code)
		}
	})

	t.Run("Login with non-existent email is rejected", func(t *testing.T) {
		app := testutil.NewTestApp(t)

		resp := app.Request("POST", "/api/auth/login", map[string]string{
			"email": "noone@test.com", "password": "whatever",
		}, "")

		if resp.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.Code)
		}
	})

	t.Run("Email comparison is case-insensitive", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		app.CreateUser("Case User", "case@test.com", "pass123", "Catholic")

		resp := app.Request("POST", "/api/auth/login", map[string]string{
			"email": "CASE@TEST.COM", "password": "pass123",
		}, "")

		if resp.Code != http.StatusOK {
			t.Errorf("email should be case-insensitive, got %d", resp.Code)
		}
	})
}

func TestUserProfile(t *testing.T) {
	t.Run("Authenticated user can view their own profile", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Profile User", "profile@test.com", "pass123", "Catholic")

		resp := app.Request("GET", "/api/auth/me", nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		result := testutil.ParseJSON(resp)
		if result["email"] != "profile@test.com" {
			t.Errorf("expected email 'profile@test.com', got %v", result["email"])
		}
	})

	t.Run("Authenticated user can update their name and denomination", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Old Name", "update@test.com", "pass123", "Catholic")

		resp := app.Request("PUT", "/api/auth/profile", map[string]string{
			"name": "New Name", "denomination": "Protestant",
		}, token)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["name"] != "New Name" {
			t.Errorf("expected name 'New Name', got %v", result["name"])
		}
		if result["denomination"] != "Protestant" {
			t.Errorf("expected denomination 'Protestant', got %v", result["denomination"])
		}
	})

	t.Run("User can set a custom location to override geolocation", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Location User", "loc@test.com", "pass123", "Catholic")

		resp := app.Request("PUT", "/api/auth/profile", map[string]any{
			"latitude": -23.5505, "longitude": -46.6333,
		}, token)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		result := testutil.ParseJSON(resp)
		if result["latitude"] == nil {
			t.Error("latitude should be set after profile update")
		}
	})

	t.Run("Unauthenticated user cannot access profile", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		resp := app.Request("GET", "/api/auth/me", nil, "")
		if resp.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.Code)
		}
	})
}

func TestLogout(t *testing.T) {
	t.Run("Authenticated user can log out", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Logout User", "logout@test.com", "pass123", "Catholic")

		resp := app.Request("POST", "/api/auth/logout", nil, token)
		if resp.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.Code)
		}
	})
}
