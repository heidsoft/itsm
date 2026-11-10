'use client';

import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Modal,
  Row,
  Select,
  Space,
  Steps,
  Table,
  Tag,
  Typography,
  App,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { GitBranch, Plus, RefreshCw, Search } from 'lucide-react';
import type { DomainConfig, EffectiveConfig } from '@/lib/api/domain-config-api';
import { DomainConfigApi } from '@/lib/api/domain-config-api';

const { Title, Text } = Typography;

export default function ConfigInheritancePage() {
  const { message } = App.useApp();
  const [configs, setConfigs] = useState<DomainConfig[]>([]);
  const [effectiveConfig, setEffectiveConfig] = useState<EffectiveConfig | null>(null);
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 空表格，再被真实数据覆盖。
  const [listLoading, setListLoading] = useState(true);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewed, setPreviewed] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [previewForm] = Form.useForm();

  const loadConfigs = async () => {
    setListLoading(true);
    try {
      const data = await DomainConfigApi.list();
      setConfigs(data);
    } catch (error) {
      console.error(error);
      message.error('Failed to load configuration');
    } finally {
      setListLoading(false);
    }
  };

  useEffect(() => {
    loadConfigs();
  }, []);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      await DomainConfigApi.save({
        ...values,
        configValue: parseJSONObject(values.configValue),
      });
      message.success('Configuration saved');
      setModalOpen(false);
      form.resetFields();
      loadConfigs();
    } catch (error) {
      console.error(error);
      message.error(error instanceof Error ? error.message : 'Failed to save configuration');
    }
  };

  const handlePreview = async () => {
    try {
      const values = await previewForm.validateFields();
      setPreviewLoading(true);
      const data = await DomainConfigApi.getEffective({
        configType: values.configType,
        configKey: values.configKey,
        departmentId: values.departmentId,
        teamId: values.teamId,
      });
      setEffectiveConfig(data);
      setPreviewed(true);
    } catch (error) {
      console.error(error);
      message.error(error instanceof Error ? error.message : 'Failed to parse valid configuration');
    } finally {
      setPreviewLoading(false);
    }
  };

  const columns: ColumnsType<DomainConfig> = [
    {
      title: 'Configuration type',
      dataIndex:'configType',
      key:'configType',
      render: value => <Tag color="blue">{value}</Tag>,
    },
    {
      title: 'Configuration key',
      dataIndex:'configKey',
      key:'configKey',
    },
    {
      title: 'Scope',
      key: 'scope',
      render: (_, record) => {
        if (record.teamId > 0) return <Tag color="purple">Team #{record.teamId}</Tag>;
        if (record.departmentId > 0) return <Tag color="geekblue">Department #{record.departmentId}</Tag>;
        if (record.tenantId > 0) return <Tag color="green">Tenant</Tag>;
        return <Tag>Global</Tag>;
      },
    },
    {
      title: 'Inheritance mode',
      dataIndex:'inheritMode',
      key:'inheritMode',
      render: value => {
        const color = value === 'override' ? 'red' : value === 'extend' ? 'gold' : 'default';
        return <Tag color={color}>{value}</Tag>;
      },
    },
    {
      title: 'Version',
      dataIndex: 'version',
      key: 'version',
      width: 90,
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
  ];

  const chainItems = useMemo(
    () => [
      { title: 'Global', description: 'Effective for all tenants' },
      { title: 'Tenant', description: 'Current tenant' },
      { title: 'Department', description: 'Department ancestor chain' },
      { title: 'Team', description: 'Team coverage' },
    ],
    []
  );

  return (
    <div className="space-y-6">
      <div>
        <Title level={2} className="!mb-2">
          <GitBranch className="mr-2" />

          Configuration Inheritance
        </Title>
        <Text type="secondary">View the configuration inheritance and final effective results of Global → Tenant → Department → Team</Text>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card
            title="Configuration list"
            extra={
              <Space>
                <Button icon={<RefreshCw size={16} />} onClick={loadConfigs} loading={listLoading}>
                  Refresh
                </Button>
                <Button type="primary" icon={<Plus size={16} />} onClick={() => setModalOpen(true)}>

                  New configuration
                </Button>
              </Space>
            }
          >
            <Table columns={columns} dataSource={configs} rowKey="id" loading={listLoading} />
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card title="Inheritance chain" style={{ marginBottom: 16 }}>
            <Steps orientation="vertical" size="small" current={3} items={chainItems} />
          </Card>

          <Card title="Valid configuration preview">
            <Form form={previewForm} layout="vertical">
              <Form.Item name="configType" label="Configuration type" rules={[{ required: true }]}>
                <Input placeholder="process_binding / approval_workflow / sla_rule" />
              </Form.Item>
              <Form.Item name="configKey" label="Configuration key" rules={[{ required: true }]}>
                <Input placeholder="default_sla" />
              </Form.Item>
              <Row gutter={12}>
                <Col span={12}>
                  <Form.Item name="departmentId" label="Department ID">
                    <InputNumber min={0} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="teamId" label="Team ID">
                    <InputNumber min={0} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>
              <Button block type="primary" icon={<Search size={16} />} loading={previewLoading} onClick={handlePreview}>

                Parse valid configuration
              </Button>
            </Form>

            {effectiveConfig && (
              <Alert
                style={{ marginTop: 16 }}
                type="success"
                title={`Source:${effectiveConfig.source} / Pattern:${effectiveConfig.inheritMode} / version:${effectiveConfig.version}`}
                description={<pre style={{ margin: 0 }}>{JSON.stringify(effectiveConfig.value, null, 2)}</pre>}
              />
            )}
            {previewed && !effectiveConfig && !previewLoading && (
              <Alert
                style={{ marginTop: 16 }}
                type="info"
                showIcon
                title="Full link miss configuration"
                description="Global → Tenant → Department → Team This configuration is not defined at any level, and the code default value is used at runtime."
              />
            )}
          </Card>
        </Col>
      </Row>

      <Modal
        title="Add/update configuration"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={handleSave}
        width={720}
      >
        <Form form={form} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="configType" label="Configuration type" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="configKey" label="Configuration key" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="departmentId" label="Department ID" initialValue={0}>
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="teamId" label="Team ID" initialValue={0}>
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="inheritMode" label="Inheritance mode" initialValue="inherit">
                <Select
                  options={[
                    { value: 'inherit', label: 'inherit' },
                    { value: 'override', label: 'override' },
                    { value: 'extend', label: 'extend' },
                  ]}
                />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="description" label="Description">
            <Input />
          </Form.Item>
          <Form.Item
            name="configValue"
            label="Configuration value JSON"
            rules={[{ required: true, message: 'Please enter JSON configuration' }]}
            initialValue="{}"
          >
            <Input.TextArea rows={8} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function parseJSONObject(value: string): Record<string, unknown> {
  const parsed = JSON.parse(value || '{}');
  if (!parsed || Array.isArray(parsed) || typeof parsed !== 'object') {
    throw new Error('Configuration value must be a JSON object');
  }
  return parsed as Record<string, unknown>;
}
