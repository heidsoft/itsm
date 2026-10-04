package seeder

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"itsm-backend/config"
	"itsm-backend/internal/initialization"

	"go.uber.org/zap"
)

// noopConnector 让装配测试拿到非 nil 的 *sql.DB，而不会真的连库。
type noopConnector struct{}

func (noopConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, errors.New("assembly test never opens a connection")
}

func (noopConnector) Driver() driver.Driver { return noopDriver{} }

type noopDriver struct{}

func (noopDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("assembly test never opens a connection")
}

func TestNewProductionAssembly(t *testing.T) {
	assembly, err := NewProductionAssembly(sql.OpenDB(noopConnector{}), nil, zap.NewNop().Sugar(), &config.Config{})
	if err != nil {
		t.Fatalf("NewProductionAssembly: %v", err)
	}
	if assembly.Store == nil || assembly.Engine == nil || assembly.Seeder == nil {
		t.Fatal("assembly must expose store, engine and seeder to both entrypoints")
	}
	if len(assembly.Components) != len(ProductionComponentNames) {
		t.Fatalf("components = %d, want %d", len(assembly.Components), len(ProductionComponentNames))
	}
	seen := make(map[string]struct{}, len(assembly.Components))
	for _, component := range assembly.Components {
		name := component.Name()
		if _, dup := seen[name]; dup {
			t.Fatalf("duplicate initializer %q", name)
		}
		seen[name] = struct{}{}
	}
	for _, name := range ProductionComponentNames {
		if _, ok := seen[name]; !ok {
			t.Fatalf("component %q missing from production assembly", name)
		}
	}
}

func TestNewProductionAssemblyRejectsNilDB(t *testing.T) {
	if _, err := NewProductionAssembly(nil, nil, zap.NewNop().Sugar(), &config.Config{}); err == nil {
		t.Fatal("nil database must fail closed")
	}
}

func TestPlatformRequest(t *testing.T) {
	tests := []struct {
		name             string
		requestedBy      string
		releaseVersion   string
		allowUnversioned bool
		wantRelease      string
		wantErr          string
	}{
		{name: "cli explicit release", requestedBy: "operator", releaseVersion: " 1.6.12 ", wantRelease: "1.6.12"},
		{name: "boot job falls back", requestedBy: "bootstrap-job", releaseVersion: "", allowUnversioned: true, wantRelease: "unversioned"},
		{name: "cli rejects empty release", requestedBy: "operator", releaseVersion: "", wantErr: "-release-version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := PlatformRequest(tt.requestedBy, tt.releaseVersion, tt.allowUnversioned)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("PlatformRequest: %v", err)
			}
			if request.ReleaseVersion != tt.wantRelease {
				t.Errorf("ReleaseVersion = %q, want %q", request.ReleaseVersion, tt.wantRelease)
			}
			if request.RequestedBy != tt.requestedBy {
				t.Errorf("RequestedBy = %q, want %q", request.RequestedBy, tt.requestedBy)
			}
			if request.Scope != (initialization.Scope{Type: "platform", ID: 0}) {
				t.Errorf("Scope = %+v, want platform/0", request.Scope)
			}
			if request.TargetVersion != CurrentTenantTemplateVersion {
				t.Errorf("TargetVersion = %q, want %q", request.TargetVersion, CurrentTenantTemplateVersion)
			}
			if request.ExecutorID == "" || !strings.Contains(request.ExecutorID, "-") {
				t.Errorf("ExecutorID = %q, want hostname-prefixed fencing id", request.ExecutorID)
			}
		})
	}
}

// TestProductionAssemblyHasSingleOwner 是装配所有权守卫：两个入口都不得再自行
// 组装 store/engine/request，否则 lease TTL、scope 和 target version 会重新漂移。
func TestProductionAssemblyHasSingleOwner(t *testing.T) {
	for _, entrypoint := range []string{
		"internal/bootstrap/app.go",
		"cmd/initialize/main.go",
	} {
		data, err := os.ReadFile(repoPath(entrypoint))
		if err != nil {
			t.Fatalf("read %s: %v", entrypoint, err)
		}
		source := string(data)
		for _, forbidden := range []string{
			"initialization.NewEngine(",
			"initialization.NewSQLStore(",
			"initialization.NewExecutorID(",
			"initialization.Request{",
			"initialization.Scope{",
		} {
			if strings.Contains(source, forbidden) {
				t.Errorf("%s assembles initialization itself with %q; use seeder.NewProductionAssembly/PlatformRequest", entrypoint, forbidden)
			}
		}
	}
}

func repoPath(rel string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", rel)
}
