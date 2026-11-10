'use client';

/**
 * 发布创建/编辑表单组件
 */

import React, { useState, useEffect } from 'react';
import {
  Alert,
  Card,
  Form,
  Input,
  Select,
  DatePicker,
  Button,
  Space,
  Switch,
  Divider,
  message,
  InputNumber,
} from 'antd';
import { useRouter, useParams } from 'next/navigation';
import dayjs from 'dayjs';
import { ArrowLeft, Lock, Save } from 'lucide-react';

import type { Release, ReleaseRequest } from '@/lib/api/release-api';
import { ReleaseApi } from '@/lib/api/release-api';
import type { Dayjs } from 'dayjs';

const { TextArea } = Input;

interface ReleaseFormValues {
  releaseNumber: string;
  title: string;
  description?: string;
  type?: Release['type'];
  environment?: Release['environment'];
  severity?: Release['severity'];
  changeId?: number;
  ownerId?: number;
  plannedReleaseDate?: Dayjs;
  plannedStartDate?: Dayjs;
  plannedEndDate?: Dayjs;
  releaseNotes?: string;
  rollbackProcedure?: string;
  validationCriteria?: string;
  affectedSystems?: string;
  affectedComponents?: string;
  deploymentSteps?: string;
  tags?: string[];
  isEmergency?: boolean;
  requiresApproval?: boolean;
}

const splitLines = (value?: string): string[] | undefined => {
  const items = value
    ?.split(/\r?\n/)
    .map(item => item.trim())
    .filter(Boolean);
  return items?.length ? items : undefined;
};

// 不允许编辑的状态列表（已发布/已部署/已完成）
const READONLY_STATUSES = ['released', 'deployed', 'completed', 'cancelled'];

