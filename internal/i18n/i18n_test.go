package i18n

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTranslationLookup(t *testing.T) {
	LoadFromMap(map[string]map[string]string{
		"pt-BR": {"hello": "Olá, %s!", "bye": "Tchau"},
		"en-US": {"hello": "Hello, %s!", "bye": "Bye"},
	})

	t.Run("Returns pt-BR translation when requested", func(t *testing.T) {
		result := T("pt-BR", "hello", "Maria")
		if result != "Olá, Maria!" {
			t.Errorf("expected 'Olá, Maria!', got %q", result)
		}
	})

	t.Run("Returns en-US translation when requested", func(t *testing.T) {
		result := T("en-US", "bye")
		if result != "Bye" {
			t.Errorf("expected 'Bye', got %q", result)
		}
	})

	t.Run("Falls back to en-US when key missing in requested locale", func(t *testing.T) {
		LoadFromMap(map[string]map[string]string{
			"pt-BR": {},
			"en-US": {"only_english": "English only"},
		})
		result := T("pt-BR", "only_english")
		if result != "English only" {
			t.Errorf("expected fallback to en-US, got %q", result)
		}
	})

	t.Run("Returns the key itself when translation is missing everywhere", func(t *testing.T) {
		result := T("pt-BR", "nonexistent.key")
		if result != "nonexistent.key" {
			t.Errorf("expected raw key, got %q", result)
		}
	})
}

func TestLanguageMiddleware(t *testing.T) {
	LoadFromMap(map[string]map[string]string{
		"pt-BR": {"test": "Teste"},
		"en-US": {"test": "Test"},
	})

	gin.SetMode(gin.TestMode)

	t.Run("Defaults to pt-BR when no language specified", func(t *testing.T) {
		r := gin.New()
		r.Use(Middleware())
		r.GET("/test", func(c *gin.Context) {
			c.String(200, GetLang(c))
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		r.ServeHTTP(w, req)

		if w.Body.String() != "pt-BR" {
			t.Errorf("expected pt-BR default, got %q", w.Body.String())
		}
	})

	t.Run("Respects ?lang= query parameter", func(t *testing.T) {
		r := gin.New()
		r.Use(Middleware())
		r.GET("/test", func(c *gin.Context) {
			c.String(200, GetLang(c))
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test?lang=en-US", nil)
		r.ServeHTTP(w, req)

		if w.Body.String() != "en-US" {
			t.Errorf("expected en-US from query, got %q", w.Body.String())
		}
	})

	t.Run("Detects language from Accept-Language header", func(t *testing.T) {
		r := gin.New()
		r.Use(Middleware())
		r.GET("/test", func(c *gin.Context) {
			c.String(200, GetLang(c))
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		r.ServeHTTP(w, req)

		if w.Body.String() != "en-US" {
			t.Errorf("expected en-US from header, got %q", w.Body.String())
		}
	})

	t.Run("Maps base language code to full locale (pt -> pt-BR)", func(t *testing.T) {
		r := gin.New()
		r.Use(Middleware())
		r.GET("/test", func(c *gin.Context) {
			c.String(200, GetLang(c))
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Accept-Language", "pt")
		r.ServeHTTP(w, req)

		if w.Body.String() != "pt-BR" {
			t.Errorf("expected pt-BR from 'pt' header, got %q", w.Body.String())
		}
	})

	t.Run("Sets Content-Language response header", func(t *testing.T) {
		r := gin.New()
		r.Use(Middleware())
		r.GET("/test", func(c *gin.Context) {
			c.String(200, "ok")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test?lang=en-US", nil)
		r.ServeHTTP(w, req)

		if cl := w.Header().Get("Content-Language"); cl != "en-US" {
			t.Errorf("expected Content-Language en-US, got %q", cl)
		}
	})
}

func TestSupportedLanguagesList(t *testing.T) {
	t.Run("pt-BR is the first supported language (default)", func(t *testing.T) {
		if SupportedLanguages[0] != "pt-BR" {
			t.Errorf("expected pt-BR as first supported language, got %q", SupportedLanguages[0])
		}
	})

	t.Run("en-US is available as fallback", func(t *testing.T) {
		found := false
		for _, l := range SupportedLanguages {
			if l == "en-US" {
				found = true
			}
		}
		if !found {
			t.Error("en-US should be in SupportedLanguages")
		}
	})
}
