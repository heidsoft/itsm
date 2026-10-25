package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/configurationitemhistory"

	"go.uber.org/zap"
)

// CIHistoryService CI历史服务
type CIHistoryService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCIHistoryService 创建CI历史服务
func NewCIHistoryService(client *ent.Client, logger *zap.SugaredLogger) *CIHistoryService {
	return &CIHistoryService{
		client: client,
		logger: logger,
	}
}

// RecordCIHistory 记录CI变更历史
func (s *CIHistoryService) RecordCIHistory(
	ctx context.Context,
	ciID, tenantID, operatorID int,
	operatorName, operation, remark string,
	before, after *ent.ConfigurationItem,
) error {
	// operator_id 无法解析时使用系统操作者约定值，保证历史必写成功
	if operatorID < 0 {
		operatorID = SystemOperatorID
	}

	// 转换before和after为map
	beforeMap := s.ciToMap(before)
	afterMap := s.ciToMap(after)

	// 计算变更的字段
	changedFields := s.getChangedFields(beforeMap, afterMap)

	// MAX+1 取版本号存在并发竞态：撞 (ci_id, version) 唯一索引时重新取版本号重试
	const maxAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		version, err := s.nextHistoryVersion(ctx, ciID, tenantID)
		if err != nil {
			s.logger.Errorw("Failed to get max CI version", "error", err, "ci_id", ciID)
			return fmt.Errorf("failed to get max version: %w", err)
		}

		_, err = s.client.ConfigurationItemHistory.Create().
			SetCiID(ciID).
			SetVersion(version).
			SetOperation(operation).
			SetBefore(beforeMap).
			SetAfter(afterMap).
			SetChangedFields(changedFields).
			SetOperatorID(operatorID).
			SetOperatorName(operatorName).
			SetRemark(remark).
			SetTenantID(tenantID).
			Save(ctx)
		if err == nil {
			return nil
		}
		lastErr = err
		if !ent.IsConstraintError(err) {
			break
		}
		s.logger.Warnw("CI history version conflict, retrying", "ci_id", ciID, "version", version, "attempt", attempt+1)
	}

	s.logger.Errorw("Failed to record CI history", "error", lastErr, "ci_id", ciID, "operation", operation)
	return fmt.Errorf("failed to record history: %w", lastErr)
}

// nextHistoryVersion 取当前最大历史版本号+1
func (s *CIHistoryService) nextHistoryVersion(ctx context.Context, ciID, tenantID int) (int, error) {
	// 确保 CI 存在且属于当前租户，防止孤儿历史记录
	exists, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.IDEQ(ciID),
			configurationitem.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to check CI existence: %w", err)
	}
	if !exists {
		return 0, fmt.Errorf("CI %d not found for tenant %d", ciID, tenantID)
	}

	// 使用 Scan + sql.NullInt64 兜底：MAX 在无记录时返回 NULL，直接 Int(ctx) 会 "converting NULL to int" 500
	var aggResult []struct {
		Max sql.NullInt64 `json:"max"`
	}
	err = s.client.ConfigurationItemHistory.Query().
		Where(
			configurationitemhistory.CiIDEQ(ciID),
			configurationitemhistory.TenantIDEQ(tenantID),
		).
		Aggregate(ent.Max(configurationitemhistory.FieldVersion)).
		Scan(ctx, &aggResult)
	if err != nil {
		return 0, err
	}
	version := 1
	if len(aggResult) > 0 && aggResult[0].Max.Valid && aggResult[0].Max.Int64 > 0 {
		version = int(aggResult[0].Max.Int64) + 1
	}
	return version, nil
}

