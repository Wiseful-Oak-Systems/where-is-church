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

func Connect(cfg *config.Config) *gorm.DB {
	gormCfg := &gorm.Config{}
	if cfg.IsProd() {
		gormCfg.Logger = logger.Default.LogMode(logger.Warn)
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN()), gormCfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
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
	)
}
