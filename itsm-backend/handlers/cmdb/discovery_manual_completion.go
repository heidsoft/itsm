package cmdb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/service"
)

// 人工补全入口：发现回写把「凑不齐必填属性」的资源记为 deferred，
// 这里把这笔账交还给运营人员——列出待补全资源、补录属性后建账。
// 必填项的事实来源是 service.CIAttributeValidator（与 CreateCI 校验同一份查询），
// 本文件不得再写一份「哪些属性必填」的逻辑。

// ErrDiscoveryProjectorNotWired 表示 CI 域装配缺失（projector 或属性定义读取器未注入）。
// 这是运维态不可用，不是用户参数错误，handler 必须映射为 5003 而不是空列表。
var ErrDiscoveryProjectorNotWired = errors.New("discovery ci projector not wired")

// CIRequirementReader 读取某 CI 类型（含继承链）生效的必填属性定义，
// 生产实现是 *service.ConfigurationItemService。
type CIRequirementReader interface {
	ListRequiredCIAttributes(ctx context.Context, tenantID, ciTypeID int) ([]service.CIAttributeRequirement, error)
}

// PendingDiscoveryAttribute 是一条必填属性在某个待补全资源上的现状：
// 值只可能来自云标签（建账时 Attributes 就是标签快照），satisfied 表示无需人工补录。
type PendingDiscoveryAttribute struct {
	service.CIAttributeRequirement
	Satisfied    bool
	CurrentValue string
}

// PendingDiscoveryResource 是一条「映射了 CI 类型、但因必填属性凑不齐而未建账」的云资源。
type PendingDiscoveryResource struct {
	Resource         *CloudResource
	TargetCITypeName string
	TargetCITypeID   int
	Attributes       []PendingDiscoveryAttribute
}

// MissingAttributeNames 返回仍需人工补录的属性名（保持 Attributes 的稳定顺序）。
func (r *PendingDiscoveryResource) MissingAttributeNames() []string {
	names := make([]string, 0, len(r.Attributes))
	for _, attr := range r.Attributes {
		if !attr.Satisfied {
			names = append(names, attr.Name)
		}
	}
	return names
}

// ManualCompletionRequest 是人工补全建账的提交内容；零值字段表示「不覆盖」。
type ManualCompletionRequest struct {
	Name        string
	Environment string
	AssignedTo  string
	Attributes  map[string]interface{}
}

// ListPending 推导出租户内「需要人工补全」的云资源全集（未分页）。
//
// 排除口径与发现回写一致：
//   - 类型未映射（discoveryCITypeByResource 没有命中）——回写本来就会 skip，不属于补全；
//   - 租户缺少目标 CI 类型——补 CI 类型是另一条运维路径；
//   - 已按 ref 或 (provider, 云上资源ID) 链接了配置项——已有账，别建重复的；
//   - 必填项全部可由标签或默认值满足——上一轮 deferred 不是必填缺口造成
//     （如唯一值冲突），列出来只会误导运营「补了就能建账」。
func (p *DiscoveryCIProjector) ListPending(ctx context.Context, tenantID int) ([]*PendingDiscoveryResource, error) {
	if p == nil || p.repo == nil || p.requirements == nil {
		return nil, ErrDiscoveryProjectorNotWired
	}
	resources, err := p.repo.ListCloudResources(ctx, tenantID, CloudResourceFilter{})
	if err != nil {
		return nil, fmt.Errorf("查询云资源失败: %w", err)
	}
	cis, err := p.repo.ListCIsForReconciliation(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("查询已建账配置项失败: %w", err)
	}
	linkedRefIDs := make(map[int]struct{}, len(cis))
	linkedIdentities := make(map[string]struct{}, len(cis))
	for _, ci := range cis {
		if ci.CloudResourceRefID > 0 {
			linkedRefIDs[ci.CloudResourceRefID] = struct{}{}
		}
		if ci.CloudResourceID != "" {
			linkedIdentities[ci.CloudResourceID] = struct{}{}
		}
	}

	// 同一目标类型只查一次属性定义（本轮调用内缓存）
	requirementsCache := make(map[int][]service.CIAttributeRequirement)

	pending := make([]*PendingDiscoveryResource, 0)
	for _, res := range resources {
		if res == nil || res.ID <= 0 {
			continue
		}
		if _, ok := linkedRefIDs[res.ID]; ok {
			continue
		}
		if _, ok := linkedIdentities[res.ResourceID]; ok {
			continue
		}
		ciTypeName, ok := CITypeForCloudResource(res.ServiceCode, res.ResourceType)
		if !ok {
			continue
		}
		ciTypeID, found := p.resolveCITypeID(ctx, tenantID, ciTypeName)
		if !found {
			continue
		}
		defs, cached := requirementsCache[ciTypeID]
		if !cached {
			var err error
			defs, err = p.requirements.ListRequiredCIAttributes(ctx, tenantID, ciTypeID)
			if err != nil {
				return nil, fmt.Errorf("查询 CI 类型 %s 必填属性失败: %w", ciTypeName, err)
			}
			requirementsCache[ciTypeID] = defs
		}
		if len(defs) == 0 {
			continue
		}

		attributes := make([]PendingDiscoveryAttribute, 0, len(defs))
		missing := 0
		for _, def := range defs {
			value, hasTag := res.Tags[def.Name]
			value = strings.TrimSpace(value)
			current := ""
			if hasTag {
				current = value
			}
			// 与 NormalizeAttributes 的通过条件对齐：默认值可填充，或标签提供非空白值
			satisfied := def.DefaultValue != "" || (hasTag && value != "")
			if !satisfied {
				missing++
			}
			attributes = append(attributes, PendingDiscoveryAttribute{
				CIAttributeRequirement: def,
				Satisfied:              satisfied,
				CurrentValue:           current,
			})
		}
		if missing == 0 {
			continue
		}
		pending = append(pending, &PendingDiscoveryResource{
			Resource:         res,
			TargetCITypeName: ciTypeName,
			TargetCITypeID:   ciTypeID,
			Attributes:       attributes,
		})
	}

	sort.Slice(pending, func(i, j int) bool {
		a, b := pending[i], pending[j]
		if a.TargetCITypeName != b.TargetCITypeName {
			return a.TargetCITypeName < b.TargetCITypeName
		}
		if a.Resource.ResourceID != b.Resource.ResourceID {
			return a.Resource.ResourceID < b.Resource.ResourceID
		}
		return a.Resource.ID < b.Resource.ID
	})
	return pending, nil
}

