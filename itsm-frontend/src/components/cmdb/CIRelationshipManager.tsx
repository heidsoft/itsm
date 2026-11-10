'use client';

/**
 * CI 关系管理组件
 * - 关系类型、出/入向关系、候选 CI、拓扑图都由 React Query 驱动
 * - 创建/删除走 useCreateRelationshipMutation / useDeleteRelationshipMutation，
 *   onSuccess 自动 invalidate 出/入向查询
 * - 拓扑环检测用 useTopologyGraphQuery 按需触发，不在挂载时拉
 * - 可选 CI 候选用 useAvailableCIsQuery，搜索值变化即改变 queryKey
 */

import React, { useState } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Tag,
  Modal,
  Form,
  Select,
  Input,
  Tooltip,
  App,
  Popconfirm,
  Typography,
  Badge,
  Tabs,
  Row,
  Col,
} from 'antd';
import { Plus, Trash2, Link, Network } from 'lucide-react';
import dayjs from 'dayjs';

import { type CIRelationship, type CIRelationshipType, type TopologyEdge } from '@/lib/api/cmdb-relationship';
import {
  useRelationshipTypesV2Query,
  useCIRelationshipsListQuery,
  useAvailableCIsQuery,
  useTopologyGraphQuery,
  useCreateRelationshipMutation,
  useDeleteRelationshipMutation,
} from '@/lib/hooks/useCMDB';
import { useOntologyRelationshipVocabulary } from '@/lib/cmdb/use-relationship-vocabulary';

const { Text, Title } = Typography;
const { TextArea } = Input;

// 在已有关系边中判断 from 是否可达 to（DFS，带访问集防止死循环）
const hasPath = (edges: TopologyEdge[], from: number, to: number): boolean => {
  if (from === to) return true;
  const adjacency = new Map<number, number[]>();
  edges.forEach(edge => {
    const next = adjacency.get(edge.source) || [];
    next.push(edge.target);
    adjacency.set(edge.source, next);
  });
  const visited = new Set<number>();
  const stack = [from];
  while (stack.length > 0) {
    const current = stack.pop()!;
    if (current === to) return true;
    if (visited.has(current)) continue;
    visited.add(current);
    (adjacency.get(current) || []).forEach(next => {
      if (!visited.has(next)) stack.push(next);
    });
  }
  return false;
};

// 关系强度标签
const strengthLabels: Record<string, { color: string; label: string }> = {
  critical: { color: 'red', label: 'Critical' },
  high: { color: 'orange', label: 'High' },
  medium: { color: 'blue', label: 'Medium' },
  low: { color: 'default', label: 'Low' },
};

interface CIRelationshipManagerProps {
  ciId: number;
  ciName: string;
  onRefresh?: () => void;
}

