//go:build migrate
// +build migrate

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"sort"

	"itsm-backend/config"
	"itsm-backend/database"
	"itsm-backend/migration"
	"itsm-backend/pkg/seeder"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	up := flag.Bool("up", false, "Apply all pending migrations")
	down := flag.Bool("down", false, "Rollback the last migration")
	status := flag.Bool("status", false, "Show migration status")
	list := flag.Bool("list", false, "List all available migrations")
	rollbackVersion := flag.String("rollback-to", "", "Rollback to a specific version")
	dryRun := flag.Bool("dry-run", false, "Show SQL without executing")
	fresh := flag.Bool("fresh", false, "Drop and recreate database, then run migrations and seed")
	seed := flag.Bool("seed", false, "Seed database with initial data")
	seedOnly := flag.Bool("seed-only", false, "Only seed data without running migrations")
	version := flag.Bool("version", false, "Show current database version")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	sugar := logger.Sugar()

	// Initialize database
	db, err := database.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	migrator := migration.NewMigrator(db, sugar)
	ctx := context.Background()

	// Ensure migrations table exists
	if err := migrator.EnsureMigrationsTable(ctx); err != nil {
		log.Fatalf("Failed to ensure migrations table: %v", err)
	}

	// Get available migrations
	available := getAvailableMigrations()

	if *fresh {
		freshDatabase(migrator, sugar)
		return
	}

	if *seed {
		seedData(sugar)
		return
	}

	if *seedOnly {
		ctx := context.Background()
		// First ensure migrations table
		if err := migrator.EnsureMigrationsTable(ctx); err != nil {
			log.Fatalf("Failed to ensure migrations table: %v", err)
		}
		count, err := migrator.RunMigrations(ctx, available)
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Printf("Applied %d migration(s)\n", count)
		seedData(sugar)
		return
	}

	if *dryRun {
		ctx := context.Background()
		fmt.Println("=== Dry Run Mode - No changes will be made ===\n")
		for _, mig := range available {
			sql, err := migrator.DryRun(ctx, mig)
			if err != nil {
				log.Fatalf("Dry run failed for %s: %v", mig.Version, err)
			}
			fmt.Printf("[%s] %s\n%s\n\n", mig.Version, mig.Description, sql)
		}
		return
	}

	if *status {
		showStatus(migrator, available)
		return
	}

	if *version {
		showVersion(migrator, getAvailableMigrations())
		return
	}

	if *list {
		listMigrations(getAvailableMigrations())
		return
	}

	if *up {
		runMigrations(migrator, available)
		return
	}

	if *down {
		if *rollbackVersion != "" {
			rollbackToVersion(migrator, available, *rollbackVersion)
		} else {
			rollbackLast(migrator, available)
		}
		return
	}

	// No command specified, show help
	fmt.Println("Migration CLI for ITSM Backend")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  go run -tags migrate cmd/migrate/main.go -up              Apply all pending migrations")
	fmt.Println("  go run -tags migrate cmd/migrate/main.go -down            Rollback the last migration")
	fmt.Println("  go run -tags migrate cmd/migrate/main.go -rollback-to v2  Rollback to version v2")
	fmt.Println("  go run -tags migrate cmd/migrate/main.go -status         Show migration status")
}

func getAvailableMigrations() []migration.Migration {
	migrations := make([]migration.Migration, len(migration.RegisteredMigrations))
	copy(migrations, migration.RegisteredMigrations)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations
}

func showStatus(migrator *migration.Migrator, available []migration.Migration) {
	ctx := context.Background()
	applied, pending, err := migrator.Status(ctx, available)
	if err != nil {
		log.Fatalf("Failed to get migration status: %v", err)
	}

	fmt.Println("=== Applied Migrations ===")
	if len(applied) == 0 {
		fmt.Println("  No migrations applied")
	} else {
		for _, m := range applied {
			fmt.Printf("  [%s] %s (applied: %s)\n", m.Version, m.Description, m.AppliedAt.Format("2006-01-02 15:04:05"))
		}
	}

	fmt.Println("")
	fmt.Println("=== Pending Migrations ===")
	if len(pending) == 0 {
		fmt.Println("  No pending migrations")
	} else {
		for _, m := range pending {
			fmt.Printf("  [%s] %s\n", m.Version, m.Description)
		}
	}
}

func runMigrations(migrator *migration.Migrator, available []migration.Migration) {
	ctx := context.Background()
	count, err := migrator.RunMigrations(ctx, available)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	fmt.Printf("Applied %d migration(s)\n", count)
}

