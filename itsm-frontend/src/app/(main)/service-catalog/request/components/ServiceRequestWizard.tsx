'use client';

import React from 'react';
import { useParams, useRouter } from 'next/navigation';
import { App, Card, Spin, Result, Button, Input, Select, Form, Row, Col } from 'antd';
import {
  ArrowLeft,
  ArrowRight,
  CheckCircle,
  Cloud,
  Database,
  HardDrive,
  Shield,
  Globe,
  Server,
} from 'lucide-react';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { ServiceItem, ServiceType } from '@/types/service-catalog';

const { TextArea } = Input;
const { Option } = Select;

// 服务类型对应的图标和标签
const serviceTypeConfig: Record<string, { icon: React.ReactNode; label: string; color: string }> = {
  vm: { icon: <Server className="w-6 h-6" />, label: '云服务器', color: '#1890ff' },
  rds: { icon: <Database className="w-6 h-6" />, label: '数据库', color: '#52c41a' },
  oss: { icon: <HardDrive className="w-6 h-6" />, label: '对象存储', color: '#faad14' },
  network: { icon: <Globe className="w-6 h-6" />, label: '网络服务', color: '#722ed1' },
  storage: { icon: <HardDrive className="w-6 h-6" />, label: '存储服务', color: '#eb2f96' },
  security: { icon: <Shield className="w-6 h-6" />, label: '安全服务', color: '#fa541c' },
  custom: { icon: <Cloud className="w-6 h-6" />, label: '自定义服务', color: '#13c2c2' },
};

interface ServiceRequestWizardProps {
  serviceId: string;
}

