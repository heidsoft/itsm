'use client';

/**
 * 服务目录申请页面
 * Bug 2 修复：原本 /service-catalog/request/[id] 路由 404
 * B10 修复：表单加上 compliance_ack / expire_at / delivery_time
 */

import React, { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import {
  Card,
  Form,
  Input,
  Button,
  Space,
  message,
  Typography,
  Breadcrumb,
  Checkbox,
  DatePicker,
  Select,
  Spin,
  Alert,
  Tag,
  Divider,
} from 'antd';
import { ArrowLeftOutlined, SendOutlined, ClockCircleOutlined } from '@ant-design/icons';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { httpClient } from '@/lib/api/http-client';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

export default function ServiceCatalogRequestPage() {
  const params = useParams();
  const router = useRouter();
  const id = Number(params?.id);
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [catalog, setCatalog] = useState<any>(null);
  const [fetching, setFetching] = useState(true);

  useEffect(() => {
    if (!id) return;
    setFetching(true);
    // 拉取服务目录详情
    httpClient
      .get<any>(`/api/v1/service-catalogs/${id}`)
      .then((data: any) => {
        setCatalog(data?.data || data);
      })
      .catch(() => {
        // 兜底：列表接口
        return httpClient.get<any>('/api/v1/service-catalogs', { page: 1, size: 100 }).then((list: any) => {
          const items = list?.data?.items || list?.items || [];
          const found = items.find((it: any) => it.id === id);
          if (found) setCatalog(found);
        });
      })
      .finally(() => setFetching(false));
  }, [id]);

  const onFinish = async (values: any) => {
    setLoading(true);
    try {
      const expireAt: Dayjs | undefined = values.expire_at;
      const payload: any = {
        serviceId: id,
        formData: {
          title: values.title,
          reason: values.reason,
          quantity: values.quantity || 1,
          expected_at: values.expected_at ? values.expected_at.toISOString() : undefined,
          cost_center: values.cost_center,
          data_classification: values.data_classification || 'internal',
          needs_public_ip: values.needs_public_ip || false,
          source_ip_whitelist: values.source_ip_whitelist
            ? values.source_ip_whitelist.split(',').map((s: string) => s.trim()).filter(Boolean)
            : undefined,
          // B10: 合规确认 + 过期时间
          compliance_ack: !!values.compliance_ack,
          expire_at: expireAt ? expireAt.toISOString() : undefined,
        },
      };

      await ServiceCatalogApi.createServiceRequest(payload);
      message.success('申请已提交，等待审批');
      router.push('/my-requests');
    } catch (e: any) {
      message.error('提交失败：' + (e?.message || '未知错误'));
    } finally {
      setLoading(false);
    }
  };

  if (fetching) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto p-6">
      <Breadcrumb
        items={[
          { title: '服务目录', href: '/service-catalog' },
          { title: '提交申请' },
        ]}
        className="mb-4"
      />
      <Card>
        <Space className="mb-4">
          <Button icon={<ArrowLeftOutlined />} onClick={() => router.push('/service-catalog')}>
            返回
          </Button>
          <Title level={3} style={{ margin: 0 }}>
            申请服务
          </Title>
        </Space>

        {catalog && (
          <Alert
            type="info"
            showIcon
            className="mb-4"
            message={
              <Space>
                <Text strong>{catalog.name}</Text>
                {catalog.delivery_time != null && (
                  <Tag icon={<ClockCircleOutlined />} color="blue">
                    交付时长 {catalog.delivery_time} 天
                  </Tag>
                )}
                {catalog.category && <Tag>{catalog.category}</Tag>}
              </Space>
            }
            description={catalog.description}
          />
        )}

        <Divider />

        <Form form={form} layout="vertical" onFinish={onFinish}>
          <Form.Item
            name="title"
            label="申请标题"
            rules={[{ required: true, message: '请输入申请标题' }]}
          >
            <Input placeholder="一句话说明申请目的" maxLength={200} />
          </Form.Item>

          <Form.Item
            name="reason"
            label="申请理由"
            rules={[{ required: true, message: '请输入申请理由' }]}
          >
            <TextArea rows={4} placeholder="请详细说明申请原因、业务场景、紧急程度" maxLength={2000} />
          </Form.Item>

          <div className="grid grid-cols-2 gap-4">
            <Form.Item name="quantity" label="数量" initialValue={1}>
              <Input type="number" min={1} max={100} />
            </Form.Item>
            <Form.Item name="expected_at" label="期望交付时间">
              <DatePicker showTime style={{ width: '100%' }} />
            </Form.Item>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <Form.Item name="cost_center" label="成本中心">
              <Input placeholder="例如 CC-1001" />
            </Form.Item>
            <Form.Item
              name="data_classification"
              label="数据分级"
              initialValue="internal"
            >
              <Select
                options={[
                  { label: '公开 (public)', value: 'public' },
                  { label: '内部 (internal)', value: 'internal' },
                  { label: '机密 (confidential)', value: 'confidential' },
                  { label: '绝密 (restricted)', value: 'restricted' },
                ]}
              />
            </Form.Item>
          </div>

          <Form.Item name="needs_public_ip" valuePropName="checked">
            <Checkbox>需要公网 IP</Checkbox>
          </Form.Item>

          <Form.Item
            name="source_ip_whitelist"
            label="来源 IP 白名单（多个以英文逗号分隔）"
            dependencies={['needs_public_ip']}
          >
            <Input placeholder="例如 1.2.3.4, 10.0.0.0/8" />
          </Form.Item>

          <Divider />

          <Form.Item
            name="expire_at"
            label="资源过期时间（到期自动回收）"
            extra="若不填写，则按服务目录默认策略"
          >
            <DatePicker
              showTime
              style={{ width: '100%' }}
              disabledDate={(d) => d && d.isBefore(dayjs().startOf('day'))}
            />
          </Form.Item>

          <Form.Item
            name="compliance_ack"
            valuePropName="checked"
            rules={[
              {
                validator: (_, value) =>
                  value
                    ? Promise.resolve()
                    : Promise.reject(new Error('请确认已知悉相关合规与安全要求')),
              },
            ]}
          >
            <Checkbox>
              我已知悉本服务的合规要求与安全策略，并承诺仅将资源用于申请所述的合法业务场景
            </Checkbox>
          </Form.Item>

          <Form.Item>
            <Space>
              <Button
                type="primary"
                htmlType="submit"
                icon={<SendOutlined />}
                loading={loading}
              >
                提交申请
              </Button>
              <Button onClick={() => router.push('/service-catalog')}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