func rollbackLast(migrator *migration.Migrator, available []migration.Migration) {
	ctx := context.Background()
	applied, _, err := migrator.Status(ctx, available)
	if err != nil {
		log.Fatalf("Failed to get migration status: %v", err)
	}

	if len(applied) == 0 {
		fmt.Println("No migrations to rollback")
		return
	}

	// Get the last applied migration
	last := applied[len(applied)-1]
	if last.RollbackSQL == "" {
		log.Fatalf("Migration %s has no rollback SQL defined", last.Version)
	}

	if err := migrator.RollbackMigration(ctx, last); err != nil {
		log.Fatalf("Rollback failed: %v", err)
	}
	fmt.Printf("Rolled back migration: %s\n", last.Version)
}

func rollbackToVersion(migrator *migration.Migrator, available []migration.Migration, targetVersion string) {
	ctx := context.Background()
	applied, _, err := migrator.Status(ctx, available)
	if err != nil {
		log.Fatalf("Failed to get migration status: %v", err)
	}

	// Find migrations to rollback (all applied after target version)
	var toRollback []migration.Migration
	for i := len(applied) - 1; i >= 0; i-- {
		if applied[i].Version <= targetVersion {
			break
		}
		toRollback = append(toRollback, applied[i])
	}

	if len(toRollback) == 0 {
		fmt.Printf("No migrations to rollback (already at version %s)\n", targetVersion)
		return
	}

	for _, m := range toRollback {
		if m.RollbackSQL == "" {
			log.Fatalf("Migration %s has no rollback SQL defined", m.Version)
		}
		if err := migrator.RollbackMigration(ctx, m); err != nil {
			log.Fatalf("Rollback failed at %s: %v", m.Version, err)
		}
		fmt.Printf("Rolled back migration: %s\n", m.Version)
	}
}

func seedData(sugar *zap.SugaredLogger) {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	client, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer client.Close()

	seederInstance := seeder.NewSeeder(client, sugar)
	seederInstance.SeedAll(context.Background())
	fmt.Println("Seed completed successfully")
}

func freshDatabase(migrator *migration.Migrator, sugar *zap.SugaredLogger) {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to postgres to drop/create database
	postgresDSN := fmt.Sprintf("host=%s port=%d user=%s dbname=postgres sslmode=%s password=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.SSLMode, cfg.Database.Password)

	postgresDB, err := sql.Open("postgres", postgresDSN)
	if err != nil {
		log.Fatalf("Failed to connect to postgres: %v", err)
	}
	defer postgresDB.Close()

	fmt.Printf("Dropping database %s...\n", cfg.Database.DBName)
	_, err = postgresDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", cfg.Database.DBName))
	if err != nil {
		log.Fatalf("Failed to drop database: %v", err)
	}

	fmt.Printf("Creating database %s...\n", cfg.Database.DBName)
	_, err = postgresDB.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.Database.DBName))
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}

	postgresDB.Close()

	// Reconnect to new database
	db, err := database.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to new database: %v", err)
	}
	defer db.Close()

	// Update migrator with new connection
	*migrator = *migration.NewMigrator(db, sugar)

	ctx := context.Background()
	if err := migrator.EnsureMigrationsTable(ctx); err != nil {
		log.Fatalf("Failed to ensure migrations table: %v", err)
	}

	available := getAvailableMigrations()
	count, err := migrator.RunMigrations(ctx, available)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	fmt.Printf("Applied %d migration(s)\n", count)

	// Seed data
	client, err := database.InitDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect for seeding: %v", err)
	}
	defer client.Close()
	seederInstance := seeder.NewSeeder(client, sugar)
	seederInstance.SeedAll(context.Background())

	fmt.Println("Fresh reset completed successfully")
}

func listMigrations(available []migration.Migration) {
	fmt.Println("=== Available Migrations ===")
	for _, mig := range available {
		fmt.Printf("  [%s] %s\n", mig.Version, mig.Description)
		if mig.RollbackSQL != "" {
			fmt.Printf("       ↳ rollback: YES\n")
		} else {
			fmt.Printf("       ↳ rollback: NO\n")
		}
	}
}

func showVersion(migrator *migration.Migrator, available []migration.Migration) {
	ctx := context.Background()
	applied, _, err := migrator.Status(ctx, available)
	if err != nil {
		log.Fatalf("Failed to get status: %v", err)
	}

	if len(applied) == 0 {
		fmt.Println("No migrations applied")
		return
	}

	latest := applied[len(applied)-1]
	fmt.Printf("Current version: %s\n", latest.Version)
	fmt.Printf("Description: %s\n", latest.Description)
	fmt.Printf("Applied at: %s\n", latest.AppliedAt.Format("2006-01-02 15:04:05"))
}
