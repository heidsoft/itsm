'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Card, Form, Input, Select, Button, Space, Typography, App } from 'antd';
import { ArrowLeft } from 'lucide-react';
import { TicketApi } from '@/lib/api/ticket-api';

const { Title, Text } = Typography;
const { TextArea } = Input;

type Priority = 'low' | 'medium' | 'high' | 'urgent';

export default function CreateTicketPage() {
  const router = useRouter();
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);

      // 检查描述长度（后端要求最少10个字符）
      if (values.description && values.description.length < 10) {
        message.warning('描述至少需要10个字符，请详细描述问题');
        setLoading(false);
        return;
      }

      const created = await TicketApi.createTicket({
        title: values.title,
        description: values.description,
        priority: values.priority,
        category: values.category,
      } as any);

      message.success('工单创建成功');
      router.push(`/tickets/${created.id}`);
    } catch (e: any) {
      console.error('Create ticket error:', e);
      // 提取更详细的错误信息
      const errorMsg = e?.message || e?.error?.message || '创建工单失败，请检查输入或重新登录';
      message.error(errorMsg);
    } finally {
      setLoading(false);
    }
  };

        return (
    <div className='max-w-4xl mx-auto p-6'>
      <Space orientation='vertical' size={16} style={{ width: '100%' }}>
        <Card>
          <Space align='center' style={{ width: '100%' }}>
            <Button icon={<ArrowLeft className='w-4 h-4' />} onClick={() => router.back()}>
              返回
            </Button>
            <div style={{ flex: 1 }}>
              <Title level={4} style={{ marginBottom: 4 }}>
                新建工单
              </Title>
              <Text type='secondary'>填写工单信息后提交，服务团队将尽快处理</Text>
            </div>
          </Space>
        </Card>

        <Card>
          <Form form={form} layout='vertical'>
            <Form.Item name='title' label='标题' rules={[{ required: true, message: '请输入标题', min: 2 }]}>
              <Input placeholder='例如：VPN 无法连接' />
                  </Form.Item>

            <Form.Item name='description' label='描述' rules={[{ required: true, message: '请输入描述（至少10个字符）' }, { min: 10, message: '描述至少需要10个字符' }]}>
              <TextArea rows={6} placeholder='请详细描述问题/需求与影响范围...' />
                  </Form.Item>

            <Form.Item name='priority' label='优先级' initialValue='medium' rules={[{ required: true }]}>
              <Select<Priority>
                options={[
                  { label: '低', value: 'low' },
                  { label: '中', value: 'medium' },
                  { label: '高', value: 'high' },
                  { label: '紧急', value: 'urgent' },
                ]}
              />
            </Form.Item>

            <Form.Item name='category' label='分类' initialValue='网络'>
              <Select
                options={[
                  { label: '网络', value: '网络' },
                  { label: '性能', value: '性能' },
                  { label: '安全', value: '安全' },
                  { label: '存储', value: '存储' },
                  { label: '连接', value: '连接' },
                ]}
              />
            </Form.Item>

          <Space>
              <Button type='primary' onClick={handleSubmit} loading={loading}>
                创建
              </Button>
              <Button onClick={() => router.push('/tickets')}>取消</Button>
          </Space>
          </Form>
      </Card>
      </Space>
    </div>
  );
}


