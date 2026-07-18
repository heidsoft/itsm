'use client';

import React, { useState } from 'react';
import { Card, Form, Input, Select, Button, App, Space } from 'antd';
import { ArrowLeft, Save } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { TicketApi } from '@/lib/api/ticket-api';
import { useI18n } from '@/lib/i18n';

const NewImprovementPage = () => {
  const router = useRouter();
  const { message } = App.useApp();
  const { t } = useI18n();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const onFinish = async (values: { title: string; description?: string; priority?: string }) => {
    try {
      setLoading(true);
      await TicketApi.createTicket({
        type: 'improvement',
        title: values.title,
        description: values.description || '',
        priority: values.priority || 'medium',
      } as unknown as Parameters<typeof TicketApi.createTicket>[0]);
      message.success('改进计划创建成功');
      router.push('/improvements');
    } catch (error) {
      console.error('Create improvement failed:', error);
      message.error('创建失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <Card>
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Button icon={<ArrowLeft />} onClick={() => router.push('/improvements')} type="text">
            返回列表
          </Button>
          <h2 className="text-2xl font-bold text-gray-800">新建改进计划</h2>

          <Form form={form} layout="vertical" onFinish={onFinish}>
            <Form.Item
              label="标题"
              name="title"
              rules={[{ required: true, message: '请输入改进计划标题' }]}
            >
              <Input placeholder="请输入改进计划标题" maxLength={200} />
            </Form.Item>

            <Form.Item label="优先级" name="priority" initialValue="medium">
              <Select
                options={[
                  { label: '低', value: 'low' },
                  { label: '中', value: 'medium' },
                  { label: '高', value: 'high' },
                  { label: '紧急', value: 'critical' },
                ]}
                placeholder="请选择优先级"
              />
            </Form.Item>

            <Form.Item label="目标描述" name="description">
              <Input.TextArea rows={4} placeholder="详细描述改进目标、范围和预期效果" />
            </Form.Item>

            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  htmlType="submit"
                  icon={<Save />}
                  loading={loading}
                >
                  保存
                </Button>
                <Button onClick={() => router.push('/improvements')}>{t('common.cancel')}</Button>
              </Space>
            </Form.Item>
          </Form>
        </Space>
      </Card>
    </div>
  );
};

export default NewImprovementPage;