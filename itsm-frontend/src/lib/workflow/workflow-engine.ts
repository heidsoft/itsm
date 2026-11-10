/**
 * 工作流引擎 - 核心执行和验证逻辑
 */

import type {
  WorkflowDefinition,
  WorkflowNode,
  WorkflowConnection,
  ValidationResult,
  ValidationError,
  NodeType,
} from '@/types/workflow';

export class WorkflowEngine {
  /**
   * 验证工作流定义
   */
  static validate(workflow: Partial<WorkflowDefinition>): ValidationResult {
    const errors: ValidationError[] = [];
    const warnings: ValidationError[] = [];

    if (!workflow.nodes || workflow.nodes.length === 0) {
      errors.push({
        type: 'error',
        message: 'Workflow must contain at least one node',
      });
      return { isValid: false, errors, warnings };
    }

    // 验证开始节点
    this.validateStartNode(workflow.nodes, errors);

    // 验证结束节点
    this.validateEndNode(workflow.nodes, warnings);

    // 验证节点配置
    workflow.nodes.forEach(node => {
      this.validateNode(node, errors, warnings);
    });

    // 验证连接
    if (workflow.connections) {
      workflow.connections.forEach(connection => {
        this.validateConnection(connection, workflow.nodes!, errors, warnings);
      });
    }

    // 验证是否有孤立节点
    this.validateOrphanedNodes(workflow.nodes, workflow.connections || [], warnings);

    // 验证是否有循环
    this.validateCycles(workflow.nodes, workflow.connections || [], warnings);

    return {
      isValid: errors.length === 0,
      errors,
      warnings,
    };
  }

  /**
   * 验证开始节点
   */
  private static validateStartNode(nodes: WorkflowNode[], errors: ValidationError[]): void {
    const startNodes = nodes.filter(n => n.type === 'start');
    if (startNodes.length === 0) {
      errors.push({
        type: 'error',
        message: 'Workflow must contain a start node',
      });
    } else if (startNodes.length > 1) {
      errors.push({
        type: 'error',
        message: 'Workflow can only have one start node',
      });
    }
  }

  /**
   * 验证结束节点
   */
  private static validateEndNode(nodes: WorkflowNode[], warnings: ValidationError[]): void {
    const endNodes = nodes.filter(n => n.type === 'end');
    if (endNodes.length === 0) {
      warnings.push({
        type: 'warning',
        message: 'It is recommended to add at least one end node',
      });
    }
  }

  /**
   * 验证单个节点
   */
  private static validateNode(
    node: WorkflowNode,
    errors: ValidationError[],
    warnings: ValidationError[]
  ): void {
    if (!node.name || node.name.trim() === '') {
      errors.push({
        nodeId: node.id,
        type: 'error',
        message: 'Node name cannot be empty',
        field: 'name',
      });
    }

    // 根据节点类型验证配置
    switch (node.type) {
      case 'task':
        this.validateTaskNode(node, errors);
        break;
      case 'approval':
        this.validateApprovalNode(node, errors);
        break;
      case 'condition':
        this.validateConditionNode(node, errors);
        break;
      case 'script':
        this.validateScriptNode(node, errors);
        break;
      case 'notification':
        this.validateNotificationNode(node, errors);
        break;
    }
  }

  /**
   * 验证任务节点
   */
  private static validateTaskNode(node: WorkflowNode, errors: ValidationError[]): void {
    const config = node.config as any;
    if (!config.assigneeType) {
      errors.push({
        nodeId: node.id,
        type: 'error',
        message: 'Task node must specify allocation type',
        field: 'assigneeType',
      });
    }
  }

  /**
   * 验证审批节点
   */
  private static validateApprovalNode(node: WorkflowNode, errors: ValidationError[]): void {
    const config = node.config as any;
    if (!config.approvers || config.approvers.length === 0) {
      errors.push({
        nodeId: node.id,
        type: 'error',
        message: 'The approval node must specify at least one approver',
        field: 'approvers',
      });
    }
  }

  /**
   * 验证条件节点
   */
  private static validateConditionNode(node: WorkflowNode, errors: ValidationError[]): void {
    const config = node.config as any;
    if (!config.conditions || config.conditions.length === 0) {
      errors.push({
        nodeId: node.id,
        type: 'error',
        message: 'The condition node must define at least one condition',
        field: 'conditions',
      });
    }
  }

  /**
   * 验证脚本节点
   */
  private static validateScriptNode(node: WorkflowNode, errors: ValidationError[]): void {
    const config = node.config as any;
    if (!config.script || config.script.trim() === '') {
      errors.push({
        nodeId: node.id,
        type: 'error',
        message: 'Script node must contain script content',
        field: 'script',
      });
    }
  }

  /**
   * 验证通知节点
   */
  private static validateNotificationNode(node: WorkflowNode, errors: ValidationError[]): void {
    const config = node.config as any;
    if (!config.recipients || config.recipients.length === 0) {
      errors.push({
        nodeId: node.id,
        type: 'error',
        message: 'The notification node must specify at least one recipient',
        field: 'recipients',
      });
    }
  }

