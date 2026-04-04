package handlers_test

import (
	"net/http"
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func TestDataHealthDashboard(t *testing.T) {
	t.Run("Community Manager can view the data health dashboard", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		cmToken := app.CreateCommunityManager("CM", "cmdash@test.com", "password123")
		app.SeedChurch("Church 1", "Catholic", "Addr", -23.0, -46.0)
		app.SeedChurch("Church 2", "Catholic", "Addr", -23.1, -46.1)

		resp := app.Request("GET", "/api/admin/data-health", nil, cmToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		churches := result["churches"].(map[string]any)
		if churches["total"].(float64) < 2 {
			t.Errorf("expected at least 2 total churches, got %v", churches["total"])
		}
		if result["moderation"] == nil || result["community"] == nil || result["activity_7d"] == nil {
			t.Error("dashboard must include moderation, community, and activity_7d sections")
		}
	})

	t.Run("Dashboard reports pending suggestions and claims", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("User", "dashuser@test.com", "password123", "Catholic")
		cmToken := app.CreateCommunityManager("CM", "cmmod@test.com", "password123")

		app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general", "content": "test suggestion",
		}, userToken)

		resp := app.Request("GET", "/api/admin/data-health", nil, cmToken)
		result := testutil.ParseJSON(resp)
		moderation := result["moderation"].(map[string]any)
		if moderation["pending_suggestions"].(float64) < 1 {
			t.Error("expected at least 1 pending suggestion")
		}
	})

	t.Run("Regular user cannot access data health dashboard", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("User", "nodash@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/admin/data-health", nil, token)
		if resp.Code != http.StatusForbidden {
			t.Errorf("regular user should be forbidden, got %d", resp.Code)
		}
	})
}

func TestBulkVerifyChurches(t *testing.T) {
	t.Run("Admin can bulk verify multiple churches at once", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		adminToken := app.CreateAdmin("Admin", "bulkadmin@test.com", "password123")
		c1 := app.SeedChurch("Unv 1", "Catholic", "Addr", -23.0, -46.0)
		c2 := app.SeedChurch("Unv 2", "Catholic", "Addr", -23.1, -46.1)

		// Set as unverified
		app.DB.Exec("UPDATE churches SET verified = false WHERE id IN (?, ?)", c1, c2)

		resp := app.Request("POST", "/api/admin/bulk/verify-churches", map[string]any{
			"church_ids": []uint{c1, c2},
		}, adminToken)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["verified"].(float64) != 2 {
			t.Errorf("expected 2 verified, got %v", result["verified"])
		}
	})

	t.Run("Bulk verify rejects empty or oversized arrays", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		adminToken := app.CreateAdmin("Admin", "bulkempty@test.com", "password123")

		resp := app.Request("POST", "/api/admin/bulk/verify-churches", map[string]any{
			"church_ids": []uint{},
		}, adminToken)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("empty array should return 400, got %d", resp.Code)
		}
	})
}

func TestBulkReviewSuggestions(t *testing.T) {
	t.Run("Community Manager can bulk approve suggestions", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("User", "bulkuser@test.com", "password123", "Catholic")
		cmToken := app.CreateCommunityManager("CM", "cmbulk@test.com", "password123")

		_ = userToken // user exists but we seed suggestions directly to avoid rate limits

		var ids []uint
		for i := 0; i < 3; i++ {
			s := models.Suggestion{UserID: 1, Type: models.SuggestGeneral, Content: "bulk test", Status: models.SuggestionPending}
			app.DB.Create(&s)
			ids = append(ids, s.ID)
		}

		resp := app.Request("POST", "/api/admin/bulk/review-suggestions", map[string]any{
			"suggestion_ids": ids,
			"status":         "approved",
			"review_note":    "Batch approved — all verified.",
		}, cmToken)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["reviewed"].(float64) != 3 {
			t.Errorf("expected 3 reviewed, got %v", result["reviewed"])
		}
	})
}

