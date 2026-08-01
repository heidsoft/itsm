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
        message.warning('该发布已进入不可编辑状态，仅可查看');
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
      message.error('加载发布详情失败');
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
        message.success('更新成功');
      } else {
        await ReleaseApi.createRelease(data);
        message.success('创建成功');
      }
      router.push('/releases');
    } catch (error) {
      message.error(isEdit ? '更新失败' : '创建失败');
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
          message="该发布已进入不可编辑状态"
          description="已发布、已部署或已完成的发布不允许修改。"
          className="mb-4"
          action={
            <Button size="small" onClick={() => router.push(`/releases/${id}`)}>
              返回详情
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
            返回列表
          </Button>
        </div>

        <Divider>基本信息</Divider>

        <Form.Item
          name="releaseNumber"
          label="发布编号"
          rules={[{ required: true, message: '请输入发布编号' }]}
        >
          <Input
            placeholder="例如: REL-20260222-001"
            data-testid="release-number-input"
          />
        </Form.Item>

        <Form.Item
          name="title"
          label="标题"
          rules={[{ required: true, message: '请输入发布标题' }]}
        >
          <Input placeholder="发布标题" data-testid="release-title-input" />
        </Form.Item>

        <Form.Item name="description" label="描述">
          <TextArea rows={3} placeholder="发布描述" />
        </Form.Item>

        <Form.Item name="type" label="发布类型">
          <Select options={[
            { value: 'major', label: '主版本 (Major)' },
            { value: 'minor', label: '次版本 (Minor)' },
            { value: 'patch', label: '补丁 (Patch)' },
            { value: 'hotfix', label: '紧急修复 (Hotfix)' },
          ]} />
        </Form.Item>

        <Form.Item name="environment" label="目标环境">
          <Select options={[
            { value: 'dev', label: '开发环境' },
            { value: 'staging', label: '预发布环境' },
            { value: 'production', label: '生产环境' },
          ]} />
        </Form.Item>

        <Form.Item name="severity" label="严重程度">
          <Select options={[
            { value: 'low', label: '低' },
            { value: 'medium', label: '中' },
            { value: 'high', label: '高' },
            { value: 'critical', label: '严重' },
          ]} />
        </Form.Item>

        <Divider>计划信息</Divider>

        <Form.Item name="plannedReleaseDate" label="计划发布日期">
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="plannedStartDate" label="计划开始时间">
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="plannedEndDate" label="计划结束时间">
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>

        <Divider>发布内容</Divider>

        <Form.Item name="releaseNotes" label="发布说明">
          <TextArea rows={4} placeholder="发布说明内容" />
        </Form.Item>

        <Form.Item name="deploymentSteps" label="部署步骤">
          <TextArea rows={4} placeholder="每行一个步骤" />
        </Form.Item>

        <Form.Item name="affectedSystems" label="受影响的系统">
          <TextArea rows={2} placeholder="每行一个系统" />
        </Form.Item>

        <Form.Item name="affectedComponents" label="受影响的组件">
          <TextArea rows={2} placeholder="每行一个组件" />
        </Form.Item>

        <Divider>回滚与验证</Divider>

        <Form.Item name="rollbackProcedure" label="回滚程序">
          <TextArea rows={4} placeholder="回滚步骤说明" />
        </Form.Item>

        <Form.Item name="validationCriteria" label="验证标准">
          <TextArea rows={3} placeholder="验证通过的标准" />
        </Form.Item>

        <Divider>其他选项</Divider>

        <Form.Item name="isEmergency" label="紧急发布" valuePropName="checked">
          <Switch />
        </Form.Item>

        <Form.Item name="requiresApproval" label="需要审批" valuePropName="checked">
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
              {isEdit ? '保存' : '创建'}
            </Button>
            <Button onClick={() => router.push('/releases')}>取消</Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  );
};

export default ReleaseForm;
