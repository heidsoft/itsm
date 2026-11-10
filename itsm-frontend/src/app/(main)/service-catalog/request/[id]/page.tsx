'use client';

/**
 * 服务目录申请页面
 */

import React, { useState, useEffect } from 'react';
import { serviceCategoryLabel } from '@/constants/service-catalog';
import { useParams, useRouter } from 'next/navigation';
import {
  Card,
  Form,
  Input,
  Button,
  Space,
  App,
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
import { ArrowLeft, Clock, Send } from 'lucide-react';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { httpClient } from '@/lib/api/http-client';
import { useAuthStore } from '@/lib/store/auth-store';
import type { CreateServiceRequestRequest } from '@/types/service-catalog';

const { Title, Text, Paragraph } = Typography;
const { TextArea } = Input;

interface CatalogInfo {
  id: number;
  name: string;
  category?: string;
  description?: string;
  deliveryTime?: number;
}

export default function ServiceCatalogRequestPage() {
  const params = useParams();
  const router = useRouter();
  const { message } = App.useApp();
  const id = Number(params?.id);
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [catalog, setCatalog] = useState<CatalogInfo | null>(null);
  const [fetching, setFetching] = useState(true);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const user = useAuthStore(state => state.user);

  useEffect(() => {
    if (!id) {
      setFetching(false);
      setFetchError('The service ID is invalid, please return to the service directory and select again.');
      return;
    }
    setFetching(true);
    setFetchError(null);
    // 拉取服务目录详情
    httpClient
      .get<unknown>(`/api/v1/service-catalogs/${id}`)
      .then((data: unknown) => {
        const raw = data as Record<string, unknown>;
        setCatalog((raw?.data ?? raw) as CatalogInfo);
      })
      .catch(() => {
        // 兜底：列表接口
        return httpClient
          .get<unknown>('/api/v1/service-catalogs', { page: 1, size: 100 })
          .then((list: unknown) => {
            const raw = list as Record<string, unknown>;
            const items = (raw?.data as Record<string, unknown>)?.items ?? raw?.items ?? [];
            const found = (items as CatalogInfo[]).find((it) => it.id === id);
            if (found) {
              setCatalog(found);
            } else {
              setFetchError('The selected service was not found. The service may have been removed from the shelves.');
            }
          });
      })
      .catch(() => setFetchError('Service information loading failed, please try again later.'))
      .finally(() => setFetching(false));
  }, [id]);

  useEffect(() => {
    if (user) {
      form.setFieldsValue({ requesterName: user.name, requesterEmail: user.email });
    }
  }, [form, user]);

  const onFinish = async (values: Record<string, unknown>) => {
    setLoading(true);
    try {
      const expireAt: Dayjs | undefined = values.expireAt as Dayjs | undefined;
      const payload: CreateServiceRequestRequest = {
        serviceId: String(id),
        formData: {
          requesterName: values.requesterName as string,
          requesterEmail: values.requesterEmail as string,
          title: values.title as string,
          reason: values.reason as string,
          quantity: (values.quantity as number) || 1,
          expectedAt: values.expectedAt ? (values.expectedAt as Dayjs).toISOString() : undefined,
          costCenter: values.costCenter as string,
          dataClassification: (values.dataClassification as string) || 'internal',
          needsPublicIp: (values.needsPublicIp as boolean) || false,
          sourceIpWhitelist: values.sourceIpWhitelist
            ? (values.sourceIpWhitelist as string)
                .split(',')
                .map((s: string) => s.trim())
                .filter(Boolean)
            : undefined,
          complianceAck: !!values.complianceAck,
          expireAt: expireAt ? expireAt.toISOString() : undefined,
        },
      };

      await ServiceCatalogApi.createServiceRequest(payload);
      message.success('The application has been submitted and is waiting for approval');
      try {
        router.push('/my-requests');
      } catch (navErr) {
        console.error('[nav]', navErr);
        message.error('Navigation failed, please try again later');
      }
    } catch (e: unknown) {
      message.error('Submission failed:' + (e instanceof Error ? e.message : 'Unknown error'));
    } finally {
      setLoading(false);
    }
  };

  if (fetching) {
    return (
      <div className='flex items-center justify-center min-h-[400px]'>
        <Spin size='large' />
      </div>
    );
  }

  return (
    <div className='max-w-3xl mx-auto p-6'>
      <Breadcrumb
        items={[{ title: 'Service Catalog', href: '/service-catalog' }, { title: 'Submitted' }]}
        className='mb-4'
      />
      <Card>
        <Space className='mb-4'>
          <Button
            icon={<ArrowLeft />}
            onClick={() => {
              try {
                router.push('/service-catalog');
              } catch (navErr) {
                console.error('[nav]', navErr);
                message.error('Navigation failed, please try again later');
              }
            }}
          >

            Back
          </Button>
          <Title level={3} style={{ margin: 0 }}>

            Apply Service
          </Title>
        </Space>

        {fetchError && (
          <Alert
            type='error'
            showIcon
            className='mb-4'
            title={fetchError}
            action={
              <Button
                onClick={() => {
                  try {
                    router.push('/service-catalog');
                  } catch (navErr) {
                    console.error('[nav]', navErr);
                    message.error('Navigation failed, please try again later');
                  }
                }}
              >

                Return to service directory
              </Button>
            }
          />
        )}

        {catalog && (
          <Alert
            type='info'
            showIcon
            className='mb-4'
            title={
              <div className='flex items-center justify-between gap-3 flex-wrap'>
                <Text strong className='!text-base'>
                  {catalog.name}
                </Text>
                <Space size={4} wrap>
                  {catalog.deliveryTime != null && catalog.deliveryTime > 0 && (
                    <Tag icon={<Clock />} color='blue'>

                      Delivery time {catalog.deliveryTime}  days
                    </Tag>
                  )}
                  {catalog.category && <Tag>{serviceCategoryLabel(catalog.category)}</Tag>}
                </Space>
              </div>
            }
            description={
              catalog.description ? (
                <div className='text-gray-600 leading-relaxed'>{catalog.description}</div>
              ) : null
            }
          />
        )}

        <Divider />

        <Form form={form} layout='vertical' onFinish={onFinish}>
          <div className='grid grid-cols-2 gap-4'>
            <Form.Item name='requesterName' label="Requester">
              <Input disabled placeholder="Current logged in user" />
            </Form.Item>
            <Form.Item name='requesterEmail' label="Contact email">
              <Input disabled placeholder="Current user’s email address" />
            </Form.Item>
          </div>
          <Form.Item
            name='title'
            label="Application title"
            rules={[{ required: true, message: 'Please enter the application title' }]}
          >
            <Input placeholder="One sentence describing the purpose of application" maxLength={200} />
          </Form.Item>

          <Form.Item
            name='reason'
            label="Reason for application"
            rules={[{ required: true, message: 'Please enter the reason for application' }]}
          >
            <TextArea
              rows={4}
              placeholder="Please explain in detail the reason for application, business scenario, and urgency"
              maxLength={2000}
            />
          </Form.Item>

          <div className='grid grid-cols-2 gap-4'>
            <Form.Item name='quantity' label="Count" initialValue={1}>
              <Input type='number' min={1} max={100} />
            </Form.Item>
            <Form.Item name='expectedAt' label="Expected delivery time">
              <DatePicker showTime style={{ width: '100%' }} />
            </Form.Item>
          </div>

          <div className='grid grid-cols-2 gap-4'>
            <Form.Item name='costCenter' label="Cost Center">
              <Input placeholder="Example CC-1001" />
            </Form.Item>
            <Form.Item name='dataClassification' label="Data classification" initialValue='internal'>
              <Select
                options={[
                  { label: 'public', value: 'public' },
                  { label: 'internal', value: 'internal' },
                  { label: 'confidential (confidential)', value: 'confidential' },
                  { label: 'Top Secret (restricted)', value: 'restricted' },
                ]}
              />
            </Form.Item>
          </div>

          <Form.Item name='needsPublicIp' valuePropName='checked'>
            <Checkbox>Public IP required</Checkbox>
          </Form.Item>

          <Form.Item
            name='sourceIpWhitelist'
            label="Source IP whitelist (multiple separated by commas)"
            dependencies={['needsPublicIp']}
          >
            <Input placeholder="For example 1.2.3.4, 10.0.0.0/8" />
          </Form.Item>

          <Divider />

          <Form.Item
            name='expireAt'
            label="Resource expiration time (automatic recycling upon expiration)"
            extra="If not filled in, the default policy of the service directory will be used."
          >
            <DatePicker
              showTime
              style={{ width: '100%' }}
              disabledDate={d => d && d.isBefore(dayjs().startOf('day'))}
            />
          </Form.Item>

          <Form.Item
            name='complianceAck'
            valuePropName='checked'
            rules={[
              {
                validator: (_, value) =>
                  value
                    ? Promise.resolve()
                    : Promise.reject(new Error('Please confirm that you are aware of the relevant compliance and security requirements')),
              },
            ]}
          >
            <Checkbox>

              I am aware of the compliance requirements and security policies of this service, and promise to use resources only for the legitimate business scenarios described in the application
            </Checkbox>
          </Form.Item>

          <Form.Item>
            <Space>
              <Button
                type='primary'
                htmlType='submit'
                icon={<Send />}
                loading={loading}
                disabled={!catalog || !!fetchError}
              >

                Submitted
              </Button>
              <Button
                onClick={() => {
                  try {
                    router.push('/service-catalog');
                  } catch (navErr) {
                    console.error('[nav]', navErr);
                    message.error('Navigation failed, please try again later');
                  }
                }}
              >

                Cancel
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
