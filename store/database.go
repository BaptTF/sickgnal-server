package store

import (
	"fmt"

	"github.com/BaptTF/sickgnal-server/config"
	"github.com/BaptTF/sickgnal-server/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes the GORM database connection and runs auto-migrations.
func InitDB(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.DBDriver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DBPath)
	case "postgres":
		dsn, err := cfg.DSN()
		if err != nil {
			return nil, err
		}
		dialector = postgres.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s (use sqlite or postgres)", cfg.DBDriver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// SQLite-specific PRAGMAs
	if cfg.DBDriver == "sqlite" {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("get underlying db: %w", err)
		}
		if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
			return nil, fmt.Errorf("set WAL mode: %w", err)
		}
		if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON"); err != nil {
			return nil, fmt.Errorf("enable foreign keys: %w", err)
		}
	}

	// Auto-migrate all models
	if err := db.AutoMigrate(
		&models.User{},
		&models.SignedPreKey{},
		&models.EphemeralPreKey{},
		&models.StoredInitialMessage{},
		&models.StoredMessage{},
		&models.AuthChallenge{},
	); err != nil {
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}

	return db, nil
}
