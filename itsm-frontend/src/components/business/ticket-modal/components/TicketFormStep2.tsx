/**
 * 工单表单 - 步骤2: 分类设置
 */

import React from 'react';
import { Form, Select, Row, Col, Alert } from 'antd';
import { TYPE_RULES, CATEGORY_RULES, PRIORITY_RULES, TICKET_TYPE_OPTIONS, PRIORITY_OPTIONS, CATEGORY_OPTIONS } from '../utils/ticket-form-utils';

interface TicketFormStep2Props {
  form: any;
}

export const TicketFormStep2: React.FC<TicketFormStep2Props> = ({ form }) => {
  return (
    <div className="space-y-6">
      <Alert
        message="提示"
        description="选择合适的分类和优先级，有助于工单快速分配到合适的处理人员。"
        type="info"
        showIcon
        className="mb-4"
      />
      <Row gutter={24}>
        <Col span={12}>
          <Form.Item
            label="工单类型"
            name="type"
            rules={TYPE_RULES}
          >
            <Select placeholder="请选择工单类型" size="large">
              {TICKET_TYPE_OPTIONS.map(opt => (
                <Select.Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            label="工单分类"
            name="category"
            rules={CATEGORY_RULES}
          >
            <Select placeholder="请选择工单分类" size="large">
              {CATEGORY_OPTIONS.map(opt => (
                <Select.Option key={opt.value} value={opt.value}>
                  {opt.label}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Col>
      </Row>
      <Form.Item
        label="优先级"
        name="priority"
        rules={PRIORITY_RULES}
        extra="根据问题紧急程度选择合适的优先级"
      >
        <Select placeholder="请选择优先级" size="large">
          {PRIORITY_OPTIONS.map(opt => (
            <Select.Option key={opt.value} value={opt.value}>
              {opt.label}
            </Select.Option>
          ))}
        </Select>
      </Form.Item>
    </div>
  );
};

TicketFormStep2.displayName = 'TicketFormStep2';
