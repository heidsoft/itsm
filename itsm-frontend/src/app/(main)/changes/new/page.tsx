'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  App,
  Button,
  Card,
  Col,
  DatePicker,
  Form,
  Input,
  Row,
  Select,
  Space,
  Typography,
} from 'antd';
import { ArrowLeft } from 'lucide-react';
import dayjs, { type Dayjs } from 'dayjs';
import { ChangeApi, type ChangeRequest } from '@/lib/api/change-api';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;
const { TextArea } = Input;

// 表单值类型：DatePicker 用 Dayjs，affectedCis 用字符串
interface ChangeFormValues {
  title: string;
  description: string;
  justification: string;
  type: ChangeRequest['type'];
  priority: ChangeRequest['priority'];
  impactScope: ChangeRequest['impactScope'];
  riskLevel: ChangeRequest['riskLevel'];
  plannedRange?: [Dayjs, Dayjs];
  affectedCisText?: string;
  implementationPlan: string;
  rollbackPlan: string;
}

const TYPE_OPTIONS: Array<{ value: ChangeRequest['type']; label: string; hint?: string }> = [
  { value: 'normal', label: 'Normal Change', hint: 'Standard Approval Process' },
  { value: 'standard', label: 'Standard Change', hint: 'Pre-authorized, low risk' },
  { value: 'emergency', label: 'Emergency Change', hint: 'Emergency approval' },
];

const PRIORITY_OPTIONS: Array<{ value: ChangeRequest['priority']; label: string }> = [
  { value: 'critical', label: 'Urgent' },
  { value: 'high', label: 'High' },
  { value: 'medium', label: 'Medium' },
  { value: 'low', label: 'Low' },
];

const IMPACT_OPTIONS: Array<{ value: ChangeRequest['impactScope']; label: string }> = [
  { value: 'high', label: 'High (affects core business)' },
  { value: 'medium', label: 'Medium (affects some services or users)' },
  { value: 'low', label: 'Low (little or no impact)' },
];

const RISK_OPTIONS: Array<{ value: ChangeRequest['riskLevel']; label: string }> = [
  { value: 'high', label: 'High' },
  { value: 'medium', label: 'Medium' },
  { value: 'low', label: 'Low' },
];

