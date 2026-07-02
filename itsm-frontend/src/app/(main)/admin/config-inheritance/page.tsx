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
  message,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { GitBranch, Plus, RefreshCw, Search } from 'lucide-react';
import { DomainConfig, DomainConfigApi, EffectiveConfig } from '@/lib/api/domain-config-api';

const { Title, Text } = Typography;

export default function ConfigInheritancePage() {
  const [configs, setConfigs] = useState<DomainConfig[]>([]);
  const [effectiveConfig, setEffectiveConfig] = useState<EffectiveConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [form] = Form.useForm();
  const [previewForm] = Form.useForm();

  const loadConfigs = async () => {
    setLoading(true);
    try {
      const data = await DomainConfigApi.list();
      setConfigs(data);
    } catch (error) {
      console.error(error);
      message.error('加载配置失败');
    } finally {
      setLoading(false);
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
        config_value: parseJSONObject(values.config_value),
      });
      message.success('配置已保存');
      setModalOpen(false);
      form.resetFields();
      loadConfigs();
    } catch (error) {
      console.error(error);
      message.error(error instanceof Error ? error.message : '保存配置失败');
    }
  };

  const handlePreview = async () => {
    try {
      const values = await previewForm.validateFields();
      setPreviewLoading(true);
      const data = await DomainConfigApi.getEffective({
        config_type: values.config_type,
        config_key: values.config_key,
        department_id: values.department_id,
        team_id: values.team_id,
      });
      setEffectiveConfig(data);
    } catch (error) {
      console.error(error);
      message.error(error instanceof Error ? error.message : '解析有效配置失败');
    } finally {
      setPreviewLoading(false);
    }
  };

  const columns: ColumnsType<DomainConfig> = [
    {
      title: '配置类型',
      dataIndex: 'config_type',
      key: 'config_type',
      render: value => <Tag color="blue">{value}</Tag>,
    },
    {
      title: '配置键',
      dataIndex: 'config_key',
      key: 'config_key',
    },
    {
      title: '作用域',
      key: 'scope',
      render: (_, record) => {
        if (record.team_id > 0) return <Tag color="purple">Team #{record.team_id}</Tag>;
        if (record.department_id > 0) return <Tag color="geekblue">Department #{record.department_id}</Tag>;
        if (record.tenant_id > 0) return <Tag color="green">Tenant</Tag>;
        return <Tag>Global</Tag>;
      },
    },
    {
      title: '继承模式',
      dataIndex: 'inherit_mode',
      key: 'inherit_mode',
      render: value => {
        const color = value === 'override' ? 'red' : value === 'extend' ? 'gold' : 'default';
        return <Tag color={color}>{value}</Tag>;
      },
    },
    {
      title: '版本',
      dataIndex: 'version',
      key: 'version',
      width: 90,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
  ];

  const chainItems = useMemo(
    () => [
      { title: 'Global', description: 'tenant_id = 0' },
      { title: 'Tenant', description: '当前租户' },
      { title: 'Department', description: '部门祖先链' },
      { title: 'Team', description: '团队覆盖' },
    ],
    []
  );

  return (
    <div style={{ padding: 24 }}>
      <div className="mb-6">
        <Title level={2} className="!mb-2">
          <GitBranch className="mr-2" />
          配置继承
        </Title>
        <Text type="secondary">查看 Global → Tenant → Department → Team 的配置继承与最终生效结果</Text>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card
            title="配置列表"
            extra={
              <Space>
                <Button icon={<RefreshCw size={16} />} onClick={loadConfigs} loading={loading}>
                  刷新
                </Button>
                <Button type="primary" icon={<Plus size={16} />} onClick={() => setModalOpen(true)}>
                  新增配置
                </Button>
              </Space>
            }
          >
            <Table columns={columns} dataSource={configs} rowKey="id" loading={loading} />
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card title="继承链" style={{ marginBottom: 16 }}>
            <Steps direction="vertical" size="small" current={3} items={chainItems} />
          </Card>

          <Card title="有效配置预览">
            <Form form={previewForm} layout="vertical">
              <Form.Item name="config_type" label="配置类型" rules={[{ required: true }]}>
                <Input placeholder="process_binding / approval_workflow / sla_rule" />
              </Form.Item>
              <Form.Item name="config_key" label="配置键" rules={[{ required: true }]}>
                <Input placeholder="default_sla" />
              </Form.Item>
              <Row gutter={12}>
                <Col span={12}>
                  <Form.Item name="department_id" label="部门ID">
                    <InputNumber min={0} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="team_id" label="团队ID">
                    <InputNumber min={0} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>
              <Button block type="primary" icon={<Search size={16} />} loading={previewLoading} onClick={handlePreview}>
                解析有效配置
              </Button>
            </Form>

            {effectiveConfig && (
              <Alert
                style={{ marginTop: 16 }}
                type="success"
                message={`来源：${effectiveConfig.source} / 模式：${effectiveConfig.inherit_mode} / 版本：${effectiveConfig.version}`}
                description={<pre style={{ margin: 0 }}>{JSON.stringify(effectiveConfig.value, null, 2)}</pre>}
              />
            )}
          </Card>
        </Col>
      </Row>

      <Modal
        title="新增/更新配置"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={handleSave}
        width={720}
      >
        <Form form={form} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="config_type" label="配置类型" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="config_key" label="配置键" rules={[{ required: true }]}>
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="department_id" label="部门ID" initialValue={0}>
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="team_id" label="团队ID" initialValue={0}>
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="inherit_mode" label="继承模式" initialValue="inherit">
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
          <Form.Item name="description" label="描述">
            <Input />
          </Form.Item>
          <Form.Item
            name="config_value"
            label="配置值 JSON"
            rules={[{ required: true, message: '请输入 JSON 配置' }]}
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
    throw new Error('配置值必须是 JSON 对象');
  }
  return parsed as Record<string, unknown>;
}
