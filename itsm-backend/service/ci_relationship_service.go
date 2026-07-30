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

	// 下游（root 影响谁）与上游（谁影响 root）均用按层 BFS，每层批量读取关系和 CI，避免逐节点 N+1 查询。
	downstream, downNodes, downEdges, downCritical, err := s.traverseImpactLayers(ctx, tenantID, ciID, maxDepth, "downstream")
	if err != nil {
		return nil, err
	}
	upstream, upNodes, upEdges, upCritical, err := s.traverseImpactLayers(ctx, tenantID, ciID, maxDepth, "upstream")
	if err != nil {
		return nil, err
	}

	// 合并双向图（按 ID 去重）
	impactNodes := []dto.TopologyNode{topologyNodeFromCI(root)}
	seenNodes := map[int]bool{impactNodes[0].ID: true}
	for _, node := range append(downNodes, upNodes...) {
		if !seenNodes[node.ID] {
			seenNodes[node.ID] = true
			impactNodes = append(impactNodes, node)
		}
	}
	impactEdges := make([]dto.TopologyEdge, 0, len(downEdges)+len(upEdges))
	seenEdges := map[int]bool{}
	for _, edge := range append(downEdges, upEdges...) {
		if !seenEdges[edge.ID] {
			seenEdges[edge.ID] = true
			impactEdges = append(impactEdges, edge)
		}
	}
	criticalDeps := append(downCritical, upCritical...)

	// 受影响的工单/事件：通过 CI 已有的 tickets/incidents 边填充
	affectedTickets := make([]dto.AffectedTicket, 0)
	tickets, err := root.QueryTickets().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query affected tickets: %w", err)
	}
	for _, tk := range tickets {
		affectedTickets = append(affectedTickets, dto.AffectedTicket{
			ID: tk.ID, Title: tk.Title, Status: tk.Status, Priority: tk.Priority,
			CreatedAt: tk.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	affectedIncidents := make([]dto.AffectedIncident, 0)
	incidents, err := root.QueryIncidents().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query affected incidents: %w", err)
	}
	for _, in := range incidents {
		affectedIncidents = append(affectedIncidents, dto.AffectedIncident{
			ID: in.ID, Title: in.Title, Status: in.Status, Severity: in.Severity,
			CreatedAt: in.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	graph := &dto.TopologyGraph{Nodes: impactNodes, Edges: impactEdges, RootCIID: ciID, Depth: maxDepth}
	graph.TotalNodes, graph.TotalEdges = len(graph.Nodes), len(graph.Edges)
	riskLevel := "low"
	if len(downstream) >= 10 {
		riskLevel = "critical"
	} else if len(downstream) >= 5 {
		riskLevel = "high"
	} else if len(downstream) > 0 {
		riskLevel = "medium"
	}
	return &dto.CIImpactAnalysisResponse{
		SourceCIID: ciID, TargetCI: ptrTopologyNode(topologyNodeFromCI(root)), Graph: graph,
		UpstreamImpact: upstream, DownstreamImpact: downstream,
		CriticalDependencies: criticalDeps, AffectedTickets: affectedTickets,
		AffectedIncidents: affectedIncidents, RiskLevel: riskLevel,
		Summary:       fmt.Sprintf("%d configuration items may be impacted, %d upstream dependencies", len(downstream), len(upstream)),
		TotalImpacted: len(downstream),
	}, nil
}

// traverseImpactLayers 按层 BFS 遍历影响链。
// direction=downstream：沿出边 source→target 前进（root 影响的 CI）；
// direction=upstream：沿入边反向前进（影响 root 的 CI）。
// 返回影响项、节点、边，以及经由 strength=critical 边到达的关键依赖项。
func (s *CIRelationshipService) traverseImpactLayers(ctx context.Context, tenantID, rootID, maxDepth int, direction string) (
	[]dto.ImpactAnalysisItem, []dto.TopologyNode, []dto.TopologyEdge, []dto.ImpactAnalysisItem, error,
) {
	visited := map[int]bool{rootID: true}
	type queueItem struct {
		ciID  int
		depth int
	}
	queue := []queueItem{{ciID: rootID, depth: 0}}

	items := make([]dto.ImpactAnalysisItem, 0)
	nodes := make([]dto.TopologyNode, 0)
	edges := make([]dto.TopologyEdge, 0)
	criticalItems := make([]dto.ImpactAnalysisItem, 0)

	for len(queue) > 0 {
		currentDepth := queue[0].depth
		frontier := make([]int, 0)
		for len(queue) > 0 && queue[0].depth == currentDepth {
			frontier = append(frontier, queue[0].ciID)
			queue = queue[1:]
		}
		if currentDepth >= maxDepth {
			continue
		}

		query := s.client.CIRelationship.Query().
			Where(
				cirelationship.TenantIDEQ(tenantID),
				cirelationship.IsActiveEQ(true),
				cirelationship.RelationshipTypeIn("impacts", "depends_on", "uses"),
			)
		if direction == "upstream" {
			query = query.Where(cirelationship.TargetCiIDIn(frontier...))
		} else {
			query = query.Where(cirelationship.SourceCiIDIn(frontier...))
		}
		relations, err := query.All(ctx)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to query %s impact relationships: %w", direction, err)
		}

		nextIDs := make([]int, 0)
		nextRelType := make(map[int]string)
		nextCritical := make(map[int]bool)
		for _, rel := range relations {
			edges = append(edges, topologyEdgeFromRelationship(rel))
			nextID := rel.TargetCiID
			if direction == "upstream" {
				nextID = rel.SourceCiID
			}
			if !visited[nextID] {
				visited[nextID] = true
				nextIDs = append(nextIDs, nextID)
				nextRelType[nextID] = rel.RelationshipType
				nextCritical[nextID] = rel.Strength == cirelationship.StrengthCritical
			}
		}
		if len(nextIDs) == 0 {
			continue
		}
		cis, err := s.client.ConfigurationItem.Query().
			Where(configurationitem.IDIn(nextIDs...), configurationitem.TenantIDEQ(tenantID)).
			All(ctx)
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to query %s impacted CIs: %w", direction, err)
		}
		for _, ci := range cis {
			nodes = append(nodes, topologyNodeFromCI(ci))
			item := dto.ImpactAnalysisItem{
				CIID: ci.ID, CIName: ci.Name, CIT: ci.CiType, Distance: currentDepth + 1,
				Direction: direction, Relationship: "impact path",
				RelationshipType: dto.CIRelationshipType(nextRelType[ci.ID]),
				ImpactLevel:      impactLevelForCI(ci),
			}
			items = append(items, item)
			if nextCritical[ci.ID] {
				criticalItems = append(criticalItems, item)
			}
			queue = append(queue, queueItem{ciID: ci.ID, depth: currentDepth + 1})
		}
	}
	return items, nodes, edges, criticalItems, nil
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
// 算法：一次性批量加载租户内所有 is_active=true 的边构建邻接表，
// 再从 target 出发内存 DFS，若能到达 source 即添加后形成环（消除逐节点 N+1 查询）。
// 关系类型语义：depends_on/contains/parent_of 等有明确层级方向；relates_to 等弱关系仍视为有向。
// 复杂度：1 次查询 + O(V+E) 内存遍历；visited 集合保证脏数据下也不会死循环。
func (s *CIRelationshipService) wouldCreateCycle(ctx context.Context, tenantID, sourceID, targetID int, _ string) (bool, error) {
	rels, err := s.client.CIRelationship.Query().
		Where(
			cirelationship.TenantIDEQ(tenantID),
			cirelationship.IsActiveEQ(true),
		).
		Select(cirelationship.FieldSourceCiID, cirelationship.FieldTargetCiID).
		All(ctx)
	if err != nil {
		return false, fmt.Errorf("load active relationship edges for cycle detection: %w", err)
	}
	adjacency := make(map[int][]int, len(rels))
	for _, rel := range rels {
		adjacency[rel.SourceCiID] = append(adjacency[rel.SourceCiID], rel.TargetCiID)
	}

	visited := make(map[int]bool)
	stack := []int{targetID}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cur == sourceID {
			return true, nil
		}
		if visited[cur] {
			continue
		}
		visited[cur] = true
		stack = append(stack, adjacency[cur]...)
	}
	return false, nil
}
