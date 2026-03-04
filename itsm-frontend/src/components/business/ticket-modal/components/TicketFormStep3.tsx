/**
 * 工单表单 - 步骤3: 分配处理
 */

import React from 'react';
import { Form, Select, DatePicker, Row, Col, Alert, Avatar } from 'antd';
import type { User } from '../types';

interface TicketFormStep3Props {
  form: any;
  userList: User[];
}

export const TicketFormStep3: React.FC<TicketFormStep3Props> = ({ form, userList }) => {
  return (
    <div className="space-y-6">
      <Alert
        message="提示"
        description="可以选择指定处理人，也可以留空由系统自动分配。"
        type="info"
        showIcon
        className="mb-4"
      />
      <Row gutter={24}>
        <Col span={12}>
          <Form.Item label="指定处理人" name="assignee_id" extra="留空将由系统智能分配">
            <Select placeholder="请选择处理人（可选）" allowClear size="large">
              {userList.map(user => (
                <Select.Option key={user.id} value={user.id}>
                  <div style={{ display: 'flex', alignItems: 'center' }}>
                    <Avatar
                      size="small"
                      style={{ backgroundColor: '#1890ff', marginRight: 8 }}
                    >
                      {user.avatar}
                    </Avatar>
                    <span>{user.name}</span>
                    <span style={{ color: '#666', marginLeft: 8 }}>({user.role})</span>
                  </div>
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item label="预计完成时间" name="estimated_time" extra="可选，用于SLA计算">
            <DatePicker
              showTime
              placeholder="请选择预计完成时间（可选）"
              size="large"
              style={{ width: '100%' }}
            />
          </Form.Item>
        </Col>
      </Row>
    </div>
  );
};

TicketFormStep3.displayName = 'TicketFormStep3';