// GetCIHistory 获取CI历史记录列表
func (s *CIHistoryService) GetCIHistory(ctx context.Context, ciID, tenantID int, page, pageSize int) (*dto.CIHistoryListResponse, error) {
	// 分页形状在本层如实归一：HTTP 入口走 common.GetPaginationFromQuery，直接调用 service 的
	// 调用方由这里兜住，否则 pageSize=0 会退化成「不加 LIMIT 的整表查询」、page<=0 是负 OFFSET。
	page, pageSize = common.ValidatePagination(page, pageSize)

	// 检查CI是否存在
	exists, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.IDEQ(ciID),
			configurationitem.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check CI existence", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to check CI existence: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("CI not found")
	}

	query := s.client.ConfigurationItemHistory.Query().
		Where(
			configurationitemhistory.CiIDEQ(ciID),
			configurationitemhistory.TenantIDEQ(tenantID),
		)

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count CI history", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to count history: %w", err)
	}

	histories, err := query.
		Offset((page-1)*pageSize).
		Limit(pageSize).
		Order(ent.Desc(configurationitemhistory.FieldVersion), ent.Asc(configurationitemhistory.FieldID)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CI history", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to list history: %w", err)
	}

	// 存量快照按列名写入（含被拆坏的缩写），在读取侧归一成 camelCase 契约键；
	// 原记录不回写，审计内容保持不变。
	for _, history := range histories {
		history.Before = normalizeSnapshot(history.Before)
		history.After = normalizeSnapshot(history.After)
		history.ChangedFields = normalizeChangedFields(history.ChangedFields)
	}

	pagination := common.NewPaginationResponse(page, pageSize, int64(total))
	return &dto.CIHistoryListResponse{
		Items:      dto.ToCIHistoryResponseList(histories),
		Total:      total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: pagination.TotalPages,
	}, nil
}

// RevertCIVersion 回滚CI到指定版本
func (s *CIHistoryService) RevertCIVersion(ctx context.Context, ciID, tenantID, operatorID int, operatorName string, req *dto.RevertCIVersionRequest) (*dto.CIResponse, error) {
	// 获取指定版本的历史记录
	history, err := s.client.ConfigurationItemHistory.Query().
		Where(
			configurationitemhistory.CiIDEQ(ciID),
			configurationitemhistory.VersionEQ(req.Version),
			configurationitemhistory.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("version %d not found", req.Version)
		}
		s.logger.Errorw("Failed to get CI history version", "error", err, "ci_id", ciID, "version", req.Version)
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	// 获取当前CI
	currentCI, err := s.client.ConfigurationItem.Query().
		Where(
			configurationitem.IDEQ(ciID),
			configurationitem.TenantIDEQ(tenantID),
		).
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to get current CI", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to get current CI: %w", err)
	}

	// 从历史记录的after字段恢复数据。
	// 键按 camelCase 契约读取，并兼容旧快照的列名键（见 snapshotValue）；
	// 整数字段必须兼容 JSON 反序列化后的 float64，否则会静默跳过。
	after := history.After
	update := s.client.ConfigurationItem.UpdateOneID(ciID)

	// 恢复字段
	if name, ok := snapshotString(after, "name"); ok {
		update.SetName(name)
	}
	if ciTypeID, ok := snapshotInt(after, "ciTypeId"); ok {
		update.SetCiTypeID(ciTypeID)
	}
	if ciType, ok := snapshotString(after, "type"); ok {
		update.SetCiType(ciType)
	}
	if status, ok := snapshotString(after, "status"); ok {
		update.SetStatus(status)
	}
	if environment, ok := snapshotString(after, "environment"); ok {
		update.SetEnvironment(environment)
	}
	if criticality, ok := snapshotString(after, "criticality"); ok {
		update.SetCriticality(criticality)
	}
	if assetTag, ok := snapshotString(after, "assetTag"); ok {
		update.SetAssetTag(assetTag)
	}
	if serialNumber, ok := snapshotString(after, "serialNumber"); ok {
		update.SetSerialNumber(serialNumber)
	}
	if model, ok := snapshotString(after, "model"); ok {
		update.SetModel(model)
	}
	if vendor, ok := snapshotString(after, "vendor"); ok {
		update.SetVendor(vendor)
	}
	if location, ok := snapshotString(after, "location"); ok {
		update.SetLocation(location)
	}
	if assignedTo, ok := snapshotString(after, "assignedTo"); ok {
		update.SetAssignedTo(assignedTo)
	}
	if ownedBy, ok := snapshotString(after, "ownedBy"); ok {
		update.SetOwnedBy(ownedBy)
	}
	if discoverySource, ok := snapshotString(after, "discoverySource"); ok {
		update.SetDiscoverySource(discoverySource)
	}
	if source, ok := snapshotString(after, "source"); ok {
		update.SetSource(source)
	}
	if attributes, ok := snapshotJSONMap(after, "attributes"); ok {
		update.SetAttributes(attributes)
	}
	if cloudProvider, ok := snapshotString(after, "cloudProvider"); ok {
		update.SetCloudProvider(cloudProvider)
	}
	if cloudAccountID, ok := snapshotString(after, "cloudAccountId"); ok {
		update.SetCloudAccountID(cloudAccountID)
	}
	if cloudRegion, ok := snapshotString(after, "cloudRegion"); ok {
		update.SetCloudRegion(cloudRegion)
	}
	if cloudZone, ok := snapshotString(after, "cloudZone"); ok {
		update.SetCloudZone(cloudZone)
	}
	if cloudResourceID, ok := snapshotString(after, "cloudResourceId"); ok {
		update.SetCloudResourceID(cloudResourceID)
	}
	if cloudResourceType, ok := snapshotString(after, "cloudResourceType"); ok {
		update.SetCloudResourceType(cloudResourceType)
	}
	if cloudMetadata, ok := snapshotJSONMap(after, "cloudMetadata"); ok {
		update.SetCloudMetadata(cloudMetadata)
	}
	if cloudTags, ok := snapshotJSONMap(after, "cloudTags"); ok {
		update.SetCloudTags(cloudTags)
	}
	if cloudMetrics, ok := snapshotJSONMap(after, "cloudMetrics"); ok {
		update.SetCloudMetrics(cloudMetrics)
	}
	if cloudSyncStatus, ok := snapshotString(after, "cloudSyncStatus"); ok {
		update.SetCloudSyncStatus(cloudSyncStatus)
	}
	if cloudResourceRefID, ok := snapshotInt(after, "cloudResourceRefId"); ok {
		update.SetCloudResourceRefID(cloudResourceRefID)
	}
	// tags字段在ConfigurationItem entity中不存在，跳过

	// 保存更新
	updatedCI, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to revert CI version", "error", err, "ci_id", ciID, "version", req.Version)
		return nil, fmt.Errorf("failed to revert version: %w", err)
	}

	// 记录回滚历史
	err = s.RecordCIHistory(
		ctx,
		ciID,
		tenantID,
		operatorID,
		operatorName,
		"revert",
		fmt.Sprintf("Reverted to version %d: %s", req.Version, req.Remark),
		currentCI,
		updatedCI,
	)
	if err != nil {
		s.logger.Warnw("Failed to record revert history", "error", err, "ci_id", ciID)
	}

	// 重新加载CI数据
	updatedCI, err = s.client.ConfigurationItem.Query().
		Where(
			configurationitem.IDEQ(ciID),
			configurationitem.TenantIDEQ(tenantID),
		).
		WithOutgoingRelations().
		WithIncomingRelations().
		WithTags().
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to reload reverted CI", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to reload CI: %w", err)
	}

	s.logger.Infow("CI reverted successfully", "ci_id", ciID, "version", req.Version, "tenant_id", tenantID)
	return dto.ToCIResponseWithRelations(updatedCI), nil
}

