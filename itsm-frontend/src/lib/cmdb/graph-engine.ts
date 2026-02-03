/**
 * CMDB关系图谱引擎
 * 提供图谱计算、布局和分析功能
 */

import type {
  ConfigurationItem,
  CIRelationship,
  RelationshipGraph,
  GraphNode,
  GraphEdge,
  CIType,
} from '@/types/cmdb';
import { RelationType } from '@/types/cmdb';

export class GraphEngine {
  /**
   * 构建关系图谱
   */
  static buildGraph(
    cis: ConfigurationItem[],
    relationships: CIRelationship[]
  ): RelationshipGraph {
    // 创建节点
    const nodes: GraphNode[] = cis.map((ci) => ({
      id: ci.id,
      ciId: ci.id,
      name: ci.name,
      type: ci.type,
      status: ci.status,
      data: ci,
      inDegree: 0,
      outDegree: 0,
    }));

    // 创建边并计算度数
    const edges: GraphEdge[] = relationships.map((rel) => {
      // 更新节点度数
      const sourceNode = nodes.find((n) => n.id === rel.sourceCI);
      const targetNode = nodes.find((n) => n.id === rel.targetCI);
      if (sourceNode) sourceNode.outDegree = (sourceNode.outDegree || 0) + 1;
      if (targetNode) targetNode.inDegree = (targetNode.inDegree || 0) + 1;

      return {
        id: rel.id,
        relationshipId: rel.id,
        source: rel.sourceCI,
        target: rel.targetCI,
        type: rel.type,
        label: this.getRelationshipLabel(rel.type),
        data: rel,
      };
    });

    // 计算统计信息
    const stats = this.calculateStats(nodes, edges);

    return {
      nodes,
      edges,
      layout: 'force',
      stats,
    };
  }

  /**
   * 应用力导向布局
   */
  static applyForceLayout(
    graph: RelationshipGraph,
    options?: {
      width?: number;
      height?: number;
      iterations?: number;
      nodeRepulsion?: number;
      linkDistance?: number;
    }
  ): RelationshipGraph {
    const {
      width = 1200,
      height = 800,
      iterations = 100,
      nodeRepulsion = 100,
      linkDistance = 100,
    } = options || {};

    // 初始化节点位置
    graph.nodes.forEach((node, i) => {
      node.x = Math.random() * width;
      node.y = Math.random() * height;
    });

    // 力导向布局算法
    for (let iter = 0; iter < iterations; iter++) {
      // 计算斥力
      for (let i = 0; i < graph.nodes.length; i++) {
        for (let j = i + 1; j < graph.nodes.length; j++) {
          const node1 = graph.nodes[i];
          const node2 = graph.nodes[j];

          const dx = node2.x! - node1.x!;
          const dy = node2.y! - node1.y!;
          const distance = Math.sqrt(dx * dx + dy * dy) || 1;

          const force = nodeRepulsion / (distance * distance);
          const fx = (dx / distance) * force;
          const fy = (dy / distance) * force;

          node1.x! -= fx;
          node1.y! -= fy;
          node2.x! += fx;
          node2.y! += fy;
        }
      }

      // 计算引力
      graph.edges.forEach((edge) => {
        const source = graph.nodes.find((n) => n.id === edge.source);
        const target = graph.nodes.find((n) => n.id === edge.target);

        if (source && target) {
          const dx = target.x! - source.x!;
          const dy = target.y! - source.y!;
          const distance = Math.sqrt(dx * dx + dy * dy) || 1;

          const force = (distance - linkDistance) / distance;
          const fx = dx * force * 0.5;
          const fy = dy * force * 0.5;

          source.x! += fx;
          source.y! += fy;
          target.x! -= fx;
          target.y! -= fy;
        }
      });

      // 限制在边界内
      graph.nodes.forEach((node) => {
        node.x = Math.max(50, Math.min(width - 50, node.x!));
        node.y = Math.max(50, Math.min(height - 50, node.y!));
      });
    }

    return graph;
  }

