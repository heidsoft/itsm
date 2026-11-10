'use client';

import { Bell, Copy, Edit, Eye, Plus, Search, Trash2 } from 'lucide-react';

import React, { useEffect, useMemo, useState } from 'react';
import { SLAApi } from '@/lib/api/sla-api';
import type { SLAAlertRule, SLADefinition } from '@/lib/api/sla-api';
import type { ColumnsType } from 'antd/es/table';
import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Descriptions,
  Drawer,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Statistic,
  Switch,
  Table,
  Tag,
  Typography,
} from 'antd';

const { Title, Text } = Typography;

const ALERT_LEVEL_CONFIG: Record<string, { label: string; color: string }> = {
  warning: { label: 'Early warning', color: 'warning' },
  critical: { label: 'Critical', color: 'error' },
  severe: { label: 'Urgent', color: 'red' },
};

const CHANNEL_CONFIG: Record<string, { label: string }> = {
  email: { label: 'Email' },
  sms: { label: 'SMS' },
  in_app: { label: 'In-App' },
  webhook: { label: 'Webhook' },
  feishu: { label: 'Feishu' },
};

const PRIORITY_TAG_COLOR: Record<string, string> = {
  urgent: 'red',
  critical: 'red',
  high: 'orange',
  medium: 'blue',
  low: 'default',
};

interface AlertRuleFormValues {
  name: string;
  slaDefinitionId: number;
  alertLevel: SLAAlertRule['alertLevel'];
  thresholdPercentage: number;
  notificationChannels: string[];
  isActive: boolean;
}

