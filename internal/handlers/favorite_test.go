package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func TestFavorites(t *testing.T) {
	t.Run("User can bookmark a church as a favorite for quick access", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Bookmarker", "bm@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Favorite Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/favorite", churchID), nil, token)
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("User cannot favorite the same church twice", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Dup", "dup@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Dup Church", "Catholic", "Addr", -23.0, -46.0)

		app.Request("POST", fmt.Sprintf("/api/churches/%d/favorite", churchID), nil, token)
		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/favorite", churchID), nil, token)
		if resp.Code != http.StatusConflict {
			t.Errorf("duplicate favorite should return 409, got %d", resp.Code)
		}
	})

	t.Run("User can remove a church from their favorites", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Remover", "rem@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Remove Church", "Catholic", "Addr", -23.0, -46.0)

		app.Request("POST", fmt.Sprintf("/api/churches/%d/favorite", churchID), nil, token)
		resp := app.Request("DELETE", fmt.Sprintf("/api/churches/%d/favorite", churchID), nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("User can list all their favorited churches", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Lister", "list@test.com", "password123", "Catholic")
		c1 := app.SeedChurch("Fav 1", "Catholic", "Addr 1", -23.0, -46.0)
		c2 := app.SeedChurch("Fav 2", "Catholic", "Addr 2", -23.1, -46.1)

		app.Request("POST", fmt.Sprintf("/api/churches/%d/favorite", c1), nil, token)
		app.Request("POST", fmt.Sprintf("/api/churches/%d/favorite", c2), nil, token)

		resp := app.Request("GET", "/api/favorites", nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		if len(results) != 2 {
			t.Errorf("expected 2 favorites, got %d", len(results))
		}
	})

	t.Run("User can check if a specific church is in their favorites", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Checker", "check@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Check Church", "Catholic", "Addr", -23.0, -46.0)

		// Not yet favorited
		resp := app.Request("GET", fmt.Sprintf("/api/churches/%d/favorite", churchID), nil, token)
		result := testutil.ParseJSON(resp)
		if result["is_favorite"] != false {
			t.Error("should not be a favorite yet")
		}

		// Favorite it
		app.Request("POST", fmt.Sprintf("/api/churches/%d/favorite", churchID), nil, token)

		// Now check again
		resp = app.Request("GET", fmt.Sprintf("/api/churches/%d/favorite", churchID), nil, token)
		result = testutil.ParseJSON(resp)
		if result["is_favorite"] != true {
			t.Error("should be a favorite after adding")
		}
	})

	t.Run("Favoriting a non-existent church returns 404", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Ghost", "ghost@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/churches/99999/favorite", nil, token)
		if resp.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", resp.Code)
		}
	})
}
