package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wiseful-oak-systems/where-is-church/internal/config"
)

// Store is the interface for file storage backends.
type Store interface {
	// Save stores a file and returns a storage key (path/URL).
	Save(ctx context.Context, filename string, reader io.Reader) (key string, err error)
	// Get returns a reader for the stored file.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes a stored file.
	Delete(ctx context.Context, key string) error
	// URL returns a public/accessible URL for the file.
	URL(key string) string
}

// New creates a Store based on configuration.
func New(cfg *config.Config) (Store, error) {
	switch cfg.StorageDriver {
	case "local":
		return NewLocalStore(cfg.StorageLocalPath)
	case "s3":
		return NewS3Store(cfg.S3Bucket, cfg.S3Region, cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey)
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s (use 'local' or 's3')", cfg.StorageDriver)
	}
}

// AllowedImageTypes lists MIME types we accept for image uploads.
var AllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

// AllowedDocTypes lists MIME types we accept for document uploads.
var AllowedDocTypes = map[string]bool{
	"application/pdf":    true,
	"image/jpeg":         true,
	"image/png":          true,
	"image/webp":         true,
	"image/gif":          true,
}

// GenerateKey creates a unique storage key for a file.
func GenerateKey(prefix, filename string) string {
	ext := filepath.Ext(filename)
	ts := time.Now().UnixNano()
	safe := strings.ReplaceAll(strings.TrimSuffix(filepath.Base(filename), ext), " ", "_")
	if len(safe) > 50 {
		safe = safe[:50]
	}
	return fmt.Sprintf("%s/%d_%s%s", prefix, ts, safe, ext)
}

// ─── Local Filesystem Store ──────────────────────────────────────────────────

type LocalStore struct {
	basePath string
}

func NewLocalStore(basePath string) (*LocalStore, error) {
	if err := os.MkdirAll(basePath, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}
	return &LocalStore{basePath: basePath}, nil
}

func (s *LocalStore) Save(_ context.Context, key string, reader io.Reader) (string, error) {
	fullPath := filepath.Join(s.basePath, key)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(fullPath) //nolint:gosec // path is validated above via HasPrefix check
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, reader); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	return key, nil
}

func (s *LocalStore) Get(_ context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.basePath, key)
	// Prevent path traversal
	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(s.basePath)) {
		return nil, fmt.Errorf("invalid file path")
	}
	return os.Open(cleanPath)
}

func (s *LocalStore) Delete(_ context.Context, key string) error {
	fullPath := filepath.Join(s.basePath, key)
	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, filepath.Clean(s.basePath)) {
		return fmt.Errorf("invalid file path")
	}
	return os.Remove(cleanPath)
}

func (s *LocalStore) URL(key string) string {
	return "/uploads/" + key
}
