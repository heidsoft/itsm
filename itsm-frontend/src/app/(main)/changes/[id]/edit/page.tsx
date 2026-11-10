'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Form,
  Input,
  Row,
  Select,
  Space,
  Typography,
} from 'antd';
import { ArrowLeft, Lock, Save } from 'lucide-react';
import dayjs, { type Dayjs } from 'dayjs';
import { AppDateRangePicker } from '@/components/ui/AppDatePicker';
import { ChangeApi, type ChangeRequest, type Change } from '@/lib/api/change-api';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;
const { TextArea } = Input;

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

// 不允许编辑的状态（已审批通过/实施中/已完成/已关闭）。
// 这些字面量必须逐字等于后端返回值（实施中是 in_progress，不存在 implementing）；
// 拼错不会报错，只会让只读守卫静默失效。
const READONLY_STATUSES: string[] = ['approved', 'in_progress', 'completed', 'closed'];

const EditChangePage: React.FC = () => {
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;
  const { t } = useI18n();
  const { message } = App.useApp();
  const [form] = Form.useForm<ChangeFormValues>();
  // 提交动作态（数据加载由 fetching 负责）
  const [actionLoading, setActionLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [changeData, setChangeData] = useState<Change | null>(null);
  const [isReadonly, setIsReadonly] = useState(false);

  // Fetch change data
  useEffect(() => {
    if (!id) return;

    const fetchChange = async () => {
      setFetching(true);
      try {
        const resp = await ChangeApi.getChange(Number(id));
        const data = resp;
        setChangeData(data);

        // 状态守卫：已审批/实施中/已完成的变更不允许编辑
        if (READONLY_STATUSES.includes(data.status)) {
          setIsReadonly(true);
          message.warning('The change has entered the uneditable state and can only be viewed');
          return;
        }

        // 解析 affectedCis
        const cisText = Array.isArray(data.affectedCis)
          ? data.affectedCis.join(', ')
          : data.affectedCis || '';

        // 解析计划时间
        const plannedRange: [Dayjs, Dayjs] | undefined =
          data.plannedStartDate && data.plannedEndDate
            ? [dayjs(data.plannedStartDate), dayjs(data.plannedEndDate)]
            : undefined;

        form.setFieldsValue({
          title: data.title,
          description: data.description,
          justification: data.justification,
          type: data.type,
          priority: data.priority,
          impactScope: data.impactScope,
          riskLevel: data.riskLevel,
          plannedRange,
          affectedCisText: cisText,
          implementationPlan: data.implementationPlan,
          rollbackPlan: data.rollbackPlan,
        });
      } catch (error) {
        message.error(t('changes.getFailed') || 'Failed to load change detail');
        router.push('/changes');
      } finally {
        setFetching(false);
      }
    };

    fetchChange();
  }, [id, form, router, message, t]);

  const handleSubmit = async (values: ChangeFormValues) => {
    if (!id) return;

    setActionLoading(true);
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
        relatedTickets: changeData?.relatedTickets || [],
      };

      await ChangeApi.updateChange(Number(id), payload);
      message.success(t('common.operationSuccess') || 'Change update successful');
      router.push(`/changes/${id}`);
    } catch (err) {
      console.error('Failed to update changes:', err);
      message.error(t('changes.createFailed') || 'Change update failed');
    } finally {
      setActionLoading(false);
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

          Return change details
        </Button>
        <Title level={2} className="!mb-1 !mt-2">

          Edit Change Request #{id}
        </Title>
        <Text type="secondary">Modify IT Infrastructure or Service Change Request</Text>
      </div>

      {/* 状态守卫提示 */}
      {isReadonly && (
        <Alert
          type="warning"
          showIcon
          icon={<Lock />}
          title="The change has entered the uneditable state"
          description="Changes that have been approved, implemented, or completed are not allowed to be modified. If you need to modify it, please initiate a change request."
          className="mb-4"
          action={
            <Button size="small" onClick={() => router.push(`/changes/${id}`)}>

              Return details
            </Button>
          }
        />
      )}

      <Card className="shadow-sm rounded-lg" loading={fetching}>
        <Form<ChangeFormValues>
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          disabled={actionLoading || isReadonly}
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
            <Input placeholder="Briefly describe the changes" allowClear />
          </Form.Item>

          <Form.Item
            label="Detailed description"
            name="description"
            rules={[{ required: true, message: 'Please fill in the detailed description' }]}
          >
            <TextArea
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
            <AppDateRangePicker
              showTime={{ format: 'HH:mm' }}
              disabledDate={(current: Dayjs) => !!current && current.isBefore(dayjs().startOf('day'))}
              placeholder={['Start Time', 'End Time']}
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
              rows={5}
              placeholder="Detailed description of how to rollback if changes fail..."
              showCount
              maxLength={3000}
            />
          </Form.Item>

          <Form.Item className="!mb-0 mt-4">
            <Space className="w-full justify-end">
              <Button onClick={() => router.back()} disabled={actionLoading}>
                Cancel
              </Button>
              {!isReadonly && (
                <Button type="primary" htmlType="submit" loading={actionLoading} icon={<Save size={16} />}>
                  Save Changes
                </Button>
              )}
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default EditChangePage;