  /**
   * 验证连接
   */
  private static validateConnection(
    connection: WorkflowConnection,
    nodes: WorkflowNode[],
    errors: ValidationError[],
    warnings: ValidationError[]
  ): void {
    const sourceNode = nodes.find(n => n.id === connection.sourceNodeId);
    const targetNode = nodes.find(n => n.id === connection.targetNodeId);

    if (!sourceNode) {
      errors.push({
        connectionId: connection.id,
        type: 'error',
        message: `Source node of the connection ${connection.sourceNodeId} does not exist`,
      });
    }

    if (!targetNode) {
      errors.push({
        connectionId: connection.id,
        type: 'error',
        message: `The target node of the connection ${connection.targetNodeId} does not exist`,
      });
    }

    // 验证开始节点不能有输入连接
    if (targetNode?.type === 'start') {
      errors.push({
        connectionId: connection.id,
        type: 'error',
        message: 'Start node cannot have input connections',
      });
    }

    // 验证结束节点不能有输出连接
    if (sourceNode?.type === 'end') {
      errors.push({
        connectionId: connection.id,
        type: 'error',
        message: 'End node cannot have output connections',
      });
    }
  }

  /**
   * 验证孤立节点
   */
  private static validateOrphanedNodes(
    nodes: WorkflowNode[],
    connections: WorkflowConnection[],
    warnings: ValidationError[]
  ): void {
    nodes.forEach(node => {
      if (node.type === 'start' || node.type === 'end') {
        return;
      }

      const hasInput = connections.some(c => c.targetNodeId === node.id);
      const hasOutput = connections.some(c => c.sourceNodeId === node.id);

      if (!hasInput || !hasOutput) {
        warnings.push({
          nodeId: node.id,
          type: 'warning',
          message: `node "${node.name}"Possibly orphaned, not connected`,
        });
      }
    });
  }

  /**
   * 验证循环（简单实现）
   */
  private static validateCycles(
    nodes: WorkflowNode[],
    connections: WorkflowConnection[],
    warnings: ValidationError[]
  ): void {
    // 构建邻接表
    const graph = new Map<string, string[]>();
    nodes.forEach(node => graph.set(node.id, []));
    connections.forEach(conn => {
      const edges = graph.get(conn.sourceNodeId) || [];
      edges.push(conn.targetNodeId);
      graph.set(conn.sourceNodeId, edges);
    });

    // DFS检测循环
    const visited = new Set<string>();
    const recursionStack = new Set<string>();

    const hasCycle = (nodeId: string): boolean => {
      visited.add(nodeId);
      recursionStack.add(nodeId);

      const neighbors = graph.get(nodeId) || [];
      for (const neighbor of neighbors) {
        if (!visited.has(neighbor)) {
          if (hasCycle(neighbor)) {
            return true;
          }
        } else if (recursionStack.has(neighbor)) {
          return true;
        }
      }

      recursionStack.delete(nodeId);
      return false;
    };

    for (const node of nodes) {
      if (!visited.has(node.id)) {
        if (hasCycle(node.id)) {
          warnings.push({
            type: 'warning',
            message: 'Loop detected in workflow, please make sure this is intentional',
          });
          break;
        }
      }
    }
  }

  /**
   * 查找下一个可执行节点
   */
  static findNextNodes(
    currentNodeId: string,
    workflow: WorkflowDefinition,
    variables: Record<string, any>
  ): string[] {
    const connections = workflow.connections.filter(c => c.sourceNodeId === currentNodeId);

    if (connections.length === 0) {
      return [];
    }

    const currentNode = workflow.nodes.find(n => n.id === currentNodeId);

    // 根据节点类型确定下一个节点
    if (currentNode?.type === 'condition') {
      return this.evaluateConditions(currentNode, connections, variables);
    }

    // 默认返回所有连接的目标节点
    return connections.map(c => c.targetNodeId);
  }

  /**
   * 评估条件
   */
  private static evaluateConditions(
    node: WorkflowNode,
    connections: WorkflowConnection[],
    variables: Record<string, any>
  ): string[] {
    const config = node.config as any;
    const conditions = config.conditions || [];

    for (const condition of conditions) {
      try {
        // 简单的表达式评估（实际应使用安全的表达式引擎）
        const result = this.evaluateExpression(condition.expression, variables);
        if (result) {
          return [condition.targetNodeId];
        }
      } catch (error) {
        // console.error('条件评估失败:', error);
      }
    }

    // 返回默认连接
    if (config.defaultTargetNodeId) {
      return [config.defaultTargetNodeId];
    }

    return [];
  }

  /**
   * 评估表达式
   */
  private static evaluateExpression(expression: string, variables: Record<string, any>): boolean {
    // 这里应该使用安全的表达式引擎
    // 简化实现，仅供示例
    try {
      const func = new Function(...Object.keys(variables), `return ${expression}`);
      return func(...Object.values(variables));
    } catch {
      return false;
    }
  }
}

export default WorkflowEngine;
