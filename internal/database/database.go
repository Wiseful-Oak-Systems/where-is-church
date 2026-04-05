package database

import (
	"log"
	"time"

	"github.com/wiseful-oak-systems/where-is-church/internal/config"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a database connection with retry logic for Docker startup ordering.
// Retries up to 30 times with 2-second intervals (60 seconds total).
func Connect(cfg *config.Config) *gorm.DB {
	gormCfg := &gorm.Config{}
	if cfg.IsProd() {
		gormCfg.Logger = logger.Default.LogMode(logger.Warn)
	}

	var db *gorm.DB
	var err error

	for attempt := 1; attempt <= 30; attempt++ {
		db, err = gorm.Open(postgres.Open(cfg.DSN()), gormCfg)
		if err == nil {
			sqlDB, dbErr := db.DB()
			if dbErr == nil && sqlDB.Ping() == nil {
				break
			}
			err = dbErr
		}

		if attempt == 30 {
			log.Fatalf("failed to connect to database after %d attempts: %v", attempt, err)
		}
		log.Printf("Database not ready (attempt %d/30): %v — retrying in 2s...", attempt, err)
		time.Sleep(2 * time.Second)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	// Enable PostGIS
	db.Exec("CREATE EXTENSION IF NOT EXISTS postgis")

	if err := runMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	log.Println("Database connected and migrated successfully")
	return db
}

func runMigrations(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Church{},
		&models.MassSchedule{},
		&models.CheckIn{},
		&models.Suggestion{},
		&models.Favorite{},
		&models.ChurchOwnership{},
		&models.Attachment{},
		&models.AuditLog{},
		&models.UserReputation{},
		&models.ChurchConfirmation{},
		&models.ChurchProposal{},
	)
}
