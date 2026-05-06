package database

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func RunMigrations() {
	if strings.TrimSpace(os.Getenv("SKIP_DB_MIGRATE")) == "1" {
		log.Println("SKIP_DB_MIGRATE=1 — skipping migrations (schema must already exist on this database)")
		return
	}

	migrateDB, err := sql.Open("postgres", postgresConnStr())
	if err != nil {
		log.Fatal("migrate db open:", err)
	}
	defer migrateDB.Close()
	migrateDB.SetMaxOpenConns(1)

	if err := migrateDB.Ping(); err != nil {
		log.Fatal("migrate db ping:", err)
	}

	driver, err := postgres.WithInstance(migrateDB, &postgres.Config{})
	if err != nil {
		log.Fatal("Error creating migration driver:", err)
	}

	migrationsPath := "file://database/migrations"
	if _, err := os.Stat("/app/backend/database/migrations"); err == nil {
		migrationsPath = "file:///app/backend/database/migrations"
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres", driver)
	if err != nil {
		log.Fatal("Error creating migration instance:", err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatal("Error applying migrations:", err)
	}

	log.Println("Migrations applied successfully!")
}

func EnsureSchemaCompat() {
	if DB == nil {
		return
	}
	if strings.TrimSpace(os.Getenv("SKIP_DB_MIGRATE")) == "1" {
		log.Println("SKIP_DB_MIGRATE=1 — skipping schema compatibility checks")
		return
	}
	stmts := []string{
		`ALTER TABLE IF EXISTS ag_dataset_images ADD COLUMN IF NOT EXISTS bbox_count INT NOT NULL DEFAULT 0`,
		`CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_vsplit_bbox_count ON ag_dataset_images (version_id, split, bbox_count)`,
		`ALTER TABLE IF EXISTS ag_projects ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE IF EXISTS ag_projects ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`,
		`ALTER TABLE IF EXISTS ag_projects ADD COLUMN IF NOT EXISTS version_counter INT NOT NULL DEFAULT 0`,
		`ALTER TABLE IF EXISTS ag_users ADD COLUMN IF NOT EXISTS first_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE IF EXISTS ag_users ADD COLUMN IF NOT EXISTS last_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE IF EXISTS ag_users ADD COLUMN IF NOT EXISTS avatar_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE IF EXISTS ag_dataset_versions ADD COLUMN IF NOT EXISTS is_draft BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE IF EXISTS ag_dataset_images ADD COLUMN IF NOT EXISTS batch_name TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE IF EXISTS ag_dataset_images ADD COLUMN IF NOT EXISTS uploaded_by_email TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE IF EXISTS ag_dataset_images ADD COLUMN IF NOT EXISTS uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()`,
		`ALTER TABLE IF EXISTS ag_dataset_images ADD COLUMN IF NOT EXISTS in_dataset BOOLEAN NOT NULL DEFAULT FALSE`,
		`ALTER TABLE IF EXISTS ag_dataset_images ADD COLUMN IF NOT EXISTS image_sha256 TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS ix_ag_dataset_versions_project_draft_created ON ag_dataset_versions (project_id, is_draft, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_version_batch_uploaded ON ag_dataset_images (version_id, batch_name, uploaded_at DESC)`,
		`CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_version_dataset_bbox ON ag_dataset_images (version_id, in_dataset, bbox_count)`,
		`CREATE INDEX IF NOT EXISTS ix_ag_dataset_images_version_dataset_sha ON ag_dataset_images (version_id, in_dataset, image_sha256)`,
	}
	for _, stmt := range stmts {
		ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
		_, err := DB.ExecContext(ctx, stmt)
		cancel()
		if err != nil {
			log.Printf("schema compat warning: %v", err)
		}
	}
}
