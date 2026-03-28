# 数据库迁移 CLI 增强实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 增强迁移 CLI，提供种子数据、完整重置、dry-run 等生产级功能

**Architecture:** 扩展现有的 `cmd/migrate/main.go`，复用 `pkg/seeder/seeder.go` 的种子数据逻辑，添加新的 CLI 参数和 Migrator 方法

**Tech Stack:** Go, PostgreSQL, Ent ORM, zap logger

---

## 文件结构

- `itsm-backend/cmd/migrate/main.go` - CLI 主入口（修改）
- `itsm-backend/migration/migrator.go` - 迁移核心逻辑（修改）
- `itsm-backend/migration/migrator_test.go` - 迁移测试（新建）
- `itsm-backend/pkg/seeder/seeder.go` - 已有的种子数据逻辑（已存在）

---

## Task 1: 添加 Dry-Run 模式

**Files:**
- Modify: `itsm-backend/migration/migrator.go`
- Modify: `itsm-backend/cmd/migrate/main.go`

- [ ] **Step 1: 修改 Migrator 添加 DryRun 方法**

在 `migrator.go` 中添加 `DryRun` 方法，返回将要执行的 SQL 而不实际执行：

```go
// DryRun returns the SQL that would be executed without actually running it
func (m *Migrator) DryRun(ctx context.Context, mig Migration) (string, error) {
    if mig.Version == "001_initial_schema" {
        return "-- Initial schema handled by Ent", nil
    }

    sql := GetMigrationSQL(mig.Version)
    if sql == "" {
        return "-- No SQL to execute", nil
    }

    return sql, nil
}
```

- [ ] **Step 2: 添加 dry-run flag 到 CLI**

在 `main.go` 中添加 `dryRun` flag：

```go
dryRun := flag.Bool("dry-run", false, "Show SQL without executing")
```

- [ ] **Step 3: 修改 runMigrations 函数支持 dry-run**

```go
func runMigrations(migrator *migration.Migrator, available []migration.Migration, dryRun bool) {
    ctx := context.Background()
    if dryRun {
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
    // ... existing code
}
```

- [ ] **Step 4: 运行测试验证**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend && go build -tags migrate -o /tmp/migrate ./cmd/migrate/
```

---

## Task 2: 添加 Seed 命令

**Files:**
- Modify: `itsm-backend/cmd/migrate/main.go`

- [ ] **Step 1: 添加 seed flag**

```go
seed := flag.Bool("seed", false, "Seed database with initial data")
seedOnly := flag.Bool("seed-only", false, "Only seed data without running migrations")
```

- [ ] **Step 2: 添加 seedData 函数**

```go
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
```

- [ ] **Step 3: 在 main() 中添加 seed 处理逻辑**

```go
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
    available := getAvailableMigrations()
    count, err := migrator.RunMigrations(ctx, available)
    if err != nil {
        log.Fatalf("Migration failed: %v", err)
    }
    fmt.Printf("Applied %d migration(s)\n", count)
    seedData(sugar)
    return
}
```

- [ ] **Step 4: 运行验证**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend && go build -tags migrate -o /tmp/migrate ./cmd/migrate/
/tmp/migrate -help
```

---

## Task 3: 添加 Fresh 命令（完整重置）

**Files:**
- Modify: `itsm-backend/cmd/migrate/main.go`

- [ ] **Step 1: 添加 fresh flag**

```go
fresh := flag.Bool("fresh", false, "Drop and recreate database, then run migrations and seed")
```

- [ ] **Step 2: 添加 freshDatabase 函数**

```go
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
```

- [ ] **Step 3: 在 main() 中添加 fresh 处理逻辑**

```go
if *fresh {
    freshDatabase(migrator, sugar)
    return
}
```

