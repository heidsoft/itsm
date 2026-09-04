'use client';

import React, { useState, useEffect, useMemo } from 'react';
import { App, Card, Select, Button, Space, Tag, Spin, Drawer, Descriptions, Empty } from 'antd';
import { RotateCcw, ExternalLink } from 'lucide-react';
import { useRouter } from 'next/navigation';
import type { Node, Edge, NodeTypes} from 'reactflow';
import ReactFlow, { Controls, Background, useNodesState, useEdgesState, MarkerType, BackgroundVariant, Handle, Position } from 'reactflow';
import 'reactflow/dist/style.css';
import dagre from 'dagre';
import { PageContainer } from '@/components/layout/PageContainer';
import { useCIsQuery, useTopologyGraphQuery } from '@/lib/hooks/useCMDB';
import { type TopologyNode } from '@/lib/api/cmdb-relationship';
import { CIStatusLabels } from '@/constants/cmdb';

const ciTypeIcons: Record<string, string> = {
  server: '🖥️', database: '🗄️', application: '📦', network: '🌐', storage: '💾', cloud: '☁️', default: '📋'
};

const ciTypeColors: Record<string, string> = {
  server: '#1890ff', database: '#52c41a', application: '#722ed1', network: '#13c2c2', storage: '#faad14', cloud: '#f5222d', default: '#8c8c8c'
};

// CMDB类型中文映射
const ciTypeNameMap: Record<string, string> = {
  server: '服务器',
  database: '数据库',
  application: '应用程序',
  network: '网络设备',
  storage: '存储设备',
  cloud: '云资源',
};

// CMDB状态中文映射
const ciStatusNameMap: Record<string, string> = {
  active: '活跃',
  inactive: '未激活',
  maintenance: '维护中',
  retired: '已下线',
};

// 关键程度中文映射
const criticalityNameMap: Record<string, string> = {
  critical: '关键',
  high: '高',
  medium: '中',
  low: '低',
};

// dagre 自动分层布局使用的节点尺寸估计值
const NODE_WIDTH = 180;
const NODE_HEIGHT = 130;

/** 使用 dagre 对节点做自动分层布局（自上而下） */
function layoutWithDagre(nodes: Node[], edges: Edge[]): Node[] {
  const graph = new dagre.graphlib.Graph();
  graph.setDefaultEdgeLabel(() => ({}));
  graph.setGraph({ rankdir: 'TB', nodesep: 60, ranksep: 100 });

  nodes.forEach(node => {
    graph.setNode(node.id, { width: NODE_WIDTH, height: NODE_HEIGHT });
  });
  edges.forEach(edge => {
    graph.setEdge(edge.source, edge.target);
  });

  dagre.layout(graph);

  return nodes.map(node => {
    const pos = graph.node(node.id);
    return {
      ...node,
      // dagre 返回中心点坐标，reactflow 需要左上角坐标
      position: { x: pos.x - NODE_WIDTH / 2, y: pos.y - NODE_HEIGHT / 2 },
    };
  });
}

const CINode = ({ data }: { data: TopologyNode & { selected?: boolean; dimmed?: boolean } }) => {
  const icon = ciTypeIcons[data.type?.toLowerCase()] || ciTypeIcons.default;
  const color = ciTypeColors[data.type?.toLowerCase()] || ciTypeColors.default;
  return (
    <div style={{ padding: '12px 16px', borderRadius: 8, background: '#fff', border: data.selected ? '2px solid ' + color : '2px solid ' + color + '40', boxShadow: data.selected ? '0 4px 12px rgba(0,0,0,0.2)' : '0 2px 8px rgba(0,0,0,0.1)', minWidth: 140, textAlign: 'center', opacity: data.dimmed ? 0.3 : 1, transition: 'opacity 0.2s' }}>
      <Handle type="target" position={Position.Top} style={{ background: color }} />
      <div style={{ fontSize: 24 }}>{icon}</div>
      <div style={{ fontWeight: 600, marginTop: 4, color: '#333' }}>{data.name}</div>
      <Tag color={color} style={{ marginTop: 4 }}>{ciTypeNameMap[data.type?.toLowerCase()] || data.typeName || data.type}</Tag>
      {data.status && <Tag color={data.status === 'active' ? 'green' : 'default'} style={{ marginTop: 2 }}>{ciStatusNameMap[data.status] || data.status}</Tag>}
      <Handle type="source" position={Position.Bottom} style={{ background: color }} />
    </div>
  );
};

