/**
 * Tests for Workflow Engine
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import { WorkflowEngine } from '../workflow-engine';

describe('WorkflowEngine', () => {
  describe('validate', () => {
    it('should return error for empty nodes', () => {
      const result = WorkflowEngine.validate({ nodes: [] });
      expect(result.isValid).toBe(false);
      expect(result.errors.length).toBeGreaterThan(0);
    });

    it('should return error for undefined nodes', () => {
      const result = WorkflowEngine.validate({});
      expect(result.isValid).toBe(false);
    });

    it('should require a start node', () => {
      const result = WorkflowEngine.validate({
        nodes: [{ id: '1', name: 'Task', type: 'task', config: { assigneeType: 'user' } }],
      } as any);
      expect(result.errors.some(e => e.message.includes('开始节点'))).toBe(true);
    });

    it('should error on multiple start nodes', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start 1', type: 'start', config: {} },
          { id: '2', name: 'Start 2', type: 'start', config: {} },
        ],
      } as any);
      expect(result.errors.some(e => e.message.includes('只能有一个开始节点'))).toBe(true);
    });

    it('should warn when no end node', () => {
      const result = WorkflowEngine.validate({
        nodes: [{ id: '1', name: 'Start', type: 'start', config: {} }],
      } as any);
      expect(result.warnings.some(e => e.message.includes('结束节点'))).toBe(true);
    });

    it('should validate task node requires assigneeType', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'Task', type: 'task', config: {} },
          { id: '3', name: 'End', type: 'end', config: {} },
        ],
        connections: [
          { id: 'c1', sourceNodeId: '1', targetNodeId: '2' },
          { id: 'c2', sourceNodeId: '2', targetNodeId: '3' },
        ],
      } as any);
      expect(result.errors.some(e => e.field === 'assigneeType')).toBe(true);
    });

    it('should validate approval node requires approvers', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'Approval', type: 'approval', config: {} },
        ],
      } as any);
      expect(result.errors.some(e => e.field === 'approvers')).toBe(true);
    });

    it('should validate condition node requires conditions', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'Condition', type: 'condition', config: {} },
        ],
      } as any);
      expect(result.errors.some(e => e.field === 'conditions')).toBe(true);
    });

    it('should validate script node requires script content', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'Script', type: 'script', config: {} },
        ],
      } as any);
      expect(result.errors.some(e => e.field === 'script')).toBe(true);
    });

    it('should validate notification node requires recipients', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'Notify', type: 'notification', config: {} },
        ],
      } as any);
      expect(result.errors.some(e => e.field === 'recipients')).toBe(true);
    });

    it('should validate node name cannot be empty', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: '', type: 'start', config: {} },
        ],
      } as any);
      expect(result.errors.some(e => e.field === 'name')).toBe(true);
    });

    it('should validate connections - source must exist', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'End', type: 'end', config: {} },
        ],
        connections: [{ id: 'c1', sourceNodeId: 'nonexist', targetNodeId: '2' }],
      } as any);
      expect(result.errors.some(e => e.message.includes('源节点'))).toBe(true);
    });

    it('should validate connections - target must exist', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'End', type: 'end', config: {} },
        ],
        connections: [{ id: 'c1', sourceNodeId: '1', targetNodeId: 'nonexist' }],
      } as any);
      expect(result.errors.some(e => e.message.includes('目标节点'))).toBe(true);
    });

    it('should error if start node has input connection', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'Task', type: 'task', config: { assigneeType: 'user' } },
        ],
        connections: [{ id: 'c1', sourceNodeId: '2', targetNodeId: '1' }],
      } as any);
      expect(result.errors.some(e => e.message.includes('开始节点不能有输入连接'))).toBe(true);
    });

    it('should error if end node has output connection', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'End', type: 'end', config: {} },
        ],
        connections: [{ id: 'c1', sourceNodeId: '2', targetNodeId: '1' }],
      } as any);
      expect(result.errors.some(e => e.message.includes('结束节点不能有输出连接'))).toBe(true);
    });

    it('should warn about orphaned nodes', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'Task', type: 'task', config: { assigneeType: 'user' } },
          { id: '3', name: 'End', type: 'end', config: {} },
        ],
        connections: [{ id: 'c1', sourceNodeId: '1', targetNodeId: '3' }],
      } as any);
      expect(result.warnings.some(e => e.message.includes('孤立'))).toBe(true);
    });

    it('should warn about cycles', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'Task A', type: 'task', config: { assigneeType: 'user' } },
          { id: '3', name: 'Task B', type: 'task', config: { assigneeType: 'user' } },
        ],
        connections: [
          { id: 'c1', sourceNodeId: '1', targetNodeId: '2' },
          { id: 'c2', sourceNodeId: '2', targetNodeId: '3' },
          { id: 'c3', sourceNodeId: '3', targetNodeId: '2' },
        ],
      } as any);
      expect(result.warnings.some(e => e.message.includes('循环'))).toBe(true);
    });

    it('should pass valid workflow', () => {
      const result = WorkflowEngine.validate({
        nodes: [
          { id: '1', name: 'Start', type: 'start', config: {} },
          { id: '2', name: 'Task', type: 'task', config: { assigneeType: 'user' } },
          { id: '3', name: 'End', type: 'end', config: {} },
        ],
        connections: [
          { id: 'c1', sourceNodeId: '1', targetNodeId: '2' },
          { id: 'c2', sourceNodeId: '2', targetNodeId: '3' },
        ],
      } as any);
      expect(result.isValid).toBe(true);
      expect(result.errors).toHaveLength(0);
    });
  });

  describe('findNextNodes', () => {
    const workflow = {
      nodes: [
        { id: '1', name: 'Start', type: 'start', config: {} },
        { id: '2', name: 'Task', type: 'task', config: { assigneeType: 'user' } },
        { id: '3', name: 'End', type: 'end', config: {} },
      ],
      connections: [
        { id: 'c1', sourceNodeId: '1', targetNodeId: '2' },
        { id: 'c2', sourceNodeId: '2', targetNodeId: '3' },
      ],
    } as any;

    it('should find next nodes from current', () => {
      const result = WorkflowEngine.findNextNodes('1', workflow, {});
      expect(result).toEqual(['2']);
    });

    it('should return empty when no connections', () => {
      const result = WorkflowEngine.findNextNodes('3', workflow, {});
      expect(result).toEqual([]);
    });

    it('should evaluate conditions for condition nodes', () => {
      const condWorkflow = {
        nodes: [
          { id: '1', name: 'Cond', type: 'condition', config: { conditions: [{ expression: 'x > 5', targetNodeId: 'target1' }], defaultTargetNodeId: 'default1' } },
        ],
        connections: [
          { id: 'c1', sourceNodeId: '1', targetNodeId: 'target1' },
          { id: 'c2', sourceNodeId: '1', targetNodeId: 'default1' },
        ],
      } as any;

      const result = WorkflowEngine.findNextNodes('1', condWorkflow, { x: 10 });
      expect(result).toEqual(['target1']);
    });

    it('should return default target when no conditions match', () => {
      const condWorkflow = {
        nodes: [
          { id: '1', name: 'Cond', type: 'condition', config: { conditions: [{ expression: 'x > 100', targetNodeId: 'target1' }], defaultTargetNodeId: 'default1' } },
        ],
        connections: [
          { id: 'c1', sourceNodeId: '1', targetNodeId: 'target1' },
          { id: 'c2', sourceNodeId: '1', targetNodeId: 'default1' },
        ],
      } as any;

      const result = WorkflowEngine.findNextNodes('1', condWorkflow, { x: 1 });
      expect(result).toEqual(['default1']);
    });

    it('should handle invalid expression gracefully', () => {
      const condWorkflow = {
        nodes: [
          { id: '1', name: 'Cond', type: 'condition', config: { conditions: [{ expression: 'invalid???', targetNodeId: 'target1' }], defaultTargetNodeId: 'default1' } },
        ],
        connections: [
          { id: 'c1', sourceNodeId: '1', targetNodeId: 'target1' },
        ],
      } as any;

      const result = WorkflowEngine.findNextNodes('1', condWorkflow, {});
      expect(result).toEqual(['default1']);
    });
  });
});