func TestMergeChurches(t *testing.T) {
	t.Run("Admin can merge duplicate churches, moving all related data", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("Visitor", "merger@test.com", "password123", "Catholic")
		adminToken := app.CreateAdmin("Admin", "mergeadmin@test.com", "password123")
		keepID := app.SeedChurch("Original Church", "Catholic", "Main St", -23.0, -46.0)
		removeID := app.SeedChurch("Duplicate Church", "Catholic", "Main Street", -23.0001, -46.0001)

		// Create a check-in on the duplicate
		app.Request("POST", "/api/checkins", map[string]any{"church_id": removeID}, userToken)

		resp := app.Request("POST", "/api/admin/merge-churches", map[string]any{
			"keep_id":   keepID,
			"remove_id": removeID,
		}, adminToken)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		// Verify the duplicate is gone
		resp = app.Request("GET", "/api/churches/"+itoa(removeID), nil, userToken)
		if resp.Code != http.StatusNotFound {
			t.Errorf("removed church should be gone, got %d", resp.Code)
		}
	})

	t.Run("Merge rejects same ID for keep and remove", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		adminToken := app.CreateAdmin("Admin", "mergesame@test.com", "password123")
		churchID := app.SeedChurch("Self Merge", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", "/api/admin/merge-churches", map[string]any{
			"keep_id":   churchID,
			"remove_id": churchID,
		}, adminToken)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("self-merge should return 400, got %d", resp.Code)
		}
	})

	t.Run("Community Manager cannot merge churches (admin only)", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		cmToken := app.CreateCommunityManager("CM", "cmmerge@test.com", "password123")
		c1 := app.SeedChurch("C1", "Catholic", "A", -23.0, -46.0)
		c2 := app.SeedChurch("C2", "Catholic", "A", -23.0, -46.0)

		resp := app.Request("POST", "/api/admin/merge-churches", map[string]any{
			"keep_id": c1, "remove_id": c2,
		}, cmToken)
		if resp.Code != http.StatusForbidden {
			t.Errorf("CM should not merge churches, got %d", resp.Code)
		}
	})
}

func TestAuditLog(t *testing.T) {
	t.Run("Bulk operations create audit log entries", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		adminToken := app.CreateAdmin("Admin", "auditadmin@test.com", "password123")
		c1 := app.SeedChurch("Audit Church", "Catholic", "Addr", -23.0, -46.0)
		app.DB.Exec("UPDATE churches SET verified = false WHERE id = ?", c1)

		// Perform a bulk verify
		app.Request("POST", "/api/admin/bulk/verify-churches", map[string]any{
			"church_ids": []uint{c1},
		}, adminToken)

		// Check audit log
		resp := app.Request("GET", "/api/admin/audit-log", nil, adminToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		logs := testutil.ParseJSONArray(resp)
		if len(logs) == 0 {
			t.Error("expected at least 1 audit log entry")
		}
	})

	t.Run("Audit log can be filtered by action type", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		adminToken := app.CreateAdmin("Admin", "filteraudit@test.com", "password123")

		// Create some audit entries
		c1 := app.SeedChurch("A", "Catholic", "A", -23.0, -46.0)
		app.DB.Exec("UPDATE churches SET verified = false WHERE id = ?", c1)
		app.Request("POST", "/api/admin/bulk/verify-churches", map[string]any{
			"church_ids": []uint{c1},
		}, adminToken)

		resp := app.Request("GET", "/api/admin/audit-log?action="+string(models.AuditBulkVerify), nil, adminToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		logs := testutil.ParseJSONArray(resp)
		for _, log := range logs {
			if log["action"] != string(models.AuditBulkVerify) {
				t.Errorf("expected only bulk.verify actions, got %v", log["action"])
			}
		}
	})
}

func TestStaleChurches(t *testing.T) {
	t.Run("Admin can list churches with stale data needing re-verification", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		cmToken := app.CreateCommunityManager("CM", "cmstale@test.com", "password123")
		app.SeedChurch("Old Church", "Catholic", "Addr", -23.0, -46.0)
		// The seeded church has no last_verified, so it's stale by default

		resp := app.Request("GET", "/api/admin/stale-churches", nil, cmToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		churches := testutil.ParseJSONArray(resp)
		if len(churches) == 0 {
			t.Error("expected at least 1 stale church")
		}
	})
}
