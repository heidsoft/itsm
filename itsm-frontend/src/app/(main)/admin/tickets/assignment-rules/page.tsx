'use client';

import React, { useEffect, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Delete, Edit, FlaskConical, Plus, RefreshCw } from 'lucide-react';
import type {
  ActionConfig,
  AssignmentRule,
  ConditionConfig,
  CreateAssignmentRuleRequest,
  UpdateAssignmentRuleRequest} from '@/lib/api/ticket-assignment-api';
import {
  TicketAssignmentApi
} from '@/lib/api/ticket-assignment-api';

const { Title, Text } = Typography;
const { TextArea } = Input;

const DEFAULT_CONDITIONS = JSON.stringify(
  [{ field: 'priority', operator: 'equals', value: 'high' }],
  null,
  2
);
const DEFAULT_ACTION = JSON.stringify({ type: 'user', value: 1 }, null, 2);

interface RuleFormValues {
  name: string;
  description?: string;
  priority?: number;
  isActive?: boolean;
  conditions: string;
  actions: string;
}

interface TestFormValues {
  ruleId: number;
  ticketId: number;
}

function formatDate(value?: string) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN', { hour12: false });
}

function parseJsonField<T>(value: string, fallback: T, fieldName: string): T {
  if (!value?.trim()) return fallback;
  try {
    return JSON.parse(value) as T;
  } catch {
    throw new Error(`${fieldName} 不是合法 JSON`);
  }
}

function stringifyJson(value: unknown, fallback: string) {
  if (value === undefined || value === null) return fallback;
  return JSON.stringify(value, null, 2);
}