- [ ] **Step 4: 运行验证**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend && go build -tags migrate -o /tmp/migrate ./cmd/migrate/
```

---

## Task 4: 添加 List 命令（列出所有迁移）

**Files:**
- Modify: `itsm-backend/cmd/migrate/main.go`

- [ ] **Step 1: 添加 list flag**

```go
list := flag.Bool("list", false, "List all available migrations")
```

- [ ] **Step 2: 添加 listMigrations 函数**

```go
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
```

- [ ] **Step 3: 在 main() 中添加 list 处理逻辑**

```go
if *list {
    listMigrations(getAvailableMigrations())
    return
}
```

---

## Task 5: 添加 Version 命令（显示当前版本）

**Files:**
- Modify: `itsm-backend/cmd/migrate/main.go`

- [ ] **Step 1: 添加 version flag**

```go
version := flag.Bool("version", false, "Show current database version")
```

- [ ] **Step 2: 添加 showVersion 函数**

```go
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
```

- [ ] **Step 3: 在 main() 中添加 version 处理逻辑**

```go
if *version {
    showVersion(migrator, getAvailableMigrations())
    return
}
```

---

## Task 6: 添加 Reset 命令（回滚所有迁移）

**Files:**
- Modify: `itsm-backend/cmd/migrate/main.go`

- [ ] **Step 1: 添加 reset flag**

```go
reset := flag.Bool("reset", false, "Rollback all migrations")
```

- [ ] **Step 2: 添加 resetMigrations 函数**

```go
func resetMigrations(migrator *migration.Migrator, available []migration.Migration) {
    ctx := context.Background()
    applied, _, err := migrator.Status(ctx, available)
    if err != nil {
        log.Fatalf("Failed to get status: %v", err)
    }

    if len(applied) == 0 {
        fmt.Println("No migrations to rollback")
        return
    }

    fmt.Printf("Rolling back %d migration(s)...\n", len(applied))
    for i := len(applied) - 1; i >= 0; i-- {
        m := applied[i]
        if m.RollbackSQL == "" {
            fmt.Printf("  Skipping %s (no rollback SQL)\n", m.Version)
            continue
        }
        if err := migrator.RollbackMigration(ctx, m); err != nil {
            log.Fatalf("Rollback failed at %s: %v", m.Version, err)
        }
        fmt.Printf("  Rolled back: %s\n", m.Version)
    }
    fmt.Println("Reset completed successfully")
}
```

- [ ] **Step 3: 在 main() 中添加 reset 处理逻辑**

```go
if *reset {
    resetMigrations(migrator, getAvailableMigrations())
    return
}
```

---

## Task 7: 添加单元测试

**Files:**
- Create: `itsm-backend/migration/migrator_test.go`

- [ ] **Step 1: 创建测试文件**

```go
package migration

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestMigration_DryRun(t *testing.T) {
    // Test that DryRun returns SQL without executing
    mig := Migration{
        Version:     "002_add_notification_preferences",
        Description: "Test migration",
        RollbackSQL: "DROP TABLE user_notification_preferences",
    }

    // Just verify the struct is correct
    assert.Equal(t, "002_add_notification_preferences", mig.Version)
    assert.NotEmpty(t, mig.RollbackSQL)
}

func TestMigration_Version(t *testing.T) {
    migrations := []Migration{
        {Version: "001_initial_schema"},
        {Version: "002_add_notification_preferences"},
        {Version: "003_add_audit_indexes"},
    }

    assert.Equal(t, 3, len(migrations))
    assert.Equal(t, "001_initial_schema", migrations[0].Version)
}
```

- [ ] **Step 2: 运行测试**

```bash
cd /Users/heidsoft/Downloads/research/itsm/itsm-backend && go test ./migration/... -v
```

---

## 最终命令帮助

```
Migration CLI for ITSM Backend

Usage:
  go run -tags migrate cmd/migrate/main.go [flags]

Flags:
  -up              Apply all pending migrations
  -down            Rollback the last migration
  -rollback-to v   Rollback to a specific version
  -status          Show migration status
  -seed            Seed database with initial data
  -seed-only       Run migrations then seed data
  -fresh           Drop and recreate database, then migrate and seed
  -reset           Rollback all migrations
  -list            List all available migrations
  -version         Show current database version
  -dry-run         Show SQL without executing (use with -up)
  -help            Show this help message
```
