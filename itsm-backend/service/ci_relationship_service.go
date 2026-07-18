package service

import (
	"context"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/cirelationship"
	"itsm-backend/ent/configurationitem"

	"go.uber.org/zap"
)

// CIRelationshipService CI关系服务
type CIRelationshipService struct {
	client *ent.Client
	logger *zap.SugaredLogger
}

// NewCIRelationshipService 创建CI关系服务
func NewCIRelationshipService(client *ent.Client, logger *zap.SugaredLogger) *CIRelationshipService {
	return &CIRelationshipService{
		client: client,
		logger: logger,
	}
}

// CreateCIRelationship 创建CI关系
func (s *CIRelationshipService) CreateCIRelationship(ctx context.Context, req *dto.CreateCIRelationshipRequest, tenantID int) (*dto.CIRelationshipResponse, error) {
	// 检查源CI是否存在
	sourceCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(req.SourceCIID), configurationitem.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("source CI not found")
		}
		s.logger.Errorw("Failed to get source CI", "error", err, "ci_id", req.SourceCIID)
		return nil, fmt.Errorf("failed to get source CI: %w", err)
	}

	// 检查目标CI是否存在
	targetCI, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(req.TargetCIID), configurationitem.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("target CI not found")
		}
		s.logger.Errorw("Failed to get target CI", "error", err, "ci_id", req.TargetCIID)
		return nil, fmt.Errorf("failed to get target CI: %w", err)
	}

	// 检查关系是否已存在
	exists, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.SourceCiIDEQ(req.SourceCIID),
			cirelationship.TargetCiIDEQ(req.TargetCIID),
			cirelationship.RelationshipTypeEQ(string(req.RelationshipType)),
			cirelationship.TenantIDEQ(tenantID),
		).
		Exist(ctx)
	if err != nil {
		s.logger.Errorw("Failed to check relation existence", "error", err)
		return nil, fmt.Errorf("failed to check relation existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("relationship already exists between CI %d and CI %d with type %s",
			req.SourceCIID, req.TargetCIID, req.RelationshipType)
	}

	// 环检测：若 targetCI 已存在通向 sourceCI 的有向路径，则本次插入会形成环
	// 语义：CI 关系图应是 DAG（拓扑）。对同类型关系（如 depends_on 全局）单独判环。
	// 实现：从 targetCIID 出发做 DFS，看是否可达 sourceCIID。
	if req.SourceCIID == req.TargetCIID {
		return nil, fmt.Errorf("relationship would form self-loop: source and target CI are identical")
	}
	cyclic, cycleErr := s.wouldCreateCycle(ctx, tenantID, req.SourceCIID, req.TargetCIID, string(req.RelationshipType))
	if cycleErr != nil {
		s.logger.Errorw("cycle detection failed", "error", cycleErr,
			"source_ci_id", req.SourceCIID, "target_ci_id", req.TargetCIID)
		return nil, fmt.Errorf("failed to detect cycle: %w", cycleErr)
	}
	if cyclic {
		return nil, fmt.Errorf("relationship would create a cycle in CI dependency graph (source=%d target=%d type=%s)",
			req.SourceCIID, req.TargetCIID, req.RelationshipType)
	}

	// 创建关系
	create := s.client.CIRelationship.Create().
		SetRelationshipType(string(req.RelationshipType)).
		SetSourceCiID(req.SourceCIID).
		SetTargetCiID(req.TargetCIID).
		SetTenantID(tenantID)

	if req.Strength != "" {
		create.SetStrength(cirelationship.Strength(req.Strength))
	}
	if req.ImpactLevel != "" {
		create.SetImpactLevel(cirelationship.ImpactLevel(req.ImpactLevel))
	}
	if req.Description != "" {
		create.SetDescription(req.Description)
	}
	if req.Metadata != nil {
		create.SetMetadata(req.Metadata)
	}
	if req.IsDiscovered != nil {
		create.SetIsDiscovered(*req.IsDiscovered)
	}

	relation, err := create.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to create CI relationship", "error", err,
			"source_ci_id", req.SourceCIID, "target_ci_id", req.TargetCIID, "type", req.RelationshipType)
		return nil, fmt.Errorf("failed to create CI relationship: %w", err)
	}

	// 加载关联的CI信息
	relation, err = s.client.CIRelationship.Query().
		Where(cirelationship.IDEQ(relation.ID)).
		WithSourceCi().
		WithTargetCi().
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to load created relation", "error", err, "relation_id", relation.ID)
		return nil, fmt.Errorf("failed to load created relation: %w", err)
	}

	s.logger.Infow("CI relationship created successfully", "relation_id", relation.ID,
		"source_ci_id", sourceCI.ID, "target_ci_id", targetCI.ID, "type", req.RelationshipType)
	return dto.ToCIRelationshipResponse(relation), nil
}

