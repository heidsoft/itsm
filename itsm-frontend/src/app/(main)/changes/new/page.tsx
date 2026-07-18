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
  { value: 'normal', label: '普通变更', hint: '标准审批流程' },
  { value: 'standard', label: '标准变更', hint: '预授权、低风险' },
  { value: 'emergency', label: '紧急变更', hint: '走应急审批' },
];

const PRIORITY_OPTIONS: Array<{ value: ChangeRequest['priority']; label: string }> = [
  { value: 'critical', label: '紧急' },
  { value: 'high', label: '高' },
  { value: 'medium', label: '中' },
  { value: 'low', label: '低' },
];

const IMPACT_OPTIONS: Array<{ value: ChangeRequest['impactScope']; label: string }> = [
  { value: 'high', label: '高（影响核心业务）' },
  { value: 'medium', label: '中（影响部分业务或用户）' },
  { value: 'low', label: '低（影响较小或无影响）' },
];

const RISK_OPTIONS: Array<{ value: ChangeRequest['riskLevel']; label: string }> = [
  { value: 'high', label: '高' },
  { value: 'medium', label: '中' },
  { value: 'low', label: '低' },
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
      console.error('提交变更失败:', err);
      message.error(t('changes.createFailed'));
    } finally {
      setLoading(false);
    }
  };

  const validateRange = (_: unknown, value?: [Dayjs, Dayjs]) => {
    if (!value || !value[0] || !value[1]) return Promise.resolve();
    if (value[1].isBefore(value[0])) {
      return Promise.reject(new Error('结束时间必须晚于开始时间'));
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
          返回变更列表
        </Button>
        <Title level={2} className="!mb-1 !mt-2">
          新建变更请求
        </Title>
        <Text type="secondary">提交新的 IT 基础设施或服务变更请求</Text>
      </div>

      <Card className="shadow-sm rounded-lg">
        <Form<ChangeFormValues>
          form={form}
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
            label="变更标题"
            name="title"
            rules={[
              { required: true, message: '请输入变更标题' },
              { max: 200, message: '标题不超过 200 字' },
            ]}
          >
            <Input placeholder="简要描述变更内容" allowClear />
          </Form.Item>

          <Form.Item
            label="详细描述"
            name="description"
            rules={[{ required: true, message: '请填写详细描述' }]}
          >
            <TextArea
              rows={4}
              placeholder="请详细说明变更的目的、范围和内容..."
              showCount
              maxLength={2000}
            />
          </Form.Item>

          <Form.Item
            label="变更理由"
            name="justification"
            tooltip="解释为什么需要此变更，例如关联的问题或业务需求"
            rules={[{ required: true, message: '请填写变更理由' }]}
          >
            <TextArea
              rows={3}
              placeholder="例如：解决问题 PRB-XXXXX，满足新业务需求等"
              showCount
              maxLength={1000}
            />
          </Form.Item>

          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Item
                label="变更类型"
                name="type"
                rules={[{ required: true, message: '请选择变更类型' }]}
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
                label="优先级"
                name="priority"
                rules={[{ required: true, message: '请选择优先级' }]}
              >
                <Select options={PRIORITY_OPTIONS} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col xs={24} md={12}>
              <Form.Item
                label="影响范围"
                name="impactScope"
                rules={[{ required: true, message: '请选择影响范围' }]}
              >
                <Select options={IMPACT_OPTIONS} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item
                label="风险等级"
                name="riskLevel"
                rules={[{ required: true, message: '请选择风险等级' }]}
              >
                <Select options={RISK_OPTIONS} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            label="计划实施时间"
            name="plannedRange"
            tooltip="计划开始与结束时间"
            rules={[{ validator: validateRange }]}
          >
            <DatePicker.RangePicker
              showTime={{ format: 'HH:mm' }}
              format="YYYY-MM-DD HH:mm"
              disabledDate={current => !!current && current.isBefore(dayjs().startOf('day'))}
              placeholder={['开始时间', '结束时间']}
              className="w-full"
            />
          </Form.Item>

          <Form.Item
            label="受影响的配置项"
            name="affectedCisText"
            tooltip="多个 CI 用逗号或空格分隔"
          >
            <Input placeholder="例如：CI-ECS-001, CI-APP-CRM" allowClear />
          </Form.Item>

          <Form.Item
            label="实施计划"
            name="implementationPlan"
            rules={[{ required: true, message: '请填写实施计划' }]}
          >
            <TextArea
              rows={5}
              placeholder="详细描述变更的实施步骤..."
              showCount
              maxLength={3000}
            />
          </Form.Item>

          <Form.Item
            label="回滚计划"
            name="rollbackPlan"
            tooltip="变更失败时如何回退"
            rules={[{ required: true, message: '请填写回滚计划' }]}
          >
            <TextArea
              rows={5}
              placeholder="详细描述如果变更失败如何回滚..."
              showCount
              maxLength={3000}
            />
          </Form.Item>

          <Form.Item className="!mb-0 mt-4">
            <Space className="w-full justify-end">
              <Button onClick={() => router.back()} disabled={loading}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" loading={loading}>
                提交变更请求
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default CreateChangePage;
