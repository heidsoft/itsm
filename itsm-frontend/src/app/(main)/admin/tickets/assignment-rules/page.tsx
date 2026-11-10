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
  App,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Delete, Edit, FlaskConical, Plus, RefreshCw } from 'lucide-react';
import { UsageGuideCard } from '@/components/common/UsageGuideCard';
import { isPermissionDeniedError } from '@/lib/api/http-client';
import { formatDateTime } from '@/lib/formatters';
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
  return formatDateTime(value) || '-';
}

function parseJsonField<T>(value: string, fallback: T, fieldName: string): T {
  if (!value?.trim()) return fallback;
  try {
    return JSON.parse(value) as T;
  } catch {
    throw new Error(`${fieldName} Not valid JSON`);
  }
}

function stringifyJson(value: unknown, fallback: string) {
  if (value === undefined || value === null) return fallback;
  return JSON.stringify(value, null, 2);
}

export default function AssignmentRulesPage() {
  const { message } = App.useApp();
  const [rules, setRules] = useState<AssignmentRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<{ denied: boolean; detail: string } | null>(null);
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
    setLoadError(null);
    try {
      const response = await TicketAssignmentApi.listRules();
      setRules(response.items);
    } catch (error) {
      // 失败不能画成「暂无规则」：空表格会让用户以为系统里确实没有规则。
      setRules([]);
      setLoadError({
        denied: isPermissionDeniedError(error),
        detail: error instanceof Error ? error.message : '',
      });
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
      const conditions = parseJsonField<ConditionConfig[]>(values.conditions, [], 'matching condition');
      const actions = parseJsonField<ActionConfig>(values.actions, { type: 'user', value: 1 }, 'Assign action');
      const payload: CreateAssignmentRuleRequest | UpdateAssignmentRuleRequest = {
        name: values.name,
        description: values.description,
        priority: values.priority ?? 1,
        isActive: values.isActive ?? true,
        conditions,
        actions,
      };

      setSaving(true);
      if (editingRule) {
        await TicketAssignmentApi.updateRule(editingRule.id, payload);
        message.success('Assignment rules updated');
      } else {
        await TicketAssignmentApi.createRule(payload as CreateAssignmentRuleRequest);
        message.success('Assignment rule created');
      }
      closeRuleModal();
      await loadRules();
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message ? error.message : 'Failed to save allocation rules');
      }
    } finally {
      setSaving(false);
    }
  };

  const deleteRule = async (id: number) => {
    try {
      await TicketAssignmentApi.deleteRule(id);
      message.success('Assignment rule deleted');
      await loadRules();
    } catch {
      message.error('Failed to delete allocation rule');
    }
  };

  const toggleRule = async (rule: AssignmentRule) => {
    try {
      await TicketAssignmentApi.updateRule(rule.id, { isActive: !rule.isActive });
      message.success(rule.isActive ? 'Rule disabled' : 'Rule enabled');
      await loadRules();
    } catch {
      message.error('Failed to switch rule status');
    }
  };

  const openTestModal = (rule: AssignmentRule) => {
    setTestResult('');
    testForm.resetFields();
    (testForm as any).setFieldValue('ruleId', rule.id);
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
          ? `Match successful ${response.assignedTo ? `, recommended to be assigned to user ID ${response.assignedTo}` : ''}。${response.reason || ''}`
          : `No match.${response.reason || ''}`
      );
    } catch {
      message.error('Test assignment rule failed');
    } finally {
      setTesting(false);
    }
  };

  const columns: ColumnsType<AssignmentRule> = [
    {
      title: 'Rule Name',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record) => (
        <div>
          <Text strong>{name}</Text>
          <div className="text-xs text-gray-500">{record.description || 'No description filled in'}</div>
        </div>
      ),
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (priority: number) => (
        <Tag color={priority >= 8 ? 'red' : priority >= 4 ? 'orange' : 'blue'}>{priority}</Tag>
      ),
    },
    {
      title: 'Action',
      dataIndex: 'actions',
      key: 'actions',
      width: 150,
      render: (actions: ActionConfig) => {
        const actionText =
          actions?.type === 'user' ? `Users ${actions.value ?? '-'}` : actions?.type || '-';
        return <Tag>{actionText}</Tag>;
      },
    },
    {
      title: 'Number of executions',
      dataIndex:'executionCount',
      key:'executionCount',
      width: 100,
      render: (count: number) => count || 0,
    },
    {
      title: 'Recently executed',
      dataIndex:'lastExecutedAt',
      key:'lastExecutedAt',
      width: 180,
      render: formatDate,
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 110,
      render: (active: boolean, record) => (
        <Switch
          aria-label={`Switching rules ${record.name} enabled status`}
          checked={active}
          checkedChildren="Enabled"
          unCheckedChildren="Disabled"
          onChange={() => toggleRule(record)}
        />
      ),
    },
    {
      title: 'Actions',
      key: 'operations',
      width: 180,
      render: (_, record) => (
        <div className="flex items-center gap-2">
          <Tooltip title="Test">
            <Button aria-label="Test" size="small" icon={<FlaskConical size={14} />} onClick={() => openTestModal(record)} />
          </Tooltip>
          <Tooltip title="Edit">
            <Button aria-label="Edit" size="small" icon={<Edit size={14} />} onClick={() => openEditModal(record)} />
          </Tooltip>
          <Popconfirm
            title="Confirm to delete this allocation rule?"
            okText="Delete"
            cancelText="Cancel"
            okButtonProps={{ danger: true }}
            onConfirm={() => deleteRule(record.id)}
          >
            <Tooltip title="Delete">
              <Button aria-label="Delete" size="small" danger icon={<Delete size={14} />} />
            </Tooltip>
          </Popconfirm>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <Card>
        <div className="mb-4 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <Title level={3} style={{ marginBottom: 4 }}>

              Work order allocation rules
            </Title>
            <Text type="secondary">Automatically select handlers based on conditions, allowing work orders to enter an operational dispatch process.</Text>
          </div>
          <div className="flex items-center gap-2">
            <Tooltip title="Refresh">
              <Button aria-label="Refresh" icon={<RefreshCw size={16} />} onClick={loadRules} />
            </Tooltip>
            <Button type="primary" icon={<Plus size={16} />} onClick={openCreateModal}>

              Create rule
            </Button>
          </div>
        </div>

        <Alert
          className="mb-4"
          type="info"
          showIcon
          title="After defining the matching conditions, enable automatic allocation in the work order type and bind this rule"
        />

        <UsageGuideCard
          style={{ marginBottom: 16 }}
          title="How to write matching conditions and allocation actions"
          intro="Condition fields: priority (priority), status (status), category_id (category ID), department_id (department ID). Operators: equals / not_equals / contains / greater_than / less_than."
          steps={[
            'Example of matching conditions (hit only when all are met at the same time): [{"field":"priority","operator":"equals","value":"urgent"},{"field":"category_id","operator":"equals","value":3}]',
            'Choose one of three allocation actions: {"type":"user","value":5} is directly assigned to user 5; {"type":"round_robin","value":[5,8,12]} is polled among the candidates; {"type":"load_balance","value":[5,8,12]} is selected according to load.',
            'The larger the priority number, the higher it is. First use "Test" to enter a real work order ID to preview the hit result, and then enable it if there is no problem.',
            'After enabling, newly created/received work orders will be automatically dispatched according to the rules, and the allocation results will be recorded in the work order history.',
          ]}
        />

        {loadError ? (
          <Alert
            type={loadError.denied ? 'warning' : 'error'}
            showIcon
            title={loadError.denied ? 'No permission to view allocation rules' : 'Failed to load allocation rules'}
            description={
              loadError.denied
                ? 'The current role is not granted read permission for the assignment rules. This is not "there are no rules in the system". Please contact the administrator to configure permissions for your role\'s opening rules.'
                : loadError.detail || 'The backend failed to return the rule list, please try again later.'
            }
            action={
              <Button
                size="small"
                icon={<RefreshCw size={14} />}
                onClick={loadRules}
                loading={loading}
              >

                Retry
              </Button>
            }
          />
        ) : rules.length === 0 && !loading ? (
          <Alert
            type="info"
            showIcon
            title="No allocation rules yet"
            description="After creating a rule, work orders can automatically recommend handlers based on priority, status, classification, or department."
          />
        ) : (
          <Table
            rowKey="id"
            columns={columns}
            dataSource={rules}
            loading={loading}
            pagination={{ pageSize: 10, showTotal: total => `Total ${total} records` }}
          />
        )}
      </Card>

      <Modal
        title={editingRule ? 'Edit allocation rules' : 'Create allocation rule'}
        open={ruleModalOpen}
        onOk={saveRule}
        onCancel={closeRuleModal}
        confirmLoading={saving}
        okText="Save"
        cancelText="Cancel"
        width={720}
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Rule Name" rules={[{ required: true, message: 'Please enter a rule name' }]}>
            <Input placeholder="For example: high-priority work orders are assigned to front-line managers" />
          </Form.Item>
          <Form.Item name="description" label="Rule Description">
            <TextArea rows={3} placeholder="Describe the business scenario to which this rule applies" />
          </Form.Item>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <Form.Item name="priority" label="Priority" tooltip="The larger the number, the first it will be executed.">
              <InputNumber min={1} max={100} style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item name="isActive" label="Enabled" valuePropName="checked">
              <Switch checkedChildren="Enabled" unCheckedChildren="Disabled" />
            </Form.Item>
          </div>
          <Form.Item
            name="conditions"
            label="Match condition JSON"
            rules={[{ required: true, message: 'Please enter matching conditions JSON' }]}
          >
            <TextArea rows={7} spellCheck={false} />
          </Form.Item>
          <Form.Item
            name="actions"
            label="Assign action JSON"
            rules={[{ required: true, message: 'Please enter the assignment action JSON' }]}
          >
            <TextArea rows={5} spellCheck={false} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="Test allocation rules"
        open={testModalOpen}
        onOk={testRule}
        onCancel={() => setTestModalOpen(false)}
        confirmLoading={testing}
        okText="Execute test"
        cancelText="Close"
        destroyOnHidden
      >
        <Form form={testForm} layout="vertical">
          <Form.Item name="ruleId" label="Rule ID" rules={[{ required: true, message: 'Please select a rule' }]}>
            <InputNumber disabled style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="ticketId" label="Ticket ID" rules={[{ required: true, message: 'Please enter the work order ID' }]}>
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
        </Form>
        {testResult ? <Alert className="mt-3" type="success" showIcon title={testResult} /> : null}
      </Modal>
    </div>
  );
}
