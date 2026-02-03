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

      const created = await TicketApi.createTicket({
        title: values.title,
        description: values.description,
        priority: values.priority,
        category: values.category,
      } as any);

      message.success('工单创建成功');
      router.push(`/tickets/${created.id}`);
    } catch (e) {
      console.error(e);
      message.error(e instanceof Error ? e.message : '创建工单失败');
    } finally {
      setLoading(false);
    }
  };

        return (
    <div className='max-w-4xl mx-auto p-6'>
      <Space orientation='vertical' size={16} style={{ width: '100%' }}>
        <Card>
          <Space align='center'>
            <Button icon={<ArrowLeft className='w-4 h-4' />} onClick={() => router.back()}>
              返回
            </Button>
            <div>
              <Title level={2} style={{ marginBottom: 0 }}>
                新建工单
              </Title>
              <Text type='secondary'>P0 轻量创建页（仅保留核心字段），避免引入 legacy 依赖</Text>
            </div>
          </Space>
        </Card>

        <Card>
          <Form form={form} layout='vertical'>
            <Form.Item name='title' label='标题' rules={[{ required: true, message: '请输入标题' }]}>
              <Input placeholder='例如：VPN 无法连接' />
                  </Form.Item>

            <Form.Item name='description' label='描述' rules={[{ required: true, message: '请输入描述' }]}>
              <TextArea rows={6} placeholder='请详细描述问题/需求与影响范围...' />
                  </Form.Item>

            <Form.Item name='priority' label='优先级' initialValue='medium' rules={[{ required: true }]}>
              <Select<Priority>>
                <Select.Option value='low'>低</Select.Option>
                <Select.Option value='medium'>中</Select.Option>
                <Select.Option value='high'>高</Select.Option>
                <Select.Option value='urgent'>紧急</Select.Option>
                    </Select>
                  </Form.Item>

            <Form.Item name='category' label='分类' initialValue='网络'>
              <Select>
                <Select.Option value='网络'>网络</Select.Option>
                <Select.Option value='性能'>性能</Select.Option>
                <Select.Option value='安全'>安全</Select.Option>
                <Select.Option value='存储'>存储</Select.Option>
                <Select.Option value='连接'>连接</Select.Option>
                    </Select>
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


