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
import { Plus, Edit, Delete, PlayCircle, PauseCircle, Settings } from 'lucide-react';
import { UsageGuideCard } from '@/components/common/UsageGuideCard';
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
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState<AutomationRule | null>(null);
  const [form] = Form.useForm();

  // 加载规则列表
  const loadRules = async () => {
    setLoading(true);
    try {
      const response = await TicketAutomationRuleApi.listRules();
      setRules(response.rules || []);
    } catch (error) {
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
      actions: rule.actions,
    });
    setModalVisible(true);
  };

  // 删除规则
  const handleDelete = async (id: number) => {
    try {
      await TicketAutomationRuleApi.deleteRule(id);
      message.success('规则删除成功');
      loadRules();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 切换规则状态
  const handleToggleActive = async (rule: AutomationRule) => {
    try {
      await TicketAutomationRuleApi.updateRule(rule.id, {
        isActive: !rule.isActive,
      });
      message.success(rule.isActive ? '规则已禁用' : '规则已启用');
      loadRules();
    } catch (error) {
      message.error('操作失败');
    }
  };

  // 保存规则
  const handleSave = async () => {
    try {
      const values = await form.validateFields();

      const ruleData: CreateAutomationRuleRequest | UpdateAutomationRuleRequest = {
        name: values.name,
        description: values.description,
        priority: values.priority || 1,
        isActive: values.isActive,
        conditions: values.conditions || [],
        actions: values.actions || [],
      };

      if (editingRule) {
        await TicketAutomationRuleApi.updateRule(editingRule.id, ruleData);
        message.success('规则更新成功');
      } else {
        await TicketAutomationRuleApi.createRule(ruleData as CreateAutomationRuleRequest);
        message.success('规则创建成功');
      }

      setModalVisible(false);
      loadRules();
    } catch (error) {
      message.error('保存失败');
    }
  };

  // 表格列定义
  const columns = [
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => <span className="font-medium">{text}</span>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      render: (text: string) => <Text type="secondary">{text || '-'}</Text>,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: number) => (
        <Tag color={priority >= 5 ? 'red' : priority >= 3 ? 'orange' : 'blue'}>P{priority}</Tag>
      ),
    },
    {
      title: '执行次数',
      dataIndex:'executionCount',
      key:'executionCount',
      width: 100,
      render: (count: number) => <Tag>{count || 0}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 100,
      render: (active: boolean, record: AutomationRule) => (
        <Switch
          checked={active}
          checkedChildren="启用"
          unCheckedChildren="禁用"
          onChange={() => handleToggleActive(record)}
        />
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_: unknown, record: AutomationRule) => (
        <Space>
          <Tooltip title="编辑">
            <Button size="small" icon={<Edit size={14} />} onClick={() => handleEdit(record)} />
          </Tooltip>
          <Popconfirm
            title="确定要删除这个规则吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Tooltip title="删除">
              <Button size="small" danger icon={<Delete size={14} />} />
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
              工单自动化规则
            </Title>
            <Text type="secondary">配置自动化规则来简化工单处理流程</Text>
          </div>
          <Button type="primary" icon={<Plus size={16} />} onClick={handleCreate}>
            创建规则
          </Button>
        </div>

        <Alert
          className="mb-4"
          type="info"
          showIcon
          message="自动化规则用于工单生命周期内的条件触发，不是另一套 BPMN 流程"
          description="先用条件定义命中的工单字段，再配置动作（例如更新字段、分配处理人或发送通知）。规则按优先级执行；BPMN 负责审批、服务履约等显式流程。创建后请用某一真实工单执行“测试”，确认命中与动作，再启用到生产。"
        />

        <UsageGuideCard
          style={{ marginBottom: 16 }}
          title="从创建到启用：推荐的配置步骤"
          intro="自动化规则 = 触发条件 + 执行动作，适用于单工单生命周期内的自动处理；需要多人审批、多阶段流转时用 BPMN 流程设计器。"
          steps={[
            '点击“创建规则”，填写名称和优先级（数字越大越先执行）。',
            '配置触发条件：字段选状态/优先级/类型/分类，操作符选等于/不等于/包含/在列表中；多个条件需同时满足。',
            '配置执行动作（可多条）：自动分配、发送通知、更新字段、升级、自动关闭。',
            '保存后先保持禁用，用一张真实工单修改对应字段验证命中效果，再切换启用。',
            '启用后通过列表的“执行次数”观察规则是否被触发；未命中时检查条件字段值是否完全匹配。',
          ]}
        />

        {rules.length === 0 && !loading ? (
          <Alert
            type="info"
            showIcon
            message="暂无自动化规则"
            description="点击上方按钮创建第一个自动化规则"
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
              showTotal: total => `共 ${total} 条记录`,
            }}
          />
        )}
      </Card>

      {/* 创建/编辑弹窗 */}
      <Modal
        title={editingRule ? '编辑自动化规则' : '创建自动化规则'}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => setModalVisible(false)}
        okText="保存"
        cancelText="取消"
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="规则名称"
            rules={[{ required: true, message: '请输入规则名称' }]}
          >
            <Input placeholder="请输入规则名称" />
          </Form.Item>

          <Form.Item name="description" label="规则描述">
            <TextArea rows={3} placeholder="请输入规则描述" />
          </Form.Item>

          <Form.Item name="priority" label="优先级" initialValue={1} tooltip="数字越大优先级越高">
            <InputNumber min={1} max={10} />
          </Form.Item>

          <Form.Item name="is_active" label="启用状态" valuePropName="checked" initialValue={true}>
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>

          <Form.Item label="触发条件" required>
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
                        <Select style={{ width: 120 }} placeholder="字段">
                          <Select.Option value="status">状态</Select.Option>
                          <Select.Option value="priority">优先级</Select.Option>
                          <Select.Option value="type">类型</Select.Option>
                          <Select.Option value="category">分类</Select.Option>
                        </Select>
                      </Form.Item>
                      <Form.Item
                        {...restField}
                        name={[name, 'operator']}
                        noStyle
                      >
                        <Select style={{ width: 100 }} placeholder="操作符">
                          <Select.Option value="equals">等于</Select.Option>
                          <Select.Option value="not_equals">不等于</Select.Option>
                          <Select.Option value="contains">包含</Select.Option>
                          <Select.Option value="in">在列表中</Select.Option>
                        </Select>
                      </Form.Item>
                      <Form.Item
                        {...restField}
                        name={[name, 'value']}
                        noStyle
                      >
                        <Input placeholder="值" style={{ width: 150 }} />
                      </Form.Item>
                      <Button type="text" danger icon={<Delete size={14} />} onClick={() => remove(name)} />
                    </Space>
                  ))}
                  <Button type="dashed" onClick={() => add()} block icon={<Plus size={14} />}>
                    添加条件
                  </Button>
                </>
              )}
            </Form.List>
          </Form.Item>

          <Form.Item label="执行动作" required>
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
                        <Select style={{ width: 140 }} placeholder="动作类型">
                          <Select.Option value="assign">自动分配</Select.Option>
                          <Select.Option value="notify">发送通知</Select.Option>
                          <Select.Option value="update_field">更新字段</Select.Option>
                          <Select.Option value="escalate">升级</Select.Option>
                          <Select.Option value="close">自动关闭</Select.Option>
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
                    添加动作
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
