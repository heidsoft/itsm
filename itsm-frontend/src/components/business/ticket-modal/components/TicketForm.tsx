/**
 * 重构后的 TicketForm 组件
 * 使用自定义 hooks 和拆分的步骤组件
 */

import React, { useEffect } from 'react';
import { Form, Steps, Button, Input, Select, DatePicker, Row, Col, Avatar } from 'antd';
import { Plus, ArrowLeft, ArrowRight, CheckCircle } from 'lucide-react';
import { useI18n } from '@/lib/i18n';

import { useTicketForm } from '../hooks/useTicketForm';
import type { TicketFormProps, User, TicketTemplate } from '../types';
import { TICKET_FORM_STEPS } from '../utils/ticket-form-utils';
import { TicketFormStep1 } from './TicketFormStep1';
import { TicketFormStep2 } from './TicketFormStep2';
import { TicketFormStep3 } from './TicketFormStep3';
import { TicketPreviewStep } from './TicketPreviewStep';

export const TicketForm: React.FC<TicketFormProps> = ({
  form,
  onSubmit,
  loading = false,
  isEditing = false,
}) => {
  const { t } = useI18n();
  const {
    currentStep,
    steps,
    userList,
    ticketTemplates,
    handleNext,
    handlePrev,
    handleSubmit,
    resetForm,
  } = useTicketForm({ form, onSubmit, loading, isEditing });

  // 如果是编辑模式，使用单页表单
  if (isEditing) {
    return <EditingForm form={form} onSubmit={onSubmit} loading={loading} userList={userList} />;
  }

  // 创建模式：使用分步表单
  return (
    <Form layout="vertical" form={form} style={{ marginTop: 20 }}>
      <Steps current={currentStep} items={steps} className="mb-8" />

      <div className="min-h-[400px]">
        {currentStep === 0 && <TicketFormStep1 form={form} />}
        {currentStep === 1 && <TicketFormStep2 form={form} />}
        {currentStep === 2 && <TicketFormStep3 form={form} userList={userList} />}
        {currentStep === 3 && <TicketPreviewStep form={form} formData={{}} userList={userList} />}
      </div>

      <div className="flex justify-between items-center pt-6 mt-6 border-t border-gray-100">
        <div>
          {currentStep > 0 && (
            <Button
              onClick={handlePrev}
              size="large"
              icon={<ArrowLeft size={16} />}
              className="rounded-lg border-gray-200 hover:border-gray-300 transition-colors duration-200"
            >
              上一步
            </Button>
          )}
        </div>
        <div className="flex gap-3">
          <Button
            onClick={resetForm}
            size="large"
            className="rounded-lg border-gray-200 hover:border-gray-300 transition-colors duration-200"
          >
            重置
          </Button>
          {currentStep < steps.length - 1 ? (
            <Button
              type="primary"
              onClick={handleNext}
              size="large"
              icon={<ArrowRight size={16} />}
              className="rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 border-0 shadow-md hover:shadow-lg transition-all duration-200"
            >
              下一步
            </Button>
          ) : (
            <Button
              type="primary"
              onClick={handleSubmit}
              size="large"
              loading={loading}
              icon={<CheckCircle size={16} />}
              className="rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 border-0 shadow-md hover:shadow-lg transition-all duration-200"
            >
              提交工单
            </Button>
          )}
        </div>
      </div>
    </Form>
  );
};

TicketForm.displayName = 'TicketForm';

// 编辑模式表单组件（提取为内部组件）
interface EditingFormProps {
  form: any;
  onSubmit: (values: any) => void;
  loading: boolean;
  userList: User[];
}

const EditingForm: React.FC<EditingFormProps> = ({
  form,
  onSubmit,
  loading,
  userList,
}) => {
  const { t } = useI18n();

  return (
    <Form layout="vertical" form={form} onFinish={onSubmit} style={{ marginTop: 20 }}>
      <Row gutter={24}>
        <Col span={12}>
          <Form.Item
            label="标题"
            name="title"
            rules={[{ required: true, message: '请输入工单标题' }]}
          >
            <Input placeholder="请输入工单标题" size="large" />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            label="类型"
            name="type"
            rules={[{ required: true, message: '请选择工单类型' }]}
          >
            <Select placeholder="请选择工单类型" size="large">
              <Select.Option value="incident">事件</Select.Option>
              <Select.Option value="service_request">服务请求</Select.Option>
              <Select.Option value="problem">问题</Select.Option>
              <Select.Option value="change">变更</Select.Option>
            </Select>
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={24}>
        <Col span={12}>
          <Form.Item
            label="分类"
            name="category"
            rules={[{ required: true, message: '请选择工单分类' }]}
          >
            <Select placeholder="请选择工单分类" size="large">
              <Select.Option value="System Access">系统访问</Select.Option>
              <Select.Option value="Hardware Equipment">硬件设备</Select.Option>
              <Select.Option value="Software Services">软件服务</Select.Option>
              <Select.Option value="Network Services">网络服务</Select.Option>
            </Select>
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item
            label="优先级"
            name="priority"
            rules={[{ required: true, message: '请选择优先级' }]}
          >
            <Select placeholder="请选择优先级" size="large">
              <Select.Option value="low">低</Select.Option>
              <Select.Option value="medium">中</Select.Option>
              <Select.Option value="high">高</Select.Option>
              <Select.Option value="urgent">紧急</Select.Option>
            </Select>
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={24}>
        <Col span={12}>
          <Form.Item label="处理人" name="assignee_id">
            <Select placeholder="请选择处理人" allowClear size="large">
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
          <Form.Item label="预计完成时间" name="estimated_time">
            <DatePicker
              showTime
              placeholder="请选择预计完成时间"
              size="large"
              style={{ width: '100%' }}
            />
          </Form.Item>
        </Col>
      </Row>

      <Form.Item
        label="描述"
        name="description"
        rules={[{ required: true, message: '请输入工单描述' }]}
      >
        <Input.TextArea rows={6} placeholder="请详细描述工单内容、问题、预期结果等..." />
      </Form.Item>

      <Form.Item style={{ marginBottom: 0 }}>
        <div className="flex justify-end gap-3 pt-6 border-t border-gray-100">
          <Button
            onClick={() => form.resetFields()}
            size="large"
            className="rounded-lg border-gray-200 hover:border-gray-300 transition-colors duration-200"
          >
            重置
          </Button>
          <Button
            type="primary"
            htmlType="submit"
            size="large"
            loading={loading}
            className="rounded-lg bg-gradient-to-r from-blue-500 to-blue-600 hover:from-blue-600 hover:to-blue-700 border-0 shadow-md hover:shadow-lg transition-all duration-200"
            icon={<Plus size={16} />}
          >
            保存修改
          </Button>
        </div>
      </Form.Item>
    </Form>
  );
};
