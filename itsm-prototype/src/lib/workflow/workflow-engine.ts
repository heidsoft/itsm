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
        message: '工作流必须至少包含一个节点',
      });
      return { isValid: false, errors, warnings };
    }

    // 验证开始节点
    this.validateStartNode(workflow.nodes, errors);

    // 验证结束节点
    this.validateEndNode(workflow.nodes, warnings);

    // 验证节点配置
    workflow.nodes.forEach((node) => {
      this.validateNode(node, errors, warnings);
    });

    // 验证连接
    if (workflow.connections) {
      workflow.connections.forEach((connection) => {
        this.validateConnection(
          connection,
          workflow.nodes!,
          errors,
          warnings
        );
      });
    }

    // 验证是否有孤立节点
    this.validateOrphanedNodes(
      workflow.nodes,
      workflow.connections || [],
      warnings
    );

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
  private static validateStartNode(
    nodes: WorkflowNode[],
    errors: ValidationError[]
  ): void {
    const startNodes = nodes.filter((n) => n.type === 'start');
    if (startNodes.length === 0) {
      errors.push({
        type: 'error',
        message: '工作流必须包含一个开始节点',
      });
    } else if (startNodes.length > 1) {
      errors.push({
        type: 'error',
        message: '工作流只能有一个开始节点',
      });
    }
  }

  /**
   * 验证结束节点
   */
  private static validateEndNode(
    nodes: WorkflowNode[],
    warnings: ValidationError[]
  ): void {
    const endNodes = nodes.filter((n) => n.type === 'end');
    if (endNodes.length === 0) {
      warnings.push({
        type: 'warning',
        message: '建议添加至少一个结束节点',
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
        message: '节点名称不能为空',
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
  private static validateTaskNode(
    node: WorkflowNode,
    errors: ValidationError[]
  ): void {
    const config = node.config as any;
    if (!config.assigneeType) {
      errors.push({
        nodeId: node.id,
        type: 'error',
        message: '任务节点必须指定分配类型',
        field: 'assigneeType',
      });
    }
  }

  /**
   * 验证审批节点
   */
  private static validateApprovalNode(
    node: WorkflowNode,
    errors: ValidationError[]
  ): void {
    const config = node.config as any;
    if (!config.approvers || config.approvers.length === 0) {
      errors.push({
        nodeId: node.id,
        type: 'error',
        message: '审批节点必须指定至少一个审批人',
        field: 'approvers',
      });
    }
  }

  /**
   * 验证条件节点
   */
  private static validateConditionNode(
    node: WorkflowNode,
    errors: ValidationError[]
  ): void {
    const config = node.config as any;
    if (!config.conditions || config.conditions.length === 0) {
      errors.push({
        nodeId: node.id,
        type: 'error',
        message: '条件节点必须至少定义一个条件',
        field: 'conditions',
      });
    }
  }

  /**
   * 验证脚本节点
   */
  private static validateScriptNode(
    node: WorkflowNode,
    errors: ValidationError[]
  ): void {
    const config = node.config as any;
    if (!config.script || config.script.trim() === '') {
      errors.push({
        nodeId: node.id,
        type: 'error',
        message: '脚本节点必须包含脚本内容',
        field: 'script',
      });
    }
  }

  /**
   * 验证通知节点
   */
  private static validateNotificationNode(
    node: WorkflowNode,
    errors: ValidationError[]
  ): void {
    const config = node.config as any;
    if (!config.recipients || config.recipients.length === 0) {
      errors.push({
        nodeId: node.id,
        type: 'error',
        message: '通知节点必须指定至少一个接收者',
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
    const sourceNode = nodes.find((n) => n.id === connection.sourceNodeId);
    const targetNode = nodes.find((n) => n.id === connection.targetNodeId);

    if (!sourceNode) {
      errors.push({
        connectionId: connection.id,
        type: 'error',
        message: `连接的源节点 ${connection.sourceNodeId} 不存在`,
      });
    }

    if (!targetNode) {
      errors.push({
        connectionId: connection.id,
        type: 'error',
        message: `连接的目标节点 ${connection.targetNodeId} 不存在`,
      });
    }

    // 验证开始节点不能有输入连接
    if (targetNode?.type === 'start') {
      errors.push({
        connectionId: connection.id,
        type: 'error',
        message: '开始节点不能有输入连接',
      });
    }

    // 验证结束节点不能有输出连接
    if (sourceNode?.type === 'end') {
      errors.push({
        connectionId: connection.id,
        type: 'error',
        message: '结束节点不能有输出连接',
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
    nodes.forEach((node) => {
      if (node.type === 'start' || node.type === 'end') {
        return;
      }

      const hasInput = connections.some((c) => c.targetNodeId === node.id);
      const hasOutput = connections.some((c) => c.sourceNodeId === node.id);

      if (!hasInput || !hasOutput) {
        warnings.push({
          nodeId: node.id,
          type: 'warning',
          message: `节点 "${node.name}" 可能是孤立的，没有连接`,
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
    nodes.forEach((node) => graph.set(node.id, []));
    connections.forEach((conn) => {
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
            message: '检测到工作流中存在循环，请确保这是有意的',
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
    const connections = workflow.connections.filter(
      (c) => c.sourceNodeId === currentNodeId
    );

    if (connections.length === 0) {
      return [];
    }

    const currentNode = workflow.nodes.find((n) => n.id === currentNodeId);

    // 根据节点类型确定下一个节点
    if (currentNode?.type === 'condition') {
      return this.evaluateConditions(currentNode, connections, variables);
    }

    // 默认返回所有连接的目标节点
    return connections.map((c) => c.targetNodeId);
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
        const result = this.evaluateExpression(
          condition.expression,
          variables
        );
        if (result) {
          return [condition.targetNodeId];
        }
      } catch (error) {
        console.error('条件评估失败:', error);
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
  private static evaluateExpression(
    expression: string,
    variables: Record<string, any>
  ): boolean {
    // 这里应该使用安全的表达式引擎
    // 简化实现，仅供示例
    try {
      const func = new Function(
        ...Object.keys(variables),
        `return ${expression}`
      );
      return func(...Object.values(variables));
    } catch {
      return false;
    }
  }
}

export default WorkflowEngine;

