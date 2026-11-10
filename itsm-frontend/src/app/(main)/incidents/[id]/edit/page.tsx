'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { Button, Card, Form, Input, Select, App, Row, Col, Space, Divider } from 'antd';
import { ArrowLeft, Save } from 'lucide-react';
import { IncidentAPI } from '@/lib/api/incident-api';
import type { Incident, UpdateIncidentRequest } from '@/lib/api/incident-api';
import { IncidentCategoryOptions } from '@/constants/taxonomy';
import { useI18n } from '@/lib/i18n';

const { TextArea } = Input;

interface IncidentFormValues {
  title: string;
  description?: string;
  status: string;
  priority: string;
  severity: string;
  category?: string;
  subcategory?: string;
  source?: string;
}

export default function IncidentEditPage() {
  const router = useRouter();
  const { t } = useI18n();
  const params = useParams();
  const id = params?.id as string;
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [incidentData, setIncidentData] = useState<Incident | null>(null);
  const { message } = App.useApp();

  // Fetch incident data
  useEffect(() => {
    if (!id) return;

    let isMounted = true;
    const fetchIncident = async () => {
      setFetching(true);
      try {
        const resp = await IncidentAPI.getIncident(Number(id));
        if (!isMounted) return;
        setIncidentData(resp);
        form.setFieldsValue({
          title: resp.title,
          description: resp.description,
          priority: resp.priority,
          severity: resp.severity,
          category: resp.category,
          subcategory: resp.subcategory,
          status: resp.status,
        });
      } catch (error) {
        if (isMounted) {
          message.error(t('common.getFailed'));
          router.push('/incidents');
        }
      } finally {
        if (isMounted) {
          setFetching(false);
        }
      }
    };

    fetchIncident();
    return () => {
      isMounted = false;
    };
  }, [id, form, router]);

  const handleSubmit = async (values: IncidentFormValues) => {
    if (!id) return;

    setLoading(true);
    try {
      // source 不在后端 UpdateIncidentRequest 契约内，转发会被静默丢弃，故不提交。
      const payload: UpdateIncidentRequest = {
        title: values.title,
        description: values.description,
        status: values.status,
        priority: values.priority,
        severity: values.severity,
        category: values.category,
        subcategory: values.subcategory,
        // 乐观锁：回传读取时的版本，后端据此判定并发冲突并返回 4090。
        version: incidentData?.version,
      };
      await IncidentAPI.updateIncident(Number(id), payload);
      message.success(t('incidents.updateSuccess'));
      router.push(`/incidents/${id}`);
    } catch (error) {
      // 后端已把冲突（4090）、越权（2003）、非法状态迁移映射成语义化业务码并给出
      // 面向用户的文案；httpClient 只保留 message，故优先展示它，兜底才用通用文案。
      const detail = error instanceof Error ? error.message : '';
      message.error(detail || t('incidents.updateFailed'));
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = () => {
    router.back();
  };

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      <div className="mb-6">
        <Button
          type="link"
          icon={<ArrowLeft />}
          onClick={() => router.back()}
          style={{ paddingLeft: 0, color: '#666' }}
        >

          Back
        </Button>
      </div>

      <Card
        title={
          <span className="text-lg font-medium">Edit event - {incidentData?.incidentNumber}</span>
        }
        loading={fetching}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
          initialValues={{
            priority: 'medium',
            severity: 'medium',
            status: 'new',
          }}
        >
          <Row gutter={24}>
            <Col span={24}>
              <Form.Item
                name="title"
                label="Incident Title"
                rules={[{ required: true, message: 'Please enter event title' }]}
              >
                <Input placeholder="Please enter event title" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                name="status"
                label="Status"
                rules={[{ required: true, message: 'Please select status' }]}
              >
                <Select placeholder="Please select status" options={[
                  { value: 'new', label: 'Create' },
                  { value: 'in_progress', label: 'In Progress' },
                  { value: 'resolved', label: 'Resolved' },
                  { value: 'closed', label: 'Closed' },
                ]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="priority"
                label="Priority"
                rules={[{ required: true, message: 'Please select priority' }]}
              >
                <Select placeholder="Please select priority" options={[
                  { value: 'low', label: 'Low' },
                  { value: 'medium', label: 'Medium' },
                  { value: 'high', label: 'High' },
                  { value: 'critical', label: 'Urgent' },
                ]} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                name="severity"
                label="Severity"
                rules={[{ required: true, message: 'Please select severity' }]}
              >
                <Select placeholder="Please select severity" options={[
                  { value: 'low', label: 'Low' },
                  { value: 'medium', label: 'Medium' },
                  { value: 'high', label: 'High' },
                  { value: 'critical', label: 'Critical' },
                ]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="category" label="Category">
                <Select placeholder="Please select a category" allowClear options={IncidentCategoryOptions} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item name="subcategory" label="Subcategory">
                <Input placeholder="Enter subcategory" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="source" label="Source">
                <Select placeholder="Please select source" allowClear options={[
                  { value: 'manual', label: 'Manually created' },
                  { value: 'monitoring', label: 'Monitoring system' },
                  { value: 'email', label: 'Email' },
                  { value: 'phone', label: 'Phone' },
                  { value: 'chat', label: 'Online Chat' },
                  { value: 'api', label: 'API' },
                ]} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name="description" label="Incident Description">
                <TextArea rows={6} placeholder="Please describe the incident in detail" />
              </Form.Item>
            </Col>
          </Row>

          <Divider />

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" icon={<Save />} loading={loading}>

                Save
              </Button>
              <Button onClick={handleCancel}>Cancel</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
