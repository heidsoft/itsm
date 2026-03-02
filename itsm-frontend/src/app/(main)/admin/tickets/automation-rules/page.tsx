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
  message,
  Popconfirm,
  Typography,
  Tooltip,
  Alert,
} from 'antd';
import { Plus, Edit, Delete, PlayCircle, PauseCircle, Settings } from 'lucide-react';
import {
  TicketAutomationRuleApi,
  AutomationRule,
  CreateAutomationRuleRequest,
  UpdateAutomationRuleRequest,
} from '@/lib/api/ticket-automation-rule-api';

const { Title, Text } = Typography;
const { TextArea } = Input;

const AutomationRulesPage: React.FC = () => {
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
      is_active: true,
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
      is_active: rule.is_active,
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
        is_active: !rule.is_active,
      });
      message.success(rule.is_active ? '规则已禁用' : '规则已启用');
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
        is_active: values.is_active,
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
      dataIndex: 'execution_count',
      key: 'execution_count',
      width: 100,
      render: (count: number) => <Tag>{count || 0}</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
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
    <div className="p-6">
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
              showTotal: total => `共 ${total} 条`,
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

          <Form.Item name="conditions" label="触发条件" tooltip="JSON格式的条件数组">
            <TextArea
              rows={4}
              placeholder='[{"field": "status", "operator": "equals", "value": "new"}]'
            />
          </Form.Item>

          <Form.Item name="actions" label="执行动作" tooltip="JSON格式的动作数组">
            <TextArea rows={4} placeholder='[{"type": "assign", "assignee_id": 1}]' />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default AutomationRulesPage;
