package database

import (
	"fmt"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	sharedError "adapter/internal/shared/error"
)

var (
	DB   *gorm.DB
	once sync.Once
)

func Init(dsn string) (*gorm.DB, error) {
	var err error

	once.Do(func() {
		gormConfig := &gorm.Config{
			Logger:      logger.Default.LogMode(logger.Silent),
			PrepareStmt: true,
		}

		db, openErr := gorm.Open(postgres.Open(dsn), gormConfig)
		if openErr != nil {
			err = sharedError.NewCustomError(500, sharedError.ErrDatabaseConnectionFailed.Code, "Failed to connect to database", openErr)
			return
		}

		sqlDB, sqlErr := db.DB()
		if sqlErr != nil {
			err = fmt.Errorf("failed to get sql DB: %w", sqlErr)
			return
		}

		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)

		if pingErr := sqlDB.Ping(); pingErr != nil {
			err = fmt.Errorf("failed to ping database: %w", pingErr)
			return
		}

		DB = db
	})

	return DB, err
}

func GetDB() *gorm.DB {
	return DB
}

func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
