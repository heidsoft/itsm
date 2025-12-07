/**
 * 工单分配规则管理页面
 * 提供分配规则的创建、编辑、删除、测试功能
 */

'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  InputNumber,
  Switch,
  message,
  Popconfirm,
  Tag,
  Tooltip,
  Typography,
  Alert,
  Tabs,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  ThunderboltOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  TicketAssignmentApi,
  AssignmentRule,
  CreateAssignmentRuleRequest,
  UpdateAssignmentRuleRequest,
} from '@/lib/api/ticket-assignment-api';
import { AssignmentRuleForm } from '@/components/business/AssignmentRuleForm';

const { Title, Text } = Typography;
const { TextArea } = Input;
const { TabPane } = Tabs;

export default function AssignmentRulesPage() {
  const [rules, setRules] = useState<AssignmentRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState<AssignmentRule | null>(null);
  const [form] = Form.useForm();

  // 加载规则列表
  const loadRules = async () => {
    setLoading(true);
    try {
      const response = await TicketAssignmentApi.listRules();
      setRules(response.rules || []);
    } catch (error) {
      console.error('Failed to load rules:', error);
      message.error('加载分配规则失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRules();
  }, []);

  // 创建规则
  const handleCreate = () => {
    setEditingRule(null);
    form.resetFields();
    form.setFieldsValue({
      priority: 0,
      is_active: true,
      conditions: [],
      actions: {},
    });
    setModalVisible(true);
  };

  // 编辑规则
  const handleEdit = (rule: AssignmentRule) => {
    setEditingRule(rule);
    form.setFieldsValue({
      name: rule.name,
      description: rule.description,
      priority: rule.priority,
      is_active: rule.is_active,
      conditions: rule.conditions,
      actions: rule.actions,
    });
    setModalVisible(true);
  };

  // 保存规则
  const handleSave = async (values: any) => {
    try {
      if (editingRule) {
        const updateData: UpdateAssignmentRuleRequest = {
          name: values.name,
          description: values.description,
          priority: values.priority,
          is_active: values.is_active,
          conditions: values.conditions,
          actions: values.actions,
        };
        await TicketAssignmentApi.updateRule(editingRule.id, updateData);
        message.success('规则更新成功');
      } else {
        const createData: CreateAssignmentRuleRequest = {
          name: values.name,
          description: values.description,
          priority: values.priority,
          is_active: values.is_active,
          conditions: values.conditions,
          actions: values.actions,
        };
        await TicketAssignmentApi.createRule(createData);
        message.success('规则创建成功');
      }
      setModalVisible(false);
      setEditingRule(null);
      form.resetFields();
      loadRules();
    } catch (error) {
      console.error('Failed to save rule:', error);
      message.error(editingRule ? '更新规则失败' : '创建规则失败');
    }
  };

  // 删除规则
  const handleDelete = async (ruleId: number) => {
    try {
      await TicketAssignmentApi.deleteRule(ruleId);
      message.success('规则删除成功');
      loadRules();
    } catch (error) {
      console.error('Failed to delete rule:', error);
      message.error('删除规则失败');
    }
  };

  // 切换规则状态
  const handleToggleStatus = async (rule: AssignmentRule) => {
    try {
      await TicketAssignmentApi.updateRule(rule.id, {
        is_active: !rule.is_active,
      });
      message.success(rule.is_active ? '规则已禁用' : '规则已启用');
      loadRules();
    } catch (error) {
      console.error('Failed to toggle rule status:', error);
      message.error('操作失败');
    }
  };

  // 表格列定义
  const columns: ColumnsType<AssignmentRule> = [
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      render: (text, record) => (
        <Space direction="vertical" size="small">
          <Text strong>{text}</Text>
          {record.description && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              {record.description}
            </Text>
          )}
        </Space>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      sorter: (a, b) => b.priority - a.priority,
      render: (priority) => (
        <Tag color={priority >= 10 ? 'red' : priority >= 5 ? 'orange' : 'blue'}>
          {priority}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      width: 100,
      render: (isActive) => (
        <Tag color={isActive ? 'success' : 'default'}>
          {isActive ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '执行次数',
      dataIndex: 'execution_count',
      key: 'execution_count',
      width: 120,
      render: (count) => <Text>{count || 0}</Text>,
    },
    {
      title: '最后执行',
      dataIndex: 'last_executed_at',
      key: 'last_executed_at',
      width: 180,
      render: (time) => (time ? new Date(time).toLocaleString('zh-CN') : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_, record) => (
        <Space>
          <Tooltip title="编辑">
            <Button
              type="link"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            />
          </Tooltip>
          <Tooltip title={record.is_active ? '禁用' : '启用'}>
            <Button
              type="link"
              onClick={() => handleToggleStatus(record)}
            >
              {record.is_active ? '禁用' : '启用'}
            </Button>
          </Tooltip>
          <Popconfirm
            title="确认删除"
            description={`确定要删除规则"${record.name}"吗？`}
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除">
              <Button type="link" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      <div className="mb-6">
        <Title level={2}>
          <Space>
            <ThunderboltOutlined style={{ color: '#1890ff' }} />
            <span>工单分配规则管理</span>
          </Space>
        </Title>
        <Text type="secondary">
          配置自动分配规则，系统将根据规则自动分配工单给合适的处理人
        </Text>
      </div>

      <Alert
        message="规则说明"
        description="分配规则按优先级从高到低执行。当工单创建或更新时，系统会按优先级顺序检查规则，找到第一个匹配的规则并执行分配动作。"
        type="info"
        icon={<InfoCircleOutlined />}
        showIcon
        style={{ marginBottom: 16 }}
      />

      <Card
        title="分配规则列表"
        extra={
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={handleCreate}
          >
            创建规则
          </Button>
        }
      >
        <Table
          columns={columns}
          dataSource={rules}
          rowKey="id"
          loading={loading}
          pagination={{
            pageSize: 20,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条规则`,
          }}
        />
      </Card>

      {/* 创建/编辑规则模态框 */}
      <Modal
        title={editingRule ? '编辑分配规则' : '创建分配规则'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingRule(null);
          form.resetFields();
        }}
        footer={null}
        width={900}
        destroyOnHidden
      >
        <AssignmentRuleForm
          form={form}
          editingRule={editingRule}
          onSave={handleSave}
          onCancel={() => {
            setModalVisible(false);
            setEditingRule(null);
            form.resetFields();
          }}
        />
      </Modal>
    </div>
  );
}

