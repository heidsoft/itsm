'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Button,
  Space,
  Tag,
  Switch,
  Modal,
  Form,
  Input,
  InputNumber,
  Select,
  App,
  Popconfirm,
  Typography,
  Tooltip,
  Alert,
} from 'antd';
import { Plus, Edit, Delete, PlayCircle, PauseCircle, Settings, RefreshCw } from 'lucide-react';
import { UsageGuideCard } from '@/components/common/UsageGuideCard';
import { isPermissionDeniedError } from '@/lib/api/http-client';
import type {
  AutomationRule,
  CreateAutomationRuleRequest,
  UpdateAutomationRuleRequest} from '@/lib/api/ticket-automation-rule-api';
import {
  TicketAutomationRuleApi
} from '@/lib/api/ticket-automation-rule-api';

const { Title, Text } = Typography;
const { TextArea } = Input;

const AutomationRulesPage: React.FC = () => {
  const { message } = App.useApp();
  const [rules, setRules] = useState<AutomationRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<{ denied: boolean; detail: string } | null>(null);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState<AutomationRule | null>(null);
  const [form] = Form.useForm();

  // 加载规则列表
  const loadRules = async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const response = await TicketAutomationRuleApi.listRules();
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

  // 创建规则
  const handleCreate = () => {
    setEditingRule(null);
    form.resetFields();
    form.setFieldsValue({
      priority: 1,
      isActive: true,
      conditions: [],
      actions: [],
    });
    setModalVisible(true);
  };

  // 编辑规则
  const handleEdit = (rule: AutomationRule) => {
    setEditingRule(rule);
    form.setFieldsValue({
      name: rule.name,
      description: rule.description,
      priority: rule.priority,
      isActive: rule.isActive,
      conditions: rule.conditions,
      // 后端 config 是对象，表单 TextArea 需要字符串：回填时序列化，保存时 parse（见 handleSave）
      actions: (rule.actions || []).map(action => ({
        ...action,
        config:
          action.config && typeof action.config === 'object'
            ? JSON.stringify(action.config)
            : ((action.config as string | undefined) ?? ''),
      })),
    });
    setModalVisible(true);
  };

  // 删除规则
  const handleDelete = async (id: number) => {
    try {
      await TicketAutomationRuleApi.deleteRule(id);
      message.success('RuleDeleteSuccess');
      loadRules();
    } catch (error) {
      message.error('Delete failed');
    }
  };

  // 切换规则状态
  const handleToggleActive = async (rule: AutomationRule) => {
    try {
      await TicketAutomationRuleApi.updateRule(rule.id, {
        isActive: !rule.isActive,
      });
      message.success(rule.isActive ? 'Rule disabled' : 'Rule enabled');
      loadRules();
    } catch (error) {
      message.error('Operation failed');
    }
  };

  // 保存规则
  const handleSave = async () => {
    try {
      const values = await form.validateFields();

      // 动作 config 在表单里是 TextArea 字符串，后端契约是对象（[]map[string]interface{}），
      // 此处统一 parse；空串按空对象处理，非法 JSON 给出可定位的字段错误。
      const actions = ((values.actions || []) as Array<{ type: string; config?: string }>).map(
        (action, index) => {
          const raw = action.config?.trim();
          let config: Record<string, unknown> = {};
          if (raw) {
            try {
              config = JSON.parse(raw) as Record<string, unknown>;
            } catch {
              message.error(`No. ${index + 1} The config of action is not a valid JSON`);
              throw new Error('invalid action config json');
            }
          }
          return { ...action, config };
        }
      );

      const ruleData: CreateAutomationRuleRequest | UpdateAutomationRuleRequest = {
        name: values.name,
        description: values.description,
        priority: values.priority ?? 1,
        isActive: values.isActive,
        conditions: values.conditions || [],
        actions,
      };

      if (editingRule) {
        await TicketAutomationRuleApi.updateRule(editingRule.id, ruleData);
        message.success('RuleUpdateSuccess');
      } else {
        await TicketAutomationRuleApi.createRule(ruleData as CreateAutomationRuleRequest);
        message.success('RuleCreateSuccess');
      }

      setModalVisible(false);
      loadRules();
    } catch (error) {
      message.error('SaveFailed');
    }
  };

  // 表格列定义
  const columns = [
    {
      title: 'Rule Name',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <span className="font-medium">{text}</span>,
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      render: (text: string) => <Text type="secondary">{text || '-'}</Text>,
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: number) => (
        <Tag color={priority >= 5 ? 'red' : priority >= 3 ? 'orange' : 'blue'}>P{priority}</Tag>
      ),
    },
    {
      title: 'Number of executions',
      dataIndex:'executionCount',
      key:'executionCount',
      width: 100,
      render: (count: number) => <Tag>{count || 0}</Tag>,
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 100,
      render: (active: boolean, record: AutomationRule) => (
        <Switch
          aria-label={`Switching rules ${record.name} enabled status`}
          checked={active}
          checkedChildren="Enabled"
          unCheckedChildren="Disabled"
          onChange={() => handleToggleActive(record)}
        />
      ),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 150,
      render: (_: unknown, record: AutomationRule) => (
        <Space>
          <Tooltip title="Edit">
            <Button aria-label="Edit" size="small" icon={<Edit size={14} />} onClick={() => handleEdit(record)} />
          </Tooltip>
          <Popconfirm
            title="Are you sure you want to delete this rule?"
            onConfirm={() => handleDelete(record.id)}
            okText="OK"
            cancelText="Cancel"
          >
            <Tooltip title="Delete">
              <Button aria-label="Delete" size="small" danger icon={<Delete size={14} />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <Card>
        <div className="flex justify-between items-center mb-4">
          <div>
            <Title level={3} style={{ marginBottom: 4 }}>

              Work order automation rules
            </Title>
            <Text type="secondary">Configure automation rules to streamline ticket processing</Text>
          </div>
          <Button type="primary" icon={<Plus size={16} />} onClick={handleCreate}>

            Create rule
          </Button>
        </div>

        <Alert
          className="mb-4"
          type="info"
          showIcon
          title="Conditions trigger automatic processing of work orders; please use BPMN process for approval and multi-stage transfer"
        />

        <UsageGuideCard
          style={{ marginBottom: 16 }}
          title="From creation to activation: Recommended configuration steps"
          intro="Automation rule = trigger condition + execution action, suitable for automatic processing within the life cycle of a single work order; use BPMN process designer when multi-person approval and multi-stage transfer are required."
          steps={[
            'Click "Create Rule" and fill in the name and priority (the larger the number, the earlier it will be executed).',
            'Configure trigger conditions: select status/priority/type/category for the field, and select equal/not equal/included/in the list for the operator; multiple conditions need to be met at the same time.',
            'Configure execution actions (can be multiple): automatic allocation, sending notifications, updating fields, upgrading, and automatic closing.',
            'Keep it disabled after saving, use a real work order to modify the corresponding fields to verify the hit effect, and then switch to enable.',
            'After enabling, observe whether the rule is triggered through the "number of executions" of the list; when it is not hit, check whether the condition field value completely matches.',
          ]}
        />

        {loadError ? (
          <Alert
            type={loadError.denied ? 'warning' : 'error'}
            showIcon
            title={loadError.denied ? 'No permission to view automation rules' : 'Failed to load automation rules'}
            description={
              loadError.denied
                ? 'The current role is not granted read permission for automation rules. This is not "there are no rules in the system". Please contact the administrator to configure permissions for your role\'s opening rules.'
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
            title="There is no automation rule yet, click the button above to create it"
          />
        ) : (
          <Table
            columns={columns}
            dataSource={rules}
            rowKey="id"
            loading={loading}
            pagination={{
              pageSize: 10,
              showSizeChanger: true,
              showTotal: total => `Total ${total} records`,
            }}
          />
        )}
      </Card>

      {/* 创建/编辑弹窗 */}
      <Modal
        title={editingRule ? 'Edit automation rules' : 'Create automation rules'}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => setModalVisible(false)}
        okText="Save"
        cancelText="Cancel"
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="Rule Name"
            rules={[{ required: true, message: 'Please enter a rule name' }]}
          >
            <Input placeholder="Please enter a rule name" />
          </Form.Item>

          <Form.Item name="description" label="Rule Description">
            <TextArea rows={3} placeholder="Please enter a rule description" />
          </Form.Item>

          <Form.Item name="priority" label="Priority" initialValue={1} tooltip="The larger the number, the higher the priority.">
            <InputNumber min={1} max={10} />
          </Form.Item>

          <Form.Item name="isActive" label="Enabled" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="Enabled" unCheckedChildren="Disabled" />
          </Form.Item>

          <Form.Item label="Trigger Condition" required>
            <Form.List
              name="conditions"
              initialValue={[{ field: 'status', operator: 'equals', value: '' }]}
            >
              {(fields, { add, remove }) => (
                <>
                  {fields.map(({ key, name, ...restField }) => (
                    <Space key={key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                      <Form.Item
                        {...restField}
                        name={[name, 'field']}
                        noStyle
                      >
                        <Select style={{ width: 120 }} placeholder="Field">
                          <Select.Option value="status">Status</Select.Option>
                          <Select.Option value="priority">Priority</Select.Option>
                          <Select.Option value="type">Type</Select.Option>
                          <Select.Option value="category">Category</Select.Option>
                        </Select>
                      </Form.Item>
                      <Form.Item
                        {...restField}
                        name={[name, 'operator']}
                        noStyle
                      >
                        <Select style={{ width: 100 }} placeholder="Operator">
                          <Select.Option value="equals">equal to</Select.Option>
                          <Select.Option value="not_equals">Not equal to</Select.Option>
                          <Select.Option value="contains">Contains</Select.Option>
                          <Select.Option value="in">In list</Select.Option>
                        </Select>
                      </Form.Item>
                      <Form.Item
                        {...restField}
                        name={[name, 'value']}
                        noStyle
                      >
                        <Input placeholder="Value" style={{ width: 150 }} />
                      </Form.Item>
                      <Button type="text" danger icon={<Delete size={14} />} onClick={() => remove(name)} />
                    </Space>
                  ))}
                  <Button type="dashed" onClick={() => add()} block icon={<Plus size={14} />}>

                    Add condition
                  </Button>
                </>
              )}
            </Form.List>
          </Form.Item>

          <Form.Item label="Action" required>
            <Form.List
              name="actions"
              initialValue={[{ type: 'notify', config: {} }]}
            >
              {(fields, { add, remove }) => (
                <>
                  {fields.map(({ key, name, ...restField }) => (
                    <Space key={key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                      <Form.Item
                        {...restField}
                        name={[name, 'type']}
                        noStyle
                      >
                        <Select style={{ width: 140 }} placeholder="Action type">
                          <Select.Option value="assign">Auto Assign</Select.Option>
                          <Select.Option value="notify">Send notification</Select.Option>
                          <Select.Option value="update_field">Update field</Select.Option>
                          <Select.Option value="escalate">Escalate</Select.Option>
                          <Select.Option value="close">Automatic shutdown</Select.Option>
                        </Select>
                      </Form.Item>
                      <Form.Item
                        {...restField}
                        name={[name, 'config']}
                        noStyle
                      >
                        <Input.TextArea
                          placeholder='{"assignee_id": 1}'
                          style={{ width: 200 }}
                          rows={1}
                        />
                      </Form.Item>
                      <Button type="text" danger icon={<Delete size={14} />} onClick={() => remove(name)} />
                    </Space>
                  ))}
                  <Button type="dashed" onClick={() => add()} block icon={<Plus size={14} />}>

                    Add action
                  </Button>
                </>
              )}
            </Form.List>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default AutomationRulesPage;
