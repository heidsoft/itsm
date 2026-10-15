package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/systemconfig"

	"go.uber.org/zap"
)

// 仪表盘个性化配置的 system_configs 键前缀与分类。
//
// 选型说明：仪表盘布局 / 配置 / 部件清单是「每用户一份」的 JSON 文档，
// 与 system_configs 的 (key, value=json, value_type, tenant_id) 语义吻合。
// 复用既有表可避免为此新增 ent schema + 迁移 + RLS 策略登记，
// 代价是 key 需要拼 user_id —— 对这个规模的键空间是可接受的。
const (
	dashboardConfigKeyPrefix = "dashboard.config"
	dashboardLayoutKeyPrefix = "dashboard.layout"
	dashboardWidgetKeyPrefix = "dashboard.widgets"

	dashboardCategory = "dashboard"
)

// DashboardConfigStore 用 system_configs 承载仪表盘个性化配置。
type DashboardConfigStore struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// LoadDashboardConfig 读取用户的仪表盘配置。
// 返回 (nil, nil) 表示该用户尚未保存过 → 调用方回落默认模板。
func (s *DashboardConfigStore) LoadDashboardConfig(ctx context.Context, tenantID, userID int) (map[string]any, error) {
	return s.loadJSON(ctx, tenantID, fmt.Sprintf("%s.%d", dashboardConfigKeyPrefix, userID))
}

// SaveDashboardConfig 覆盖写入用户的仪表盘配置。
func (s *DashboardConfigStore) SaveDashboardConfig(ctx context.Context, tenantID, userID int, cfg map[string]any) error {
	return s.saveJSON(ctx, tenantID, fmt.Sprintf("%s.%d", dashboardConfigKeyPrefix, userID), "仪表盘配置", cfg)
}

// LoadDashboardLayout 读取用户的仪表盘布局。返回 (nil, nil) 表示未保存过。
func (s *DashboardConfigStore) LoadDashboardLayout(ctx context.Context, tenantID, userID int) (map[string]any, error) {
	return s.loadJSON(ctx, tenantID, fmt.Sprintf("%s.%d", dashboardLayoutKeyPrefix, userID))
}

// SaveDashboardLayout 覆盖写入用户的仪表盘布局。
func (s *DashboardConfigStore) SaveDashboardLayout(ctx context.Context, tenantID, userID int, layout map[string]any) error {
	return s.saveJSON(ctx, tenantID, fmt.Sprintf("%s.%d", dashboardLayoutKeyPrefix, userID), "仪表盘布局", layout)
}

// LoadDashboardWidgets 读取用户的自定义部件清单。返回 (nil, nil) 表示未保存过。
func (s *DashboardConfigStore) LoadDashboardWidgets(ctx context.Context, tenantID, userID int) ([]map[string]any, error) {
	raw, err := s.loadRaw(ctx, tenantID, fmt.Sprintf("%s.%d", dashboardWidgetKeyPrefix, userID))
	if err != nil || raw == "" {
		return nil, err
	}
	var list []map[string]any
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		s.logger.Warnf("仪表盘部件配置 JSON 非法 (tenant=%d user=%d): %v", tenantID, userID, err)
		return nil, nil
	}
	return list, nil
}

// SaveDashboardWidgets 覆盖写入用户的自定义部件清单。
func (s *DashboardConfigStore) SaveDashboardWidgets(ctx context.Context, tenantID, userID int, widgets []map[string]any) error {
	raw, err := json.Marshal(widgets)
	if err != nil {
		return fmt.Errorf("序列化仪表盘部件失败: %w", err)
	}
	return s.writeRaw(ctx, tenantID, fmt.Sprintf("%s.%d", dashboardWidgetKeyPrefix, userID), "仪表盘自定义部件", string(raw))
}

// DeleteDashboardWidgets 清除用户的自定义部件清单（回落默认部件集）。
func (s *DashboardConfigStore) DeleteDashboardWidgets(ctx context.Context, tenantID, userID int) error {
	row, err := s.find(ctx, tenantID, fmt.Sprintf("%s.%d", dashboardWidgetKeyPrefix, userID))
	if err != nil {
		return err // 含 not-found（已视为幂等成功）
	}
	if err := s.client.SystemConfig.DeleteOneID(row.ID).Exec(ctx); err != nil {
		return fmt.Errorf("删除仪表盘部件配置失败: %w", err)
	}
	return nil
}

// ---- 内部通用读写 ----

func (s *DashboardConfigStore) find(ctx context.Context, tenantID int, key string) (*ent.SystemConfig, error) {
	row, err := s.client.SystemConfig.Query().
		Where(
			systemconfig.KeyEQ(key),
			systemconfig.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询配置失败: %w", err)
	}
	return row, nil
}

// loadRaw 读原始 JSON 字符串。行不存在或 value 为空返回 ""。
func (s *DashboardConfigStore) loadRaw(ctx context.Context, tenantID int, key string) (string, error) {
	row, err := s.find(ctx, tenantID, key)
	if err != nil || row == nil || row.Value == "" {
		return "", err
	}
	return row.Value, nil
}

func (s *DashboardConfigStore) loadJSON(ctx context.Context, tenantID int, key string) (map[string]any, error) {
	raw, err := s.loadRaw(ctx, tenantID, key)
	if err != nil || raw == "" {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		s.logger.Warnf("配置 JSON 非法 (key=%s): %v", key, err)
		return nil, nil
	}
	return out, nil
}

func (s *DashboardConfigStore) saveJSON(ctx context.Context, tenantID int, key, desc string, v map[string]any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	return s.writeRaw(ctx, tenantID, key, desc, string(raw))
}

// writeRaw 先查后写地覆盖一份配置。
// 不依赖 ent upsert —— 本项目 ent 未开启 sql/upsert，生成代码没有 OnConflict。
func (s *DashboardConfigStore) writeRaw(ctx context.Context, tenantID int, key, desc, value string) error {
	now := time.Now()
	existing, err := s.find(ctx, tenantID, key)

	switch {
	case err != nil:
		return err
	case existing != nil:
		if _, err := s.client.SystemConfig.UpdateOneID(existing.ID).
			SetValue(value).
			SetValueType("json").
			SetCategory(dashboardCategory).
			SetDescription(desc).
			SetUpdatedAt(now).
			Save(ctx); err != nil {
			return fmt.Errorf("更新配置失败: %w", err)
		}
		return nil
	default:
		if _, err := s.client.SystemConfig.Create().
			SetKey(key).
			SetValue(value).
			SetValueType("json").
			SetCategory(dashboardCategory).
			SetDescription(desc).
			SetTenantID(tenantID).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx); err != nil {
			return fmt.Errorf("创建配置失败: %w", err)
		}
		return nil
	}
}

// NewDashboardConfigStore 创建仪表盘配置 store。
func NewDashboardConfigStore(client *ent.Client, logger *zap.SugaredLogger) *DashboardConfigStore {
	return &DashboardConfigStore{client: client, logger: logger}
}
