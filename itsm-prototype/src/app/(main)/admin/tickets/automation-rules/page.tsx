/**
 * 工单自动化规则管理页面
 * 提供自动化规则的创建、编辑、删除、测试功能
 */

'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Modal,
  message,
  Popconfirm,
  Tag,
  Tooltip,
  Typography,
  Alert,
  Badge,
  Switch,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  ThunderboltOutlined,
  InfoCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  TicketAutomationRuleApi,
  AutomationRule,
  CreateAutomationRuleRequest,
  UpdateAutomationRuleRequest,
} from '@/lib/api/ticket-automation-rule-api';
import { AutomationRuleForm } from '@/components/business/AutomationRuleForm';

const { Title, Text } = Typography;

export default function AutomationRulesPage() {
  const [rules, setRules] = useState<AutomationRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState<AutomationRule | null>(null);

  // 加载规则列表
  const loadRules = async () => {
    setLoading(true);
    try {
      const response = await TicketAutomationRuleApi.listRules();
      setRules(response.rules || []);
    } catch (error) {
      console.error('Failed to load rules:', error);
      message.error('加载自动化规则失败');
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
    setModalVisible(true);
  };

  // 编辑规则
  const handleEdit = (rule: AutomationRule) => {
    setEditingRule(rule);
    setModalVisible(true);
  };

  // 保存规则
  const handleSave = async (values: any) => {
    try {
      if (editingRule) {
        const updateData: UpdateAutomationRuleRequest = {
          name: values.name,
          description: values.description,
          priority: values.priority,
          is_active: values.is_active,
          conditions: values.conditions,
          actions: values.actions,
        };
        await TicketAutomationRuleApi.updateRule(editingRule.id, updateData);
        message.success('规则更新成功');
      } else {
        const createData: CreateAutomationRuleRequest = {
          name: values.name,
          description: values.description,
          priority: values.priority,
          is_active: values.is_active,
          conditions: values.conditions,
          actions: values.actions,
        };
        await TicketAutomationRuleApi.createRule(createData);
        message.success('规则创建成功');
      }
      setModalVisible(false);
      setEditingRule(null);
      loadRules();
    } catch (error) {
      console.error('Failed to save rule:', error);
      message.error(editingRule ? '更新规则失败' : '创建规则失败');
    }
  };

  // 删除规则
  const handleDelete = async (ruleId: number) => {
    try {
      await TicketAutomationRuleApi.deleteRule(ruleId);
      message.success('规则删除成功');
      loadRules();
    } catch (error) {
      console.error('Failed to delete rule:', error);
      message.error('删除规则失败');
    }
  };

  // 切换规则状态
  const handleToggleStatus = async (rule: AutomationRule) => {
    try {
      await TicketAutomationRuleApi.updateRule(rule.id, {
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
  const columns: ColumnsType<AutomationRule> = [
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
      title: '条件数',
      key: 'conditions_count',
      width: 100,
      render: (_, record) => (
        <Badge count={record.conditions?.length || 0} showZero />
      ),
    },
    {
      title: '动作数',
      key: 'actions_count',
      width: 100,
      render: (_, record) => (
        <Badge count={record.actions?.length || 0} showZero />
      ),
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      width: 100,
      render: (isActive, record) => (
        <Switch
          checked={isActive}
          onChange={() => handleToggleStatus(record)}
          checkedChildren={<CheckCircleOutlined />}
          unCheckedChildren={<CloseCircleOutlined />}
        />
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
            <span>工单自动化规则管理</span>
          </Space>
        </Title>
        <Text type="secondary">
          配置自动化规则，系统将在工单创建或更新时自动执行匹配的规则
        </Text>
      </div>

      <Alert
        message="规则说明"
        description="自动化规则按优先级从高到低执行。当工单创建或更新时，系统会按优先级顺序检查规则，找到匹配的规则并执行相应的动作（如自动分类、自动分配、自动升级等）。"
        type="info"
        icon={<InfoCircleOutlined />}
        showIcon
        style={{ marginBottom: 16 }}
      />

      <Card
        title="自动化规则列表"
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
        title={editingRule ? '编辑自动化规则' : '创建自动化规则'}
        open={modalVisible}
        onCancel={() => {
          setModalVisible(false);
          setEditingRule(null);
        }}
        footer={null}
        width={1000}
        destroyOnClose
      >
        <AutomationRuleForm
          editingRule={editingRule}
          onSave={handleSave}
          onCancel={() => {
            setModalVisible(false);
            setEditingRule(null);
          }}
        />
      </Modal>
    </div>
  );
}