// GetCIRelationshipByID 根据ID获取CI关系
func (s *CIRelationshipService) GetCIRelationshipByID(ctx context.Context, id, tenantID int) (*dto.CIRelationshipResponse, error) {
	relation, err := s.client.CIRelationship.Query().
		Where(cirelationship.IDEQ(id), cirelationship.TenantIDEQ(tenantID)).
		WithSourceCi().
		WithTargetCi().
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		s.logger.Errorw("Failed to get CI relationship", "error", err, "relation_id", id)
		return nil, fmt.Errorf("failed to get CI relationship: %w", err)
	}

	return dto.ToCIRelationshipResponse(relation), nil
}

// ListCIRelationshipsByCIID 根据CI ID获取关系列表
func (s *CIRelationshipService) ListCIRelationshipsByCIID(ctx context.Context, ciID, tenantID int, direction string) ([]*dto.CIRelationshipResponse, error) {
	query := s.client.CIRelationship.Query().
		Where(cirelationship.TenantIDEQ(tenantID), cirelationship.IsActiveEQ(true))

	if direction == "outgoing" {
		query = query.Where(cirelationship.SourceCiIDEQ(ciID))
	} else if direction == "incoming" {
		query = query.Where(cirelationship.TargetCiIDEQ(ciID))
	} else {
		query = query.Where(
			cirelationship.Or(
				cirelationship.SourceCiIDEQ(ciID),
				cirelationship.TargetCiIDEQ(ciID),
			),
		)
	}

	relations, err := query.
		WithSourceCi().
		WithTargetCi().
		Order(ent.Desc(cirelationship.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CI relationships", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to list CI relationships: %w", err)
	}

	return dto.ToCIRelationshipResponseList(relations), nil
}

