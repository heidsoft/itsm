'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { Button, Card, Form, Input, Select, message, Row, Col, Space, Divider } from 'antd';
import { ArrowLeftOutlined, SaveOutlined } from '@ant-design/icons';
import { IncidentAPI } from '@/lib/api/incident-api';
import type { Incident, UpdateIncidentRequest } from '@/lib/api/incident-api';
import { useI18n } from '@/lib/i18n';

const { TextArea } = Input;
const { Option } = Select;

export default function IncidentEditPage() {
  const router = useRouter();
  const { t } = useI18n();
  const params = useParams();
  const id = params?.id as string;
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(false);
  const [incidentData, setIncidentData] = useState<Incident | null>(null);

  // Fetch incident data
  useEffect(() => {
    if (!id) return;

    let isMounted = true;
    const fetchIncident = async () => {
      setFetching(true);
      try {
        const resp = await IncidentAPI.getIncident(Number(id));
        if (!isMounted) return;
        const data = resp as any;
        setIncidentData(data);
        form.setFieldsValue({
          title: data.title,
          description: data.description,
          priority: data.priority,
          severity: data.severity,
          category: data.category,
          subcategory: data.subcategory,
          status: data.status,
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

  const handleSubmit = async (values: any) => {
    if (!id) return;

    setLoading(true);
    try {
      await IncidentAPI.updateIncident(Number(id), values);
      message.success(t('incidents.updateSuccess'));
      router.push(`/incidents/${id}`);
    } catch (error) {
      message.error(t('incidents.updateFailed'));
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
          icon={<ArrowLeftOutlined />}
          onClick={() => router.back()}
          style={{ paddingLeft: 0, color: '#666' }}
        >
          返回
        </Button>
      </div>

      <Card
        title={
          <span className="text-lg font-medium">编辑事件 - {incidentData?.incident_number}</span>
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
                label="事件标题"
                rules={[{ required: true, message: '请输入事件标题' }]}
              >
                <Input placeholder="请输入事件标题" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                name="status"
                label="状态"
                rules={[{ required: true, message: '请选择状态' }]}
              >
                <Select placeholder="请选择状态">
                  <Option value="new">新建</Option>
                  <Option value="in_progress">进行中</Option>
                  <Option value="resolved">已解决</Option>
                  <Option value="closed">已关闭</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="priority"
                label="优先级"
                rules={[{ required: true, message: '请选择优先级' }]}
              >
                <Select placeholder="请选择优先级">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="urgent">紧急</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item
                name="severity"
                label="严重程度"
                rules={[{ required: true, message: '请选择严重程度' }]}
              >
                <Select placeholder="请选择严重程度">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">严重</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="category" label="分类">
                <Select placeholder="请选择分类" allowClear>
                  <Option value="performance">性能</Option>
                  <Option value="connectivity">连接</Option>
                  <Option value="security">安全</Option>
                  <Option value="storage">存储</Option>
                  <Option value="network">网络</Option>
                  <Option value="application">应用</Option>
                  <Option value="database">数据库</Option>
                  <Option value="other">其他</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={12}>
              <Form.Item name="subcategory" label="子分类">
                <Input placeholder="请输入子分类" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="source" label="来源">
                <Select placeholder="请选择来源" allowClear>
                  <Option value="manual">手动创建</Option>
                  <Option value="monitoring">监控系统</Option>
                  <Option value="email">邮件</Option>
                  <Option value="phone">电话</Option>
                  <Option value="chat">在线聊天</Option>
                  <Option value="api">API</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={24}>
            <Col span={24}>
              <Form.Item name="description" label="事件描述">
                <TextArea rows={6} placeholder="请详细描述事件情况" />
              </Form.Item>
            </Col>
          </Row>

          <Divider />

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" icon={<SaveOutlined />} loading={loading}>
                保存
              </Button>
              <Button onClick={handleCancel}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}
