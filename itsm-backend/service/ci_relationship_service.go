package service

import (
	"context"
	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/incident"
	"itsm-backend/ent/ticket"
	"log"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

// CIRelationshipService CI关系服务
type CIRelationshipService struct {
	client *ent.Client
}

// NewCIRelationshipService 创建CI关系服务
func NewCIRelationshipService(client *ent.Client) *CIRelationshipService {
	return &CIRelationshipService{client: client}
}

// CreateCIRelationship 创建CI关系
func (s *CIRelationshipService) CreateCIRelationship(ctx context.Context, req *dto.CreateCIRelationshipRequest, tenantID int) (*dto.CIRelationshipResponse, error) {
	// 检查源CI是否存在
	sourceCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(req.SourceCIID), configurationitem.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("源CI不存在")
		}
		return nil, errors.Wrap(err, "查询源CI失败")
	}

	// 检查目标CI是否存在
	targetCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(req.TargetCIID), configurationitem.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("目标CI不存在")
		}
		return nil, errors.Wrap(err, "查询目标CI失败")
	}

	// 检查关系是否已存在
	exists, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.And(
				cirelationship.SourceCiID(req.SourceCIID),
				cirelationship.TargetCiID(req.TargetCIID),
				cirelationship.RelationshipType(string(req.RelationshipType)),
			),
		).
		Exist(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "检查关系存在性失败")
	}
	if exists {
		return nil, errors.New("该关系已存在")
	}

	// 创建关系
	strength := string(req.Strength)
	if strength == "" {
		strength = "medium"
	}
	impactLevel := string(req.ImpactLevel)
	if impactLevel == "" {
		impactLevel = "medium"
	}

	relationship, err := s.client.CIRelationship.Create().
		SetSourceCiID(req.SourceCIID).
		SetTargetCiID(req.TargetCIID).
		SetRelationshipType(string(req.RelationshipType)).
		SetStrength(cirelationship.Strength(strength)).
		SetImpactLevel(cirelationship.ImpactLevel(impactLevel)).
		SetDescription(req.Description).
		SetMetadata(req.Metadata).
		Save(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "创建CI关系失败")
	}

	return s.toCIRelationshipResponse(relationship, sourceCI, targetCI)
}

// UpdateCIRelationship 更新CI关系
func (s *CIRelationshipService) UpdateCIRelationship(ctx context.Context, id int, req *dto.UpdateCIRelationshipRequest, tenantID int) (*dto.CIRelationshipResponse, error) {
	// 检查关系是否存在
	relationship, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.And(
				cirelationship.ID(id),
				cirelationship.HasSourceCiWith(configurationitem.TenantID(tenantID)),
			),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("CI关系不存在")
		}
		return nil, errors.Wrap(err, "查询CI关系失败")
	}

	// 获取关联的CI
	sourceCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(relationship.SourceCiID)).
		Only(ctx)
	if err != nil {
		sourceCI = nil
	}

	targetCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(relationship.TargetCiID)).
		Only(ctx)
	if err != nil {
		targetCI = nil
	}

	// 更新关系
	if req.RelationshipType != nil {
		relationship.RelationshipType = string(*req.RelationshipType)
	}
	if req.Strength != nil {
		relationship.Strength = cirelationship.Strength(*req.Strength)
	}
	if req.ImpactLevel != nil {
		relationship.ImpactLevel = cirelationship.ImpactLevel(*req.ImpactLevel)
	}
	if req.Description != nil {
		relationship.Description = *req.Description
	}
	if req.Metadata != nil {
		relationship.Metadata = *req.Metadata
	}
	if req.IsActive != nil {
		relationship.IsActive = *req.IsActive
	}

	_, err = s.client.CIRelationship.UpdateOne(relationship).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "更新CI关系失败")
	}

	// 重新查询
	relationship, err = s.client.CIRelationship.Query().
		Where(cirelationship.ID(id)).
		Only(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "重新查询CI关系失败")
	}

	return s.toCIRelationshipResponse(relationship, sourceCI, targetCI)
}

// DeleteCIRelationship 删除CI关系
func (s *CIRelationshipService) DeleteCIRelationship(ctx context.Context, id int, tenantID int) error {
	// 检查关系是否存在
	relationship, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.And(
				cirelationship.ID(id),
				cirelationship.HasSourceCiWith(configurationitem.TenantID(tenantID)),
			),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errors.New("CI关系不存在")
		}
		return errors.Wrap(err, "查询CI关系失败")
	}

	// 删除关系
	err = s.client.CIRelationship.DeleteOne(relationship).
		Exec(ctx)
	if err != nil {
		return errors.Wrap(err, "删除CI关系失败")
	}

	return nil
}

