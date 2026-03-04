/**
 * 工单表单 - 步骤1: 基本信息
 */

import React from 'react';
import { Form, Input, Alert } from 'antd';
import type { TicketFormValues } from '../types';
import { DESCRIPTION_RULES } from '../utils/ticket-form-utils';

interface TicketFormStep1Props {
  form: any;
}

export const TicketFormStep1: React.FC<TicketFormStep1Props> = ({ form }) => {
  return (
    <div className="space-y-6">
      <Alert
        message="提示"
        description="请详细描述您遇到的问题或需求，这将帮助处理人员更快地理解和解决问题。"
        type="info"
        showIcon
        className="mb-4"
      />
      <Form.Item
        label="工单标题"
        name="title"
        rules={[{ required: true, message: '请输入工单标题' }]}
        extra="简洁明了地描述问题或需求"
      >
        <Input placeholder="例如：无法登录系统" size="large" />
      </Form.Item>
      <Form.Item
        label="详细描述"
        name="description"
        rules={DESCRIPTION_RULES}
        extra="请详细描述问题现象、发生时间、影响范围等信息"
      >
        <Input.TextArea
          rows={8}
          placeholder="请详细描述工单内容、问题、预期结果等..."
          showCount
          maxLength={1000}
        />
      </Form.Item>
    </div>
  );
};

TicketFormStep1.displayName = 'TicketFormStep1';
