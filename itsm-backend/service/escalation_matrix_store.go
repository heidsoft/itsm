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

// escalationMatrixConfigKey 是升级矩阵在 system_configs 表中的配置键。
const escalationMatrixConfigKey = "escalation.matrix"

// escalationMatrixCategory 用于在 /admin/system-config 的分类筛选中定位该配置。
const escalationMatrixCategory = "escalation"

// SystemConfigEscalationMatrixStore 用 system_configs 表承载升级矩阵的持久化。
//
// 选型说明：不新增 ent schema —— 升级矩阵是每租户一份的 JSON 文档，
// 与 system_configs 的 (key, value, value_type, tenant_id) 语义完全吻合，
// 复用既有表同时避免引入新迁移与 RLS 策略登记成本。
type SystemConfigEscalationMatrixStore struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// Load 读取并反序列化该租户的升级矩阵。
//
// 返回约定：
//   - (nil, nil)  租户尚未配置 → 调用方回落 DefaultEscalationMatrix
//   - (m, nil)    命中并成功反序列化
//   - (nil, err)  DB 异常或 value 非法 JSON → 调用方降级默认矩阵并记日志
func (s *SystemConfigEscalationMatrixStore) Load(ctx context.Context, tenantID int) (EscalationMatrix, error) {
	row, err := s.client.SystemConfig.Query().
		Where(
			systemconfig.KeyEQ(escalationMatrixConfigKey),
			systemconfig.TenantIDEQ(tenantID),
			systemconfig.DeletedAtIsNil(),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询升级矩阵配置失败: %w", err)
	}
	if row.Value == "" {
		return nil, nil
	}

	var matrix EscalationMatrix
	if err := json.Unmarshal([]byte(row.Value), &matrix); err != nil {
		return nil, fmt.Errorf("升级矩阵配置 JSON 非法 (config_id=%d): %w", row.ID, err)
	}
	if matrix == nil {
		return nil, nil
	}
	return matrix, nil
}

// Save 覆盖写入该租户的升级矩阵（存在则更新，不存在则创建）。
func (s *SystemConfigEscalationMatrixStore) Save(ctx context.Context, tenantID int, matrix EscalationMatrix) error {
	raw, err := json.Marshal(matrix)
	if err != nil {
		return fmt.Errorf("序列化升级矩阵失败: %w", err)
	}
	value := string(raw)
	now := time.Now()

	// 先查后写：按 (tenant_id, key) 定位现有行。
	// 不依赖 ent 的 upsert —— 本项目 ent 未开启 sql/upsert，生成代码没有
	// OnConflict/ResolveWithIgnore，写了也编译不过。
	existing, err := s.client.SystemConfig.Query().
		Where(
			systemconfig.KeyEQ(escalationMatrixConfigKey),
			systemconfig.TenantIDEQ(tenantID),
		).
		Only(ctx)

	switch {
	case err == nil:
		upd := s.client.SystemConfig.UpdateOneID(existing.ID).
			SetValue(value).
			SetValueType("json").
			SetCategory(escalationMatrixCategory).
			SetDescription("升级矩阵（优先级 → 升级级别链）").
			SetUpdatedAt(now)
		if _, err := upd.Save(ctx); err != nil {
			return fmt.Errorf("更新升级矩阵配置失败: %w", err)
		}
		return nil

	case ent.IsNotFound(err):
		_, err := s.client.SystemConfig.Create().
			SetKey(escalationMatrixConfigKey).
			SetValue(value).
			SetValueType("json").
			SetCategory(escalationMatrixCategory).
			SetDescription("升级矩阵（优先级 → 升级级别链）").
			SetTenantID(tenantID).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("创建升级矩阵配置失败: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("查询升级矩阵配置失败: %w", err)
	}
}

// Delete 清除该租户的自定义矩阵（回落默认矩阵）。
// 行不存在视为已删除，幂等返回 nil。
func (s *SystemConfigEscalationMatrixStore) Delete(ctx context.Context, tenantID int) error {
	existing, err := s.client.SystemConfig.Query().
		Where(
			systemconfig.KeyEQ(escalationMatrixConfigKey),
			systemconfig.TenantIDEQ(tenantID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("查询升级矩阵配置失败: %w", err)
	}
	if err := s.client.SystemConfig.DeleteOneID(existing.ID).Exec(ctx); err != nil {
		return fmt.Errorf("删除升级矩阵配置失败: %w", err)
	}
	return nil
}

// NewSystemConfigEscalationMatrixStore 创建基于 system_configs 的升级矩阵 store。
func NewSystemConfigEscalationMatrixStore(client *ent.Client, logger *zap.SugaredLogger) *SystemConfigEscalationMatrixStore {
	return &SystemConfigEscalationMatrixStore{client: client, logger: logger}
}