// GetCIRelationships 获取CI的所有关系
func (s *CIRelationshipService) GetCIRelationships(ctx context.Context, req *dto.GetCIRelationshipsRequest, tenantID int) (*dto.GetCIRelationshipsResponse, error) {
	// 检查CI是否存在
	_, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(req.CIID), configurationitem.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("CI不存在")
		}
		return nil, errors.Wrap(err, "查询CI失败")
	}

	response := &dto.GetCIRelationshipsResponse{}

	// 查询出边关系
	if req.IncludeOutgoing {
		query := s.client.CIRelationship.Query().
			Where(cirelationship.SourceCiID(req.CIID))

		if req.RelationshipType != "" {
			query = query.Where(cirelationship.RelationshipType(string(req.RelationshipType)))
		}
		if req.ActiveOnly {
			query = query.Where(cirelationship.IsActive(true))
		}

		relationships, err := query.
			WithTargetCi().
			All(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "查询出边关系失败")
		}

		response.TotalOutgoing = len(relationships)
		for _, rel := range relationships {
			resp, _ := s.toCIRelationshipResponse(rel, nil, rel.Edges.TargetCi)
			if resp != nil {
				response.OutgoingRelations = append(response.OutgoingRelations, *resp)
			}
		}
	}

	// 查询入边关系
	if req.IncludeIncoming {
		query := s.client.CIRelationship.Query().
			Where(cirelationship.TargetCiID(req.CIID))

		if req.RelationshipType != "" {
			query = query.Where(cirelationship.RelationshipType(string(req.RelationshipType)))
		}
		if req.ActiveOnly {
			query = query.Where(cirelationship.IsActive(true))
		}

		relationships, err := query.
			WithSourceCi().
			All(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "查询入边关系失败")
		}

		response.TotalIncoming = len(relationships)
		for _, rel := range relationships {
			resp, _ := s.toCIRelationshipResponse(rel, rel.Edges.SourceCi, nil)
			if resp != nil {
				response.IncomingRelations = append(response.IncomingRelations, *resp)
			}
		}
	}

	return response, nil
}

// TopologyGraphData 用于构建拓扑图的临时结构
type TopologyGraphData struct {
	Nodes      []dto.TopologyNode
	Edges      []dto.TopologyEdge
	NodesMap   map[int]*dto.TopologyNode
}

// GetTopologyGraph 获取拓扑图
func (s *CIRelationshipService) GetTopologyGraph(ctx context.Context, ciID int, depth int, tenantID int) (*dto.TopologyGraph, error) {
	if depth > 3 {
		depth = 3
	}

	// 验证根节点CI存在
	_, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(ciID), configurationitem.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("CI不存在")
		}
		return nil, errors.Wrap(err, "查询CI失败")
	}

	graph := &dto.TopologyGraph{
		RootCIID: ciID,
		Depth:    depth,
	}

	// BFS遍历构建图
	visited := make(map[int]bool)
	queue := []struct {
		ciID  int
		depth int
	}{{ciID, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.ciID] {
			continue
		}
		visited[current.ciID] = true

		// 获取CI详情
		ci, err := s.client.ConfigurationItem.Query().
			Where(configurationitem.ID(current.ciID)).
			Only(ctx)
		if err != nil {
			continue
		}

		// 添加节点
		node := s.toTopologyNode(ci)
		graph.Nodes = append(graph.Nodes, node)

		if current.depth >= depth {
			continue
		}

		// 查询出边
		outgoing, err := s.client.CIRelationship.Query().
			Where(
				cirelationship.And(
					cirelationship.SourceCiID(current.ciID),
					cirelationship.IsActive(true),
				),
			).
			WithTargetCi().
			All(ctx)
		if err == nil {
			for _, rel := range outgoing {
				edge := s.toTopologyEdge(rel)
				graph.Edges = append(graph.Edges, edge)
				if !visited[rel.TargetCiID] {
					queue = append(queue, struct {
						ciID  int
						depth int
					}{rel.TargetCiID, current.depth + 1})
				}
			}
		}

		// 查询入边
		incoming, err := s.client.CIRelationship.Query().
			Where(
				cirelationship.And(
					cirelationship.TargetCiID(current.ciID),
					cirelationship.IsActive(true),
				),
			).
			WithSourceCi().
			All(ctx)
		if err == nil {
			for _, rel := range incoming {
				// 双向边：反转方向显示
				edge := dto.TopologyEdge{
					ID:                rel.ID,
					Source:            rel.TargetCiID,
					Target:            rel.SourceCiID,
					RelationshipType:  rel.RelationshipType,
					RelationshipLabel: s.getReverseRelationshipLabel(rel.RelationshipType),
					Strength:          string(rel.Strength),
					ImpactLevel:       string(rel.ImpactLevel),
				}
				graph.Edges = append(graph.Edges, edge)
				if !visited[rel.SourceCiID] {
					queue = append(queue, struct {
						ciID  int
						depth int
					}{rel.SourceCiID, current.depth + 1})
				}
			}
		}
	}

	graph.TotalNodes = len(graph.Nodes)
	graph.TotalEdges = len(graph.Edges)

	return graph, nil
}