const ReleaseForm: React.FC = () => {
  const router = useRouter();
  const { id } = useParams() as { id: string };
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState<Release | null>(null);
  const [isReadonly, setIsReadonly] = useState(false);
  const isEdit = !!id;

  useEffect(() => {
    if (id) {
      loadDetail();
    }
  }, [id]);

  const loadDetail = async () => {
    setLoading(true);
    try {
      const data = await ReleaseApi.getRelease(Number(id));
      setDetail(data);

      // 状态守卫：已发布/已部署/已完成的发布不允许编辑
      if (READONLY_STATUSES.includes(data.status)) {
        setIsReadonly(true);
        message.warning('This publication has entered a non-editable state and can only be viewed');
        return;
      }

      // 设置表单值
      form.setFieldsValue({
        ...data,
        plannedReleaseDate: data.plannedReleaseDate
          ? dayjs(data.plannedReleaseDate)
          : undefined,
        plannedStartDate: data.plannedStartDate ? dayjs(data.plannedStartDate) : undefined,
        plannedEndDate: data.plannedEndDate ? dayjs(data.plannedEndDate) : undefined,
        deploymentSteps: data.deploymentSteps?.join('\n'),
        affectedSystems: data.affectedSystems?.join('\n'),
        affectedComponents: data.affectedComponents?.join('\n'),
      });
    } catch (error) {
      message.error('Failed to load publication details');
    } finally {
      setLoading(false);
    }
  };

  const onFinish = async (values: ReleaseFormValues) => {
    setLoading(true);
    try {
      const data: ReleaseRequest = {
        releaseNumber: values.releaseNumber,
        title: values.title,
        description: values.description,
        type: values.type,
        environment: values.environment,
        severity: values.severity,
        changeId: values.changeId,
        ownerId: values.ownerId,
        plannedReleaseDate: values.plannedReleaseDate?.toISOString(),
        plannedStartDate: values.plannedStartDate?.toISOString(),
        plannedEndDate: values.plannedEndDate?.toISOString(),
        releaseNotes: values.releaseNotes,
        rollbackProcedure: values.rollbackProcedure,
        validationCriteria: values.validationCriteria,
        affectedSystems: splitLines(values.affectedSystems),
        affectedComponents: splitLines(values.affectedComponents),
        deploymentSteps: splitLines(values.deploymentSteps),
        tags: values.tags,
        isEmergency: values.isEmergency,
        requiresApproval: values.requiresApproval,
      };

      if (isEdit) {
        await ReleaseApi.updateRelease(Number(id), data);
        message.success('Updated');
      } else {
        await ReleaseApi.createRelease(data);
        message.success('Created');
      }
      router.push('/releases');
    } catch (error) {
      message.error(isEdit ? 'Update failed' : 'Creation failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card>
      {/* 状态守卫提示 */}
      {isReadonly && (
        <Alert
          type="warning"
          showIcon
          icon={<Lock />}
          title="The publication has entered a non-editable state"
          description="Modification is not allowed for published, deployed or completed publications."
          className="mb-4"
          action={
            <Button size="small" onClick={() => router.push(`/releases/${id}`)}>

              Return details
            </Button>
          }
        />
      )}

      <Form
        form={form}
        data-testid="release-form"
        layout="vertical"
        onFinish={onFinish}
        disabled={isReadonly}
        initialValues={{
          type: 'minor',
          environment: 'staging',
          severity: 'medium',
          isEmergency: false,
          requiresApproval: true,
        }}
      >
        <div style={{ marginBottom: 16 }}>
          <Button icon={<ArrowLeft />} onClick={() => router.push('/releases')}>

            Back to List
          </Button>
        </div>

        <Divider>Basic Info</Divider>

        <Form.Item
          name="releaseNumber"
          label="Release number"
          rules={[{ required: true, message: 'Please enter the release number' }]}
        >
          <Input
            placeholder="Example: REL-20260222-001"
            data-testid="release-number-input"
          />
        </Form.Item>

        <Form.Item
          name="title"
          label="Title"
          rules={[{ required: true, message: 'Please enter a publication title' }]}
        >
          <Input placeholder="Post title" data-testid="release-title-input" />
        </Form.Item>

        <Form.Item name="description" label="Description">
          <TextArea rows={3} placeholder="Release description" />
        </Form.Item>

        <Form.Item name="type" label="Release type">
          <Select options={[
            { value: 'major', label: 'Major version' },
            { value: 'minor', label: 'Minor version (Minor)' },
            { value: 'patch', label: 'Patch' },
            { value: 'hotfix', label: 'Hotfix' },
          ]} />
        </Form.Item>

        <Form.Item name="environment" label="Target environment">
          <Select options={[
            { value: 'dev', label: 'Development environment' },
            { value: 'staging', label: 'Pre-release environment' },
            { value: 'production', label: 'Production environment' },
          ]} />
        </Form.Item>

        <Form.Item name="severity" label="Severity">
          <Select options={[
            { value: 'low', label: 'Low' },
            { value: 'medium', label: 'Medium' },
            { value: 'high', label: 'High' },
            { value: 'critical', label: 'Critical' },
          ]} />
        </Form.Item>

        <Divider>Plan information</Divider>

        <Form.Item name="plannedReleaseDate" label="Planned release date">
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="plannedStartDate" label="Planned start time">
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="plannedEndDate" label="Planned end time">
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>

        <Divider>Post content</Divider>

        <Form.Item name="releaseNotes" label="Release Notes">
          <TextArea rows={4} placeholder="Release Notes Content" />
        </Form.Item>

        <Form.Item name="deploymentSteps" label="Deployment steps">
          <TextArea rows={4} placeholder="One step per line" />
        </Form.Item>

        <Form.Item name="affectedSystems" label="Affected systems">
          <TextArea rows={2} placeholder="One system per line" />
        </Form.Item>

        <Form.Item name="affectedComponents" label="Affected components">
          <TextArea rows={2} placeholder="One component per row" />
        </Form.Item>

        <Divider>Rollback and Verification</Divider>

        <Form.Item name="rollbackProcedure" label="Rollback procedure">
          <TextArea rows={4} placeholder="Rollback step instructions" />
        </Form.Item>

        <Form.Item name="validationCriteria" label="Verification Standard">
          <TextArea rows={3} placeholder="Verification passed standard" />
        </Form.Item>

        <Divider>Other options</Divider>

        <Form.Item name="isEmergency" label="Urgent release" valuePropName="checked">
          <Switch />
        </Form.Item>

        <Form.Item name="requiresApproval" label="Approval required" valuePropName="checked">
          <Switch />
        </Form.Item>

        <Form.Item>
          <Space>
            <Button
              type="primary"
              htmlType="submit"
              icon={<Save />}
              loading={loading}
              data-testid="release-submit-button"
            >
              {isEdit ? 'Save' : 'Create'}
            </Button>
            <Button onClick={() => router.push('/releases')}>Cancel</Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );
};

export default ReleaseForm;
