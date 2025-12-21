/**
 * 自动化规则表单组件
 * 提供规则条件、动作的配置界面
 */

'use client';

import React, { useState, useEffect } from 'react';
import {
  Form,
  Input,
  InputNumber,
  Switch,
  Button,
  Space,
  Card,
  Select,
  Row,
  Col,
  Typography,
  Divider,
  Tag,
  message,
  Modal,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  SaveOutlined,
  CloseOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons';
import type { AutomationRule } from '@/lib/api/ticket-automation-rule-api';
import { TicketAutomationRuleApi } from '@/lib/api/ticket-automation-rule-api';

const { TextArea } = Input;
const { Option } = Select;
const { Title, Text } = Typography;

interface AutomationRuleFormProps {
  editingRule: AutomationRule | null;
  onSave: (values: any) => void;
  onCancel: () => void;
}

export const AutomationRuleForm: React.FC<AutomationRuleFormProps> = ({
  editingRule,
  onSave,
  onCancel,
}) => {
  const [form] = Form.useForm();
  const [testModalVisible, setTestModalVisible] = useState(false);
  const [testTicketId, setTestTicketId] = useState<number | null>(null);

  useEffect(() => {
    if (editingRule) {
      form.setFieldsValue({
        name: editingRule.name,
        description: editingRule.description,
        priority: editingRule.priority,
        is_active: editingRule.is_active,
        conditions: editingRule.conditions || [],
        actions: editingRule.actions || [],
      });
    } else {
      form.resetFields();
      form.setFieldsValue({
        priority: 0,
        is_active: true,
        conditions: [],
        actions: [],
      });
    }
  }, [editingRule, form]);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      onSave(values);
    } catch (error) {
      console.error('Form validation failed:', error);
    }
  };

  // 测试规则
  const handleTest = async () => {
    if (!testTicketId || !editingRule) {
      message.warning('请输入工单ID');
      return;
    }

    try {
      const response = await TicketAutomationRuleApi.testRule({
        rule_id: editingRule.id,
        ticket_id: testTicketId,
      });

      Modal.info({
        title: '测试结果',
        content: (
          <div>
            <p>
              <strong>匹配状态：</strong>
              <Tag color={response.matched ? 'success' : 'default'}>
                {response.matched ? '匹配' : '不匹配'}
              </Tag>
            </p>
            {response.reason && (
              <p>
                <strong>原因：</strong>
                {response.reason}
              </p>
            )}
            {response.actions && response.actions.length > 0 && (
              <div>
                <strong>将执行的动作：</strong>
                <ul>
                  {response.actions.map((action, index) => (
                    <li key={index}>{action}</li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        ),
      });
    } catch (error) {
      console.error('Failed to test rule:', error);
      message.error('测试规则失败');
    }
  };

  // 条件字段选项
  const conditionFields = [
    { value: 'status', label: '工单状态' },
    { value: 'priority', label: '优先级' },
    { value: 'category_id', label: '工单分类' },
    { value: 'department_id', label: '部门' },
    { value: 'requester_id', label: '申请人' },
    { value: 'assignee_id', label: '处理人' },
    { value: 'title', label: '标题关键词' },
  ];

  // 操作类型选项
  const actionTypes = [
    { value: 'set_category', label: '设置分类' },
    { value: 'set_priority', label: '设置优先级' },
    { value: 'assign', label: '分配给用户' },
    { value: 'auto_assign', label: '自动分配' },
    { value: 'escalate', label: '升级优先级' },
    { value: 'send_notification', label: '发送通知' },
    { value: 'set_status', label: '设置状态' },
  ];

  return (
    <>
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        initialValues={{
          priority: 0,
          is_active: true,
          conditions: [],
          actions: [],
        }}
      >
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="name"
              label="规则名称"
              rules={[{ required: true, message: '请输入规则名称' }]}
            >
              <Input placeholder="例如：高优先级工单自动升级" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="priority"
              label="优先级"
              rules={[{ required: true, message: '请输入优先级' }]}
              tooltip="数字越大优先级越高，系统按优先级从高到低执行规则"
            >
              <InputNumber min={0} max={100} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item name="description" label="规则描述">
          <TextArea rows={2} placeholder="规则描述（可选）" />
        </Form.Item>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="is_active"
              label="启用状态"
              valuePropName="checked"
            >
              <Switch checkedChildren="启用" unCheckedChildren="禁用" />
            </Form.Item>
          </Col>
          {editingRule && (
            <Col span={12}>
              <Space>
                <Button
                  icon={<PlayCircleOutlined />}
                  onClick={() => setTestModalVisible(true)}
                >
                  测试规则
                </Button>
              </Space>
            </Col>
          )}
        </Row>

        <Divider>条件设置</Divider>

        <Form.Item
          name="conditions"
          label="触发条件"
          rules={[{ required: true, message: '请至少添加一个条件' }]}
        >
          <Form.List name="conditions">
            {(fields, { add, remove }) => (
              <div>
                {fields.map((field, index) => (
                  <Card
                    key={field.key}
                    size="small"
                    style={{ marginBottom: 8 }}
                    extra={
                      <Button
                        type="link"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={() => remove(field.name)}
                      />
                    }
                  >
                    <Row gutter={8} align="middle">
                      <Col span={8}>
                        <Form.Item
                          {...field}
                          name={[field.name, 'field']}
                          rules={[{ required: true }]}
                        >
                          <Select placeholder="选择字段">
                            {conditionFields.map(f => (
                              <Option key={f.value} value={f.value}>
                                {f.label}
                              </Option>
                            ))}
                          </Select>
                        </Form.Item>
                      </Col>
                      <Col span={6}>
                        <Form.Item
                          {...field}
                          name={[field.name, 'operator']}
                          rules={[{ required: true }]}
                        >
                          <Select placeholder="操作符">
                            <Option value="equals">等于</Option>
                            <Option value="not_equals">不等于</Option>
                            <Option value="contains">包含</Option>
                            <Option value="in">属于</Option>
                            <Option value="not_in">不属于</Option>
                            <Option value="greater_than">大于</Option>
                            <Option value="less_than">小于</Option>
                          </Select>
                        </Form.Item>
                      </Col>
                      <Col span={10}>
                        <Form.Item
                          {...field}
                          name={[field.name, 'value']}
                          rules={[{ required: true }]}
                        >
                          <Input placeholder="值" />
                        </Form.Item>
                      </Col>
                    </Row>
                  </Card>
                ))}
                <Button
                  type="dashed"
                  onClick={() => add()}
                  block
                  icon={<PlusOutlined />}
                >
                  添加条件
                </Button>
              </div>
            )}
          </Form.List>
        </Form.Item>

        <Divider>动作设置</Divider>

        <Form.Item
          name="actions"
          label="执行动作"
          rules={[{ required: true, message: '请至少添加一个动作' }]}
        >
          <Form.List name="actions">
            {(fields, { add, remove }) => (
              <div>
                {fields.map((field, index) => {
                  const actionType = form.getFieldValue(['actions', index, 'type']);
                  return (
                    <Card
                      key={field.key}
                      size="small"
                      style={{ marginBottom: 8 }}
                      extra={
                        <Button
                          type="link"
                          danger
                          icon={<DeleteOutlined />}
                          onClick={() => remove(field.name)}
                        />
                      }
                    >
                      <Row gutter={8} align="middle">
                        <Col span={8}>
                          <Form.Item
                            {...field}
                            name={[field.name, 'type']}
                            rules={[{ required: true }]}
                          >
                            <Select placeholder="选择动作类型">
                              {actionTypes.map(type => (
                                <Option key={type.value} value={type.value}>
                                  {type.label}
                                </Option>
                              ))}
                            </Select>
                          </Form.Item>
                        </Col>
                        <Col span={16}>
                          {actionType === 'set_category' && (
                            <Form.Item
                              {...field}
                              name={[field.name, 'category_id']}
                              rules={[{ required: true }]}
                            >
                              <Input placeholder="分类ID" />
                            </Form.Item>
                          )}
                          {actionType === 'set_priority' && (
                            <Form.Item
                              {...field}
                              name={[field.name, 'priority']}
                              rules={[{ required: true }]}
                            >
                              <Select placeholder="选择优先级">
                                <Option value="low">低</Option>
                                <Option value="medium">中</Option>
                                <Option value="high">高</Option>
                                <Option value="urgent">紧急</Option>
                              </Select>
                            </Form.Item>
                          )}
                          {actionType === 'assign' && (
                            <Form.Item
                              {...field}
                              name={[field.name, 'user_id']}
                              rules={[{ required: true }]}
                            >
                              <Input placeholder="用户ID" />
                            </Form.Item>
                          )}
                          {actionType === 'send_notification' && (
                            <Form.Item
                              {...field}
                              name={[field.name, 'content']}
                              rules={[{ required: true }]}
                            >
                              <Input placeholder="通知内容" />
                            </Form.Item>
                          )}
                          {actionType === 'set_status' && (
                            <Form.Item
                              {...field}
                              name={[field.name, 'status']}
                              rules={[{ required: true }]}
                            >
                              <Select placeholder="选择状态">
                                <Option value="open">待处理</Option>
                                <Option value="in_progress">处理中</Option>
                                <Option value="resolved">已解决</Option>
                                <Option value="closed">已关闭</Option>
                              </Select>
                            </Form.Item>
                          )}
                          {(actionType === 'auto_assign' || actionType === 'escalate') && (
                            <Text type="secondary" style={{ fontSize: 12 }}>
                              此动作无需额外配置
                            </Text>
                          )}
                        </Col>
                      </Row>
                    </Card>
                  );
                })}
                <Button
                  type="dashed"
                  onClick={() => add({ type: 'set_category' })}
                  block
                  icon={<PlusOutlined />}
                >
                  添加动作
                </Button>
              </div>
            )}
          </Form.List>
        </Form.Item>

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" icon={<SaveOutlined />}>
              {editingRule ? '更新规则' : '创建规则'}
            </Button>
            <Button onClick={onCancel} icon={<CloseOutlined />}>
              取消
            </Button>
          </Space>
        </Form.Item>
      </Form>

      {/* 测试规则模态框 */}
      <Modal
        title="测试规则"
        open={testModalVisible}
        onOk={handleTest}
        onCancel={() => {
          setTestModalVisible(false);
          setTestTicketId(null);
        }}
        okText="测试"
        cancelText="取消"
      >
        <Form.Item label="工单ID" required>
          <Input
            type="number"
            placeholder="请输入要测试的工单ID"
            value={testTicketId || ''}
            onChange={(e) => setTestTicketId(parseInt(e.target.value) || null)}
          />
        </Form.Item>
      </Modal>
    </>
  );
};