// AnalyzeImpact 影响分析
func (s *CIRelationshipService) AnalyzeImpact(ctx context.Context, ciID int, tenantID int) (*dto.ImpactAnalysisResponse, error) {
	// 获取目标CI
	ci, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.ID(ciID), configurationitem.TenantID(tenantID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.New("CI不存在")
		}
		return nil, errors.Wrap(err, "查询CI失败")
	}

	targetNode := s.toTopologyNode(ci)
	response := &dto.ImpactAnalysisResponse{
		TargetCI:            &targetNode,
		UpstreamImpact:     []dto.ImpactAnalysisItem{},
		DownstreamImpact:   []dto.ImpactAnalysisItem{},
		CriticalDependencies: []dto.ImpactAnalysisItem{},
	}

	// 分析上游影响
	upstream, _ := s.analyzeUpstreamImpact(ctx, ciID, 0, tenantID)
	response.UpstreamImpact = upstream

	// 分析下游影响
	downstream, _ := s.analyzeDownstreamImpact(ctx, ciID, 0, tenantID)
	response.DownstreamImpact = downstream

	// 找出关键依赖
	for _, item := range downstream {
		if item.ImpactLevel == "critical" && item.Relationship == "depends_on" {
			response.CriticalDependencies = append(response.CriticalDependencies, item)
		}
	}

	// 获取受影响的工单
	tickets, err := s.client.Ticket.Query().
		Where(
			ticket.HasConfigurationItemsWith(configurationitem.ID(ciID)),
		).
		Limit(10).
		All(ctx)
	if err == nil {
		for _, t := range tickets {
			response.AffectedTickets = append(response.AffectedTickets, dto.AffectedTicket{
				ID:        t.ID,
				Title:     t.Title,
				Status:    t.Status,
				Priority:  t.Priority,
				CreatedAt: t.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	// 获取受影响的事件
	incidents, err := s.client.Incident.Query().
		Where(
			incident.ConfigurationItemID(ciID),
		).
		Limit(10).
		All(ctx)
	if err == nil {
		for _, i := range incidents {
			response.AffectedIncidents = append(response.AffectedIncidents, dto.AffectedIncident{
				ID:        i.ID,
				Title:     i.Title,
				Status:    i.Status,
				Severity:  i.Severity,
				CreatedAt: i.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}

	// 计算风险等级
	response.RiskLevel = s.calculateRiskLevel(response)
	response.Summary = s.generateImpactSummary(response)

	return response, nil
}

// BatchCreateCIRelationships 批量创建CI关系
func (s *CIRelationshipService) BatchCreateCIRelationships(ctx context.Context, req *dto.BatchCreateCIRelationshipRequest, tenantID int) (*dto.BatchCreateCIRelationshipResponse, error) {
	response := &dto.BatchCreateCIRelationshipResponse{
		CreatedCount: 0,
		FailedCount:  0,
		Errors:       []string{},
	}

	for i, rel := range req.Relationships {
		_, err := s.CreateCIRelationship(ctx, &rel, tenantID)
		if err != nil {
			response.FailedCount++
			response.Errors = append(response.Errors, "关系创建失败: "+err.Error())
		} else {
			response.CreatedCount++
		}
		log.Printf("创建CI关系 %d/%d: %d -> %d (%s)", i+1, len(req.Relationships), rel.SourceCIID, rel.TargetCIID, rel.RelationshipType)
	}

	return response, nil
}

// GetRelationshipTypes 获取所有关系类型
func (s *CIRelationshipService) GetRelationshipTypes() []dto.RelationshipTypeInfo {
	return []dto.RelationshipTypeInfo{
		{Type: dto.DependsOn, Name: "依赖", Description: "A依赖于B，B故障会影响A", Direction: "bi-directional", Icon: "link"},
		{Type: dto.Hosts, Name: "托管", Description: "A托管于B（如应用部署在服务器）", Direction: "uni-directional", Icon: "server"},
		{Type: dto.HostedOn, Name: "所属", Description: "A所属B（如服务器在机架）", Direction: "uni-directional", Icon: "database"},
		{Type: dto.ConnectsTo, Name: "连接", Description: "A连接到B（如网络连接）", Direction: "bi-directional", Icon: "network"},
		{Type: dto.RunsOn, Name: "运行", Description: "A运行在B上", Direction: "uni-directional", Icon: "play-circle"},
		{Type: dto.Contains, Name: "包含", Description: "A包含B", Direction: "uni-directional", Icon: "folder"},
		{Type: dto.PartOf, Name: "组成", Description: "A是B的一部分", Direction: "uni-directional", Icon: "block"},
		{Type: dto.Impacts, Name: "影响", Description: "A影响B", Direction: "uni-directional", Icon: "warning"},
		{Type: dto.ImpactedBy, Name: "受影响于", Description: "A受B影响", Direction: "uni-directional", Icon: "alert"},
		{Type: dto.Owns, Name: "拥有", Description: "A拥有B", Direction: "uni-directional", Icon: "key"},
		{Type: dto.OwnedBy, Name: "被拥有", Description: "A被B拥有", Direction: "uni-directional", Icon: "lock"},
		{Type: dto.Uses, Name: "使用", Description: "A使用B", Direction: "uni-directional", Icon: "tool"},
		{Type: dto.UsedBy, Name: "被使用", Description: "A被B使用", Direction: "uni-directional", Icon: "setting"},
	}
}

// 辅助函数

func (s *CIRelationshipService) toCIRelationshipResponse(rel *ent.CIRelationship, sourceCI, targetCI *ent.ConfigurationItem) (*dto.CIRelationshipResponse, error) {
	sourceCIName := ""
	sourceCIType := ""
	targetCIName := ""
	targetCIType := ""

	if sourceCI != nil {
		sourceCIName = sourceCI.Name
		sourceCIType = sourceCI.CiType
	}
	if targetCI != nil {
		targetCIName = targetCI.Name
		targetCIType = targetCI.CiType
	}

	return &dto.CIRelationshipResponse{
		ID:                rel.ID,
		SourceCIID:        rel.SourceCiID,
		SourceCIName:      sourceCIName,
		SourceCIType:      sourceCIType,
		TargetCIID:        rel.TargetCiID,
		TargetCIName:      targetCIName,
		TargetCIType:      targetCIType,
		RelationshipType:  dto.CIRelationshipType(rel.RelationshipType),
		RelationshipTypeName: s.getRelationshipTypeName(rel.RelationshipType),
		Strength:          dto.RelationshipStrength(rel.Strength),
		ImpactLevel:       dto.ImpactLevel(rel.ImpactLevel),
		IsActive:         rel.IsActive,
		IsDiscovered:     rel.IsDiscovered,
		Description:       rel.Description,
		Metadata:          rel.Metadata,
		CreatedAt:        rel.CreatedAt,
		UpdatedAt:        rel.UpdatedAt,
	}, nil
}

func (s *CIRelationshipService) toTopologyNode(ci *ent.ConfigurationItem) dto.TopologyNode {
	return dto.TopologyNode{
		ID:         ci.ID,
		Name:       ci.Name,
		Type:       ci.CiType,
		TypeName:   ci.CiType,
		Status:     ci.Status,
		Criticality: ci.Criticality,
		Attributes: ci.Attributes,
	}
}

func (s *CIRelationshipService) toTopologyEdge(rel *ent.CIRelationship) dto.TopologyEdge {
	return dto.TopologyEdge{
		ID:                rel.ID,
		Source:            rel.SourceCiID,
		Target:            rel.TargetCiID,
		RelationshipType:  rel.RelationshipType,
		RelationshipLabel: s.getRelationshipTypeName(rel.RelationshipType),
		Strength:          string(rel.Strength),
		ImpactLevel:       string(rel.ImpactLevel),
	}
}

func (s *CIRelationshipService) getRelationshipTypeName(relType string) string {
	typeNames := map[string]string{
		"depends_on":  "依赖",
		"hosts":       "托管",
		"hosted_on":   "所属",
		"connects_to": "连接",
		"runs_on":     "运行",
		"contains":    "包含",
		"part_of":     "组成",
		"impacts":     "影响",
		"impacted_by": "受影响于",
		"owns":        "拥有",
		"owned_by":    "被拥有",
		"uses":        "使用",
		"used_by":     "被使用",
	}
	if name, ok := typeNames[relType]; ok {
		return name
	}
	return relType
}

func (s *CIRelationshipService) getReverseRelationshipLabel(relType string) string {
	reverseLabels := map[string]string{
		"depends_on":  "被依赖",
		"hosts":       "部署在",
		"hosted_on":   "包含",
		"connects_to": "已连接",
		"runs_on":     "运行于",
		"contains":    "被包含",
		"part_of":     "包含",
		"impacts":     "受影响于",
		"impacted_by": "影响",
		"owns":        "被拥有",
		"owned_by":    "拥有",
		"uses":        "被使用",
		"used_by":     "使用",
	}
	if label, ok := reverseLabels[relType]; ok {
		return label
	}
	return relType
}

func (s *CIRelationshipService) analyzeUpstreamImpact(ctx context.Context, ciID int, distance int, tenantID int) ([]dto.ImpactAnalysisItem, error) {
	var items []dto.ImpactAnalysisItem

	relatives, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.And(
				cirelationship.TargetCiID(ciID),
				cirelationship.IsActive(true),
			),
		).
		WithSourceCi().
		All(ctx)
	if err != nil {
		return nil, err
	}

	for _, rel := range relatives {
		if rel.Edges.SourceCi == nil {
			continue
		}

		item := dto.ImpactAnalysisItem{
			CIID:         rel.SourceCiID,
			CIName:       rel.Edges.SourceCi.Name,
			CIT:          rel.Edges.SourceCi.CiType,
			Relationship: s.getReverseRelationshipLabel(rel.RelationshipType),
			ImpactLevel:  dto.ImpactLevel(rel.ImpactLevel),
			Distance:     distance,
			Direction:    "upstream",
		}
		items = append(items, item)

		// 递归获取上游
		if distance < 3 {
			subItems, _ := s.analyzeUpstreamImpact(ctx, rel.SourceCiID, distance+1, tenantID)
			items = append(items, subItems...)
		}
	}

	return items, nil
}

func (s *CIRelationshipService) analyzeDownstreamImpact(ctx context.Context, ciID int, distance int, tenantID int) ([]dto.ImpactAnalysisItem, error) {
	var items []dto.ImpactAnalysisItem

	relatives, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.And(
				cirelationship.SourceCiID(ciID),
				cirelationship.IsActive(true),
			),
		).
		WithTargetCi().
		All(ctx)
	if err != nil {
		return nil, err
	}

	for _, rel := range relatives {
		if rel.Edges.TargetCi == nil {
			continue
		}

		item := dto.ImpactAnalysisItem{
			CIID:         rel.TargetCiID,
			CIName:       rel.Edges.TargetCi.Name,
			CIT:          rel.Edges.TargetCi.CiType,
			Relationship: s.getRelationshipTypeName(rel.RelationshipType),
			ImpactLevel:  dto.ImpactLevel(rel.ImpactLevel),
			Distance:     distance,
			Direction:    "downstream",
		}
		items = append(items, item)

		// 递归获取下游
		if distance < 3 {
			subItems, _ := s.analyzeDownstreamImpact(ctx, rel.TargetCiID, distance+1, tenantID)
			items = append(items, subItems...)
		}
	}

	return items, nil
}

func (s *CIRelationshipService) calculateRiskLevel(response *dto.ImpactAnalysisResponse) string {
	criticalCount := 0
	highCount := 0

	for _, item := range response.DownstreamImpact {
		if item.ImpactLevel == "critical" {
			criticalCount++
		} else if item.ImpactLevel == "high" {
			highCount++
		}
	}

	if criticalCount > 0 {
		return "critical"
	}
	if highCount > 2 {
		return "high"
	}
	if highCount > 0 {
		return "medium"
	}
	return "low"
}

func (s *CIRelationshipService) generateImpactSummary(response *dto.ImpactAnalysisResponse) string {
	upstreamCount := len(response.UpstreamImpact)
	downstreamCount := len(response.DownstreamImpact)
	criticalCount := len(response.CriticalDependencies)
	ticketCount := len(response.AffectedTickets)
	incidentCount := len(response.AffectedIncidents)

	summary := "该CI"
	if upstreamCount > 0 {
		summary += "被" + strconv.Itoa(upstreamCount) + "个系统依赖"
	}
	if downstreamCount > 0 {
		if upstreamCount > 0 {
			summary += "，"
		}
		summary += "依赖" + strconv.Itoa(downstreamCount) + "个组件"
	}
	if criticalCount > 0 {
		summary += "，包含" + strconv.Itoa(criticalCount) + "个关键依赖"
	}
	if ticketCount > 0 {
		summary += "，关联" + strconv.Itoa(ticketCount) + "个未关闭工单"
	}
	if incidentCount > 0 {
		summary += "，关联" + strconv.Itoa(incidentCount) + "个未解决事件"
	}

	return summary
}
