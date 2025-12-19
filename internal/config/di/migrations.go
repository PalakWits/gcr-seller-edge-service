package di

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func MigrateDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	source, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	version, dirty, verr := m.Version()
	if verr == nil && dirty {
		fmt.Printf("Detected dirty migration at version %d. Cleaning it automatically...\n", version)

		if err := m.Force(int(version)); err != nil {
			fmt.Printf("Force failed, trying direct SQL...\n")
			query := fmt.Sprintf("UPDATE schema_migrations SET dirty = false WHERE version = %d", version)
			if _, execErr := sqlDB.Exec(query); execErr != nil {
				fmt.Printf("Failed to clean migration %d: %v\n", version, execErr)
			} else {
				fmt.Printf("Successfully cleaned migration %d via SQL\n", version)
			}
		} else {
			fmt.Printf("Successfully cleaned migration %d\n", version)
		}
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		version, dirty, verr := m.Version()
		if verr == nil {
			return fmt.Errorf("migration failed at version %d (dirty: %v): %w", version, dirty, err)
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	version, dirty, _ = m.Version()
	if err == migrate.ErrNoChange {
		fmt.Printf("Database is up to date at version %d (dirty: %v)\n", version, dirty)
	} else {
		fmt.Printf("Migrations completed successfully to version %d\n", version)
	}

	return nil
}

func GetMigrationMode() string {
	return "migrate"
}