  /**
   * 应用层次布局
   */
  static applyHierarchicalLayout(
    graph: RelationshipGraph,
    options?: {
      width?: number;
      height?: number;
      levelSpacing?: number;
      nodeSpacing?: number;
    }
  ): RelationshipGraph {
    const {
      width = 1200,
      height = 800,
      levelSpacing = 150,
      nodeSpacing = 100,
    } = options || {};

    // 拓扑排序，确定层级
    const levels = this.topologicalSort(graph);

    // 为每层分配位置
    levels.forEach((level, levelIndex) => {
      const y = levelSpacing * levelIndex;
      const totalWidth = (level.length - 1) * nodeSpacing;
      const startX = (width - totalWidth) / 2;

      level.forEach((nodeId, index) => {
        const node = graph.nodes.find((n) => n.id === nodeId);
        if (node) {
          node.x = startX + index * nodeSpacing;
          node.y = y;
        }
      });
    });

    return graph;
  }

  /**
   * 拓扑排序
   */
  private static topologicalSort(graph: RelationshipGraph): string[][] {
    const levels: string[][] = [];
    const visited = new Set<string>();
    const inDegree = new Map<string, number>();

    // 计算入度
    graph.nodes.forEach((node) => {
      inDegree.set(node.id, 0);
    });
    graph.edges.forEach((edge) => {
      const current = inDegree.get(edge.target) || 0;
      inDegree.set(edge.target, current + 1);
    });

    // 找出所有入度为0的节点作为第一层
    while (visited.size < graph.nodes.length) {
      const currentLevel: string[] = [];

      graph.nodes.forEach((node) => {
        if (!visited.has(node.id) && (inDegree.get(node.id) || 0) === 0) {
          currentLevel.push(node.id);
          visited.add(node.id);
        }
      });

      if (currentLevel.length === 0) {
        // 如果有环，把剩余节点放在一层
        graph.nodes.forEach((node) => {
          if (!visited.has(node.id)) {
            currentLevel.push(node.id);
            visited.add(node.id);
          }
        });
      }

      levels.push(currentLevel);

      // 减少相邻节点的入度
      currentLevel.forEach((nodeId) => {
        graph.edges
          .filter((e) => e.source === nodeId)
          .forEach((edge) => {
            const current = inDegree.get(edge.target) || 0;
            inDegree.set(edge.target, Math.max(0, current - 1));
          });
      });
    }

    return levels;
  }

  /**
   * 查找最短路径（BFS）
   */
  static findShortestPath(
    graph: RelationshipGraph,
    sourceId: string,
    targetId: string
  ): string[] | null {
    const queue: Array<{ node: string; path: string[] }> = [
      { node: sourceId, path: [sourceId] },
    ];
    const visited = new Set<string>([sourceId]);

    while (queue.length > 0) {
      const { node: current, path } = queue.shift()!;

      if (current === targetId) {
        return path;
      }

      // 找到所有相邻节点
      graph.edges
        .filter((e) => e.source === current)
        .forEach((edge) => {
          if (!visited.has(edge.target)) {
            visited.add(edge.target);
            queue.push({
              node: edge.target,
              path: [...path, edge.target],
            });
          }
        });
    }

    return null;
  }

  /**
   * 查找所有路径（DFS）
   */
  static findAllPaths(
    graph: RelationshipGraph,
    sourceId: string,
    targetId: string,
    maxDepth = 10
  ): string[][] {
    const paths: string[][] = [];

    const dfs = (current: string, path: string[], depth: number) => {
      if (depth > maxDepth) return;
      if (current === targetId) {
        paths.push([...path]);
        return;
      }

      graph.edges
        .filter((e) => e.source === current)
        .forEach((edge) => {
          if (!path.includes(edge.target)) {
            dfs(edge.target, [...path, edge.target], depth + 1);
          }
        });
    };

    dfs(sourceId, [sourceId], 0);
    return paths;
  }

  /**
   * 检测环
   */
  static detectCycles(graph: RelationshipGraph): string[][] {
    const cycles: string[][] = [];
    const visited = new Set<string>();
    const recursionStack = new Set<string>();

    const dfs = (nodeId: string, path: string[]): void => {
      visited.add(nodeId);
      recursionStack.add(nodeId);

      graph.edges
        .filter((e) => e.source === nodeId)
        .forEach((edge) => {
          if (!visited.has(edge.target)) {
            dfs(edge.target, [...path, edge.target]);
          } else if (recursionStack.has(edge.target)) {
            // 找到环
            const cycleStart = path.indexOf(edge.target);
            if (cycleStart !== -1) {
              cycles.push(path.slice(cycleStart));
            }
          }
        });

      recursionStack.delete(nodeId);
    };

    graph.nodes.forEach((node) => {
      if (!visited.has(node.id)) {
        dfs(node.id, [node.id]);
      }
    });

    return cycles;
  }

