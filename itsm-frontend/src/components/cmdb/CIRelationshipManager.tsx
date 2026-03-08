'use client';

/**
 * CI关系管理组件
 */

import React, { useState, useEffect, useCallback } from 'react';
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
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  LinkOutlined,
  NodeIndexOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';

import {
  CIRelationshipAPI,
  type CIRelationship,
  type CIRelationshipType,
  type RelationshipTypeInfo,
  type CreateRelationshipRequest,
} from '@/lib/api/cmdb-relationship';

const { Text, Title } = Typography;
const { Option } = Select;
const { TextArea } = Input;

// 关系类型中文映射
const relationshipTypeLabels: Record<string, string> = {
  depends_on: '依赖',
  hosts: '托管',
  hosted_on: '所属',
  connects_to: '连接',
  runs_on: '运行',
  contains: '包含',
  part_of: '组成',
  impacts: '影响',
  impacted_by: '受影响于',
  owns: '拥有',
  owned_by: '被拥有',
  uses: '使用',
  used_by: '被使用',
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
      setOutgoingRelations(data.outgoing_relations);
      setIncomingRelations(data.incoming_relations);
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

  // 创建关系
  const handleCreate = async (values: any) => {
    try {
      const targetCiId = values.target_ci_id;
      delete values.target_ci_id;

      const data: CreateRelationshipRequest = {
        source_ci_id: createType === 'outgoing' ? ciId : targetCiId,
        target_ci_id: createType === 'outgoing' ? targetCiId : ciId,
        relationship_type: values.relationship_type,
        strength: values.strength,
        impact_level: values.impact_level,
        description: values.description,
      };

      await CIRelationshipAPI.createRelationship(data);
      message.success('关系创建成功');
      setCreateModalOpen(false);
      form.resetFields();
      loadRelations();
      onRefresh?.();
    } catch (error) {
      message.error('创建关系失败');
    }
  };

  // 删除关系
  const handleDelete = async (relationId: number) => {
    try {
      await CIRelationshipAPI.deleteRelationship(relationId);
      message.success('关系已删除');
      loadRelations();
      onRefresh?.();
    } catch (error) {
      message.error('删除关系失败');
    }
  };

  // 表格列定义
  const columns = [
    {
      title: '关系类型',
      dataIndex: 'relationship_type_name',
      key: 'relationship_type',
      width: 100,
    },
    {
      title: '关联CI',
      key: 'related_ci',
      render: (_: unknown, record: CIRelationship) => (
        <Space orientation="vertical" size={0}>
          <Text strong>
            {createType === 'outgoing' ? record.target_ci_name : record.source_ci_name}
          </Text>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {createType === 'outgoing' ? record.target_ci_type : record.source_ci_type}
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
      dataIndex: 'impact_level',
      key: 'impact_level',
      width: 80,
      render: (level: string) => {
        const config = strengthLabels[level] || strengthLabels.low;
        return <Badge color={config.color} text={config.label} />;
      },
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      width: 80,
      render: (active: boolean) => (
        <Badge status={active ? 'success' : 'default'} text={active ? '启用' : '禁用'} />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, record: CIRelationship) => (
        <Space>
          <Tooltip title="删除">
            <Popconfirm
              title="确认删除"
              description="确定要删除此关系吗？"
              onConfirm={() => handleDelete(record.id)}
            >
              <Button type="text" danger icon={<DeleteOutlined />} size="small" />
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
          <LinkOutlined /> 出向关系 ({outgoingRelations.length})
        </span>
      ),
      children: (
        <Table
          columns={columns}
          dataSource={outgoingRelations}
          rowKey="id"
          scroll={{ x: 'max-content' }}
          pagination={false}
          size="small"
          loading={loading}
        />
      ),
    },
    {
      key: 'incoming',
      label: (
        <span>
          <NodeIndexOutlined /> 入向关系 ({incomingRelations.length})
        </span>
      ),
      children: (
        <Table
          columns={columns}
          dataSource={incomingRelations}
          rowKey="id"
          scroll={{ x: 'max-content' }}
          pagination={false}
          size="small"
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
            <LinkOutlined />
            <span>CI关系管理</span>
            <Text type="secondary">({ciName})</Text>
          </Space>
        }
        extra={
          <Space>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => handleOpenCreate('outgoing')}
            >
              添加出向关系
            </Button>
            <Button icon={<PlusOutlined />} onClick={() => handleOpenCreate('incoming')}>
              添加入向关系
            </Button>
          </Space>
        }
      >
        <Tabs items={tabItems} defaultActiveKey="outgoing" />
      </Card>

      {/* 创建关系模态框 */}
      <Modal
        title={createType === 'outgoing' ? '添加出向关系' : '添加入向关系'}
        open={createModalOpen}
        onCancel={() => setCreateModalOpen(false)}
        footer={null}
        width={500}
      >
        <Form form={form} layout="vertical" onFinish={handleCreate}>
          <Form.Item
            name="target_ci_id"
            label={createType === 'outgoing' ? '目标CI' : '源CI'}
            rules={[{ required: true, message: '请选择CI' }]}
          >
            <Select
              showSearch
              placeholder="搜索CI名称"
              optionFilterProp="children"
              onSearch={loadAvailableCIs}
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
            name="relationship_type"
            label="关系类型"
            rules={[{ required: true, message: '请选择关系类型' }]}
          >
            <Select placeholder="选择关系类型" style={{ width: '100%' }}>
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

          <Space style={{ width: '100%' }} size="large">
            <Form.Item name="strength" label="关系强度" style={{ width: '50%' }}>
              <Select placeholder="选择强度" defaultValue="medium">
                <Option value="critical">关键</Option>
                <Option value="high">高</Option>
                <Option value="medium">中</Option>
                <Option value="low">低</Option>
              </Select>
            </Form.Item>

            <Form.Item name="impact_level" label="影响程度" style={{ width: '50%' }}>
              <Select placeholder="选择程度" defaultValue="medium">
                <Option value="critical">致命</Option>
                <Option value="high">高</Option>
                <Option value="medium">中</Option>
                <Option value="low">低</Option>
              </Select>
            </Form.Item>
          </Space>

          <Form.Item name="description" label="描述">
            <TextArea rows={3} placeholder="输入关系描述（可选）" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setCreateModalOpen(false)}>取消</Button>
              <Button type="primary" htmlType="submit">
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