const CIRelationshipManager: React.FC<CIRelationshipManagerProps> = ({
  ciId,
  ciName,
  onRefresh,
}) => {
  const { message } = App.useApp();
  // 把后端 ontology 关系词表写入 runtimeVocabulary（词表单一源）。
  // Select 渲染仍用 V2 端点（已有完整 name/description/direction），
  // 但 graph-engine 等其他模块读 relationshipLabel() 时拿到的是 ontology 权威值。
  // 返回值不直接使用——副作用即目的（注入运行时词表）。
  useOntologyRelationshipVocabulary();
  // React Query：关系类型（10 分钟缓存）
  const typesQuery = useRelationshipTypesV2Query();
  const relationshipTypes = typesQuery.data ?? [];

  // React Query：出/入向关系列表
  const relationsQuery = useCIRelationshipsListQuery(ciId, {
    includeOutgoing: true,
    includeIncoming: true,
    activeOnly: false,
  });
  const outgoingRelations: CIRelationship[] = relationsQuery.data?.outgoingRelations ?? [];
  const incomingRelations: CIRelationship[] = relationsQuery.data?.incomingRelations ?? [];

  // React Query：拓扑图（仅在创建模态打开时按需获取，避免空挂载）
  const [topologyEnabled, setTopologyEnabled] = useState(false);
  const topologyQuery = useTopologyGraphQuery(ciId, 5, topologyEnabled);

  // React Query mutations
  const createMutation = useCreateRelationshipMutation();
  const deleteMutation = useDeleteRelationshipMutation();

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createType, setCreateType] = useState<'outgoing' | 'incoming'>('outgoing');
  const [form] = Form.useForm();
  const [searchTerm, setSearchTerm] = useState('');
  const availableCIsQuery = useAvailableCIsQuery(ciId, searchTerm || undefined, createModalOpen);
  const availableCIs = (availableCIsQuery.data ?? []).map(c => ({
    id: c.id,
    name: c.name,
    type: c.type ?? '',
  }));

  // 打开创建模态框：激活按需拓扑查询；触发一次初始拉取
  const handleOpenCreate = (type: 'outgoing' | 'incoming') => {
    setCreateType(type);
    setCreateModalOpen(true);
    form.resetFields();
    setSearchTerm('');
    setTopologyEnabled(true);
  };

  // 关闭模态：清查询态以释放内存
  const handleCloseCreate = () => {
    setCreateModalOpen(false);
    setTopologyEnabled(false);
  };

  // 创建关系（环检测 → 提交 → 失败由 mutation onError 弹出）
  const handleCreate = async (values: {
    targetCiId: number;
    relationshipType: string;
    strength?: 'critical' | 'high' | 'medium' | 'low';
    impactLevel?: 'critical' | 'high' | 'medium' | 'low';
    description?: string;
  }) => {
    const targetCiId = values.targetCiId;
    const sourceCiId = createType === 'outgoing' ? ciId : targetCiId;
    const destCiId = createType === 'outgoing' ? targetCiId : ciId;

    // 环检测：若拓扑数据源可达，新增 source→target 会成环
    const graph = topologyQuery.data;
    if (graph?.edges && hasPath(graph.edges, destCiId, sourceCiId)) {
      message.error('Unable to create relationship: it would create a circular dependency on an existing relationship');
      return;
    }

    try {
      await createMutation.mutateAsync({
        sourceCiId,
        targetCiId: destCiId,
        relationshipType: values.relationshipType as CIRelationshipType,
        strength: values.strength,
        impactLevel: values.impactLevel,
        description: values.description,
      });
      handleCloseCreate();
      form.resetFields();
      onRefresh?.();
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : 'Failed to create relationship');
    }
  };

  const handleDelete = async (relationId: number) => {
    try {
      await deleteMutation.mutateAsync(relationId);
      onRefresh?.();
    } catch {
      // error 已由 mutation onError 展示
    }
  };

  const isLoading = relationsQuery.isLoading;

  // 表格列定义
  const columns = [
    {
      title: 'Relationship Type',
      dataIndex: 'relationshipTypeName',
      key: 'relationshipType',
      width: 100,
    },
    {
      title: 'Associated CI',
      key: 'relatedCi',
      render: (_: unknown, record: CIRelationship) => (
        <Space orientation='vertical' size={0}>
          <Text strong>
            {record.sourceCiId === ciId ? record.targetCiName : record.sourceCiName}
          </Text>
          <Text type='secondary' style={{ fontSize: 12 }}>
            {record.sourceCiId === ciId ? record.targetCiType : record.sourceCiType}
          </Text>
        </Space>
      ),
    },
    {
      title: 'Strength',
      dataIndex: 'strength',
      key: 'strength',
      width: 80,
      render: (strength: string) => {
        const config = strengthLabels[strength] || strengthLabels.low;
        return <Tag color={config.color}>{config.label}</Tag>;
      },
    },
    {
      title: 'Impact',
      dataIndex: 'impactLevel',
      key: 'impactLevel',
      width: 80,
      render: (level: string) => {
        const config = strengthLabels[level] || strengthLabels.low;
        return <Badge color={config.color} text={config.label} />;
      },
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 80,
      render: (active: boolean) => (
        <Badge status={active ? 'success' : 'default'} text={active ? 'Enabled' : 'Disabled'} />
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 100,
      render: (_: unknown, record: CIRelationship) => (
        <Space>
          <Tooltip title="Delete">
            <Popconfirm
              title="Confirm Delete"
              description="Are you sure you want to delete this relationship?"
              onConfirm={() => handleDelete(record.id)}
            >
              <Button
                type='text'
                danger
                icon={<Trash2 />}
                size='small'
                loading={deleteMutation.isPending && deleteMutation.variables === record.id}
                aria-label="Delete relationship"
              />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ];

  const tabItems = [
    {
      key: 'outgoing',
      label: (
        <span>
          <Link />  Outbound relationship ({outgoingRelations.length})
        </span>
      ),
      children: (
        <Table
          columns={columns}
          dataSource={outgoingRelations}
          rowKey='id'
          scroll={{ x: 'max-content' }}
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: total => `Total ${total} items` }}
          size='small'
          loading={isLoading}
        />
      ),
    },
    {
      key: 'incoming',
      label: (
        <span>
          <Network />  Incoming relationship ({incomingRelations.length})
        </span>
      ),
      children: (
        <Table
          columns={columns}
          dataSource={incomingRelations}
          rowKey='id'
          scroll={{ x: 'max-content' }}
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: total => `Total ${total} items` }}
          size='small'
          loading={isLoading}
        />
      ),
    },
  ];

  return (
    <>
      <Card
        title={
          <Space>
            <Link />
            <span>CI relationship management</span>
            <Text type='secondary'>({ciName})</Text>
          </Space>
        }
        extra={
          <Space>
            <Button type='primary' icon={<Plus />} onClick={() => handleOpenCreate('outgoing')}>

              Add outbound relationship
            </Button>
            <Button icon={<Plus />} onClick={() => handleOpenCreate('incoming')}>

              Add inbound relationship
            </Button>
          </Space>
        }
      >
        <Tabs items={tabItems} defaultActiveKey='outgoing' />
      </Card>

      {/* 创建关系模态框 */}
      <Modal
        title={createType === 'outgoing' ? 'Add outbound relationship' : 'Add inbound relationship'}
        open={createModalOpen}
        onCancel={handleCloseCreate}
        footer={null}
        width={500}
      >
        <Form
          form={form}
          layout='vertical'
          onFinish={handleCreate}
          initialValues={{ strength: 'medium', impactLevel: 'medium' }}
        >
          <Form.Item
            name='targetCiId'
            label={createType === 'outgoing' ? 'Target CI' : 'Source CI'}
            rules={[{ required: true, message: 'Please select CI' }]}
          >
            <Select
              showSearch
              placeholder="Search CI name"
              optionFilterProp='children'
              onSearch={setSearchTerm}
              filterOption={false}
              loading={availableCIsQuery.isFetching}
              style={{ width: '100%' }}
             options={availableCIs.map(ci => ({ value: ci.id, label: <Space>
                    <span>{ci.name}</span>
                    <Tag>{ci.type}</Tag>
                  </Space> }))} />
          </Form.Item>

          <Form.Item
            name='relationshipType'
            label="Relationship Type"
            rules={[{ required: true, message: 'Please select a relationship type' }]}
          >
            <Select placeholder="Select relationship type" style={{ width: '100%' }} options={relationshipTypes.map(type => ({ value: type.type, label: <Tooltip title={type.description ? type.description : undefined}>
                    <Space>
                      <span>{type.name}</span>
                      <Tag>{type.direction === 'bi-directional' ? 'Bidirectional' : 'One-way'}</Tag>
                    </Space>
                  </Tooltip> }))} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name='strength' label="Relationship Strength">
                <Select placeholder="Select intensity" style={{ width: '100%' }} options={[{ value: "critical", label: "Critical" }, { value: "high", label: "High" }, { value: "medium", label: "Medium" }, { value: "low", label: "Low" }]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='impactLevel' label="Impact Level">
                <Select placeholder="Degree of selection" style={{ width: '100%' }} options={[{ value: "critical", label: "Fatal" }, { value: "high", label: "High" }, { value: "medium", label: "Medium" }, { value: "low", label: "Low" }]} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name='description' label="Description">
            <TextArea rows={3} placeholder="Enter relationship description (optional)" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={handleCloseCreate}>Cancel</Button>
              <Button type='primary' htmlType='submit' loading={createMutation.isPending}>

                Create
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default CIRelationshipManager;
