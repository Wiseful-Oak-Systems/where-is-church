package handlers_test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wiseful-oak-systems/where-is-church/internal/testutil"
)

func createMultipartRequest(t *testing.T, path, token string, fields map[string]string, fileName string, fileContent []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}

	if fileName != "" {
		part, err := writer.CreateFormFile("file", fileName)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(part, bytes.NewReader(fileContent)); err != nil {
			t.Fatal(err)
		}
	}
	_ = writer.Close()

	req := httptest.NewRequest("POST", path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestAttachmentUpload(t *testing.T) {
	t.Run("User can upload a document attached to a suggestion", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Uploader", "up@test.com", "password123", "Catholic")

		// Create a suggestion first
		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "new_church", "content": "New church with photo evidence",
		}, token)
		sugg := testutil.ParseJSON(resp)["suggestion"].(map[string]any)
		suggID := fmt.Sprintf("%d", int(sugg["id"].(float64)))

		// Upload a fake JPEG (JPEG magic bytes)
		jpegData := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)

		req := createMultipartRequest(t, "/api/attachments", token, map[string]string{
			"attachable_type": "suggestion",
			"attachable_id":   suggID,
		}, "photo.jpg", jpegData)

		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		result := testutil.ParseJSON(w)
		if result["filename"] != "photo.jpg" {
			t.Errorf("expected filename 'photo.jpg', got %v", result["filename"])
		}
		if result["url"] == nil || result["url"] == "" {
			t.Error("response should include a file URL")
		}
	})

	t.Run("User can upload evidence for a church ownership claim", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Claimer", "cl@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("Claim Church", "Catholic", "Addr", -23.0, -46.0)

		// Create a claim
		resp := app.Request("POST", fmt.Sprintf("/api/churches/%d/claim", churchID), map[string]any{
			"evidence": "I have documents",
		}, token)
		claim := testutil.ParseJSON(resp)
		claimID := fmt.Sprintf("%d", int(claim["id"].(float64)))

		pngData := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte{0x00}, 100)...)

		req := createMultipartRequest(t, "/api/attachments", token, map[string]string{
			"attachable_type": "claim",
			"attachable_id":   claimID,
		}, "parish_letter.png", pngData)

		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Upload rejects files with disallowed content types", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("BadFile", "bad@test.com", "password123", "Catholic")
		churchID := app.SeedChurch("C", "Catholic", "A", -23.0, -46.0)

		// Create a suggestion to attach to
		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general", "content": "test",
		}, token)
		sugg := testutil.ParseJSON(resp)["suggestion"].(map[string]any)
		suggID := fmt.Sprintf("%d", int(sugg["id"].(float64)))
		_ = churchID

		// Upload a plain text file (not allowed)
		req := createMultipartRequest(t, "/api/attachments", token, map[string]string{
			"attachable_type": "suggestion",
			"attachable_id":   suggID,
		}, "hack.txt", []byte("this is a text file not an image"))

		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("disallowed file type should return 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("Upload requires attachable_type and attachable_id", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Missing", "miss@test.com", "password123", "Catholic")

		jpegData := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)
		req := createMultipartRequest(t, "/api/attachments", token, map[string]string{}, "photo.jpg", jpegData)

		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("missing metadata should return 400, got %d", w.Code)
		}
	})
}

func TestAttachmentList(t *testing.T) {
	t.Run("User can list all attachments for a specific suggestion", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Lister", "ls@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "new_church", "content": "With photos",
		}, token)
		sugg := testutil.ParseJSON(resp)["suggestion"].(map[string]any)
		suggID := fmt.Sprintf("%d", int(sugg["id"].(float64)))

		// Upload two files
		for _, name := range []string{"photo1.jpg", "photo2.jpg"} {
			jpegData := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)
			req := createMultipartRequest(t, "/api/attachments", token, map[string]string{
				"attachable_type": "suggestion",
				"attachable_id":   suggID,
			}, name, jpegData)
			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)
		}

		resp = app.Request("GET", "/api/attachments?attachable_type=suggestion&attachable_id="+suggID, nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		results := testutil.ParseJSONArray(resp)
		if len(results) != 2 {
			t.Errorf("expected 2 attachments, got %d", len(results))
		}
	})
}

func TestAttachmentDelete(t *testing.T) {
	t.Run("User can delete their own attachment", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token := app.CreateUser("Deleter", "del@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general", "content": "Delete test",
		}, token)
		sugg := testutil.ParseJSON(resp)["suggestion"].(map[string]any)
		suggID := fmt.Sprintf("%d", int(sugg["id"].(float64)))

		jpegData := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)
		req := createMultipartRequest(t, "/api/attachments", token, map[string]string{
			"attachable_type": "suggestion",
			"attachable_id":   suggID,
		}, "todelete.jpg", jpegData)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		att := testutil.ParseJSON(w)
		attID := int(att["id"].(float64))

		resp = app.Request("DELETE", fmt.Sprintf("/api/attachments/%d", attID), nil, token)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("User cannot delete another user's attachment", func(t *testing.T) {
		app := testutil.NewTestApp(t)
		token1 := app.CreateUser("Owner", "own@test.com", "password123", "Catholic")
		token2 := app.CreateUser("Other", "other@test.com", "password123", "Catholic")

		resp := app.Request("POST", "/api/suggestions", map[string]any{
			"type": "general", "content": "Other test",
		}, token1)
		sugg := testutil.ParseJSON(resp)["suggestion"].(map[string]any)
		suggID := fmt.Sprintf("%d", int(sugg["id"].(float64)))

		jpegData := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, bytes.Repeat([]byte{0x00}, 100)...)
		req := createMultipartRequest(t, "/api/attachments", token1, map[string]string{
			"attachable_type": "suggestion",
			"attachable_id":   suggID,
		}, "private.jpg", jpegData)
		w := httptest.NewRecorder()
		app.Router.ServeHTTP(w, req)
		att := testutil.ParseJSON(w)
		attID := int(att["id"].(float64))

		resp = app.Request("DELETE", fmt.Sprintf("/api/attachments/%d", attID), nil, token2)
		if resp.Code != http.StatusForbidden {
			t.Errorf("other user should be forbidden, got %d", resp.Code)
		}
	})
}
