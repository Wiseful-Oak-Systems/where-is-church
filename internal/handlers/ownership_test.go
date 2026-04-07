package handlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func TestChurchOwnershipClaim(t *testing.T) {
	t.Run("User can claim ownership of a church by providing evidence", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Parish Admin", "parish@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("St. Mary's", "Catholic", "Main St", -23.0, -46.0)

		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/claim", churchID), map[string]any{
			"evidence": "I am the parish secretary. Contact: (11) 9999-0000",
		}, token)

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["status"] != "pending" {
			t.Errorf("new claim should be pending, got %v", result["status"])
		}
		if result["evidence"] == nil || result["evidence"] == "" {
			t.Error("evidence should be stored with the claim")
		}
	})

	t.Run("User cannot claim the same church twice", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Dup Claimer", "dup@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Dup Church", "Catholic", "Addr", -23.0, -46.0)

		app.Request("POST", fmt.Sprintf("/api/churches/%d/claim", churchID), map[string]any{
			"evidence": "I am the pastor",
		}, token)

		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/claim", churchID), map[string]any{
			"evidence": "Second attempt",
		}, token)

		if resp.Code != http.StatusConflict {
			t.Errorf("duplicate claim should return 409, got %d", resp.Code)
		}
	})

	t.Run("Claim requires evidence explaining the ownership relationship", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("No Evidence", "noev@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Evidence Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/claim", churchID), map[string]any{}, token)
		if resp.Code != http.StatusBadRequest {
			t.Errorf("missing evidence should return 400, got %d", resp.Code)
		}
	})

	t.Run("User can view their own ownership claims and their statuses", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Claims User", "claims@test.com", "password123", "Catholic")
		c1 := app.SeedChurch("Claim Church 1", "Catholic", "Addr", -23.0, -46.0)
		c2 := app.SeedChurch("Claim Church 2", "Catholic", "Addr", -23.1, -46.1)

		app.Request("POST", fmt.Sprintf("/api/churches/%d/claim", c1), map[string]any{"evidence": "E1"}, token)
		app.Request("POST", fmt.Sprintf("/api/churches/%d/claim", c2), map[string]any{"evidence": "E2"}, token)

		resp := app.Request("GET", "/api/my-churches/claims", nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		if len(results) != 2 {
			t.Errorf("expected 2 claims, got %d", len(results))
		}
	})
}

func TestClaimReviewByCommunityManager(t *testing.T) {
	t.Run("Community Manager can list all pending ownership claims", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("Claimer", "claimer@test.com", "password123", "Catholic")
		cmToken := app.CreateCommunityManager("CM", "cm@test.com", "password123")
		churchID := app.SeedChurch("Pending Church", "Catholic", "Addr", -23.0, -46.0)

		app.Request("POST", fmt.Sprintf("/api/churches/%d/claim", churchID), map[string]any{
			"evidence": "I run this parish",
		}, userToken)

		resp := app.Request("GET", "/api/admin/claims?status=pending", nil, cmToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		if len(results) == 0 {
			t.Error("expected at least 1 pending claim")
		}
	})

	t.Run("Community Manager can approve a claim, promoting the user to church_owner", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("Future Owner", "owner@test.com", "password123", "Catholic")
		cmToken := app.CreateCommunityManager("CM Approver", "cma@test.com", "password123")
		churchID := app.SeedChurch("Approved Church", "Catholic", "Addr", -23.0, -46.0)

		// Submit claim
		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/claim", churchID), map[string]any{
			"evidence": "I am the parish priest, Fr. João",
		}, userToken)
		claim := testutil.ParseJSON(resp)
		claimID := int(claim["id"].(float64))

		// Approve
		resp = app.Request("PUT", fmt.Sprintf("/api/admin/claims/%d", claimID), map[string]any{
			"status":      "approved",
			"review_note": "Verified via phone call to the parish.",
		}, cmToken)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		result := testutil.ParseJSON(resp)
		if result["status"] != "approved" {
			t.Errorf("expected approved, got %v", result["status"])
		}
	})

	t.Run("Community Manager can reject a claim with a reason", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("Bad Claimer", "bad@test.com", "password123", "Catholic")
		cmToken := app.CreateCommunityManager("CM Rejector", "cmr@test.com", "password123")
		churchID := app.SeedChurch("Rejected Church", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/claim", churchID), map[string]any{
			"evidence": "Trust me bro",
		}, userToken)
		claim := testutil.ParseJSON(resp)
		claimID := int(claim["id"].(float64))

		resp = app.Request("PUT", fmt.Sprintf("/api/admin/claims/%d", claimID), map[string]any{
			"status":      "rejected",
			"review_note": "Insufficient evidence. Please provide verifiable contact info.",
		}, cmToken)

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		result := testutil.ParseJSON(resp)
		if result["status"] != "rejected" {
			t.Errorf("expected rejected, got %v", result["status"])
		}
	})

	t.Run("Regular user cannot review ownership claims", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("Regular", "reg@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/admin/claims", nil, userToken)
		if resp.Code != http.StatusForbidden {
			t.Errorf("regular user should be forbidden from claims, got %d", resp.Code)
		}
	})
}