// ciToMap 生成 CI 快照。
//
// 快照原样出现在 GET /cis/:id/history 的 before/after 中，因此键必须是 API
// 契约的 camelCase，而不是持久化列名。历史实现用「Go 字段名 → snake_case」，
// 既把 ci_type_id 这类列名泄漏进 API，又把连续大写缩写拆坏
// （ID → i_d、CINumber → c_i_number），调用方无法映射回 DTO。
//
// 现在直接以 ToCIResponse 的 JSON 投影为准：键与 GET /cis/:id 完全一致，
// DTO 新增字段会自动进入快照，不需要再维护第二份字段清单。
func (s *CIHistoryService) ciToMap(ci *ent.ConfigurationItem) map[string]interface{} {
	if ci == nil {
		return nil
	}
	raw, err := json.Marshal(dto.ToCIResponse(ci))
	if err != nil {
		s.logger.Errorw("Failed to marshal CI snapshot", "error", err, "ci_id", ci.ID)
		return nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		s.logger.Errorw("Failed to unmarshal CI snapshot", "error", err, "ci_id", ci.ID)
		return nil
	}
	return out
}

// getChangedFields 比较两个map，返回变更的字段列表
func (s *CIHistoryService) getChangedFields(before, after map[string]interface{}) []string {
	if before == nil {
		// 创建操作，所有字段都是变更的
		fields := make([]string, 0, len(after))
		for k := range after {
			fields = append(fields, k)
		}
		return fields
	}
	if after == nil {
		// 删除操作
		return []string{"all"}
	}

	changed := make([]string, 0)
	// 检查before中的字段是否在after中变更
	for k, vBefore := range before {
		vAfter, exists := after[k]
		if !exists || !reflect.DeepEqual(vBefore, vAfter) {
			changed = append(changed, k)
		}
	}
	// 检查after中新增的字段
	for k := range after {
		if _, exists := before[k]; !exists {
			changed = append(changed, k)
		}
	}

	return changed
}

