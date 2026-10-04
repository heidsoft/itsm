package seeder

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"itsm-backend/config"
	"itsm-backend/ent"
	"itsm-backend/internal/initialization"

	"go.uber.org/zap"
)

// productionLeaseTTL 是初始化组件租约时长，两个装配入口必须一致，否则 fencing
// 行为随入口漂移。
const productionLeaseTTL = 30 * time.Second

// ProductionAssembly 是生产初始化装配的唯一所有者：服务启动内联 seed 与
// cmd/initialize CLI 共用同一套 store/components/engine，禁止再出现第二份装配。
type ProductionAssembly struct {
	Engine     *initialization.Engine
	Components []initialization.Initializer
	Store      *initialization.SQLStore
	Seeder     *Seeder
}

func NewProductionAssembly(
	db *sql.DB,
	client *ent.Client,
	sugar *zap.SugaredLogger,
	appConfig *config.Config,
) (*ProductionAssembly, error) {
	store, err := initialization.NewSQLStore(db)
	if err != nil {
		return nil, fmt.Errorf("create initialization store: %w", err)
	}
	productSeeder := NewSeeder(client, sugar, appConfig)
	components, err := ProductionInitializers(productSeeder)
	if err != nil {
		return nil, fmt.Errorf("create production initializers: %w", err)
	}
	engine, err := initialization.NewEngine(store, components, productionLeaseTTL)
	if err != nil {
		return nil, fmt.Errorf("create initialization engine: %w", err)
	}
	return &ProductionAssembly{
		Engine:     engine,
		Components: components,
		Store:      store,
		Seeder:     productSeeder,
	}, nil
}

// PlatformScope 是平台级初始化的唯一作用域。
func PlatformScope() initialization.Scope {
	return initialization.Scope{Type: "platform", ID: 0}
}

// PlatformRequest 组装平台初始化请求。executor id 前缀取主机名，取不到时回落到
// requestedBy，保证审计身份仍然可读。
func PlatformRequest(requestedBy, releaseVersion string, allowUnversioned bool) (initialization.Request, error) {
	releaseVersion = strings.TrimSpace(releaseVersion)
	if releaseVersion == "" {
		if !allowUnversioned {
			return initialization.Request{}, fmt.Errorf(
				"release version is required (-release-version or ITSM_RELEASE_VERSION)",
			)
		}
		releaseVersion = "unversioned"
	}
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		hostname = requestedBy
	}
	executorID, err := initialization.NewExecutorID(hostname)
	if err != nil {
		return initialization.Request{}, fmt.Errorf("create initialization executor id: %w", err)
	}
	return initialization.Request{
		Scope:          PlatformScope(),
		TargetVersion:  CurrentTenantTemplateVersion,
		ReleaseVersion: releaseVersion,
		RequestedBy:    requestedBy,
		ExecutorID:     executorID,
	}, nil
}