// CompleteManually 用人工补录的属性把一条待建账云资源写成配置项。
// 校验/发号/历史仍然只有 ConfigurationItemService.CreateCI 一个所有者；
// 这里只决定「带哪些值」：云标签打底，人工提交覆盖同名键。
func (p *DiscoveryCIProjector) CompleteManually(
	ctx context.Context,
	tenantID, resourceID, operatorID int,
	req ManualCompletionRequest,
) (*dto.CIResponse, error) {
	if p == nil || p.repo == nil || p.writer == nil || p.requirements == nil {
		return nil, ErrDiscoveryProjectorNotWired
	}
	res, err := p.repo.GetCloudResource(ctx, tenantID, resourceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, common.NewNotFoundError("云资源")
		}
		return nil, fmt.Errorf("查询云资源失败: %w", err)
	}

	ciTypeName, ok := CITypeForCloudResource(res.ServiceCode, res.ResourceType)
	if !ok {
		return nil, common.NewValidationError(
			fmt.Sprintf("资源类型 %s|%s 未登记 CI 类型映射，请手工创建配置项", res.ServiceCode, res.ResourceType), nil)
	}
	ciTypeID, found := p.resolveCITypeID(ctx, tenantID, ciTypeName)
	if !found {
		return nil, common.NewValidationError(
			fmt.Sprintf("租户尚未创建 CI 类型 %s，无法按发现结果建账", ciTypeName), nil)
	}
	if linked := p.findLinkedCI(ctx, res); linked != nil {
		return nil, common.NewConflictError("云资源已建账",
			fmt.Sprintf("该资源已关联配置项 #%d（%s），请直接编辑该配置项", linked.ID, linked.CINumber))
	}

	createReq := p.newCreateCIRequest(res, ciTypeID)
	if name := strings.TrimSpace(req.Name); name != "" {
		createReq.Name = name
	}
	if env := strings.TrimSpace(req.Environment); env != "" {
		createReq.Environment = env
	}
	if owner := strings.TrimSpace(req.AssignedTo); owner != "" {
		createReq.AssignedTo = owner
	}
	merged := make(map[string]interface{}, len(createReq.Attributes)+len(req.Attributes))
	for k, v := range createReq.Attributes {
		merged[k] = v
	}
	for k, v := range req.Attributes {
		merged[k] = v
	}
	createReq.Attributes = merged

	created, err := p.writer.CreateCI(ctx, createReq, tenantID)
	if err != nil {
		if errors.Is(err, service.ErrCIAttributeValidation) {
			return nil, common.NewValidationError(err.Error(), err)
		}
		p.logger.Warnw("discovery manual completion failed",
			"tenant_id", tenantID, "cloud_resource_id", resourceID, "operator_id", operatorID, "error", err)
		return nil, fmt.Errorf("创建配置项失败: %w", err)
	}
	p.logger.Infow("configuration item created by manual completion",
		"tenant_id", tenantID, "ci_id", created.ID, "cloud_resource_id", resourceID, "operator_id", operatorID)
	return created, nil
}