// legacySnapshotKeyFixups 还原旧快照（2026-10 之前）里被拆坏的缩写键。
//
// 旧实现按 Go 字段名逐个大写字母前插下划线，连续大写缩写会被拆成单字母段：
// ID→i_d、CINumber→c_i_number。这类键无法用通用 snake→camel 还原，只能显式列出。
// 仅用于读取存量快照；新快照一律写 camelCase，存量数据自然过期后可删除。
var legacySnapshotKeyFixups = map[string]string{
	"id":                 "i_d",
	"type":               "ci_type", // DTO 的 type 对应 Ent 字段 CiType
	"ciTypeId":           "ci_type_i_d",
	"ciNumber":           "c_i_number",
	"sourceId":           "source_i_d",
	"cloudAccountId":     "cloud_account_i_d",
	"cloudResourceId":    "cloud_resource_i_d",
	"cloudResourceRefId": "cloud_resource_ref_i_d",
}

// snakeCase 把 camelCase 键转成下划线形式，用于匹配旧快照里的常规键
// （asset_tag、cloud_provider 等）。
func snakeCase(key string) string {
	var b strings.Builder
	for i, r := range key {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

// snapshotValue 读取快照字段：先取 camelCase 契约键，再回退到旧快照键。
func snapshotValue(snapshot map[string]interface{}, key string) (interface{}, bool) {
	if snapshot == nil {
		return nil, false
	}
	if v, ok := snapshot[key]; ok {
		return v, true
	}
	if v, ok := snapshot[snakeCase(key)]; ok {
		return v, true
	}
	if legacy, ok := legacySnapshotKeyFixups[key]; ok {
		if v, ok := snapshot[legacy]; ok {
			return v, true
		}
	}
	return nil, false
}

func snapshotString(snapshot map[string]interface{}, key string) (string, bool) {
	v, ok := snapshotValue(snapshot, key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// snapshotInt 兼容 JSON 反序列化后的 float64（Ent 的 JSON 字段读取后是 float64），
// 避免旧实现 `after["x"].(int)` 在读取时静默跳过整数字段。
func snapshotInt(snapshot map[string]interface{}, key string) (int, bool) {
	v, ok := snapshotValue(snapshot, key)
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
}

func snapshotJSONMap(snapshot map[string]interface{}, key string) (map[string]interface{}, bool) {
	v, ok := snapshotValue(snapshot, key)
	if !ok {
		return nil, false
	}
	m, ok := v.(map[string]interface{})
	return m, ok
}

// normalizeSnapshot 把存量快照的旧键（列名 / 被拆坏的缩写）统一成 camelCase，
// 仅用于读取侧展示；不回写数据库，历史记录保持原始审计内容。
func normalizeSnapshot(snapshot map[string]interface{}) map[string]interface{} {
	if snapshot == nil {
		return nil
	}
	out := make(map[string]interface{}, len(snapshot))
	for key, value := range snapshot {
		if !strings.Contains(key, "_") {
			out[key] = value
			continue
		}
		out[legacyKeyToCamel(key)] = value
	}
	return out
}

// legacyKeyToCamel 把旧快照的 snake 键还原为 camelCase 契约键。
func legacyKeyToCamel(key string) string {
	for camel, legacy := range legacySnapshotKeyFixups {
		if key == legacy {
			return camel
		}
	}
	parts := strings.Split(key, "_")
	var b strings.Builder
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			b.WriteString(part)
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		b.WriteString(part[1:])
	}
	return b.String()
}

// normalizeChangedFields 同步归一化变更字段列表，避免 API 同时出现
// "ci_type_id" 与 "ciTypeId" 两种风格。
func normalizeChangedFields(fields []string) []string {
	if fields == nil {
		return nil
	}
	out := make([]string, len(fields))
	for i, field := range fields {
		if strings.Contains(field, "_") {
			out[i] = legacyKeyToCamel(field)
			continue
		}
		out[i] = field
	}
	return out
}