  /**
   * 计算中心性（度中心性）
   */
  static calculateCentrality(graph: RelationshipGraph): Map<string, number> {
    const centrality = new Map<string, number>();

    graph.nodes.forEach((node) => {
      const degree = (node.inDegree || 0) + (node.outDegree || 0);
      centrality.set(node.id, degree);
    });

    return centrality;
  }

  /**
   * 查找关键节点
   */
  static findCriticalNodes(
    graph: RelationshipGraph,
    threshold = 5
  ): GraphNode[] {
    const centrality = this.calculateCentrality(graph);
    return graph.nodes.filter(
      (node) => (centrality.get(node.id) || 0) >= threshold
    );
  }

  /**
   * 计算图谱统计
   */
  private static calculateStats(
    nodes: GraphNode[],
    edges: GraphEdge[]
  ): RelationshipGraph['stats'] {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const nodesByType: Record<CIType, number> = {} as any;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const edgesByType: Record<RelationType, number> = {} as any;

    nodes.forEach((node) => {
      nodesByType[node.type] = (nodesByType[node.type] || 0) + 1;
    });

    edges.forEach((edge) => {
      edgesByType[edge.type] = (edgesByType[edge.type] || 0) + 1;
    });

    return {
      totalNodes: nodes.length,
      totalEdges: edges.length,
      nodesByType,
      edgesByType,
    };
  }

  /**
   * 获取关系类型标签
   */
  private static getRelationshipLabel(type: RelationType): string {
    const labels: Record<RelationType, string> = {
      [RelationType.CONNECTED_TO]: '连接到',
      [RelationType.INSTALLED_ON]: '安装在',
      [RelationType.HOSTED_ON]: '托管在',
      [RelationType.RUNS_ON]: '运行在',
      [RelationType.CONTAINS]: '包含',
      [RelationType.DEPENDS_ON]: '依赖于',
      [RelationType.PROVIDES_TO]: '提供给',
      [RelationType.USES]: '使用',
      [RelationType.MANAGES]: '管理',
      [RelationType.SUPPORTS]: '支持',
      [RelationType.OWNED_BY]: '归属于',
      [RelationType.LOCATED_IN]: '位于',
      [RelationType.MEMBER_OF]: '成员',
      [RelationType.CUSTOM]: '自定义',
    };
    return labels[type] || type;
  }

  /**
   * 过滤图谱
   */
  static filterGraph(
    graph: RelationshipGraph,
    filters: {
      nodeTypes?: CIType[];
      relationshipTypes?: RelationType[];
      minConnections?: number;
    }
  ): RelationshipGraph {
    let filteredNodes = graph.nodes;
    let filteredEdges = graph.edges;

    // 按节点类型过滤
    if (filters.nodeTypes && filters.nodeTypes.length > 0) {
      filteredNodes = filteredNodes.filter((node) =>
        filters.nodeTypes!.includes(node.type)
      );
      const nodeIds = new Set(filteredNodes.map((n) => n.id));
      filteredEdges = filteredEdges.filter(
        (edge) => nodeIds.has(edge.source) && nodeIds.has(edge.target)
      );
    }

    // 按关系类型过滤
    if (filters.relationshipTypes && filters.relationshipTypes.length > 0) {
      filteredEdges = filteredEdges.filter((edge) =>
        filters.relationshipTypes!.includes(edge.type)
      );
    }

    // 按连接数过滤
    if (filters.minConnections) {
      const connectionCount = new Map<string, number>();
      filteredEdges.forEach((edge) => {
        connectionCount.set(
          edge.source,
          (connectionCount.get(edge.source) || 0) + 1
        );
        connectionCount.set(
          edge.target,
          (connectionCount.get(edge.target) || 0) + 1
        );
      });

      filteredNodes = filteredNodes.filter(
        (node) =>
          (connectionCount.get(node.id) || 0) >= filters.minConnections!
      );
      const nodeIds = new Set(filteredNodes.map((n) => n.id));
      filteredEdges = filteredEdges.filter(
        (edge) => nodeIds.has(edge.source) && nodeIds.has(edge.target)
      );
    }

    return {
      nodes: filteredNodes,
      edges: filteredEdges,
      layout: graph.layout,
      stats: this.calculateStats(filteredNodes, filteredEdges),
    };
  }
}

export default GraphEngine;
