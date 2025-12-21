/**
 * 分配规则表单组件
 * 提供规则条件、动作的配置界面
 */

'use client';

import React, { useState } from 'react';
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
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  SaveOutlined,
  CloseOutlined,
} from '@ant-design/icons';
import type { AssignmentRule } from '@/lib/api/ticket-assignment-api';

const { TextArea } = Input;
const { Option } = Select;
const { Title, Text } = Typography;

interface AssignmentRuleFormProps {
  form: any;
  editingRule: AssignmentRule | null;
  onSave: (values: any) => void;
  onCancel: () => void;
}

export const AssignmentRuleForm: React.FC<AssignmentRuleFormProps> = ({
  form,
  editingRule,
  onSave,
  onCancel,
}) => {
  const [conditionType, setConditionType] = useState<string>('status');
  const [actionType, setActionType] = useState<string>('assign');

  // 条件字段选项
  const conditionFields = [
    { value: 'status', label: '工单状态' },
    { value: 'priority', label: '优先级' },
    { value: 'category_id', label: '工单分类' },
    { value: 'department_id', label: '部门' },
    { value: 'requester_id', label: '申请人' },
    { value: 'title', label: '标题关键词' },
  ];

  // 操作类型选项
  const actionTypes = [
    { value: 'assign', label: '分配给用户' },
    { value: 'assign_to_team', label: '分配给团队' },
    { value: 'assign_by_skill', label: '按技能分配' },
    { value: 'assign_by_workload', label: '按工作负载分配' },
  ];

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      onSave(values);
    } catch (error) {
      console.error('Form validation failed:', error);
    }
  };

  // 添加条件
  const handleAddCondition = () => {
    const conditions = form.getFieldValue('conditions') || [];
    form.setFieldsValue({
      conditions: [
        ...conditions,
        {
          field: conditionType,
          operator: 'equals',
          value: '',
        },
      ],
    });
  };

  // 删除条件
  const handleRemoveCondition = (index: number) => {
    const conditions = form.getFieldValue('conditions') || [];
    conditions.splice(index, 1);
    form.setFieldsValue({ conditions });
  };

  return (
    <Form
      form={form}
      layout="vertical"
      onFinish={handleSubmit}
      initialValues={{
        priority: 0,
        is_active: true,
        conditions: [],
        actions: {},
      }}
    >
      <Form.Item
        name="name"
        label="规则名称"
        rules={[{ required: true, message: '请输入规则名称' }]}
      >
        <Input placeholder="例如：高优先级工单分配给技术团队" />
      </Form.Item>

      <Form.Item name="description" label="规则描述">
        <TextArea rows={2} placeholder="规则描述（可选）" />
      </Form.Item>

      <Row gutter={16}>
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
        <Col span={12}>
          <Form.Item
            name="is_active"
            label="启用状态"
            valuePropName="checked"
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Col>
      </Row>

      <Divider>条件设置</Divider>

      <Form.Item
        name="conditions"
        label="匹配条件"
        rules={[{ required: true, message: '请至少添加一个条件' }]}
      >
        <Form.List name="conditions">
          {(fields, { add, remove }) => (
            <div>
              {fields.map((field, index) => {
                const condition = form.getFieldValue(['conditions', index]);
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
                        onClick={() => {
                          remove(field.name);
                        }}
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
                );
              })}
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
        name={['actions', 'type']}
        label="分配方式"
        rules={[{ required: true, message: '请选择分配方式' }]}
      >
        <Select
          placeholder="选择分配方式"
          onChange={(value) => setActionType(value)}
        >
          {actionTypes.map(type => (
            <Option key={type.value} value={type.value}>
              {type.label}
            </Option>
          ))}
        </Select>
      </Form.Item>

      {actionType === 'assign' && (
        <Form.Item
          name={['actions', 'user_id']}
          label="分配给用户"
          rules={[{ required: true, message: '请选择用户' }]}
        >
          <Input placeholder="用户ID（需要集成用户选择器）" />
        </Form.Item>
      )}

      {actionType === 'assign_to_team' && (
        <Form.Item
          name={['actions', 'team_id']}
          label="分配给团队"
          rules={[{ required: true, message: '请选择团队' }]}
        >
          <Input placeholder="团队ID（需要集成团队选择器）" />
        </Form.Item>
      )}

      {actionType === 'assign_by_skill' && (
        <Form.Item
          name={['actions', 'skill_required']}
          label="所需技能"
        >
          <Input placeholder="技能名称或ID" />
        </Form.Item>
      )}

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
  );
};

