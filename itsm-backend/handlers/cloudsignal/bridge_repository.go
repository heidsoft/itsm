package cloudsignal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"itsm-backend/database"
)

// bridgeRepository 用原生 SQL 读取事件与配置项的只读投影。
//
// 建单桥只需要「这个告警是否已经建过单」和「这台实例对应哪个 CI」，
// 不值得为此把两个领域的 Ent 模型拉进本包，因此在这里做窄投影查询。
type bridgeRepository struct {
	db *sql.DB
}

func NewBridgeRepository(db *sql.DB) BridgeRepository {
	if db == nil {
		return nil
	}
	return &bridgeRepository{db: db}
}

func (r *bridgeRepository) FindIncidentByCloudAlertID(ctx context.Context, tenantID int, alertKey string) (int64, error) {
	alertKey = strings.TrimSpace(alertKey)
	if tenantID <= 0 || alertKey == "" {
		return 0, fmt.Errorf("invalid incident lookup input")
	}
	const query = `
		SELECT id FROM incidents
		WHERE tenant_id = $1 AND metadata->>'cloudAlertId' = $2
		ORDER BY id ASC
		LIMIT 1`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (int64, error) {
		var id int64
		err := q.QueryRowContext(ctx, query, tenantID, alertKey).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return id, err
	})
}

func (r *bridgeRepository) FindCIByCloudResource(ctx context.Context, tenantID int, provider, resourceID string) (int64, error) {
	resourceID = strings.TrimSpace(resourceID)
	if tenantID <= 0 || resourceID == "" {
		return 0, fmt.Errorf("invalid configuration item lookup input")
	}
	// provider 为空时按实例 ID 匹配任意厂商：实例 ID 本身已经足够唯一，
	// 但老数据可能没有写 cloud_provider，过度收紧会让 CI 关联静默丢失。
	const query = `
		SELECT id FROM configuration_items
		WHERE tenant_id = $1 AND cloud_resource_id = $2 AND ($3 = '' OR cloud_provider = $3)
		ORDER BY id ASC
		LIMIT 1`
	return database.WithTenantSQL(ctx, r.db, tenantID, func(q database.SQLExecutor) (int64, error) {
		var id int64
		err := q.QueryRowContext(ctx, query, tenantID, resourceID, strings.TrimSpace(provider)).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return id, err
	})
}
