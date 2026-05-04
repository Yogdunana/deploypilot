package database

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DriverFactory creates a *gorm.DB from a DSN and config.
type DriverFactory func(dsn string, config *gorm.Config) (*gorm.DB, error)

var drivers = map[string]DriverFactory{}

// RegisterDriver registers a database driver factory.
func RegisterDriver(name string, factory DriverFactory) {
	drivers[name] = factory
}

func init() {
	RegisterDriver("sqlite", func(dsn string, cfg *gorm.Config) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open(dsn), cfg)
	})
	RegisterDriver("postgres", func(dsn string, cfg *gorm.Config) (*gorm.DB, error) {
		return gorm.Open(postgres.Open(dsn), cfg)
	})
}

// Connect establishes a database connection based on the driver type.
// Supported drivers: "sqlite", "postgres".
func Connect(driver, dsn string) (*gorm.DB, error) {
	if dsn == "" {
		return nil, errors.New("database DSN must not be empty")
	}

	var db *gorm.DB
	var err error

	factory, ok := drivers[driver]
	if !ok {
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}
	db, err = factory(dsn, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database (%s): %w", driver, err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return db, nil
}