export default function AssignmentRulesPage() {
  const [rules, setRules] = useState<AssignmentRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [ruleModalOpen, setRuleModalOpen] = useState(false);
  const [testModalOpen, setTestModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<AssignmentRule | null>(null);
  const [testResult, setTestResult] = useState<string>('');
  const [form] = Form.useForm<RuleFormValues>();
  const [testForm] = Form.useForm<TestFormValues>();

  const loadRules = async () => {
    setLoading(true);
    try {
      const response = await TicketAssignmentApi.listRules();
      setRules(response.rules || []);
    } catch {
      message.error('加载分配规则失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRules();
  }, []);

  const openCreateModal = () => {
    setEditingRule(null);
    form.setFieldsValue({
      name: '',
      description: '',
      priority: 1,
      isActive: true,
      conditions: DEFAULT_CONDITIONS,
      actions: DEFAULT_ACTION,
    });
    setRuleModalOpen(true);
  };

  const openEditModal = (rule: AssignmentRule) => {
    setEditingRule(rule);
    form.setFieldsValue({
      name: rule.name,
      description: rule.description,
      priority: rule.priority,
      isActive: rule.isActive,
      conditions: stringifyJson(rule.conditions, DEFAULT_CONDITIONS),
      actions: stringifyJson(rule.actions, DEFAULT_ACTION),
    });
    setRuleModalOpen(true);
  };

  const closeRuleModal = () => {
    setRuleModalOpen(false);
    setEditingRule(null);
  };

  const saveRule = async () => {
    try {
      const values = await form.validateFields();
      const conditions = parseJsonField<ConditionConfig[]>(values.conditions, [], '匹配条件');
      const actions = parseJsonField<ActionConfig>(values.actions, { type: 'user', value: 1 }, '分配动作');
      const payload: CreateAssignmentRuleRequest | UpdateAssignmentRuleRequest = {
        name: values.name,
        description: values.description,
        priority: values.priority || 1,
        isActive: values.isActive ?? true,
        conditions,
        actions,
      };

      setSaving(true);
      if (editingRule) {
        await TicketAssignmentApi.updateRule(editingRule.id, payload);
        message.success('分配规则已更新');
      } else {
        await TicketAssignmentApi.createRule(payload as CreateAssignmentRuleRequest);
        message.success('分配规则已创建');
      }
      closeRuleModal();
      await loadRules();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message);
      }
    } finally {
      setSaving(false);
    }
  };

  const deleteRule = async (id: number) => {
    try {
      await TicketAssignmentApi.deleteRule(id);
      message.success('分配规则已删除');
      await loadRules();
    } catch {
      message.error('删除分配规则失败');
    }
  };

  const toggleRule = async (rule: AssignmentRule) => {
    try {
      await TicketAssignmentApi.updateRule(rule.id, { isActive: !rule.isActive });
      message.success(rule.isActive ? '规则已禁用' : '规则已启用');
      await loadRules();
    } catch {
      message.error('切换规则状态失败');
    }
  };

  const openTestModal = (rule: AssignmentRule) => {
    setTestResult('');
    testForm.resetFields();
    (testForm as any).setFieldValue('rule_id', rule.id);
    setTestModalOpen(true);
  };

  const testRule = async () => {
    try {
      const values = await testForm.validateFields();
      setTesting(true);
      const response = await TicketAssignmentApi.testRule({
        ruleId: values.ruleId,
        ticketId: values.ticketId,
      });
      setTestResult(
        response.matched
          ? `匹配成功${response.assignedTo ? `，推荐分配给用户 ID ${response.assignedTo}` : ''}。${response.reason || ''}`
          : `未匹配。${response.reason || ''}`
      );
    } catch {
      message.error('测试分配规则失败');
    } finally {
      setTesting(false);
    }
  };

  const columns: ColumnsType<AssignmentRule> = [
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record) => (
        <div>
          <Text strong>{name}</Text>
          <div className="text-xs text-gray-500">{record.description || '未填写描述'}</div>
        </div>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (priority: number) => (
        <Tag color={priority >= 8 ? 'red' : priority >= 4 ? 'orange' : 'blue'}>{priority}</Tag>
      ),
    },
    {
      title: '动作',
      dataIndex: 'actions',
      key: 'actions',
      width: 150,
      render: (actions: ActionConfig) => {
        const actionText =
          actions?.type === 'user' ? `用户 ${actions.value ?? '-'}` : actions?.type || '-';
        return <Tag>{actionText}</Tag>;
      },
    },
    {
      title: '执行次数',
      dataIndex:'executionCount',
      key:'executionCount',
      width: 100,
      render: (count: number) => count || 0,
    },
    {
      title: '最近执行',
      dataIndex:'lastExecutedAt',
      key:'lastExecutedAt',
      width: 180,
      render: formatDate,
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 110,
      render: (active: boolean, record) => (
        <Switch
          checked={active}
          checkedChildren="启用"
          unCheckedChildren="禁用"
          onChange={() => toggleRule(record)}
        />
      ),
    },
    {
      title: '操作',
      key: 'operations',
      width: 180,
      render: (_, record) => (
        <div className="flex items-center gap-2">
          <Tooltip title="测试">
            <Button size="small" icon={<FlaskConical size={14} />} onClick={() => openTestModal(record)} />
          </Tooltip>
          <Tooltip title="编辑">
            <Button size="small" icon={<Edit size={14} />} onClick={() => openEditModal(record)} />
          </Tooltip>
          <Popconfirm
            title="确认删除该分配规则？"
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            onConfirm={() => deleteRule(record.id)}
          >
            <Tooltip title="删除">
              <Button size="small" danger icon={<Delete size={14} />} />
            </Tooltip>
          </Popconfirm>
        </div>
      ),
    },
  ];

  return (
    <div className="p-6">
      <Card>
        <div className="mb-4 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <Title level={3} style={{ marginBottom: 4 }}>
              工单分配规则
            </Title>
            <Text type="secondary">按条件自动选择处理人，支撑工单进入可运营的分派流程。</Text>
          </div>
          <div className="flex items-center gap-2">
            <Tooltip title="刷新">
              <Button icon={<RefreshCw size={16} />} onClick={loadRules} />
            </Tooltip>
            <Button type="primary" icon={<Plus size={16} />} onClick={openCreateModal}>
              创建规则
            </Button>
          </div>
        </div>

        {rules.length === 0 && !loading ? (
          <Alert
            type="info"
            showIcon
            message="暂无分配规则"
            description="创建规则后，工单可按优先级、状态、分类或部门等条件自动推荐处理人。"
          />
        ) : (
          <Table
            rowKey="id"
            columns={columns}
            dataSource={rules}
            loading={loading}
            pagination={{ pageSize: 10, showTotal: total => `共 ${total} 条` }}
          />
        )}
      </Card>

      <Modal
        title={editingRule ? '编辑分配规则' : '创建分配规则'}
        open={ruleModalOpen}
        onOk={saveRule}
        onCancel={closeRuleModal}
        confirmLoading={saving}
        okText="保存"
        cancelText="取消"
        width={720}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="规则名称" rules={[{ required: true, message: '请输入规则名称' }]}>
            <Input placeholder="例如：高优先级工单分配给一线负责人" />
          </Form.Item>
          <Form.Item name="description" label="规则描述">
            <TextArea rows={3} placeholder="说明该规则适用的业务场景" />
          </Form.Item>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <Form.Item name="priority" label="优先级" tooltip="数字越大越先执行">
              <InputNumber min={1} max={100} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item name="is_active" label="启用状态" valuePropName="checked">
              <Switch checkedChildren="启用" unCheckedChildren="禁用" />
            </Form.Item>
          </div>
          <Form.Item
            name="conditions"
            label="匹配条件 JSON"
            rules={[{ required: true, message: '请输入匹配条件 JSON' }]}
          >
            <TextArea rows={7} spellCheck={false} />
          </Form.Item>
          <Form.Item
            name="actions"
            label="分配动作 JSON"
            rules={[{ required: true, message: '请输入分配动作 JSON' }]}
          >
            <TextArea rows={5} spellCheck={false} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="测试分配规则"
        open={testModalOpen}
        onOk={testRule}
        onCancel={() => setTestModalOpen(false)}
        confirmLoading={testing}
        okText="执行测试"
        cancelText="关闭"
        destroyOnHidden
      >
        <Form form={testForm} layout="vertical">
          <Form.Item name="ruleId" label="规则 ID" rules={[{ required: true, message: '请选择规则' }]}>
            <InputNumber disabled style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="ticketId" label="工单 ID" rules={[{ required: true, message: '请输入工单 ID' }]}>
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
        {testResult ? <Alert className="mt-3" type="success" showIcon message={testResult} /> : null}
      </Modal>
    </div>
  );
}
