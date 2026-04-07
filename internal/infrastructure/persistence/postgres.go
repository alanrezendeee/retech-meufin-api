package persistence

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func OpenPostgres(dsn string, logLevel logger.LogLevel) (*gorm.DB, error) {
	cfg := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}
	db, err := gorm.Open(postgres.Open(dsn), cfg)
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db, nil
}
