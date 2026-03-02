'use client';

/**
 * CI拓扑图组件
 * 使用reactflow渲染服务拓扑图和依赖关系
 */

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Card, Button, Space, Select, Tooltip, Tag, Badge, Alert,
  Drawer, Descriptions, List, Typography, Empty, Spin, message
} from 'antd';
import {
  ZoomInOutlined, ZoomOutOutlined, FullscreenOutlined,
  ReloadOutlined, NodeIndexOutlined, WarningOutlined
} from '@ant-design/icons';
import type { Node, Edge } from 'reactflow';
import ReactFlow, {
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  Handle,
  Position,
  useReactFlow,
  ReactFlowProvider,
  NodeProps
} from 'reactflow';
import 'reactflow/dist/style.css';

import { CIRelationshipAPI, type TopologyGraph, type TopologyNode, type TopologyEdge, type ImpactAnalysisResponse } from '@/lib/api/cmdb-relationship';

const { Text, Title } = Typography;

// CI节点自定义组件
const CINode: React.FC<NodeProps<any>> = ({ data }) => {
  const statusColors: Record<string, string> = {
    operational: '#52c41a',
    warning: '#faad14',
    error: '#ff4d4f',
    unknown: '#8c8c8c',
  };

  const criticalityColors: Record<string, string> = {
    critical: '#ff4d4f',
    high: '#fa8c16',
    medium: '#1890ff',
    low: '#52c41a',
  };

  const statusColor = statusColors[data.status] || '#8c8c8c';
  const criticalityColor = criticalityColors[data.criticality] || '#8c8c8c';

  return (
    <div style={{
      padding: '12px 16px',
      background: '#fff',
      borderRadius: 8,
      border: `2px solid ${statusColor}`,
      boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
      minWidth: 140,
      position: 'relative',
    }}>
      <Handle type="target" position={Position.Top} style={{ background: statusColor }} />

      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <Badge color={statusColor} />
        <div>
          <Text strong style={{ fontSize: 13 }}>{data.name}</Text>
          <div style={{ fontSize: 11, color: '#888' }}>{data.type}</div>
        </div>
      </div>

      <Tooltip title={`重要性: ${data.criticality}`}>
        <div style={{
          position: 'absolute',
          top: -8,
          right: -8,
          width: 20,
          height: 20,
          borderRadius: '50%',
          background: criticalityColor,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
        }}>
          <span style={{ color: '#fff', fontSize: 10 }}>!</span>
        </div>
      </Tooltip>

      {data.onClick && (
        <div
          style={{
            position: 'absolute',
            inset: 0,
            borderRadius: 8,
            cursor: 'pointer',
          }}
          onClick={data.onClick}
        />
      )}
    </div>
  );
};

// 节点类型
const nodeTypes = {
  ci: CINode,
};

interface TopologyGraphViewProps {
  ciId: number;
  initialDepth?: number;
  onNodeClick?: (node: TopologyNode) => void;
  height?: number;
}

