/**
 * 工单表单 - 步骤4: 确认提交（预览）
 */

import React from 'react';
import { Form, Card, Upload, Alert, Button } from 'antd';
import type { TicketFormValues, User } from '../types';
import { FileText } from 'lucide-react';

interface TicketPreviewStepProps {
  form: any;
  formData: Record<string, any>;
  userList: User[];
}

export const TicketPreviewStep: React.FC<TicketPreviewStepProps> = ({
  form,
  formData,
  userList,
}) => {
  const allValues = { ...formData, ...form.getFieldsValue() };

  return (
    <div className="space-y-6">
      <Alert
        message="请确认信息"
        description="确认无误后点击提交，如有问题可返回修改。"
        type="warning"
        showIcon
        className="mb-4"
      />
      <Card title="工单信息预览" className="bg-gray-50">
        <div className="space-y-4">
          <div>
            <span className="text-gray-600 font-medium">标题：</span>
            <span className="ml-2">{allValues.title || '未填写'}</span>
          </div>
          <div>
            <span className="text-gray-600 font-medium">类型：</span>
            <span className="ml-2">{allValues.type || '未选择'}</span>
          </div>
          <div>
            <span className="text-gray-600 font-medium">分类：</span>
            <span className="ml-2">{allValues.category || '未选择'}</span>
          </div>
          <div>
            <span className="text-gray-600 font-medium">优先级：</span>
            <span className="ml-2">{allValues.priority || '未选择'}</span>
          </div>
          {allValues.assignee_id && (
            <div>
              <span className="text-gray-600 font-medium">处理人：</span>
              <span className="ml-2">
                {userList.find(u => u.id === allValues.assignee_id)?.name || '未选择'}
              </span>
            </div>
          )}
          <div>
            <span className="text-gray-600 font-medium">描述：</span>
            <div className="mt-2 p-3 bg-white rounded border border-gray-200">
              {allValues.description || '未填写'}
            </div>
          </div>
        </div>
      </Card>
      <Form.Item label="附件（可选）" name="attachments">
        <Upload
          multiple
          beforeUpload={() => false}
          maxCount={5}
          accept="image/*,.pdf,.doc,.docx,.xls,.xlsx"
        >
          <Button icon={<FileText size={16} />}>上传附件</Button>
        </Upload>
      </Form.Item>
    </div>
  );
};

TicketPreviewStep.displayName = 'TicketPreviewStep';
