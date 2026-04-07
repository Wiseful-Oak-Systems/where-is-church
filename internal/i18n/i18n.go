package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

const (
	DefaultLang = "pt-BR"
	FallbackLang = "en-US"
	LangContextKey = "lang"
)

// SupportedLanguages lists all available locales.
var SupportedLanguages = []string{"pt-BR", "en-US"}

// Bundle holds all loaded translations keyed by locale then message key.
type Bundle struct {
	mu           sync.RWMutex
	translations map[string]map[string]string // locale -> key -> message
}

var global *Bundle

// Load reads all JSON translation files from the given directory.
// Each file should be named like "pt-BR.json", "en-US.json".
func Load(dir string) (*Bundle, error) {
	b := &Bundle{translations: make(map[string]map[string]string)}

	for _, lang := range SupportedLanguages {
		path := filepath.Join(dir, lang+".json")
		data, err := os.ReadFile(path) //nolint:gosec // path is constructed from hardcoded SupportedLanguages list, not user input
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", path, err)
		}
		var messages map[string]string
		if err := json.Unmarshal(data, &messages); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", path, err)
		}
		b.translations[lang] = messages
	}

	global = b
	return b, nil
}

// LoadFromMap creates a Bundle from an in-memory map (useful for testing).
func LoadFromMap(translations map[string]map[string]string) *Bundle {
	b := &Bundle{translations: translations}
	global = b
	return b
}

// T returns the translated message for the given key in the given locale.
// Falls back to en-US, then returns the key itself if not found.
func (b *Bundle) T(lang, key string, args ...any) string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if msgs, ok := b.translations[lang]; ok {
		if msg, ok := msgs[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(msg, args...)
			}
			return msg
		}
	}

	// Fallback to en-US
	if lang != FallbackLang {
		if msgs, ok := b.translations[FallbackLang]; ok {
			if msg, ok := msgs[key]; ok {
				if len(args) > 0 {
					return fmt.Sprintf(msg, args...)
				}
				return msg
			}
		}
	}

	return key
}

// T is the global translation function — uses the global bundle.
func T(lang, key string, args ...any) string {
	if global == nil {
		return key
	}
	return global.T(lang, key, args...)
}

// GetLang extracts the language from a Gin context.
func GetLang(c *gin.Context) string {
	if lang, exists := c.Get(LangContextKey); exists {
		return lang.(string)
	}
	return DefaultLang
}

// Middleware detects the user's preferred language from:
// 1. ?lang= query parameter
// 2. Accept-Language header
// 3. Default (pt-BR for Brazil focus)
func Middleware() gin.HandlerFunc {
	supported := make(map[string]bool)
	for _, l := range SupportedLanguages {
		supported[l] = true
		// Also support the base language code (e.g., "pt" -> "pt-BR")
		supported[strings.Split(l, "-")[0]] = true
	}

	baseToFull := map[string]string{
		"pt": "pt-BR",
		"en": "en-US",
	}

	return func(c *gin.Context) {
		lang := DefaultLang

		// 1. Query parameter
		if q := c.Query("lang"); q != "" {
			if supported[q] {
				if full, ok := baseToFull[q]; ok {
					lang = full
				} else {
					lang = q
				}
			}
		} else {
			// 2. Accept-Language header
			accept := c.GetHeader("Accept-Language")
			if accept != "" {
				lang = parseAcceptLanguage(accept, supported, baseToFull)
			}
		}

		c.Set(LangContextKey, lang)
		c.Header("Content-Language", lang)
		c.Next()
	}
}

func parseAcceptLanguage(header string, supported map[string]bool, baseToFull map[string]string) string {
	// Simple parser: takes first supported language from the header
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(strings.Split(part, ";")[0])
		if supported[tag] {
			if full, ok := baseToFull[tag]; ok {
				return full
			}
			return tag
		}
		// Try base language (e.g., "pt-PT" -> "pt" -> "pt-BR")
		base := strings.Split(tag, "-")[0]
		if full, ok := baseToFull[base]; ok {
			return full
		}
	}
	return DefaultLang
}