export default function EscalationRulesPage() {
  const { message } = App.useApp();
  const [form] = Form.useForm<AlertRuleFormValues>();

  const [rules, setRules] = useState<SLAAlertRule[]>([]);
  const [definitions, setDefinitions] = useState<SLADefinition[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [togglingId, setTogglingId] = useState<number | null>(null);
  const [duplicatingId, setDuplicatingId] = useState<number | null>(null);

  const [searchTerm, setSearchTerm] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all');
  const [levelFilter, setLevelFilter] = useState<SLAAlertRule['alertLevel'] | undefined>(undefined);
  const [definitionFilter, setDefinitionFilter] = useState<number | undefined>(undefined);

  const [showEditModal, setShowEditModal] = useState(false);
  const [selectedRule, setSelectedRule] = useState<SLAAlertRule | null>(null);
  const [showDetailDrawer, setShowDetailDrawer] = useState(false);

  const definitionMap = useMemo(() => {
    const map = new Map<number, SLADefinition>();
    definitions.forEach((def) => map.set(def.id, def));
    return map;
  }, [definitions]);

  const loadRules = async () => {
    setLoading(true);
    try {
      const [ruleData, defData] = await Promise.all([
        SLAApi.getAlertRules(),
        // 取较大页容量，避免默认分页截断筛选下拉中的 SLA 定义选项
        SLAApi.getSLADefinitions({ page: 1, pageSize: 100 }),
      ]);
      setRules(ruleData ?? []);
      setDefinitions(defData?.items ?? []);
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to load alarm rules');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRules();
  }, []);

  const stats = useMemo(() => {
    const active = rules.filter((r) => r.isActive).length;
    const linked = new Set(rules.map((r) => r.slaDefinitionId));
    return {
      total: rules.length,
      active,
      inactive: rules.length - active,
      linkedDefinitions: linked.size,
    };
  }, [rules]);

  const filteredRules = useMemo(() => {
    const term = searchTerm.trim().toLowerCase();
    return rules.filter((rule) => {
      if (term && !rule.name.toLowerCase().includes(term)) return false;
      if (statusFilter === 'active' && !rule.isActive) return false;
      if (statusFilter === 'inactive' && rule.isActive) return false;
      if (levelFilter && rule.alertLevel !== levelFilter) return false;
      if (definitionFilter !== undefined && rule.slaDefinitionId !== definitionFilter) return false;
      return true;
    });
  }, [rules, searchTerm, statusFilter, levelFilter, definitionFilter]);

  const openCreateModal = () => {
    setSelectedRule(null);
    form.resetFields();
    setShowEditModal(true);
  };

  const openEditModal = (rule: SLAAlertRule) => {
    setSelectedRule(rule);
    form.setFieldsValue({
      name: rule.name,
      slaDefinitionId: rule.slaDefinitionId,
      alertLevel: rule.alertLevel,
      thresholdPercentage: rule.thresholdPercentage,
      notificationChannels: rule.notificationChannels,
      isActive: rule.isActive,
    });
    setShowEditModal(true);
  };

  const openDetailDrawer = (rule: SLAAlertRule) => {
    setSelectedRule(rule);
    setShowDetailDrawer(true);
  };

  const handleStatusToggle = async (rule: SLAAlertRule) => {
    const next = !rule.isActive;
    setTogglingId(rule.id);
    try {
      await SLAApi.updateAlertRule(rule.id, { isActive: next });
      message.success(next ? `Rule "${rule.name}"Enabled"` : `Rule "${rule.name}"Deactivated`);
      await loadRules();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to update rule status');
    } finally {
      setTogglingId(null);
    }
  };

  const handleDuplicate = async (rule: SLAAlertRule) => {
    setDuplicatingId(rule.id);
    try {
      await SLAApi.createAlertRule({
        name: `${rule.name} (Copy)`,
        slaDefinitionId: rule.slaDefinitionId,
        alertLevel: rule.alertLevel,
        thresholdPercentage: rule.thresholdPercentage,
        notificationChannels: rule.notificationChannels,
        isActive: false,
      });
      message.success('Copied as a new rule (disabled by default, please enable it manually after confirmation)');
      await loadRules();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to copy rule');
    } finally {
      setDuplicatingId(null);
    }
  };

  const handleDelete = async (rule: SLAAlertRule) => {
    try {
      await SLAApi.deleteAlertRule(rule.id);
      message.success(`Rule "${rule.name}"Deleted`);
      await loadRules();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to delete rule');
    }
  };

  const handleSave = async () => {
    let values: AlertRuleFormValues;
    try {
      values = await form.validateFields();
    } catch {
      return;
    }
    setSaving(true);
    try {
      if (selectedRule) {
        await SLAApi.updateAlertRule(selectedRule.id, {
          name: values.name,
          alertLevel: values.alertLevel,
          thresholdPercentage: values.thresholdPercentage,
          notificationChannels: values.notificationChannels,
          isActive: values.isActive,
        });
        message.success('Rule updated');
      } else {
        await SLAApi.createAlertRule({
          name: values.name,
          slaDefinitionId: values.slaDefinitionId,
          alertLevel: values.alertLevel,
          thresholdPercentage: values.thresholdPercentage,
          notificationChannels: values.notificationChannels,
          isActive: values.isActive,
        });
        message.success('Rule created');
      }
      setShowEditModal(false);
      setSelectedRule(null);
      form.resetFields();
      await loadRules();
    } catch (error) {
      message.error(error instanceof Error ? error.message : 'Failed to save rules');
    } finally {
      setSaving(false);
    }
  };

  const columns: ColumnsType<SLAAlertRule> = [
    {
      title: 'Rule',
      dataIndex: 'name',
      key: 'name',
      render: (_, record) => (
        <div>
          <Text strong>{record.name}</Text>
          <br />
          <Text type="secondary" style={{ fontSize: 12 }}>
            ID: {record.id}
          </Text>
        </div>
      ),
    },
    {
      title: 'Associated SLA definition',
      dataIndex: 'slaDefinitionId',
      key: 'slaDefinitionId',
      render: (slaDefinitionId: number) => {
        const def = definitionMap.get(slaDefinitionId);
        if (!def) return <Text type="secondary">SLA Definition #{slaDefinitionId}(deleted)</Text>;
        return (
          <Space>
            <Text>{def.name}</Text>
            <Tag color={PRIORITY_TAG_COLOR[def.priority] ?? 'default'}>{def.priority}</Tag>
          </Space>
        );
      },
    },
    {
      title: 'Alarm policy',
      key: 'strategy',
      render: (_, record) => {
        const level = ALERT_LEVEL_CONFIG[record.alertLevel] ?? { label: record.alertLevel, color: 'default' };
        return (
          <Space wrap>
            <Tag color={level.color}>{level.label}</Tag>
            <Text style={{ fontSize: 12 }}>Threshold {record.thresholdPercentage}%</Text>
            {record.notificationChannels.map((channel) => (
              <Tag key={channel} style={{ fontSize: 12 }}>
                {CHANNEL_CONFIG[channel]?.label ?? channel}
              </Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: 'Status',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 90,
      render: (isActive: boolean) =>
        isActive ? <Tag color="success">Enabled</Tag> : <Tag color="default">Disabled</Tag>,
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 280,
      render: (_, record) => (
        <Space>
          <Button type="link" size="small" icon={<Eye size={14} />} onClick={() => openDetailDrawer(record)}>

            Details
          </Button>
          <Button type="link" size="small" icon={<Edit size={14} />} onClick={() => openEditModal(record)}>

            Edit
          </Button>
          <Button
            type="link"
            size="small"
            icon={<Copy size={14} />}
            loading={duplicatingId === record.id}
            onClick={() => handleDuplicate(record)}
          >

            Copy
          </Button>
          <Button type="link" size="small" loading={togglingId === record.id} onClick={() => handleStatusToggle(record)}>
            {record.isActive ? 'Disable' : 'Enabled'}
          </Button>
          <Popconfirm
            title="Are you sure you want to delete this alarm rule?"
            description={`Rule "${record.name}」（ID: ${record.id}) cannot be recovered after deletion, and the alarms on the corresponding SLA definition will stop triggering.`}
            okText="Delete"
            cancelText="Cancel"
            okButtonProps={{ danger: true }}
            onConfirm={() => handleDelete(record)}
          >
            <Button type="link" size="small" danger icon={<Trash2 size={14} />}>

              Delete
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const detailDefinition = selectedRule ? definitionMap.get(selectedRule.slaDefinitionId) : undefined;

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ marginBottom: 8 }}>
          <Space>
            <Bell size={20} />

            SLA Alert Rule
          </Space>
        </Title>
        <Text type="secondary">

          Configure alarm thresholds for SLA definition: when the work order time reaches the threshold percentage, an early warning/serious/emergency alarm will be issued by channel
        </Text>
        <Alert
          type="info"
          showIcon
          style={{ marginTop: 12 }}
          title="This page configures SLA threshold alarms, not the upgrade matrix."
          description={
            <span>

              Escalation matrix (priority → notification role chain, e.g. no response for 15 minutes escalates to department manager) in
              <a href="/admin/escalation-matrices">Upgrade Matrix</a>

              Page configuration. The two are often confused, but the trigger timing and output are completely different: alarm rules output notifications, and the upgrade matrix drives work order upgrades.
            </span>
          }
        />
      </div>

      {/* 统计卡必须门控：还没拉到数据时渲染成「告警规则总数 0」会被误读成「没有配置任何告警规则」 */}
      {!loading && (
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="Total number of alarm rules" value={stats.total} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="Enabled" value={stats.active} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="Disabled" value={stats.inactive} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="Associated SLA definition" value={stats.linkedDefinitions} />
          </Card>
        </Col>
      </Row>
      )}

      <Card style={{ marginBottom: 16 }}>
        <Row gutter={[16, 12]} align="middle">
          <Col xs={24} md={8}>
            <Input
              placeholder="Search rule name"
              allowClear
              prefix={<Search size={14} />}
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </Col>
          <Col xs={12} md={4}>
            <Select
              style={{ width: '100%' }}
              value={statusFilter}
              onChange={setStatusFilter}
              options={[
                { value: 'all', label: 'All Status' },
                { value: 'active', label: 'Enabled' },
                { value: 'inactive', label: 'Disabled' },
              ]}
            />
          </Col>
          <Col xs={12} md={4}>
            <Select
              style={{ width: '100%' }}
              allowClear
              placeholder="All levels"
              value={levelFilter}
              onChange={(value) => setLevelFilter(value)}
              options={Object.entries(ALERT_LEVEL_CONFIG).map(([value, cfg]) => ({ value, label: cfg.label }))}
            />
          </Col>
          <Col xs={12} md={5}>
            <Select
              style={{ width: '100%' }}
              allowClear
              showSearch
              optionFilterProp="label"
              placeholder="All SLA definitions"
              value={definitionFilter}
              onChange={(value) => setDefinitionFilter(value)}
              options={definitions.map((def) => ({ value: def.id, label: def.name }))}
            />
          </Col>
          <Col xs={12} md={3} style={{ textAlign: 'right' }}>
            <Button type="primary" icon={<Plus size={14} />} onClick={openCreateModal}>

              Create Rule
            </Button>
          </Col>
        </Row>
      </Card>

      <Card>
        <Table
          rowKey="id"
          columns={columns}
          dataSource={filteredRules}
          loading={loading}
          pagination={{ showSizeChanger: true, showTotal: (total) => `Total ${total} Rules` }}
          locale={{ emptyText: 'There is no upgrade rule yet. Click "New Rule" to configure alarms for SLA definition.' }}
        />
      </Card>

      <Modal
        title={selectedRule ? 'Edit upgrade rules' : 'Create new upgrade rule'}
        open={showEditModal}
        onCancel={() => {
          setShowEditModal(false);
          setSelectedRule(null);
        }}
        onOk={handleSave}
        confirmLoading={saving}
        width={640}
        okText="Save"
        cancelText="Cancel"
        destroyOnHidden
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            alertLevel: 'warning',
            thresholdPercentage: 80,
            notificationChannels: ['email'],
            isActive: false,
          }}
        >
          <Form.Item name="name" label="Rule Name" rules={[{ required: true, message: 'Please enter a rule name' }]}>
            <Input placeholder="For example: P1 response timeout warning" />
          </Form.Item>
          <Form.Item
            name="slaDefinitionId"
            label="Associated SLA definition"
            rules={[{ required: true, message: 'Please select an SLA definition' }]}
          >
            <Select
              showSearch
              optionFilterProp="label"
              disabled={!!selectedRule}
              placeholder="Please select an SLA definition to monitor"
              options={definitions.map((def) => ({ value: def.id, label: `${def.name}（${def.priority}）` }))}
            />
          </Form.Item>
          {selectedRule && (
            <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>

              The associated SLA definition cannot be changed after it is created; if you need to change it, please copy the rule and re-associate it.
            </Text>
          )}
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="alertLevel" label="Alarm level" rules={[{ required: true, message: 'Please select the alarm level' }]}>
                <Select
                  options={Object.entries(ALERT_LEVEL_CONFIG).map(([value, cfg]) => ({ value, label: cfg.label }))}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="thresholdPercentage"
                label="Alarm threshold (%)"
                rules={[{ required: true, message: 'Please enter the alarm threshold' }]}
              >
                <InputNumber min={0} max={100} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="isActive" label="Enabled" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            name="notificationChannels"
            label="Notification Channels"
            rules={[{ required: true, message: 'Please select at least one notification channel' }]}
          >
            <Select
              mode="multiple"
              options={Object.entries(CHANNEL_CONFIG).map(([value, cfg]) => ({ value, label: cfg.label }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title="Upgrade rule details"
        open={showDetailDrawer}
        onClose={() => setShowDetailDrawer(false)}
        size={480}
      >
        {selectedRule && (
          <Descriptions
            column={1}
            bordered
            size="small"
            items={[
              { key: 'name', label: 'Rule Name', children: selectedRule.name },
              { key: 'id', label: 'Rule ID', children: selectedRule.id },
              {
                key: 'definition',
                label: 'Associated SLA definition',
                children: detailDefinition
                  ? detailDefinition.name
                  : `SLA Definition #${selectedRule.slaDefinitionId}(deleted)`,
              },
              {
                key: 'level',
                label: 'Alarm level',
                children: (
                  <Tag color={ALERT_LEVEL_CONFIG[selectedRule.alertLevel]?.color ?? 'default'}>
                    {ALERT_LEVEL_CONFIG[selectedRule.alertLevel]?.label ?? selectedRule.alertLevel}
                  </Tag>
                ),
              },
              { key: 'threshold', label: 'Alarm threshold', children: `${selectedRule.thresholdPercentage}%` },
              {
                key: 'channels',
                label: 'Notification Channels',
                children: (
                  <Space wrap>
                    {selectedRule.notificationChannels.map((channel) => (
                      <Tag key={channel}>{CHANNEL_CONFIG[channel]?.label ?? channel}</Tag>
                    ))}
                  </Space>
                ),
              },
              {
                key: 'status',
                label: 'Status',
                children: selectedRule.isActive ? <Tag color="success">Enabled</Tag> : <Tag color="default">Disabled</Tag>,
              },
            ]}
          />
        )}
      </Drawer>
    </div>
  );
}