func TestCommunityManagerOversight(t *testing.T) {
	t.Run("Community Manager can list all users like an admin", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		cmToken := app.CreateCommunityManager("CM", "cmlist@test.com", "password123")
		app.CreateUser("User1", "u1@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/admin/users", nil, cmToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("community manager should access user list, got %d", resp.Code)
		}
	})

	t.Run("Community Manager can promote users to moderator", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		cmToken := app.CreateCommunityManager("CM Promoter", "cmprom@test.com", "password123")
		app.CreateUser("Loyal User", "loyal@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/admin/users", nil, cmToken)
		users := testutil.ParseJSONArray(resp)
		var loyalID float64
		for _, u := range users {
			if u["email"] == "loyal@test.com" {
				loyalID = u["id"].(float64)
				break
			}
		}

		resp = app.Request("PUT", fmt.Sprintf("/api/admin/users/%d/role", int(loyalID)), map[string]any{
			"role": "moderator",
		}, cmToken)
		if resp.Code != http.StatusOK {
			t.Errorf("CM should promote to moderator, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("Community Manager cannot promote users to admin", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		cmToken := app.CreateCommunityManager("CM Limited", "cmlim@test.com", "password123")
		app.CreateUser("Target", "target@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/admin/users", nil, cmToken)
		users := testutil.ParseJSONArray(resp)
		var targetID float64
		for _, u := range users {
			if u["email"] == "target@test.com" {
				targetID = u["id"].(float64)
				break
			}
		}

		resp = app.Request("PUT", fmt.Sprintf("/api/admin/users/%d/role", int(targetID)), map[string]any{
			"role": "admin",
		}, cmToken)
		if resp.Code != http.StatusForbidden {
			t.Errorf("CM should not be able to assign admin role, got %d", resp.Code)
		}
	})

	t.Run("Community Manager cannot promote users to community_manager", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		cmToken := app.CreateCommunityManager("CM Self", "cmself@test.com", "password123")
		app.CreateUser("Target2", "target2@test.com", "password123", "Catholic")

		resp := app.Request("GET", "/api/admin/users", nil, cmToken)
		users := testutil.ParseJSONArray(resp)
		var targetID float64
		for _, u := range users {
			if u["email"] == "target2@test.com" {
				targetID = u["id"].(float64)
				break
			}
		}

		resp = app.Request("PUT", fmt.Sprintf("/api/admin/users/%d/role", int(targetID)), map[string]any{
			"role": "community_manager",
		}, cmToken)
		if resp.Code != http.StatusForbidden {
			t.Errorf("CM should not self-replicate, got %d", resp.Code)
		}
	})

	t.Run("Community Manager can verify churches", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		cmToken := app.CreateCommunityManager("CM Verifier", "cmv@test.com", "password123")
		churchID := app.SeedChurch("Unverified", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("PUT", fmt.Sprintf("/api/admin/churches/%d/verify", churchID), nil, cmToken)
		if resp.Code != http.StatusOK {
			t.Errorf("CM should verify churches, got %d", resp.Code)
		}
	})

	t.Run("Community Manager cannot delete churches (admin only)", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		cmToken := app.CreateCommunityManager("CM Delete", "cmdel@test.com", "password123")
		churchID := app.SeedChurch("Protected", "Catholic", "Addr", -23.0, -46.0)

		resp := app.Request("DELETE", fmt.Sprintf("/api/admin/churches/%d", churchID), nil, cmToken)
		if resp.Code != http.StatusForbidden {
			t.Errorf("CM should NOT delete churches, got %d", resp.Code)
		}
	})
}

func TestApprovedChurchOwnerPermissions(t *testing.T) {
	t.Run("Approved church owner can see their owned churches list", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		userToken := app.CreateUser("Owner", "ownerview@test.com", "password123", "Catholic")
		cmToken := app.CreateCommunityManager("CM", "cmown@test.com", "password123")
		churchID := app.SeedChurch("Owned Church", "Catholic", "Addr", -23.0, -46.0)

		// Claim and approve
		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/claim", churchID), map[string]any{
			"evidence": "I am the pastor",
		}, userToken)
		claim := testutil.ParseJSON(resp)
		claimID := int(claim["id"].(float64))

		app.Request("PUT", fmt.Sprintf("/api/admin/claims/%d", claimID), map[string]any{
			"status": "approved",
		}, cmToken)

		// Re-login to get updated role token
		resp = app.Request("POST", "/api/auth/login", map[string]string{
			"email": "ownerview@test.com", "password": "password123",
		}, "")
		result := testutil.ParseJSON(resp)
		ownerToken := result["token"].(string)

		// List my churches
		resp = app.Request("GET", "/api/my-churches", nil, ownerToken)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		churches := testutil.ParseJSONArray(resp)
		if len(churches) != 1 {
			t.Errorf("expected 1 owned church, got %d", len(churches))
		}
	})
}
