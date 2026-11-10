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
      message.success('Improvement plan created successfully');
      router.push('/improvements');
    } catch (error) {
      console.error('Create improvement failed:', error);
      message.error('Creation failed, please try again later');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <Card>
        <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
          <Button icon={<ArrowLeft />} onClick={() => router.push('/improvements')} type="text">

            Back to List
          </Button>
          <h2 className="text-2xl font-bold text-gray-800">Create a new improvement plan</h2>

          <Form form={form} layout="vertical" onFinish={onFinish}>
            <Form.Item
              label="Title"
              name="title"
              rules={[{ required: true, message: 'Please enter the title of the improvement plan' }]}
            >
              <Input placeholder="Please enter the title of the improvement plan" maxLength={200} />
            </Form.Item>

            <Form.Item label="Priority" name="priority" initialValue="medium">
              <Select
                options={[
                  { label: 'Low', value: 'low' },
                  { label: 'Medium', value: 'medium' },
                  { label: 'High', value: 'high' },
                  { label: 'Urgent', value: 'critical' },
                ]}
                placeholder="Please select priority"
              />
            </Form.Item>

            <Form.Item label="Description" name="description">
              <Input.TextArea rows={4} placeholder="Detailed description of improvement goals, scope and expected effects" />
            </Form.Item>

            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  htmlType="submit"
                  icon={<Save />}
                  loading={loading}
                >

                  Save
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