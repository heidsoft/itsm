'use client';

/**
 * CI拓扑图组件
 * 使用reactflow渲染服务拓扑图和依赖关系
 */

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Card,
  Button,
  Space,
  Select,
  Tooltip,
  Tag,
  Badge,
  Alert,
  Drawer,
  Descriptions,
  List,
  Typography,
  Empty,
  Spin,
  message,
} from 'antd';
import { RotateCcw, AlertTriangle, Maximize, ZoomIn, ZoomOut, Network } from 'lucide-react';
import type { Node, Edge, NodeProps } from 'reactflow';
import { useTheme } from '@/lib/design-system/theme';
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
} from 'reactflow';
import 'reactflow/dist/style.css';

import {
  CIRelationshipAPI,
  type TopologyGraph,
  type TopologyNode,
  type TopologyEdge,
  type ImpactAnalysisResponse,
} from '@/lib/api/cmdb-relationship';

const { Text, Title } = Typography;

// CI节点自定义组件
const CINode: React.FC<NodeProps<any>> = ({ data }) => {
  const { isDark } = useTheme();
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

  const nodeBg = isDark ? '#1e293b' : '#fff';
  const nodeText = isDark ? '#f1f5f9' : '#1e293b';
  const nodeSubText = isDark ? '#94a3b8' : '#888';
  const nodeShadow = isDark ? '0 2px 8px rgba(0,0,0,0.45)' : '0 2px 8px rgba(0,0,0,0.1)';

  return (
    <div
      style={{
        padding: '12px 16px',
        background: nodeBg,
        borderRadius: 8,
        border: `2px solid ${statusColor}`,
        boxShadow: nodeShadow,
        minWidth: 140,
        position: 'relative',
      }}
    >
      <Handle type='target' position={Position.Top} style={{ background: statusColor }} />

      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <Badge color={statusColor} />
        <div>
          <Text strong style={{ fontSize: 13, color: nodeText }}>
            {data.name}
          </Text>
          <div style={{ fontSize: 11, color: nodeSubText }}>{data.type}</div>
        </div>
      </div>

      <Tooltip title={`Importance: ${data.criticality}`}>
        <div
          style={{
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
          }}
        >
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
  const [loadError, setLoadError] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<TopologyNode | null>(null);
  const [impactAnalysis, setImpactAnalysis] = useState<ImpactAnalysisResponse | null>(null);
  const [impactDrawerOpen, setImpactDrawerOpen] = useState(false);
  const { isDark } = useTheme();

  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  // 加载拓扑数据
  const loadTopology = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
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
      const flowEdges: Edge[] = data.edges.map(edge => ({
        id: `e${edge.source}-${edge.target}`,
        source: String(edge.source),
        target: String(edge.target),
        type: 'smoothstep',
        animated: false,
        style: {
          stroke: getEdgeColor(edge.strength),
          strokeWidth: getEdgeWidth(edge.strength),
        },
        label: edge.relationshipLabel,
        labelStyle: {
          fill: isDark ? '#94a3b8' : '#666',
          fontSize: 11,
        },
      }));
      setEdges(flowEdges);
    } catch (error) {
      console.error('Load topology graph failed:', error);
      setGraph(null);
      setNodes([]);
      setEdges([]);
      setLoadError('Topology data loading failed, please try again later.');
      message.error('Failed to load topology map');
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
      message.error('Failed to obtain impact analysis');
    }
  };

  const riskLevelColors: Record<string, string> = {
    critical: 'error',
    high: 'warning',
    medium: 'processing',
    low: 'success',
  };

  const riskLevelLabels: Record<string, string> = {
    critical: 'Fatal',
    high: 'High',
    medium: 'Medium',
    low: 'Low',
  };

  return (
    <>
      <Card
        title={
          <Space>
            <Network />
            <span>Service topology diagram</span>
            {graph && <Tag color='blue'>{graph.totalNodes}  nodes</Tag>}
          </Space>
        }
        extra={
          <Space>
            <Select
              value={depth}
              onChange={setDepth}
              style={{ width: 100 }}
              options={[
                { value: 1, label: '1st floor' },
                { value: 2, label: '2nd floor' },
                { value: 3, label: '3rd floor' },
              ]}
            />
            <Tooltip title="Refresh">
              <Button
                icon={<RotateCcw />}
                onClick={loadTopology}
                loading={loading}
                disabled={loading}
                aria-label="Refresh topology"
              />
            </Tooltip>
          </Space>
        }
      >
        <div
          style={{
            height,
            border: `1px solid ${isDark ? '#334155' : '#f0f0f0'}`,
            borderRadius: 8,
            overflow: 'hidden',
          }}
        >
          {loading ? (
            <div
              style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height }}
            >
              <Spin size='large' description="Loading topology map..." />
            </div>
          ) : loadError ? (
            <div
              style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height }}
            >
              <Alert
                type='warning'
                showIcon
                title="Topology map is temporarily unavailable"
                description={loadError}
                action={
                  <Button size='small' onClick={loadTopology}>

                    Retry
                  </Button>
                }
              />
            </div>
          ) : nodes.length === 0 ? (
            <div
              style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height }}
            >
              <Empty description="No topology data yet" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            </div>
          ) : (
            <ReactFlow
              nodes={nodes}
              edges={edges}
              nodeTypes={nodeTypes}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              fitView
              attributionPosition='bottom-left'
            >
              <Background color={isDark ? '#0f172a' : '#f0f0f0'} gap={20} />
              <Controls />
              <MiniMap
                nodeColor={node => {
                  const statusColors: Record<string, string> = {
                    operational: '#52c41a',
                    warning: '#faad14',
                    error: '#ff4d4f',
                  };
                  const status = String(node.data?.status || 'unknown');
                  return statusColors[status] || '#8c8c8c';
                }}
              />
            </ReactFlow>
          )}
        </div>

        {/* 图例 */}
        <div
          style={{
            marginTop: 16,
            padding: '8px 16px',
            background: isDark ? '#1e293b' : '#fafafa',
            borderRadius: 8,
          }}
        >
          <Space split='|'>
            <Space>
              <span>Node status:</span>
              <Badge color='#52c41a' text="OK" />
              <Badge color='#faad14' text="Warning" />
              <Badge color='#ff4d4f' text="Bug" />
            </Space>
            <Space>
              <span>Relationship Strength:</span>
              <span style={{ color: '#ff4d4f' }}>● Key</span>
              <span style={{ color: '#fa8c16' }}>● High</span>
              <span style={{ color: '#1890ff' }}>● Medium</span>
              <span style={{ color: '#8c8c8c' }}>● Low</span>
            </Space>
          </Space>
        </div>
      </Card>

      {/* 影响分析抽屉 */}
      <Drawer
        title={
          <Space>
            <AlertTriangle />
            <span>Impact Analysis - {selectedNode?.name}</span>
            {impactAnalysis && (
              <Tag color={riskLevelColors[impactAnalysis.riskLevel]}>
                {riskLevelLabels[impactAnalysis.riskLevel]}Risk
              </Tag>
            )}
          </Space>
        }
        open={impactDrawerOpen}
        onClose={() => setImpactDrawerOpen(false)}
        styles={{ wrapper: { width: 600 } }}
      >
        {impactAnalysis ? (
          <div>
            {/* 风险摘要 */}
            <Alert
              type={
                riskLevelColors[impactAnalysis.riskLevel] as
                  | 'error'
                  | 'warning'
                  | 'info'
                  | 'success'
              }
              title="Impact summary"
              description={impactAnalysis.summary}
              showIcon
              style={{ marginBottom: 16 }}
            />

            {/* 上游影响 */}
            <Card title="Upstream impact (depending on this CI)" size='small' style={{ marginBottom: 16 }}>
              {impactAnalysis.upstreamImpact.length > 0 ? (
                <List
                  dataSource={impactAnalysis.upstreamImpact}
                  renderItem={item => (
                    <List.Item>
                      <Descriptions column={1} size='small'>
                        <Descriptions.Item label="CI name">{item.ciName}</Descriptions.Item>
                        <Descriptions.Item label="Relationship">{item.relationship}</Descriptions.Item>
                        <Descriptions.Item label="Impact Level">
                          <Tag color={riskLevelColors[item.impactLevel]}>
                            {riskLevelLabels[item.impactLevel]}
                          </Tag>
                        </Descriptions.Item>
                      </Descriptions>
                    </List.Item>
                  )}
                />
              ) : (
                <Empty description="No upstream impact" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              )}
            </Card>

            {/* 下游影响 */}
            <Card title="Downstream impact (this CI depends on)" size='small' style={{ marginBottom: 16 }}>
              {impactAnalysis.downstreamImpact.length > 0 ? (
                <List
                  dataSource={impactAnalysis.downstreamImpact}
                  renderItem={item => (
                    <List.Item>
                      <Descriptions column={1} size='small'>
                        <Descriptions.Item label="CI name">{item.ciName}</Descriptions.Item>
                        <Descriptions.Item label="Relationship">{item.relationship}</Descriptions.Item>
                        <Descriptions.Item label="Impact Level">
                          <Tag color={riskLevelColors[item.impactLevel]}>
                            {riskLevelLabels[item.impactLevel]}
                          </Tag>
                        </Descriptions.Item>
                      </Descriptions>
                    </List.Item>
                  )}
                />
              ) : (
                <Empty description="No downstream effects" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              )}
            </Card>

            {/* 关键依赖 */}
            {impactAnalysis.criticalDependencies.length > 0 && (
              <Card title="Critical dependency" size='small' style={{ marginBottom: 16 }}>
                <Alert
                  type='error'
                  title="The following are key dependencies, please be careful when making changes."
                  description={
                    <ul style={{ margin: '8px 0', paddingLeft: 20 }}>
                      {impactAnalysis.criticalDependencies.map((dep, idx) => (
                        <li key={idx}>
                          {dep.ciName} ({dep.relationship})
                        </li>
                      ))}
                    </ul>
                  }
                />
              </Card>
            )}

            {/* 受影响的工单 */}
            {impactAnalysis.affectedTickets.length > 0 && (
              <Card title="Affected Tickets" size='small' style={{ marginBottom: 16 }}>
                <List
                  size='small'
                  dataSource={impactAnalysis.affectedTickets}
                  renderItem={ticket => (
                    <List.Item>
                      <List.Item.Meta
                        title={`#${ticket.id} ${ticket.title}`}
                        description={`Status: ${ticket.status} | Priority: ${ticket.priority}`}
                      />
                    </List.Item>
                  )}
                />
              </Card>
            )}

            {/* 受影响的事件 */}
            {impactAnalysis.affectedIncidents.length > 0 && (
              <Card title="Affected events" size='small'>
                <List
                  size='small'
                  dataSource={impactAnalysis.affectedIncidents}
                  renderItem={incident => (
                    <List.Item>
                      <List.Item.Meta
                        title={`#${incident.id} ${incident.title}`}
                        description={`Status: ${incident.status} | Severity: ${incident.severity}`}
                      />
                    </List.Item>
                  )}
                />
              </Card>
            )}
          </div>
        ) : (
          <Spin description="Loading impact analysis..." />
        )}
      </Drawer>
    </>
  );
};

const TopologyGraphView: React.FC<TopologyGraphViewProps> = props => {
  return (
    <ReactFlowProvider>
      <TopologyGraphViewInner {...props} />
    </ReactFlowProvider>
  );
};

export default TopologyGraphView;
