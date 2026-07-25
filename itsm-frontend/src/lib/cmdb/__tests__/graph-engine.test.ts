/**
 * Tests for CMDB Graph Engine
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

// Mock RelationType enum
jest.mock('@/types/cmdb', () => ({
  RelationType: {
    CONNECTED_TO: 'connected_to',
    INSTALLED_ON: 'installed_on',
    HOSTED_ON: 'hosted_on',
    RUNS_ON: 'runs_on',
    CONTAINS: 'contains',
    DEPENDS_ON: 'depends_on',
    PROVIDES_TO: 'provides_to',
    USES: 'uses',
    MANAGES: 'manages',
    SUPPORTS: 'supports',
    OWNED_BY: 'owned_by',
    LOCATED_IN: 'located_in',
    MEMBER_OF: 'member_of',
    CUSTOM: 'custom',
  },
}));

import { GraphEngine } from '../graph-engine';

const mockCIs = [
  { id: 'ci1', name: 'Server A', type: 'server', status: 'active' },
  { id: 'ci2', name: 'App B', type: 'application', status: 'active' },
  { id: 'ci3', name: 'DB C', type: 'database', status: 'active' },
  { id: 'ci4', name: 'Orphan D', type: 'server', status: 'inactive' },
] as any[];

const mockRelationships = [
  { id: 'rel1', sourceCI: 'ci1', targetCI: 'ci2', type: 'hosted_on' },
  { id: 'rel2', sourceCI: 'ci2', targetCI: 'ci3', type: 'depends_on' },
] as any[];

describe('GraphEngine', () => {
  describe('buildGraph', () => {
    it('should build graph with correct nodes and edges', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);

      expect(graph.nodes).toHaveLength(4);
      expect(graph.edges).toHaveLength(2);
      expect(graph.layout).toBe('force');
    });

    it('should calculate node degrees correctly', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);

      const ci1Node = graph.nodes.find(n => n.id === 'ci1');
      const ci2Node = graph.nodes.find(n => n.id === 'ci2');
      const ci3Node = graph.nodes.find(n => n.id === 'ci3');

      expect(ci1Node?.outDegree).toBe(1);
      expect(ci1Node?.inDegree).toBe(0);
      expect(ci2Node?.outDegree).toBe(1);
      expect(ci2Node?.inDegree).toBe(1);
      expect(ci3Node?.inDegree).toBe(1);
    });

    it('should calculate stats correctly', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);

      expect(graph.stats.totalNodes).toBe(4);
      expect(graph.stats.totalEdges).toBe(2);
      expect(graph.stats.nodesByType['server']).toBe(2);
      expect(graph.stats.nodesByType['application']).toBe(1);
      expect(graph.stats.edgesByType['hosted_on']).toBe(1);
      expect(graph.stats.edgesByType['depends_on']).toBe(1);
    });

    it('should handle empty inputs', () => {
      const graph = GraphEngine.buildGraph([], []);
      expect(graph.nodes).toHaveLength(0);
      expect(graph.edges).toHaveLength(0);
      expect(graph.stats.totalNodes).toBe(0);
    });
  });

  describe('applyForceLayout', () => {
    it('should assign positions to nodes', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const result = GraphEngine.applyForceLayout(graph, { iterations: 5 });

      result.nodes.forEach(node => {
        expect(node.x).toBeDefined();
        expect(node.y).toBeDefined();
        expect(node.x).toBeGreaterThanOrEqual(50);
        expect(node.y).toBeGreaterThanOrEqual(50);
      });
    });

    it('should respect boundary limits', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const width = 800;
      const height = 600;
      const result = GraphEngine.applyForceLayout(graph, { width, height, iterations: 10 });

      result.nodes.forEach(node => {
        expect(node.x).toBeLessThanOrEqual(width - 50);
        expect(node.y).toBeLessThanOrEqual(height - 50);
      });
    });

    it('should use default options when none provided', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const result = GraphEngine.applyForceLayout(graph);
      expect(result.nodes.length).toBe(4);
    });
  });

  describe('applyHierarchicalLayout', () => {
    it('should assign positions based on topology', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const result = GraphEngine.applyHierarchicalLayout(graph);

      const ci1Node = result.nodes.find(n => n.id === 'ci1');
      expect(ci1Node?.x).toBeDefined();
      expect(ci1Node?.y).toBeDefined();
    });

    it('should use default options', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const result = GraphEngine.applyHierarchicalLayout(graph);
      expect(result.nodes.length).toBe(4);
    });
  });

  describe('findShortestPath', () => {
    it('should find path between connected nodes', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const path = GraphEngine.findShortestPath(graph, 'ci1', 'ci3');

      expect(path).toEqual(['ci1', 'ci2', 'ci3']);
    });

    it('should return null when no path exists', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const path = GraphEngine.findShortestPath(graph, 'ci3', 'ci1');

      expect(path).toBeNull();
    });

    it('should return source node when source equals target', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const path = GraphEngine.findShortestPath(graph, 'ci1', 'ci1');

      expect(path).toEqual(['ci1']);
    });
  });

  describe('findAllPaths', () => {
    it('should find all paths between nodes', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const paths = GraphEngine.findAllPaths(graph, 'ci1', 'ci3');

      expect(paths).toHaveLength(1);
      expect(paths[0]).toEqual(['ci1', 'ci2', 'ci3']);
    });

    it('should return empty when no paths exist', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const paths = GraphEngine.findAllPaths(graph, 'ci4', 'ci1');

      expect(paths).toHaveLength(0);
    });

    it('should respect maxDepth', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const paths = GraphEngine.findAllPaths(graph, 'ci1', 'ci3', 1);

      expect(paths).toHaveLength(0);
    });
  });

  describe('detectCycles', () => {
    it('should detect no cycles in acyclic graph', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const cycles = GraphEngine.detectCycles(graph);

      expect(cycles).toHaveLength(0);
    });

    it('should detect cycles in cyclic graph', () => {
      const cyclicRels = [
        { id: 'rel1', sourceCI: 'ci1', targetCI: 'ci2', type: 'depends_on' },
        { id: 'rel2', sourceCI: 'ci2', targetCI: 'ci3', type: 'depends_on' },
        { id: 'rel3', sourceCI: 'ci3', targetCI: 'ci1', type: 'depends_on' },
      ] as any[];
      const graph = GraphEngine.buildGraph(mockCIs.slice(0, 3), cyclicRels);
      const cycles = GraphEngine.detectCycles(graph);

      expect(cycles.length).toBeGreaterThan(0);
    });
  });

  describe('calculateCentrality', () => {
    it('should calculate degree centrality', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const centrality = GraphEngine.calculateCentrality(graph);

      expect(centrality.get('ci2')).toBe(2); // 1 in + 1 out
      expect(centrality.get('ci4')).toBe(0); // no connections
    });
  });

  describe('findCriticalNodes', () => {
    it('should find nodes above threshold', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const critical = GraphEngine.findCriticalNodes(graph, 2);

      expect(critical.some(n => n.id === 'ci2')).toBe(true);
    });

    it('should return empty when no nodes meet threshold', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const critical = GraphEngine.findCriticalNodes(graph, 10);

      expect(critical).toHaveLength(0);
    });
  });

  describe('filterGraph', () => {
    it('should filter by node types', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const filtered = GraphEngine.filterGraph(graph, { nodeTypes: ['server' as any] });

      expect(filtered.nodes.every(n => n.type === 'server')).toBe(true);
      expect(filtered.edges).toHaveLength(0); // no edges between two servers
    });

    it('should filter by relationship types', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const filtered = GraphEngine.filterGraph(graph, { relationshipTypes: ['hosted_on' as any] });

      expect(filtered.edges).toHaveLength(1);
      expect(filtered.edges[0].type).toBe('hosted_on');
    });

    it('should filter by minConnections', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const filtered = GraphEngine.filterGraph(graph, { minConnections: 2 });

      // ci2 has 2 connections (1 as source, 1 as target)
      expect(filtered.nodes.some(n => n.id === 'ci2')).toBe(true);
    });

    it('should return all when no filters provided', () => {
      const graph = GraphEngine.buildGraph(mockCIs, mockRelationships);
      const filtered = GraphEngine.filterGraph(graph, {});

      expect(filtered.nodes).toHaveLength(4);
      expect(filtered.edges).toHaveLength(2);
    });
  });
});