export const ServiceRequestWizard: React.FC<ServiceRequestWizardProps> = ({ serviceId }) => {
  const params = useParams();
  const router = useRouter();
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const sid = serviceId || (params.serviceId as string);

  const [catalog, setCatalog] = React.useState<ServiceItem | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [currentStep, setCurrentStep] = React.useState(0);
  const [submitting, setSubmitting] = React.useState(false);

  // 获取服务目录详情
  React.useEffect(() => {
    const fetchCatalog = async () => {
      try {
        setLoading(true);
        const response = await ServiceCatalogApi.getServices({ page: 1, pageSize: 100 });
        const foundCatalog = response.services.find(c => Number(c.id) === Number(sid));
        if (foundCatalog) setCatalog(foundCatalog);
        else setError('服务目录不存在');
      } catch (err) {
        setError('获取服务目录失败');
        console.error('Failed to fetch catalog:', err);
      } finally {
        setLoading(false);
      }
    };

    if (sid) {
      fetchCatalog();
    }
  }, [sid]);

  // 获取服务类型配置
  const getServiceTypeInfo = () => {
    const serviceType = catalog?.serviceType || 'custom';
    return serviceTypeConfig[serviceType] || serviceTypeConfig.custom;
  };

  // 提交服务请求
  const handleSubmit = async () => {
    if (!catalog) return;

    try {
      setSubmitting(true);
      const values = await form.validateFields();

      const requestData = {
        serviceId: String(catalog.id),
        formData: {
          title: catalog.name,
          reason: values.reason || '',
          cost_center: values.costCenter,
          data_classification: values.dataClassification || 'internal',
          needs_public_ip: values.needsPublicIP || false,
          source_ip_whitelist: values.sourceIPWhitelist
            ? values.sourceIPWhitelist.split(',').map((s: string) => s.trim())
            : [],
          expire_at: values.expireAt ? new Date(values.expireAt).toISOString() : undefined,
          compliance_ack: values.complianceAck || false,
        },
      };

      await ServiceCatalogApi.createServiceRequest(requestData);

      message.success(`服务请求 "${catalog.name}" 已成功提交！`);
      router.push('/my-requests');
    } catch (err) {
      console.error('Failed to create service request:', err);
      message.error('提交服务请求失败，请稍后重试');
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Spin size="large" />
      </div>
    );
  }

  if (error || !catalog) {
    return (
      <Result
        status="error"
        title="加载失败"
        subTitle={error || '服务不存在'}
        extra={[
          <Button type="primary" key="back" onClick={() => router.push('/service-catalog')}>
            返回服务目录
          </Button>,
        ]}
      />
    );
  }

  const typeInfo = getServiceTypeInfo();

  return (
    <div className="max-w-4xl mx-auto">
      {/* 头部 */}
      <div className="mb-6">
        <Button
          type="link"
          icon={<ArrowLeft className="w-4 h-4" />}
          onClick={() => router.push('/service-catalog')}
          className="pl-0"
        >
          返回服务目录
        </Button>
      </div>

      {/* 服务信息卡片 */}
      <Card className="mb-6">
        <div className="flex items-center gap-4">
          <div
            className="w-12 h-12 rounded-lg flex items-center justify-center"
            style={{ backgroundColor: `${typeInfo.color}20`, color: typeInfo.color }}
          >
            {typeInfo.icon}
          </div>
          <div>
            <h2 className="text-xl font-semibold m-0">{catalog.name}</h2>
            <p className="text-gray-500 text-sm m-0">{typeInfo.label}</p>
          </div>
          <div className="ml-auto text-right">
            <div className="text-sm text-gray-500">预计交付</div>
            <div className="font-medium">{catalog.availability?.resolutionTime || '-'} 小时</div>
          </div>
        </div>
      </Card>

      {/* 步骤指示器 */}
      <div className="flex items-center gap-4 mb-6">
        <div
          className={`w-8 h-8 rounded-full flex items-center justify-center ${currentStep >= 0 ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}
        >
          1
        </div>
        <div className="flex-1 h-1 bg-gray-200">
          <div className="h-full bg-blue-600" style={{ width: currentStep >= 1 ? '100%' : '0%' }} />
        </div>
        <div
          className={`w-8 h-8 rounded-full flex items-center justify-center ${currentStep >= 1 ? 'bg-blue-600 text-white' : 'bg-gray-200'}`}
        >
          2
        </div>
      </div>

      {/* 表单内容 */}
      <Card>
        <Form form={form} layout="vertical">
          {/* 服务类型说明 */}
          <div className="bg-blue-50 border border-blue-100 rounded-lg p-4 mb-6">
            <div className="flex items-center gap-2 text-blue-800 mb-2">
              {typeInfo.icon}
              <span className="font-medium">{typeInfo.label}申请</span>
            </div>
            <p className="text-sm text-blue-600 m-0">
              请填写以下配置信息，以便我们为您提供准确的{typeInfo.label}服务。
            </p>
          </div>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="costCenter" label="成本中心">
                <Input placeholder="例如：CC-1001" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="dataClassification"
                label="数据分级"
                rules={[{ required: true, message: '请选择数据分级' }]}
              >
                <Select placeholder="请选择">
                  <Option value="public">公开（public）</Option>
                  <Option value="internal">内部（internal）</Option>
                  <Option value="confidential">机密（confidential）</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="expireAt"
                label="到期时间"
                rules={[{ required: true, message: '请选择到期时间' }]}
              >
                <Input type="datetime-local" style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="needsPublicIP" label="需要公网访问" valuePropName="checked">
                <Select placeholder="是否需要">
                  <Option value={true}>是</Option>
                  <Option value={false}>否</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="sourceIPWhitelist" label="源IP白名单">
            <TextArea placeholder="例如：1.2.3.4/32, 5.6.7.0/24（逗号分隔）" rows={2} />
          </Form.Item>

          <Form.Item name="reason" label="申请理由">
            <TextArea rows={4} placeholder="请说明申请此服务的业务背景和目的" />
          </Form.Item>

          <Form.Item name="complianceAck" valuePropName="checked">
            <div className="flex items-start gap-2">
              <input type="checkbox" className="mt-1" />
              <span className="text-sm text-gray-600">
                我已阅读并同意服务条款，包括数据处理范围、最小权限原则、到期回收和成本归属等要求。
              </span>
            </div>
          </Form.Item>

          <div className="flex justify-end">
            <Button type="primary" loading={submitting} onClick={handleSubmit}>
              提交申请
            </Button>
          </div>
        </Form>
      </Card>
    </div>
  );
};

export default ServiceRequestWizard;