const TopologyGraphViewInner: React.FC<TopologyGraphViewProps> = ({
  ciId,
  initialDepth = 2,
  onNodeClick,
  height = 600,
}) => {
  const [loading, setLoading] = useState(true);
  const [graph, setGraph] = useState<TopologyGraph | null>(null);
  const [depth, setDepth] = useState(initialDepth);
  const [selectedNode, setSelectedNode] = useState<TopologyNode | null>(null);
  const [impactAnalysis, setImpactAnalysis] = useState<ImpactAnalysisResponse | null>(null);
  const [impactDrawerOpen, setImpactDrawerOpen] = useState(false);

  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  // 加载拓扑数据
  const loadTopology = useCallback(async () => {
    setLoading(true);
    try {
      const data = await CIRelationshipAPI.getTopologyGraph(ciId, depth);
      setGraph(data);

      // 转换节点
      const flowNodes: Node<any>[] = data.nodes.map((node, index) => ({
        id: String(node.id),
        type: 'ci',
        position: calculateNodePosition(index, data.nodes.length),
        data: {
          ...node,
          onClick: () => handleNodeClick(node),
        },
      }));
      setNodes(flowNodes);

      // 转换边
      const flowEdges: Edge[] = data.edges.map((edge) => ({
        id: `e${edge.source}-${edge.target}`,
        source: String(edge.source),
        target: String(edge.target),
        type: 'smoothstep',
        animated: false,
        style: {
          stroke: getEdgeColor(edge.strength),
          strokeWidth: getEdgeWidth(edge.strength),
        },
        label: edge.relationship_label,
        labelStyle: {
          fill: '#666',
          fontSize: 11,
        },
      }));
      setEdges(flowEdges);
    } catch (error) {
      message.error('加载拓扑图失败');
    } finally {
      setLoading(false);
    }
  }, [ciId, depth, setNodes, setEdges]);

  useEffect(() => {
    loadTopology();
  }, [loadTopology]);

  // 计算节点位置
  const calculateNodePosition = (index: number, total: number): { x: number; y: number } => {
    const centerX = 400;
    const centerY = 300;
    const radius = Math.min(300, total * 30);
    const angle = (index * 2 * Math.PI) / total - Math.PI / 2;

    return {
      x: centerX + radius * Math.cos(angle),
      y: centerY + radius * Math.sin(angle),
    };
  };

  // 获取边颜色
  const getEdgeColor = (strength: string): string => {
    const colors: Record<string, string> = {
      critical: '#ff4d4f',
      high: '#fa8c16',
      medium: '#1890ff',
      low: '#8c8c8c',
    };
    return colors[strength] || '#8c8c8c';
  };

  // 获取边宽度
  const getEdgeWidth = (strength: string): number => {
    const widths: Record<string, number> = {
      critical: 4,
      high: 3,
      medium: 2,
      low: 1,
    };
    return widths[strength] || 1;
  };

  // 处理节点点击
  const handleNodeClick = async (node: TopologyNode) => {
    setSelectedNode(node);
    setImpactDrawerOpen(true);

    try {
      const analysis = await CIRelationshipAPI.analyzeImpact(node.id);
      setImpactAnalysis(analysis);
    } catch (error) {
      message.error('获取影响分析失败');
    }
  };

  const riskLevelColors: Record<string, string> = {
    critical: 'error',
    high: 'warning',
    medium: 'processing',
    low: 'success',
  };

  const riskLevelLabels: Record<string, string> = {
    critical: '致命',
    high: '高',
    medium: '中',
    low: '低',
  };

  return (
    <>
      <Card
        title={
          <Space>
            <NodeIndexOutlined />
            <span>服务拓扑图</span>
            {graph && (
              <Tag color="blue">{graph.total_nodes} 个节点</Tag>
            )}
          </Space>
        }
        extra={
          <Space>
            <Select
              value={depth}
              onChange={setDepth}
              style={{ width: 100 }}
              options={[
                { value: 1, label: '1层' },
                { value: 2, label: '2层' },
                { value: 3, label: '3层' },
              ]}
            />
            <Tooltip title="刷新">
              <Button icon={<ReloadOutlined />} onClick={loadTopology} />
            </Tooltip>
          </Space>
        }
      >
        <div style={{ height, border: '1px solid #f0f0f0', borderRadius: 8, overflow: 'hidden' }}>
          {loading ? (
            <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height }}>
              <Spin size="large" tip="加载拓扑图..." />
            </div>
          ) : (
            <ReactFlow
              nodes={nodes}
              edges={edges}
              nodeTypes={nodeTypes}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              fitView
              attributionPosition="bottom-left"
            >
              <Background color="#f0f0f0" gap={20} />
              <Controls />
              <MiniMap
                nodeColor={(node) => {
                  const statusColors: Record<string, string> = {
                    operational: '#52c41a',
                    warning: '#faad14',
                    error: '#ff4d4f',
                  };
                  return statusColors[node.data.status] || '#8c8c8c';
                }}
              />
            </ReactFlow>
          )}
        </div>

        {/* 图例 */}
        <div style={{ marginTop: 16, padding: '8px 16px', background: '#fafafa', borderRadius: 8 }}>
          <Space split="|">
            <Space>
              <span>节点状态:</span>
              <Badge color="#52c41a" text="正常" />
              <Badge color="#faad14" text="警告" />
              <Badge color="#ff4d4f" text="故障" />
            </Space>
            <Space>
              <span>关系强度:</span>
              <span style={{ color: '#ff4d4f' }}>● 关键</span>
              <span style={{ color: '#fa8c16' }}>● 高</span>
              <span style={{ color: '#1890ff' }}>● 中</span>
              <span style={{ color: '#8c8c8c' }}>● 低</span>
            </Space>
          </Space>
        </div>
      </Card>

      {/* 影响分析抽屉 */}
      <Drawer
        title={
          <Space>
            <WarningOutlined />
            <span>影响分析 - {selectedNode?.name}</span>
            {impactAnalysis && (
              <Tag color={riskLevelColors[impactAnalysis.risk_level]}>
                {riskLevelLabels[impactAnalysis.risk_level]}风险
              </Tag>
            )}
          </Space>
        }
        open={impactDrawerOpen}
        onClose={() => setImpactDrawerOpen(false)}
        width={600}
      >
        {impactAnalysis ? (
          <div>
            {/* 风险摘要 */}
            <Alert
              type={riskLevelColors[impactAnalysis.risk_level] as 'error' | 'warning' | 'info' | 'success'}
              message="影响摘要"
              description={impactAnalysis.summary}
              showIcon
              style={{ marginBottom: 16 }}
            />

            {/* 上游影响 */}
            <Card title="上游影响（依赖此CI的）" size="small" style={{ marginBottom: 16 }}>
              {impactAnalysis.upstream_impact.length > 0 ? (
                <List
                  dataSource={impactAnalysis.upstream_impact}
                  renderItem={(item) => (
                    <List.Item>
                      <Descriptions column={1} size="small">
                        <Descriptions.Item label="CI名称">{item.ci_name}</Descriptions.Item>
                        <Descriptions.Item label="关系">{item.relationship}</Descriptions.Item>
                        <Descriptions.Item label="影响程度">
                          <Tag color={riskLevelColors[item.impact_level]}>
                            {riskLevelLabels[item.impact_level]}
                          </Tag>
                        </Descriptions.Item>
                      </Descriptions>
                    </List.Item>
                  )}
                />
              ) : (
                <Empty description="无上游影响" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              )}
            </Card>

            {/* 下游影响 */}
            <Card title="下游影响（此CI依赖的）" size="small" style={{ marginBottom: 16 }}>
              {impactAnalysis.downstream_impact.length > 0 ? (
                <List
                  dataSource={impactAnalysis.downstream_impact}
                  renderItem={(item) => (
                    <List.Item>
                      <Descriptions column={1} size="small">
                        <Descriptions.Item label="CI名称">{item.ci_name}</Descriptions.Item>
                        <Descriptions.Item label="关系">{item.relationship}</Descriptions.Item>
                        <Descriptions.Item label="影响程度">
                          <Tag color={riskLevelColors[item.impact_level]}>
                            {riskLevelLabels[item.impact_level]}
                          </Tag>
                        </Descriptions.Item>
                      </Descriptions>
                    </List.Item>
                  )}
                />
              ) : (
                <Empty description="无下游影响" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              )}
            </Card>

            {/* 关键依赖 */}
            {impactAnalysis.critical_dependencies.length > 0 && (
              <Card title="关键依赖" size="small" style={{ marginBottom: 16 }}>
                <Alert
                  type="error"
                  message="以下为关键依赖，变更需谨慎"
                  description={
                    <ul style={{ margin: '8px 0', paddingLeft: 20 }}>
                      {impactAnalysis.critical_dependencies.map((dep, idx) => (
                        <li key={idx}>{dep.ci_name} ({dep.relationship})</li>
                      ))}
                    </ul>
                  }
                />
              </Card>
            )}

            {/* 受影响的工单 */}
            {impactAnalysis.affected_tickets.length > 0 && (
              <Card title="受影响的工单" size="small" style={{ marginBottom: 16 }}>
                <List
                  size="small"
                  dataSource={impactAnalysis.affected_tickets}
                  renderItem={(ticket) => (
                    <List.Item>
                      <List.Item.Meta
                        title={`#${ticket.id} ${ticket.title}`}
                        description={`状态: ${ticket.status} | 优先级: ${ticket.priority}`}
                      />
                    </List.Item>
                  )}
                />
              </Card>
            )}

            {/* 受影响的事件 */}
            {impactAnalysis.affected_incidents.length > 0 && (
              <Card title="受影响的事件" size="small">
                <List
                  size="small"
                  dataSource={impactAnalysis.affected_incidents}
                  renderItem={(incident) => (
                    <List.Item>
                      <List.Item.Meta
                        title={`#${incident.id} ${incident.title}`}
                        description={`状态: ${incident.status} | 严重程度: ${incident.severity}`}
                      />
                    </List.Item>
                  )}
                />
              </Card>
            )}
          </div>
        ) : (
          <Spin tip="加载影响分析..." />
        )}
      </Drawer>
    </>
  );
};

const TopologyGraphView: React.FC<TopologyGraphViewProps> = (props) => {
  return (
    <ReactFlowProvider>
      <TopologyGraphViewInner {...props} />
    </ReactFlowProvider>
  );
};

export default TopologyGraphView;
