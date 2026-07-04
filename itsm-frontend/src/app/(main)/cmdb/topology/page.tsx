'use client';

import React, { useState, useCallback } from 'react';
import { Card, Select, Button, Space, Tag, Spin, message, Drawer, Descriptions, Empty } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import type { Node, Edge, NodeTypes} from 'reactflow';
import ReactFlow, { Controls, Background, useNodesState, useEdgesState, MarkerType, BackgroundVariant, Panel, Handle, Position } from 'reactflow';
import 'reactflow/dist/style.css';
import { PageContainer } from '@/components/layout/PageContainer';
import { CMDBApi } from '@/lib/api/cmdb-api';
import { CIRelationshipAPI, type TopologyNode } from '@/lib/api/cmdb-relationship';

const ciTypeIcons: Record<string, string> = {
  server: '🖥️', database: '🗄️', application: '📦', network: '🌐', storage: '💾', cloud: '☁️', default: '📋'
};

const ciTypeColors: Record<string, string> = {
  server: '#1890ff', database: '#52c41a', application: '#722ed1', network: '#13c2c2', storage: '#faad14', cloud: '#f5222d', default: '#8c8c8c'
};

const CINode = ({ data }: { data: TopologyNode & { selected?: boolean } }) => {
  const icon = ciTypeIcons[data.type?.toLowerCase()] || ciTypeIcons.default;
  const color = ciTypeColors[data.type?.toLowerCase()] || ciTypeColors.default;
  return (
    <div style={{ padding: '12px 16px', borderRadius: 8, background: '#fff', border: data.selected ? '2px solid ' + color : '2px solid ' + color + '40', boxShadow: '0 2px 8px rgba(0,0,0,0.1)', minWidth: 140, textAlign: 'center' }}>
      <Handle type="target" position={Position.Top} style={{ background: color }} />
      <div style={{ fontSize: 24 }}>{icon}</div>
      <div style={{ fontWeight: 600, marginTop: 4, color: '#333' }}>{data.name}</div>
      <Tag color={color} style={{ marginTop: 4 }}>{data.type_name || data.type}</Tag>
      {data.status && <Tag color={data.status === 'active' ? 'green' : 'default'} style={{ marginTop: 2 }}>{data.status}</Tag>}
      <Handle type="source" position={Position.Bottom} style={{ background: color }} />
    </div>
  );
};

const nodeTypes: NodeTypes = { ciNode: CINode };

export default function TopologyPage() {
  const [loading, setLoading] = useState(false);
  const [selectedCI, setSelectedCI] = useState<number | null>(null);
  const [ciList, setCIList] = useState<{ id: number; name: string; type: string }[]>([]);
  const [depth, setDepth] = useState(2);
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [selectedNodeData, setSelectedNodeData] = useState<TopologyNode | null>(null);

  const loadCIList = useCallback(async () => {
    try {
      const result = await CMDBApi.getCIs({});
      const items = result.items || result.cis || [];
      setCIList(items.map((ci: any) => ({ id: ci.id, name: ci.name, type: ci.ci_type || ci.type })));
    } catch (error) { console.error('Failed to load CI list:', error); }
  }, []);

  const loadTopology = useCallback(async () => {
    if (!selectedCI) return;
    setLoading(true);
    try {
      const graph = await CIRelationshipAPI.getTopologyGraph(selectedCI, depth);
      const flowNodes: Node[] = graph.nodes.map((node, index) => ({
        id: String(node.id), type: 'ciNode', position: { x: 250 + (index % 4) * 200, y: 100 + Math.floor(index / 4) * 150 }, data: { ...node }
      }));
      const flowEdges: Edge[] = graph.edges.map(edge => ({
        id: 'e' + edge.id, source: String(edge.source), target: String(edge.target), label: edge.relationship_label, type: 'smoothstep',
        animated: edge.impact_level === 'critical', style: { stroke: getEdgeColor(edge.strength) },
        markerEnd: { type: MarkerType.ArrowClosed, color: getEdgeColor(edge.strength) }
      }));
      setNodes(flowNodes); setEdges(flowEdges);
    } catch (error) { console.error('Failed to load topology:', error); message.error('加载拓扑图失败'); setNodes([]); setEdges([]); }
    finally { setLoading(false); }
  }, [selectedCI, depth, setNodes, setEdges]);

  React.useEffect(() => { loadCIList(); }, [loadCIList]);
  React.useEffect(() => { if (selectedCI) loadTopology(); }, [selectedCI, depth, loadTopology]);

  const onNodeClick = useCallback((_: any, node: Node) => { setSelectedNodeData(node.data as TopologyNode); setDrawerVisible(true); }, []);

  return (
    <PageContainer title="CMDB 拓扑图" description="可视化配置项之间的依赖关系">
      <Card className="shadow-sm rounded-lg mb-4">
        <Space wrap>
          <Select placeholder="选择根配置项" showSearch style={{ width: 300 }} value={selectedCI} onChange={setSelectedCI}
            options={ciList.map(ci => ({ value: ci.id, label: ci.name + ' (' + ci.type + ')' }))} allowClear />
          <Select placeholder="关系深度" value={depth} onChange={setDepth} style={{ width: 120 }}
            options={[{ value: 1, label: '1 层' }, { value: 2, label: '2 层' }, { value: 3, label: '3 层' }, { value: 4, label: '4 层' }]} />
          <Button icon={<ReloadOutlined />} onClick={loadTopology} loading={loading} disabled={!selectedCI}>刷新</Button>
        </Space>
      </Card>
      <Card className="shadow-sm rounded-lg" style={{ height: 'calc(100vh - 250px)' }}>
        {loading ? <div style={{ textAlign: 'center', padding: 100 }}><Spin size="large" /><div style={{ marginTop: 16 }}>加载拓扑图中...</div></div>
        : selectedCI ? <div style={{ height: '100%' }}><ReactFlow nodes={nodes} edges={edges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} onNodeClick={onNodeClick}
            nodeTypes={nodeTypes} fitView fitViewOptions={{ padding: 0.2 }} attributionPosition="bottom-left"><Controls /><Background variant={BackgroundVariant.Dots} gap={20} size={1} /></ReactFlow></div>
        : <Empty description="请选择一个配置项查看其拓扑关系" style={{ paddingTop: 100 }} />}
      </Card>
      <Drawer title="配置项详情" placement="right" width={400} open={drawerVisible} onClose={() => setDrawerVisible(false)}>
        {selectedNodeData && <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="名称">{selectedNodeData.name}</Descriptions.Item>
          <Descriptions.Item label="类型">{selectedNodeData.type_name}</Descriptions.Item>
          <Descriptions.Item label="状态"><Tag color={selectedNodeData.status === 'active' ? 'green' : 'default'}>{selectedNodeData.status}</Tag></Descriptions.Item>
          <Descriptions.Item label="关键程度"><Tag color={selectedNodeData.criticality === 'critical' ? 'red' : 'default'}>{selectedNodeData.criticality}</Tag></Descriptions.Item>
        </Descriptions>}
      </Drawer>
    </PageContainer>
  );
}

function getEdgeColor(strength: string): string {
  switch (strength) { case 'critical': return '#f5222d'; case 'high': return '#fa8c16'; case 'medium': return '#1890ff'; case 'low': return '#8c8c8c'; default: return '#8c8c8c'; }
}
