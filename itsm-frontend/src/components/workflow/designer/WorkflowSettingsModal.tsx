// 流程设置模态框
// Workflow Settings Modal

'use client';

import React from 'react';
import { Modal, Form, Tabs, Select, Input, Checkbox, Row, Col } from 'antd';
import type { FormInstance } from 'antd';

const { Option } = Select;

interface WorkflowSettingsModalProps {
  visible: boolean;
  onClose: () => void;
  onSave: () => void;
  form: FormInstance;
}

export default function WorkflowSettingsModal({
  visible,
  onClose,
  onSave,
  form,
}: WorkflowSettingsModalProps) {
  return (
    <Modal
      title="流程设置"
      open={visible}
      onOk={onSave}
      onCancel={onClose}
      width={800}
      okText="保存"
      cancelText="取消"
    >
      <Form form={form} layout="vertical">
        <Tabs
          items={[
            {
              key: 'approval',
              label: '审批配置',
              children: (
                <>
                  <Form.Item
                    label="审批类型"
                    name={['approval_config', 'approval_type']}
                    rules={[{ required: true, message: '请选择审批类型' }]}
                  >
                    <Select>
                      <Option value="single">单人审批</Option>
                      <Option value="parallel">并行审批</Option>
                      <Option value="sequential">串行审批</Option>
                      <Option value="conditional">条件审批</Option>
                    </Select>
                  </Form.Item>

                  <Form.Item label="审批人" name={['approval_config', 'approvers']}>
                    <Select mode="multiple" placeholder="选择审批人">
                      {/* 用户列表通过 Context 获取 */}
                    </Select>
                  </Form.Item>
                </>
              ),
            },
            {
              key: 'sla',
              label: 'SLA配置',
              children: (
                <>
                  <Row gutter={16}>
                    <Col span={12}>
                      <Form.Item label="响应时间(小时)" name={['sla_config', 'response_time_hours']}>
                        <Input type="number" />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item label="解决时间(小时)" name={['sla_config', 'resolution_time_hours']}>
                        <Input type="number" />
                      </Form.Item>
                    </Col>
                  </Row>

                  <Form.Item
                    label="工作时间设置"
                    name={['sla_config', 'business_hours_only']}
                    valuePropName="checked"
                  >
                    <Checkbox>仅工作时间</Checkbox>
                  </Form.Item>
                </>
              ),
            },
          ]}
        />
      </Form>
    </Modal>
  );
}