// ListAllCIRelationships 获取所有CI关系列表
func (s *CIRelationshipService) ListAllCIRelationships(ctx context.Context, tenantID int, page, pageSize int, relationType string) (*dto.CIRelationshipListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	query := s.client.CIRelationship.Query().
		Where(cirelationship.TenantIDEQ(tenantID), cirelationship.IsActiveEQ(true))

	if relationType != "" {
		query = query.Where(cirelationship.RelationshipTypeEQ(relationType))
	}

	total, err := query.Count(ctx)
	if err != nil {
		s.logger.Errorw("Failed to count CI relationships", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to count CI relationships: %w", err)
	}

	relations, err := query.
		WithSourceCi().
		WithTargetCi().
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(ent.Desc(cirelationship.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Errorw("Failed to list CI relationships", "error", err, "tenant_id", tenantID)
		return nil, fmt.Errorf("failed to list CI relationships: %w", err)
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	return &dto.CIRelationshipListResponse{
		Items:      dto.ToCIRelationshipResponseList(relations),
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UpdateCIRelationship 更新CI关系
func (s *CIRelationshipService) UpdateCIRelationship(ctx context.Context, id, tenantID int, req *dto.UpdateCIRelationshipRequest) (*dto.CIRelationshipResponse, error) {
	update := s.client.CIRelationship.UpdateOneID(id).
		Where(cirelationship.TenantIDEQ(tenantID))

	if req.Strength != nil {
		update.SetStrength(cirelationship.Strength(*req.Strength))
	}
	if req.ImpactLevel != nil {
		update.SetImpactLevel(cirelationship.ImpactLevel(*req.ImpactLevel))
	}
	if req.Description != nil {
		update.SetDescription(*req.Description)
	}
	if req.Metadata != nil {
		update.SetMetadata(*req.Metadata)
	}
	if req.IsActive != nil {
		update.SetIsActive(*req.IsActive)
	}

	relation, err := update.Save(ctx)
	if err != nil {
		s.logger.Errorw("Failed to update CI relationship", "error", err, "relation_id", id)
		return nil, fmt.Errorf("failed to update CI relationship: %w", err)
	}

	// 重新加载关联信息
	relation, err = s.client.CIRelationship.Query().
		Where(cirelationship.IDEQ(relation.ID)).
		WithSourceCi().
		WithTargetCi().
		First(ctx)
	if err != nil {
		s.logger.Errorw("Failed to load updated relation", "error", err, "relation_id", relation.ID)
		return nil, fmt.Errorf("failed to load updated relation: %w", err)
	}

	s.logger.Infow("CI relationship updated successfully", "relation_id", relation.ID, "tenant_id", tenantID)
	return dto.ToCIRelationshipResponse(relation), nil
}

// DeleteCIRelationship 删除CI关系
func (s *CIRelationshipService) DeleteCIRelationship(ctx context.Context, id, tenantID int) error {
	err := s.client.CIRelationship.DeleteOneID(id).
		Where(cirelationship.TenantIDEQ(tenantID)).
		Exec(ctx)
	if err != nil {
		s.logger.Errorw("Failed to delete CI relationship", "error", err, "relation_id", id)
		return fmt.Errorf("failed to delete CI relationship: %w", err)
	}

	s.logger.Infow("CI relationship deleted successfully", "relation_id", id, "tenant_id", tenantID)
	return nil
}

// GetCIImpactAnalysis 获取CI影响分析
const maxCIImpactAnalysisDepth = 10

func topologyNodeFromCI(ci *ent.ConfigurationItem) dto.TopologyNode {
	return dto.TopologyNode{
		ID: ci.ID, Name: ci.Name, Type: ci.CiType, TypeName: ci.CiType,
		Status: ci.Status, Criticality: ci.Criticality, Attributes: ci.Attributes,
	}
}

func topologyEdgeFromRelationship(rel *ent.CIRelationship) dto.TopologyEdge {
	return dto.TopologyEdge{
		ID: rel.ID, Source: rel.SourceCiID, Target: rel.TargetCiID,
		RelationshipType: rel.RelationshipType, RelationshipLabel: rel.RelationshipType,
		Strength: string(rel.Strength), ImpactLevel: string(rel.ImpactLevel),
	}
}

// GetCITopology 返回统一的图结构，并对租户、深度和环路做强制约束。
func (s *CIRelationshipService) GetCITopology(ctx context.Context, ciID, tenantID, maxDepth int) (*dto.TopologyGraph, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("tenant ID is required")
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}
	if maxDepth > maxCIImpactAnalysisDepth {
		maxDepth = maxCIImpactAnalysisDepth
	}
	root, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(ciID), configurationitem.TenantIDEQ(tenantID)).Only(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get topology root CI: %w", err)
	}

	nodes := map[int]dto.TopologyNode{root.ID: topologyNodeFromCI(root)}
	edges := make(map[int]dto.TopologyEdge)
	visited := map[int]bool{root.ID: true}
	frontier := []int{root.ID}
	for depth := 0; depth < maxDepth && len(frontier) > 0; depth++ {
		relations, err := s.client.CIRelationship.Query().Where(
			cirelationship.TenantIDEQ(tenantID), cirelationship.IsActiveEQ(true),
			cirelationship.Or(cirelationship.SourceCiIDIn(frontier...), cirelationship.TargetCiIDIn(frontier...)),
		).All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query topology relationships: %w", err)
		}
		nextIDs := make([]int, 0)
		for _, rel := range relations {
			edges[rel.ID] = topologyEdgeFromRelationship(rel)
			for _, id := range []int{rel.SourceCiID, rel.TargetCiID} {
				if !visited[id] {
					visited[id] = true
					nextIDs = append(nextIDs, id)
				}
			}
		}
		if len(nextIDs) == 0 {
			break
		}
		cis, err := s.client.ConfigurationItem.Query().
			Where(configurationitem.IDIn(nextIDs...), configurationitem.TenantIDEQ(tenantID)).All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query topology CIs: %w", err)
		}
		frontier = frontier[:0]
		for _, ci := range cis {
			nodes[ci.ID] = topologyNodeFromCI(ci)
			frontier = append(frontier, ci.ID)
		}
	}

	graph := &dto.TopologyGraph{RootCIID: ciID, Depth: maxDepth}
	for _, node := range nodes {
		graph.Nodes = append(graph.Nodes, node)
	}
	for _, edge := range edges {
		if _, sourceOK := nodes[edge.Source]; sourceOK {
			if _, targetOK := nodes[edge.Target]; targetOK {
				graph.Edges = append(graph.Edges, edge)
			}
		}
	}
	graph.TotalNodes, graph.TotalEdges = len(graph.Nodes), len(graph.Edges)
	return graph, nil
}

func (s *CIRelationshipService) GetCIImpactAnalysis(ctx context.Context, ciID, tenantID int, maxDepth int) (*dto.CIImpactAnalysisResponse, error) {
	if tenantID <= 0 {
		return nil, fmt.Errorf("tenant ID is required")
	}
	if maxDepth <= 0 {
		maxDepth = 3
	}
	if maxDepth > maxCIImpactAnalysisDepth {
		maxDepth = maxCIImpactAnalysisDepth
	}
	// 检查CI是否存在
	root, err := s.client.ConfigurationItem.Query().
		Where(configurationitem.IDEQ(ciID), configurationitem.TenantIDEQ(tenantID)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("CI not found")
		}
		s.logger.Errorw("Failed to get CI for impact analysis", "error", err, "ci_id", ciID)
		return nil, fmt.Errorf("failed to get CI: %w", err)
	}

	// 使用按层广度优先搜索遍历影响链。每层批量读取关系和 CI，避免逐节点 N+1 查询。
	visited := map[int]bool{ciID: true}
	queue := []struct {
		ciID  int
		depth int
		path  []int
	}{{ciID: ciID, depth: 0, path: []int{ciID}}}

	impactedCIs := make([]dto.ImpactAnalysisItem, 0)
	impactEdges := make([]dto.TopologyEdge, 0)
	impactNodes := []dto.TopologyNode{topologyNodeFromCI(root)}

	for len(queue) > 0 {
		currentDepth := queue[0].depth
		frontier := make([]int, 0)
		paths := make(map[int][]int)
		for len(queue) > 0 && queue[0].depth == currentDepth {
			item := queue[0]
			queue = queue[1:]
			frontier = append(frontier, item.ciID)
			paths[item.ciID] = item.path
		}
		if currentDepth >= maxDepth {
			continue
		}

		relations, err := s.client.CIRelationship.Query().
			Where(
				cirelationship.SourceCiIDIn(frontier...),
				cirelationship.TenantIDEQ(tenantID),
				cirelationship.IsActiveEQ(true),
				cirelationship.RelationshipTypeIn("impacts", "depends_on", "uses"),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query impact relationships: %w", err)
		}

		nextPaths := make(map[int][]int)
		for _, rel := range relations {
			impactEdges = append(impactEdges, topologyEdgeFromRelationship(rel))
			if !visited[rel.TargetCiID] {
				visited[rel.TargetCiID] = true
				newPath := append([]int(nil), paths[rel.SourceCiID]...)
				newPath = append(newPath, rel.TargetCiID)
				nextPaths[rel.TargetCiID] = newPath
			}
		}

		nextIDs := make([]int, 0, len(nextPaths))
		for id := range nextPaths {
			nextIDs = append(nextIDs, id)
		}
		if len(nextIDs) == 0 {
			continue
		}
		cis, err := s.client.ConfigurationItem.Query().
			Where(configurationitem.IDIn(nextIDs...), configurationitem.TenantIDEQ(tenantID)).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query impacted CIs: %w", err)
		}
		for _, ci := range cis {
			path := nextPaths[ci.ID]
			impactNodes = append(impactNodes, topologyNodeFromCI(ci))
			impactedCIs = append(impactedCIs, dto.ImpactAnalysisItem{
				CIID: ci.ID, CIName: ci.Name, CIT: ci.CiType, Distance: currentDepth + 1,
				Direction: "downstream", Relationship: "impact path", ImpactLevel: impactLevelForCI(ci),
			})
			queue = append(queue, struct {
				ciID  int
				depth int
				path  []int
			}{ciID: ci.ID, depth: currentDepth + 1, path: path})
		}
	}

	graph := &dto.TopologyGraph{Nodes: impactNodes, Edges: impactEdges, RootCIID: ciID, Depth: maxDepth}
	graph.TotalNodes, graph.TotalEdges = len(graph.Nodes), len(graph.Edges)
	riskLevel := "low"
	if len(impactedCIs) >= 10 {
		riskLevel = "critical"
	} else if len(impactedCIs) >= 5 {
		riskLevel = "high"
	} else if len(impactedCIs) > 0 {
		riskLevel = "medium"
	}
	return &dto.CIImpactAnalysisResponse{
		SourceCIID: ciID, TargetCI: ptrTopologyNode(topologyNodeFromCI(root)), Graph: graph,
		UpstreamImpact: []dto.ImpactAnalysisItem{}, DownstreamImpact: impactedCIs,
		CriticalDependencies: []dto.ImpactAnalysisItem{}, AffectedTickets: []dto.AffectedTicket{},
		AffectedIncidents: []dto.AffectedIncident{}, RiskLevel: riskLevel,
		Summary: fmt.Sprintf("%d configuration items may be impacted", len(impactedCIs)), TotalImpacted: len(impactedCIs),
	}, nil
}

