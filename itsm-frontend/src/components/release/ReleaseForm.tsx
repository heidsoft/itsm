'use client';

/**
 * 发布创建/编辑表单组件
 */

import React, { useState, useEffect } from 'react';
import {
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
import { ArrowLeftOutlined, SaveOutlined } from '@ant-design/icons';

import { ReleaseApi, Release, ReleaseRequest } from '@/lib/api/release-api';

const { TextArea } = Input;
const { Option } = Select;

const ReleaseForm: React.FC = () => {
  const router = useRouter();
  const { id } = useParams() as { id: string };
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [detail, setDetail] = useState<Release | null>(null);
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
      // 设置表单值
      form.setFieldsValue({
        ...data,
        planned_release_date: data.planned_release_date
          ? dayjs(data.planned_release_date)
          : undefined,
        planned_start_date: data.planned_start_date ? dayjs(data.planned_start_date) : undefined,
        planned_end_date: data.planned_end_date ? dayjs(data.planned_end_date) : undefined,
      });
    } catch (error) {
      message.error('加载发布详情失败');
    } finally {
      setLoading(false);
    }
  };

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      const data: ReleaseRequest = {
        release_number: values.release_number,
        title: values.title,
        description: values.description,
        type: values.type,
        environment: values.environment,
        severity: values.severity,
        change_id: values.change_id,
        owner_id: values.owner_id,
        planned_release_date: values.planned_release_date?.toISOString(),
        planned_start_date: values.planned_start_date?.toISOString(),
        planned_end_date: values.planned_end_date?.toISOString(),
        release_notes: values.release_notes,
        rollback_procedure: values.rollback_procedure,
        validation_criteria: values.validation_criteria,
        affected_systems: values.affected_systems,
        affected_components: values.affected_components,
        deployment_steps: values.deployment_steps,
        tags: values.tags,
        is_emergency: values.is_emergency,
        requires_approval: values.requires_approval,
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
      <Form
        form={form}
        layout="vertical"
        onFinish={onFinish}
        initialValues={{
          type: 'minor',
          environment: 'staging',
          severity: 'medium',
          is_emergency: false,
          requires_approval: true,
        }}
      >
        <div style={{ marginBottom: 16 }}>
          <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/releases')}>
            返回列表
          </Button>
        </div>

        <Divider>基本信息</Divider>

        <Form.Item
          name="release_number"
          label="发布编号"
          rules={[{ required: true, message: '请输入发布编号' }]}
        >
          <Input placeholder="例如: REL-20260222-001" />
        </Form.Item>

        <Form.Item
          name="title"
          label="标题"
          rules={[{ required: true, message: '请输入发布标题' }]}
        >
          <Input placeholder="发布标题" />
        </Form.Item>

        <Form.Item name="description" label="描述">
          <TextArea rows={3} placeholder="发布描述" />
        </Form.Item>

        <Form.Item name="type" label="发布类型">
          <Select>
            <Option value="major">主版本 (Major)</Option>
            <Option value="minor">次版本 (Minor)</Option>
            <Option value="patch">补丁 (Patch)</Option>
            <Option value="hotfix">紧急修复 (Hotfix)</Option>
          </Select>
        </Form.Item>

        <Form.Item name="environment" label="目标环境">
          <Select>
            <Option value="dev">开发环境</Option>
            <Option value="staging">预发布环境</Option>
            <Option value="production">生产环境</Option>
          </Select>
        </Form.Item>

        <Form.Item name="severity" label="严重程度">
          <Select>
            <Option value="low">低</Option>
            <Option value="medium">中</Option>
            <Option value="high">高</Option>
            <Option value="critical">严重</Option>
          </Select>
        </Form.Item>

        <Divider>计划信息</Divider>

        <Form.Item name="planned_release_date" label="计划发布日期">
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="planned_start_date" label="计划开始时间">
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="planned_end_date" label="计划结束时间">
          <DatePicker showTime style={{ width: '100%' }} />
        </Form.Item>

        <Divider>发布内容</Divider>

        <Form.Item name="release_notes" label="发布说明">
          <TextArea rows={4} placeholder="发布说明内容" />
        </Form.Item>

        <Form.Item name="deployment_steps" label="部署步骤">
          <TextArea rows={4} placeholder="每行一个步骤" />
        </Form.Item>

        <Form.Item name="affected_systems" label="受影响的系统">
          <TextArea rows={2} placeholder="每行一个系统" />
        </Form.Item>

        <Form.Item name="affected_components" label="受影响的组件">
          <TextArea rows={2} placeholder="每行一个组件" />
        </Form.Item>

        <Divider>回滚与验证</Divider>

        <Form.Item name="rollback_procedure" label="回滚程序">
          <TextArea rows={4} placeholder="回滚步骤说明" />
        </Form.Item>

        <Form.Item name="validation_criteria" label="验证标准">
          <TextArea rows={3} placeholder="验证通过的标准" />
        </Form.Item>

        <Divider>其他选项</Divider>

        <Form.Item name="is_emergency" label="紧急发布" valuePropName="checked">
          <Switch />
        </Form.Item>

        <Form.Item name="requires_approval" label="需要审批" valuePropName="checked">
          <Switch />
        </Form.Item>

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={loading}>
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
