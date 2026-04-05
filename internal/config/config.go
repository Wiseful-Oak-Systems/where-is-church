package config

import (
	"fmt"
	"os"
	"strconv"
)

const EnvProduction = "production"

type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	DBMaxOpenConns int
	DBMaxIdleConns int
	JWTSecret      string
	JWTExpiry      int
	Port           string
	CookieSecure   bool
	Environment    string // "development" or "production"

	// Storage: "local" or "s3"
	StorageDriver    string
	StorageLocalPath string // local filesystem path for uploads
	S3Bucket         string
	S3Region         string
	S3Endpoint       string // custom endpoint for S3-compatible services (MinIO, etc.)
	S3AccessKey      string
	S3SecretKey      string
	MaxUploadSizeMB  int
	SeedCountry      string // ISO 3166-1 code for auto-seeding (e.g., "BR")
	AutoSeed         bool   // auto-seed from OSM when DB is empty
}

func Load() *Config {
	return &Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "wic"),
		DBPassword:     getEnv("DB_PASSWORD", "wic_dev_password"),
		DBName:         getEnv("DB_NAME", "whereischurch"),
		DBSSLMode:      getEnv("DB_SSLMODE", "disable"),
		DBMaxOpenConns: getEnvInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns: getEnvInt("DB_MAX_IDLE_CONNS", 5),
		JWTSecret:      getEnv("JWT_SECRET", "change-me-to-a-random-secret"),
		JWTExpiry:      getEnvInt("JWT_EXPIRY_HOURS", 24),
		Port:           getEnv("PORT", "8080"),
		CookieSecure:   getEnvBool("COOKIE_SECURE", false),
		Environment:    getEnv("ENVIRONMENT", "development"),

		StorageDriver:    getEnv("STORAGE_DRIVER", "local"),
		StorageLocalPath: getEnv("STORAGE_LOCAL_PATH", "./uploads"),
		S3Bucket:         getEnv("S3_BUCKET", ""),
		S3Region:         getEnv("S3_REGION", "us-east-1"),
		S3Endpoint:       getEnv("S3_ENDPOINT", ""),
		S3AccessKey:      getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:      getEnv("S3_SECRET_KEY", ""),
		MaxUploadSizeMB:  getEnvInt("MAX_UPLOAD_SIZE_MB", 10),
		SeedCountry:      getEnv("SEED_COUNTRY", "BR"),
		AutoSeed:         getEnvBool("AUTO_SEED", true),
	}
}

// Validate checks that critical configuration values are safe for use.
func (c *Config) Validate() error {
	if c.JWTSecret == "change-me-to-a-random-secret" || len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be set to a random string of at least 32 characters")
	}
	if c.JWTExpiry <= 0 {
		return fmt.Errorf("JWT_EXPIRY_HOURS must be a positive integer")
	}
	if c.Environment == EnvProduction && !c.CookieSecure {
		return fmt.Errorf("COOKIE_SECURE must be true in production")
	}
	if c.Environment == EnvProduction && c.DBSSLMode == "disable" {
		return fmt.Errorf("DB_SSLMODE should not be 'disable' in production")
	}
	return nil
}

func (c *Config) DSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=" + c.DBSSLMode
}

func (c *Config) IsProd() bool {
	return c.Environment == EnvProduction
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