const nodeTypes: NodeTypes = { ciNode: CINode };

export default function TopologyPage() {
  const router = useRouter();
  const { message: messageApi } = App.useApp();
  // 把全局 message 代理到 messageApi（保持旧调用语义）
  const message = { error: (msg: string) => messageApi.error(msg) };

  const [selectedCI, setSelectedCI] = useState<number | null>(null);
  const [searchInput, setSearchInput] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');

  // CI 选项防抖（300ms）
  useEffect(() => {
    const t = setTimeout(() => setDebouncedSearch(searchInput), 300);
    return () => clearTimeout(t);
  }, [searchInput]);

  // React Query：CI 选项（按搜索值变化自动失效/拉取）
  const cisQuery = useCIsQuery({
    search: debouncedSearch || undefined,
    size: 20,
  });
  const ciOptions = useMemo(
    () =>
      ((cisQuery.data?.items ?? []) as Array<{
        id: number;
        name: string;
        type?: string;
        ciType?: string;
        status?: string;
      }>).map(ci => ({
        id: ci.id,
        name: ci.name,
        type: ci.ciType || ci.type || '',
        status: ci.status,
      })),
    [cisQuery.data]
  );

  const [depth, setDepth] = useState(2);
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [selectedNodeData, setSelectedNodeData] = useState<TopologyNode | null>(null);
  // 当前点击高亮的节点 id（一跳影响面高亮）
  const [highlightNodeId, setHighlightNodeId] = useState<string | null>(null);

  // React Query：拓扑图（仅在选中 CI 时拉取，按 depth 缓存）
  const topologyQuery = useTopologyGraphQuery(selectedCI ?? 0, depth, !!selectedCI);
  const loading = topologyQuery.isFetching;

  // 拓扑数据落入 react-flow 节点/边
  useEffect(() => {
    if (!topologyQuery.data) {
      setNodes([]);
      setEdges([]);
      return;
    }
    const graph = topologyQuery.data;
    const flowEdges: Edge[] = graph.edges.map(edge => ({
      id: 'e' + edge.id,
      source: String(edge.source),
      target: String(edge.target),
      label: edge.relationshipLabel,
      type: 'smoothstep',
      animated: edge.impactLevel === 'critical',
      style: { stroke: getEdgeColor(edge.strength) },
      markerEnd: { type: MarkerType.ArrowClosed, color: getEdgeColor(edge.strength) },
    }));
    const rawNodes: Node[] = graph.nodes.map(node => ({
      id: String(node.id),
      type: 'ciNode',
      position: { x: 0, y: 0 },
      data: { ...node },
    }));
    setNodes(layoutWithDagre(rawNodes, flowEdges));
    setEdges(flowEdges);
    setHighlightNodeId(null);
  }, [topologyQuery.data, setNodes, setEdges]);

  // 错误反馈（失败弹一次）
  useEffect(() => {
    if (topologyQuery.isError) message.error('加载拓扑图失败');
  }, [topologyQuery.isError, message]);

  const onNodeClick = React.useCallback((_: unknown, node: Node) => {
    setSelectedNodeData(node.data as TopologyNode);
    setHighlightNodeId(node.id);
    setDrawerVisible(true);
  }, []);

  const onPaneClick = React.useCallback(() => {
    setHighlightNodeId(null);
  }, []);

  const handleRefresh = () => {
    topologyQuery.refetch();
  };

  // 一跳邻居集合（含自身）
  const neighborIds = useMemo(() => {
    if (!highlightNodeId) return null;
    const set = new Set<string>([highlightNodeId]);
    edges.forEach(edge => {
      if (edge.source === highlightNodeId) set.add(edge.target);
      if (edge.target === highlightNodeId) set.add(edge.source);
    });
    return set;
  }, [highlightNodeId, edges]);

  // 高亮态渲染：直接上下游保持醒目，其余节点/边淡化
  const displayNodes = useMemo(
    () =>
      nodes.map(node => ({
        ...node,
        data: {
          ...node.data,
          selected: node.id === highlightNodeId,
          dimmed: neighborIds ? !neighborIds.has(node.id) : false,
        },
      })),
    [nodes, highlightNodeId, neighborIds]
  );

  const displayEdges = useMemo(
    () =>
      edges.map(edge => {
        const isNeighborEdge =
          highlightNodeId !== null &&
          (edge.source === highlightNodeId || edge.target === highlightNodeId);
        return {
          ...edge,
          style: {
            ...edge.style,
            opacity: highlightNodeId && !isNeighborEdge ? 0.15 : 1,
            strokeWidth: isNeighborEdge ? 2.5 : 1.5,
          },
        };
      }),
    [edges, highlightNodeId]
  );

  return (
    <PageContainer title="CMDB 拓扑图" description="查看一个配置项周围的依赖关系图，支持 1-4 层关系深度。">
      <Card className="shadow-sm rounded-lg mb-4">
        <Space wrap>
          <Select
            placeholder="输入名称搜索配置项"
            showSearch
            style={{ width: 300 }}
            value={selectedCI}
            onChange={setSelectedCI}
            onSearch={setSearchInput}
            filterOption={false}
            loading={cisQuery.isFetching}
            notFoundContent={cisQuery.isFetching ? '搜索中...' : '无匹配配置项'}
            allowClear
            options={ciOptions.map(ci => ({
              value: ci.id,
              label: (
                <span>
                  {ci.name}
                  <Tag style={{ marginLeft: 8 }}>{ci.type}</Tag>
                  {ci.status && (CIStatusLabels as Record<string, string>)[ci.status] && (
                    <Tag>{(CIStatusLabels as Record<string, string>)[ci.status]}</Tag>
                  )}
                </span>
              ),
            }))}
          />
          <Select placeholder="关系深度" value={depth} onChange={setDepth} style={{ width: 120 }}
            options={[{ value: 1, label: '1 层' }, { value: 2, label: '2 层' }, { value: 3, label: '3 层' }, { value: 4, label: '4 层' }]} />
          <Button icon={<RotateCcw />} onClick={handleRefresh} loading={loading} disabled={!selectedCI}>刷新</Button>
          {highlightNodeId && <Tag color="blue" closable onClose={() => setHighlightNodeId(null)}>已高亮直接上下游（点击空白处取消）</Tag>}
        </Space>
      </Card>
      <Card className="shadow-sm rounded-lg" style={{ height: 'calc(100vh - 250px)' }}>
        {loading ? <div style={{ textAlign: 'center', padding: 100 }}><Spin size="large" /><div style={{ marginTop: 16 }}>加载拓扑图中...</div></div>
        : selectedCI ? <div style={{ height: '100%' }}><ReactFlow nodes={displayNodes} edges={displayEdges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} onNodeClick={onNodeClick} onPaneClick={onPaneClick}
            nodeTypes={nodeTypes} fitView fitViewOptions={{ padding: 0.2 }} attributionPosition="bottom-left"><Controls /><Background variant={BackgroundVariant.Dots} gap={20} size={1} /></ReactFlow></div>
        : <Empty description="请选择一个配置项查看其拓扑关系" style={{ paddingTop: 100 }} />}
      </Card>
      <Drawer
        title="配置项详情"
        placement="right"
        width={400}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
        extra={selectedNodeData && (
          <Button
            type="primary"
            size="small"
            icon={<ExternalLink className="w-3.5 h-3.5" />}
            onClick={() => router.push(`/cmdb/cis/${selectedNodeData.id}`)}
          >
            查看 CI 详情
          </Button>
        )}
      >
        {selectedNodeData && <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="ID">{selectedNodeData.id}</Descriptions.Item>
          <Descriptions.Item label="名称">{selectedNodeData.name}</Descriptions.Item>
          <Descriptions.Item label="类型">{ciTypeNameMap[selectedNodeData.type?.toLowerCase()] || selectedNodeData.typeName}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag color={selectedNodeData.status === 'active' ? 'green' : 'default'}>{ciStatusNameMap[selectedNodeData.status] || selectedNodeData.status}</Tag></Descriptions.Item>
          <Descriptions.Item label="关键程度"><Tag color={selectedNodeData.criticality === 'critical' ? 'red' : 'default'}>{criticalityNameMap[selectedNodeData.criticality] || selectedNodeData.criticality}</Tag></Descriptions.Item>
        </Descriptions>}
      </Drawer>
    </PageContainer>
  );
}

function getEdgeColor(strength: string): string {
  switch (strength) { case 'critical': return '#f5222d'; case 'high': return '#fa8c16'; case 'medium': return '#1890ff'; case 'low': return '#8c8c8c'; default: return '#8c8c8c'; }
}
