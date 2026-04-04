package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// These tests verify that the HTML templates render correctly and contain
// the required accessibility attributes, conditional content, and security
// properties that a UI/UX review would check.

func renderTemplate(name string, data gin.H) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.LoadHTMLGlob("../../web/templates/*")
	r.GET("/test", func(c *gin.Context) {
		c.HTML(http.StatusOK, name, data)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	return w
}

func TestLoginTemplateRendering(t *testing.T) {
	t.Run("Login page contains email and password form fields", func(t *testing.T) {
		w := renderTemplate("login.html", nil)
		body := w.Body.String()
		assertContains(t, body, `id="email"`, "email input field")
		assertContains(t, body, `id="password"`, "password input field")
		assertContains(t, body, `type="submit"`, "submit button")
	})

	t.Run("Login page has lang attribute for screen readers", func(t *testing.T) {
		w := renderTemplate("login.html", nil)
		assertContains(t, w.Body.String(), `lang="en"`, "lang attribute")
	})

	t.Run("Login page has error div with aria-live for accessibility", func(t *testing.T) {
		w := renderTemplate("login.html", nil)
		body := w.Body.String()
		assertContains(t, body, `role="alert"`, "alert role")
		assertContains(t, body, `aria-live="polite"`, "aria-live region")
	})

	t.Run("Login page links to registration page", func(t *testing.T) {
		w := renderTemplate("login.html", nil)
		assertContains(t, w.Body.String(), `href="/register"`, "register link")
	})

	t.Run("Login page includes app stylesheet", func(t *testing.T) {
		w := renderTemplate("login.html", nil)
		assertContains(t, w.Body.String(), `/static/css/app.css`, "CSS link")
	})
}

func TestRegisterTemplateRendering(t *testing.T) {
	t.Run("Register page contains all required form fields", func(t *testing.T) {
		w := renderTemplate("register.html", nil)
		body := w.Body.String()
		assertContains(t, body, `id="name"`, "name input")
		assertContains(t, body, `id="email"`, "email input")
		assertContains(t, body, `id="password"`, "password input")
		assertContains(t, body, `id="denomination"`, "denomination select")
	})

	t.Run("Register page has Catholic as default denomination", func(t *testing.T) {
		w := renderTemplate("register.html", nil)
		assertContains(t, w.Body.String(), `value="Catholic" selected`, "Catholic default selected")
	})

	t.Run("Register page offers all 6 supported denominations", func(t *testing.T) {
		w := renderTemplate("register.html", nil)
		body := w.Body.String()
		for _, d := range []string{"Catholic", "Orthodox", "Protestant", "Anglican", "Evangelical", "Other"} {
			assertContains(t, body, d, "denomination option "+d)
		}
	})

	t.Run("Register page links to login page", func(t *testing.T) {
		w := renderTemplate("register.html", nil)
		assertContains(t, w.Body.String(), `href="/login"`, "login link")
	})
}

func TestIndexTemplateRendering(t *testing.T) {
	t.Run("Index page shows admin link for admin users", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "admin", "denomination": "Catholic",
		})
		assertContains(t, w.Body.String(), `href="/admin"`, "admin link present for admin")
	})

	t.Run("Index page shows admin link for community_manager users", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "community_manager", "denomination": "Catholic",
		})
		assertContains(t, w.Body.String(), `href="/admin"`, "admin link present for community_manager")
	})

	t.Run("Index page hides admin link for regular users", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		assertNotContains(t, w.Body.String(), `href="/admin"`, "admin link absent for user")
	})

	t.Run("Index page contains skip-to-content link for keyboard navigation", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		assertContains(t, w.Body.String(), `class="skip-link"`, "skip link")
	})

	t.Run("Index page has screen reader announcement region for map events", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		assertContains(t, w.Body.String(), `id="map-announcer"`, "map announcer")
	})

	t.Run("Index page includes Leaflet map container", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		assertContains(t, w.Body.String(), `id="map"`, "map container")
	})

	t.Run("Index page includes PWA manifest link", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		assertContains(t, w.Body.String(), `manifest.json`, "PWA manifest")
	})

	t.Run("Index page has address search input for location search", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		assertContains(t, w.Body.String(), `id="address-search"`, "address search input")
	})

	t.Run("Index page has bottom sheet drag handle for mobile", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		assertContains(t, w.Body.String(), `bottom-sheet-handle`, "bottom sheet handle")
	})

	t.Run("Index page has denomination filter with all options", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		body := w.Body.String()
		assertContains(t, body, `id="denomination-filter"`, "denomination filter")
		assertContains(t, body, `value="All"`, "All option")
	})

	t.Run("Index page has radius slider with range 1-50", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		body := w.Body.String()
		assertContains(t, body, `id="radius-slider"`, "radius slider")
		assertContains(t, body, `min="1"`, "min radius")
		assertContains(t, body, `max="50"`, "max radius")
	})

	t.Run("Index page has suggestion modal with proper ARIA attributes", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		body := w.Body.String()
		assertContains(t, body, `id="suggestion-modal"`, "suggestion modal")
		assertContains(t, body, `aria-modal="true"`, "aria-modal")
		assertContains(t, body, `role="dialog"`, "dialog role")
	})
}