const CreateChangePage: React.FC = () => {
  const router = useRouter();
  const { t } = useI18n();
  const { message } = App.useApp();
  const [form] = Form.useForm<ChangeFormValues>();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (values: ChangeFormValues) => {
    setLoading(true);
    try {
      const [start, end] = values.plannedRange ?? [];
      const affectedCis =
        values.affectedCisText
          ?.split(/[,，\s]+/)
          .map(s => s.trim())
          .filter(Boolean) ?? [];

      const payload: ChangeRequest = {
        title: values.title.trim(),
        description: values.description.trim(),
        justification: values.justification.trim(),
        type: values.type,
        priority: values.priority,
        impactScope: values.impactScope,
        riskLevel: values.riskLevel,
        plannedStartDate: start ? start.toISOString() : undefined,
        plannedEndDate: end ? end.toISOString() : undefined,
        implementationPlan: values.implementationPlan.trim(),
        rollbackPlan: values.rollbackPlan.trim(),
        affectedCis,
        relatedTickets: [],
      };

      await ChangeApi.createChange(payload);
      message.success(t('changes.createSuccess'));
      router.push('/changes');
    } catch (err) {
      console.error('Failed to commit changes:', err);
      message.error(t('changes.createFailed'));
    } finally {
      setLoading(false);
    }
  };

  const validateRange = (_: unknown, value?: [Dayjs, Dayjs]) => {
    if (!value || !value[0] || !value[1]) return Promise.resolve();
    if (value[1].isBefore(value[0])) {
      return Promise.reject(new Error('End time must be later than start time'));
    }
    return Promise.resolve();
  };

  return (
    <div className="p-6 md:p-10 bg-gray-50 min-h-full">
      <div className="mb-6">
        <Button
          type="link"
          icon={<ArrowLeft size={16} />}
          onClick={() => router.back()}
          className="!px-0"
        >

          Back to change list
        </Button>
        <Title level={2} className="!mb-1 !mt-2">

          New change request
        </Title>
        <Text type="secondary">Submit a new IT infrastructure or service change request</Text>
      </div>

      <Card className="shadow-sm rounded-lg">
        <Form<ChangeFormValues>
          form={form}
          data-testid="change-create-form"
          layout="vertical"
          initialValues={{
            type: 'normal',
            priority: 'medium',
            impactScope: 'medium',
            riskLevel: 'medium',
          }}
          onFinish={handleSubmit}
          disabled={loading}
          scrollToFirstError
        >
          <Form.Item
            label="Change title"
            name="title"
            rules={[
              { required: true, message: 'Please enter a change title' },
              { max: 200, message: 'Title should not exceed 200 words' },
            ]}
          >
            <Input
              placeholder="Briefly describe the changes"
              allowClear
              data-testid="change-title-input"
            />
          </Form.Item>

          <Form.Item
            label="Detailed description"
            name="description"
            rules={[{ required: true, message: 'Please fill in the detailed description' }]}
          >
            <TextArea
              data-testid="change-description-input"
              rows={4}
              placeholder="Please explain in detail the purpose, scope and content of the change..."
              showCount
              maxLength={2000}
            />
          </Form.Item>

          <Form.Item
            label="Reason for change"
            name="justification"
            tooltip="Explain why this change is needed, such as associated issues or business needs"
            rules={[{ required: true, message: 'Please fill in the reason for the change' }]}
          >
            <TextArea
              data-testid="change-justification-input"
              rows={3}
              placeholder="For example: solve the problem PRB-XXXXX, meet new business needs, etc."
              showCount
              maxLength={1000}
            />
          </Form.Item>

          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Item
                label="Change Type"
                name="type"
                rules={[{ required: true, message: 'Please select a change type' }]}
              >
                <Select
                  options={TYPE_OPTIONS.map(o => ({
                    value: o.value,
                    label: (
                      <span>
                        {o.label}
                        {o.hint && (
                          <Text type="secondary" className="ml-2 text-xs">
                            {o.hint}
                          </Text>
                        )}
                      </span>
                    ),
                  }))}
                />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item
                label="Priority"
                name="priority"
                rules={[{ required: true, message: 'Please select priority' }]}
              >
                <Select options={PRIORITY_OPTIONS} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Item
                label="Impact Range"
                name="impactScope"
                rules={[{ required: true, message: 'Please select impact scope' }]}
              >
                <Select options={IMPACT_OPTIONS} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item
                label="Risk Level"
                name="riskLevel"
                rules={[{ required: true, message: 'Please select a risk level' }]}
              >
                <Select options={RISK_OPTIONS} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label="Scheduled Date"
            name="plannedRange"
            tooltip="Planned start and end time"
            rules={[{ validator: validateRange }]}
          >
            <DatePicker.RangePicker
              showTime={{ format: 'HH:mm' }}
              format="YYYY-MM-DD HH:mm"
              disabledDate={current => !!current && current.isBefore(dayjs().startOf('day'))}
              placeholder={['Start Time', 'End Time']}
              className="w-full"
            />
          </Form.Item>

          <Form.Item
            label="Affected configuration items"
            name="affectedCisText"
            tooltip="Multiple CIs separated by commas or spaces"
          >
            <Input placeholder="For example: CI-ECS-001, CI-APP-CRM" allowClear />
          </Form.Item>

          <Form.Item
            label="Implementation Plan"
            name="implementationPlan"
            rules={[{ required: true, message: 'Please fill in the implementation plan' }]}
          >
            <TextArea
              data-testid="change-implementation-input"
              rows={5}
              placeholder="Describe in detail the implementation steps of the change..."
              showCount
              maxLength={3000}
            />
          </Form.Item>

          <Form.Item
            label="Rollback Plan"
            name="rollbackPlan"
            tooltip="How to roll back when changes fail"
            rules={[{ required: true, message: 'Please fill in the rollback plan' }]}
          >
            <TextArea
              data-testid="change-rollback-input"
              rows={5}
              placeholder="Detailed description of how to rollback if changes fail..."
              showCount
              maxLength={3000}
            />
          </Form.Item>

          <Form.Item className="!mb-0 mt-4">
            <Space className="w-full justify-end">
              <Button onClick={() => router.back()} disabled={loading}>

                Cancel
              </Button>
              <Button
                type="primary"
                htmlType="submit"
                loading={loading}
                data-testid="change-submit-button"
              >

                Submit change request
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default CreateChangePage;
