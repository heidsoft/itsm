'use client';

/**
 * CI关系管理组件
 */

import React, { useState, useEffect, useCallback, useRef } from 'react';
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
  message,
  Popconfirm,
  Typography,
  Badge,
  Tabs,
  Row,
  Col,
} from 'antd';
import { Plus, Trash2, Eye, Link, Network } from 'lucide-react';
import dayjs from 'dayjs';

import {
  CIRelationshipAPI,
  type CIRelationship,
  type CIRelationshipType,
  type RelationshipTypeInfo,
  type CreateRelationshipRequest,
  type TopologyEdge,
} from '@/lib/api/cmdb-relationship';

const { Text, Title } = Typography;
const { Option } = Select;
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

// 关系类型中文映射
const relationshipTypeLabels: Record<string, string> = {
  dependsOn: '依赖',
  hosts: '托管',
  hostedOn: '所属',
  connectsTo: '连接',
  runsOn: '运行',
  contains: '包含',
  partOf: '组成',
  impacts: '影响',
  impactedBy: '受影响于',
  owns: '拥有',
  ownedBy: '被拥有',
  uses: '使用',
  usedBy: '被使用',
};

// 关系强度标签
const strengthLabels: Record<string, { color: string; label: string }> = {
  critical: { color: 'red', label: '关键' },
  high: { color: 'orange', label: '高' },
  medium: { color: 'blue', label: '中' },
  low: { color: 'default', label: '低' },
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
  const [loading, setLoading] = useState(true);
  const [outgoingRelations, setOutgoingRelations] = useState<CIRelationship[]>([]);
  const [incomingRelations, setIncomingRelations] = useState<CIRelationship[]>([]);
  const [relationshipTypes, setRelationshipTypes] = useState<RelationshipTypeInfo[]>([]);

  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createType, setCreateType] = useState<'outgoing' | 'incoming'>('outgoing');
  const [form] = Form.useForm();

  const [availableCIs, setAvailableCIs] = useState<{ id: number; name: string; type: string }[]>(
    []
  );
  const [creating, setCreating] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(
    () => () => {
      if (searchTimerRef.current) {
        clearTimeout(searchTimerRef.current);
      }
    },
    []
  );

  // 加载关系类型
  useEffect(() => {
    const loadTypes = async () => {
      try {
        const types = await CIRelationshipAPI.getRelationshipTypes();
        setRelationshipTypes(types);
      } catch (error) {
        message.error('加载关系类型失败');
      }
    };
    loadTypes();
  }, []);

  // 加载关系数据
  const loadRelations = useCallback(async () => {
    setLoading(true);
    try {
      const data = await CIRelationshipAPI.getCIRelationships(ciId, {
        includeOutgoing: true,
        includeIncoming: true,
        activeOnly: false,
      });
      setOutgoingRelations(data.outgoingRelations);
      setIncomingRelations(data.incomingRelations);
    } catch (error) {
      message.error('加载关系失败');
    } finally {
      setLoading(false);
    }
  }, [ciId]);

  useEffect(() => {
    loadRelations();
  }, [loadRelations]);

  // 打开创建模态框
  const handleOpenCreate = (type: 'outgoing' | 'incoming') => {
    setCreateType(type);
    setCreateModalOpen(true);
    form.resetFields();

    // 加载可用CI列表
    loadAvailableCIs();
  };

  // 加载可用CI列表
  const loadAvailableCIs = async (search?: string) => {
    const isMounted = true;
    try {
      const cIs = await CIRelationshipAPI.getAvailableCIs(ciId, search);
      if (isMounted) {
        setAvailableCIs(cIs.map(c => ({ id: c.id, name: c.name, type: c.type })));
      }
    } catch (error) {
      if (isMounted) {
        message.error('加载可用CI失败，请稍后重试');
      }
    }
  };

  // CI 搜索防抖（300ms）
  const handleSearchAvailableCIs = (search: string) => {
    if (searchTimerRef.current) {
      clearTimeout(searchTimerRef.current);
    }
    searchTimerRef.current = setTimeout(() => {
      loadAvailableCIs(search);
    }, 300);
  };

  // 创建关系
  const handleCreate = async (values: any) => {
    setCreating(true);
    try {
      const targetCiId = values.targetCiId;

      const sourceCiId = createType === 'outgoing' ? ciId : targetCiId;
      const destCiId = createType === 'outgoing' ? targetCiId : ciId;

      // 环检测：若已有关系图中 target 可达 source，新增 source→target 会成环
      try {
        const graph = await CIRelationshipAPI.getTopologyGraph(ciId, 5);
        if (hasPath(graph.edges || [], destCiId, sourceCiId)) {
          message.error('无法创建关系：该关系会与现有关系形成循环依赖');
          return;
        }
      } catch {
        // 拓扑接口不可用时不阻塞创建，由后端兜底校验
      }

      const data: CreateRelationshipRequest = {
        sourceCiId,
        targetCiId: destCiId,
        relationshipType: values.relationshipType,
        strength: values.strength,
        impactLevel: values.impactLevel,
        description: values.description,
      };

      await CIRelationshipAPI.createRelationship(data);
      message.success('关系创建成功');
      setCreateModalOpen(false);
      form.resetFields();
      loadRelations();
      onRefresh?.();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '创建关系失败');
    } finally {
      setCreating(false);
    }
  };

  // 删除关系
  const handleDelete = async (relationId: number) => {
    if (deletingId !== null) return;
    setDeletingId(relationId);
    try {
      await CIRelationshipAPI.deleteRelationship(relationId);
      message.success('关系已删除');
      loadRelations();
      onRefresh?.();
    } catch (error) {
      message.error('删除关系失败');
    } finally {
      setDeletingId(null);
    }
  };

  // 表格列定义
  const columns = [
    {
      title: '关系类型',
      dataIndex: 'relationshipTypeName',
      key: 'relationshipType',
      width: 100,
    },
    {
      title: '关联CI',
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
      title: '强度',
      dataIndex: 'strength',
      key: 'strength',
      width: 80,
      render: (strength: string) => {
        const config = strengthLabels[strength] || strengthLabels.low;
        return <Tag color={config.color}>{config.label}</Tag>;
      },
    },
    {
      title: '影响',
      dataIndex: 'impactLevel',
      key: 'impactLevel',
      width: 80,
      render: (level: string) => {
        const config = strengthLabels[level] || strengthLabels.low;
        return <Badge color={config.color} text={config.label} />;
      },
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 80,
      render: (active: boolean) => (
        <Badge status={active ? 'success' : 'default'} text={active ? '启用' : '禁用'} />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, record: CIRelationship) => (
        <Space>
          <Tooltip title='删除'>
            <Popconfirm
              title='确认删除'
              description='确定要删除此关系吗？'
              onConfirm={() => handleDelete(record.id)}
            >
              <Button
                type='text'
                danger
                icon={<Trash2 />}
                size='small'
                loading={deletingId === record.id}
                aria-label='删除关系'
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
          <Link /> 出向关系 ({outgoingRelations.length})
        </span>
      ),
      children: (
        <Table
          columns={columns}
          dataSource={outgoingRelations}
          rowKey='id'
          scroll={{ x: 'max-content' }}
          pagination={false}
          size='small'
          loading={loading}
        />
      ),
    },
    {
      key: 'incoming',
      label: (
        <span>
          <Network /> 入向关系 ({incomingRelations.length})
        </span>
      ),
      children: (
        <Table
          columns={columns}
          dataSource={incomingRelations}
          rowKey='id'
          scroll={{ x: 'max-content' }}
          pagination={false}
          size='small'
          loading={loading}
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
            <span>CI关系管理</span>
            <Text type='secondary'>({ciName})</Text>
          </Space>
        }
        extra={
          <Space>
            <Button type='primary' icon={<Plus />} onClick={() => handleOpenCreate('outgoing')}>
              添加出向关系
            </Button>
            <Button icon={<Plus />} onClick={() => handleOpenCreate('incoming')}>
              添加入向关系
            </Button>
          </Space>
        }
      >
        <Tabs items={tabItems} defaultActiveKey='outgoing' />
      </Card>

      {/* 创建关系模态框 */}
      <Modal
        title={createType === 'outgoing' ? '添加出向关系' : '添加入向关系'}
        open={createModalOpen}
        onCancel={() => setCreateModalOpen(false)}
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
            label={createType === 'outgoing' ? '目标CI' : '源CI'}
            rules={[{ required: true, message: '请选择CI' }]}
          >
            <Select
              showSearch
              placeholder='搜索CI名称'
              optionFilterProp='children'
              onSearch={handleSearchAvailableCIs}
              style={{ width: '100%' }}
            >
              {availableCIs.map(ci => (
                <Option key={ci.id} value={ci.id}>
                  <Space>
                    <span>{ci.name}</span>
                    <Tag>{ci.type}</Tag>
                  </Space>
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name='relationshipType'
            label='关系类型'
            rules={[{ required: true, message: '请选择关系类型' }]}
          >
            <Select placeholder='选择关系类型' style={{ width: '100%' }}>
              {relationshipTypes.map(type => (
                <Option key={type.type} value={type.type}>
                  <Tooltip title={type.description}>
                    <Space>
                      <span>{type.name}</span>
                      <Tag>{type.direction === 'bi-directional' ? '双向' : '单向'}</Tag>
                    </Space>
                  </Tooltip>
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name='strength' label='关系强度'>
                <Select placeholder='选择强度' style={{ width: '100%' }}>
                  <Option value='critical'>关键</Option>
                  <Option value='high'>高</Option>
                  <Option value='medium'>中</Option>
                  <Option value='low'>低</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name='impactLevel' label='影响程度'>
                <Select placeholder='选择程度' style={{ width: '100%' }}>
                  <Option value='critical'>致命</Option>
                  <Option value='high'>高</Option>
                  <Option value='medium'>中</Option>
                  <Option value='low'>低</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name='description' label='描述'>
            <TextArea rows={3} placeholder='输入关系描述（可选）' />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setCreateModalOpen(false)}>取消</Button>
              <Button type='primary' htmlType='submit' loading={creating}>
                创建
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default CIRelationshipManager;