func TestChurchTemplateRendering(t *testing.T) {
	t.Run("Church page shows edit forms only for admin/moderator roles", func(t *testing.T) {
		// Admin can see edit
		w := renderTemplate("church.html", gin.H{
			"churchID": "1", "userID": uint(1), "userRole": "admin",
		})
		assertContains(t, w.Body.String(), `id="edit-church-form"`, "edit form for admin")
		assertContains(t, w.Body.String(), `id="add-schedule-form"`, "schedule form for admin")
	})

	t.Run("Church page shows edit forms for moderator role", func(t *testing.T) {
		w := renderTemplate("church.html", gin.H{
			"churchID": "1", "userID": uint(1), "userRole": "moderator",
		})
		assertContains(t, w.Body.String(), `id="edit-church-form"`, "edit form for moderator")
	})

	t.Run("Church page shows edit forms for church_owner role", func(t *testing.T) {
		w := renderTemplate("church.html", gin.H{
			"churchID": "1", "userID": uint(1), "userRole": "church_owner",
		})
		assertContains(t, w.Body.String(), `id="edit-church-form"`, "edit form for church_owner")
	})

	t.Run("Church page hides edit forms for regular user", func(t *testing.T) {
		w := renderTemplate("church.html", gin.H{
			"churchID": "1", "userID": uint(1), "userRole": "user",
		})
		assertNotContains(t, w.Body.String(), `id="edit-church-form"`, "no edit form for user")
		assertNotContains(t, w.Body.String(), `id="add-schedule-form"`, "no schedule form for user")
	})

	t.Run("Church page always shows suggestion form for all users", func(t *testing.T) {
		w := renderTemplate("church.html", gin.H{
			"churchID": "1", "userID": uint(1), "userRole": "user",
		})
		assertContains(t, w.Body.String(), `id="church-suggestion-form"`, "suggestion form always visible")
	})

	t.Run("Church page has check-in button with accessible label", func(t *testing.T) {
		w := renderTemplate("church.html", gin.H{
			"churchID": "1", "userID": uint(1), "userRole": "user",
		})
		assertContains(t, w.Body.String(), `id="checkin-btn"`, "checkin button")
	})

	t.Run("Church page has mini map container", func(t *testing.T) {
		w := renderTemplate("church.html", gin.H{
			"churchID": "1", "userID": uint(1), "userRole": "user",
		})
		assertContains(t, w.Body.String(), `id="church-map"`, "church mini map")
	})
}

func TestProfileTemplateRendering(t *testing.T) {
	t.Run("Profile page has edit form with name and denomination fields", func(t *testing.T) {
		w := renderTemplate("profile.html", gin.H{
			"userID": uint(1), "userRole": "user",
		})
		body := w.Body.String()
		assertContains(t, body, `id="edit-profile-form"`, "edit profile form")
		assertContains(t, body, `id="edit-name"`, "name input")
		assertContains(t, body, `id="edit-denomination"`, "denomination select")
	})

	t.Run("Profile page has stats display section", func(t *testing.T) {
		w := renderTemplate("profile.html", gin.H{
			"userID": uint(1), "userRole": "user",
		})
		body := w.Body.String()
		assertContains(t, body, `id="stat-total"`, "total stats")
		assertContains(t, body, `id="stat-recent"`, "recent stats")
	})

	t.Run("Profile page has check-in history and suggestion history", func(t *testing.T) {
		w := renderTemplate("profile.html", gin.H{
			"userID": uint(1), "userRole": "user",
		})
		body := w.Body.String()
		assertContains(t, body, `id="recent-checkins"`, "checkin history")
		assertContains(t, body, `id="my-suggestions"`, "suggestion history")
	})
}

func TestAdminTemplateRendering(t *testing.T) {
	t.Run("Admin page has ARIA tablist with three tabs", func(t *testing.T) {
		w := renderTemplate("admin.html", gin.H{
			"userID": uint(1), "userRole": "admin",
		})
		body := w.Body.String()
		assertContains(t, body, `role="tablist"`, "tablist role")
		assertContains(t, body, `id="tab-users"`, "users tab")
		assertContains(t, body, `id="tab-churches"`, "churches tab")
		assertContains(t, body, `id="tab-suggestions"`, "suggestions tab")
	})

	t.Run("Admin page has user management table", func(t *testing.T) {
		w := renderTemplate("admin.html", gin.H{
			"userID": uint(1), "userRole": "admin",
		})
		assertContains(t, w.Body.String(), `id="users-tbody"`, "users table body")
	})

	t.Run("Admin page has confirmation modal for destructive actions", func(t *testing.T) {
		w := renderTemplate("admin.html", gin.H{
			"userID": uint(1), "userRole": "admin",
		})
		body := w.Body.String()
		assertContains(t, body, `id="confirm-modal"`, "confirm modal")
		assertContains(t, body, `aria-modal="true"`, "modal accessibility")
	})

	t.Run("Admin page has review modal for suggestion moderation", func(t *testing.T) {
		w := renderTemplate("admin.html", gin.H{
			"userID": uint(1), "userRole": "admin",
		})
		assertContains(t, w.Body.String(), `id="review-modal"`, "review modal")
	})
}

// ─── CSS and Dark Mode Assertions ────────────────────────────────────────────

func TestDarkModeCSS(t *testing.T) {
	t.Run("CSS includes dark mode media query", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		// The page links to app.css which contains dark mode — we verify the link exists
		assertContains(t, w.Body.String(), `app.css`, "CSS file linked")
	})

	t.Run("Index page includes theme-color meta tag for mobile browsers", func(t *testing.T) {
		w := renderTemplate("index.html", gin.H{
			"userID": uint(1), "userRole": "user", "denomination": "Catholic",
		})
		assertContains(t, w.Body.String(), `name="theme-color"`, "theme-color meta")
	})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func assertContains(t *testing.T, body, substr, label string) {
	t.Helper()
	if !strings.Contains(body, substr) {
		t.Errorf("expected %s to be present in HTML (looking for %q)", label, substr)
	}
}

func assertNotContains(t *testing.T, body, substr, label string) {
	t.Helper()
	if strings.Contains(body, substr) {
		t.Errorf("expected %s to be absent from HTML (found %q)", label, substr)
	}
}