func ptrTopologyNode(node dto.TopologyNode) *dto.TopologyNode { return &node }

func impactLevelForCI(ci *ent.ConfigurationItem) dto.ImpactLevel {
	switch ci.Criticality {
	case "critical":
		return dto.ImpactCritical
	case "high":
		return dto.ImpactHigh
	case "medium":
		return dto.ImpactMedium
	default:
		return dto.ImpactLow
	}
}

// wouldCreateCycle 检测新增 source→target 边是否会导致 CI 关系图形成环。
// 算法：从 target 出发做有向 DFS，若能到达 source，即添加后回到 source 形成环。
// 只对同 tenant 内、is_active=true 的关系进行遍历。
// 关系类型语义：depends_on/contains/parent_of 等有明确层级方向；relates_to 等弱关系仍视为有向。
// 复杂度：O(V+E)，深度上限 512 兜底防脏数据死循环。
func (s *CIRelationshipService) wouldCreateCycle(ctx context.Context, tenantID, sourceID, targetID int, _ string) (bool, error) {
	const maxDepth = 512
	visited := make(map[int]bool)
	stack := []int{targetID}
	depth := 0
	for len(stack) > 0 {
		if depth++; depth > maxDepth {
			return false, fmt.Errorf("cycle detection exceeded max depth %d (possible corrupted graph)", maxDepth)
		}
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur == sourceID {
			return true, nil
		}
		if visited[cur] {
			continue
		}
		visited[cur] = true

		// 出边：cur → next
		nexts, err := s.client.CIRelationship.Query().
			Where(
				cirelationship.SourceCiIDEQ(cur),
				cirelationship.TenantIDEQ(tenantID),
				cirelationship.IsActiveEQ(true),
			).
			Select(cirelationship.FieldTargetCiID).
			Ints(ctx)
		if err != nil {
			return false, fmt.Errorf("query outgoing edges of ci=%d: %w", cur, err)
		}
		stack = append(stack, nexts...)
	}
	return false, nil
}
