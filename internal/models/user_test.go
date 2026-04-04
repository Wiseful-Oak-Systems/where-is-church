package models_test

import (
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/models"
)

func TestUserPassword(t *testing.T) {
	t.Run("User can set a password and it is stored as a bcrypt hash", func(t *testing.T) {
		user := &models.User{}
		err := user.SetPassword("secure123")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if user.Password == "" {
			t.Fatal("expected password hash to be set")
		}
		if user.Password == "secure123" {
			t.Fatal("password must not be stored in plain text")
		}
	})

	t.Run("User can verify their password after setting it", func(t *testing.T) {
		user := &models.User{}
		user.SetPassword("mypassword")

		if !user.CheckPassword("mypassword") {
			t.Fatal("correct password should be accepted")
		}
	})

	t.Run("Wrong password is rejected", func(t *testing.T) {
		user := &models.User{}
		user.SetPassword("correctpassword")

		if user.CheckPassword("wrongpassword") {
			t.Fatal("incorrect password should be rejected")
		}
	})

	t.Run("Empty password is rejected when checking against a hashed password", func(t *testing.T) {
		user := &models.User{}
		user.SetPassword("something")

		if user.CheckPassword("") {
			t.Fatal("empty password should be rejected")
		}
	})
}

func TestUserRoles(t *testing.T) {
	t.Run("Default role constants are correctly defined", func(t *testing.T) {
		if models.RoleUser != "user" {
			t.Errorf("expected RoleUser to be 'user', got %q", models.RoleUser)
		}
		if models.RoleModerator != "moderator" {
			t.Errorf("expected RoleModerator to be 'moderator', got %q", models.RoleModerator)
		}
		if models.RoleAdmin != "admin" {
			t.Errorf("expected RoleAdmin to be 'admin', got %q", models.RoleAdmin)
		}
	})
}

func TestUserDenomination(t *testing.T) {
	t.Run("User struct supports denomination field for religious preference", func(t *testing.T) {
		user := models.User{
			Name:         "Maria",
			Email:        "maria@example.com",
			Denomination: "Catholic",
		}
		if user.Denomination != "Catholic" {
			t.Errorf("expected denomination 'Catholic', got %q", user.Denomination)
		}
	})
}

func TestUserLocationFields(t *testing.T) {
	t.Run("User can have optional latitude and longitude for custom location", func(t *testing.T) {
		lat := -23.5505
		lng := -46.6333
		user := models.User{
			Name:      "Carlos",
			Latitude:  &lat,
			Longitude: &lng,
		}
		if user.Latitude == nil || *user.Latitude != lat {
			t.Error("expected latitude to be set")
		}
		if user.Longitude == nil || *user.Longitude != lng {
			t.Error("expected longitude to be set")
		}
	})

	t.Run("User location is optional and can be nil", func(t *testing.T) {
		user := models.User{Name: "João"}
		if user.Latitude != nil {
			t.Error("latitude should be nil by default")
		}
	})
}
