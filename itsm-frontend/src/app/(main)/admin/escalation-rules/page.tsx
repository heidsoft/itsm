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
  warning: { label: '预警', color: 'warning' },
  critical: { label: '严重', color: 'error' },
  severe: { label: '紧急', color: 'red' },
};

const CHANNEL_CONFIG: Record<string, { label: string }> = {
  email: { label: '邮件' },
  sms: { label: '短信' },
  in_app: { label: '站内信' },
  webhook: { label: 'Webhook' },
  feishu: { label: '飞书' },
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
      message.error(error instanceof Error ? error.message : '加载告警规则失败');
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
      message.success(next ? `规则「${rule.name}」已启用` : `规则「${rule.name}」已停用`);
      await loadRules();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '更新规则状态失败');
    } finally {
      setTogglingId(null);
    }
  };

  const handleDuplicate = async (rule: SLAAlertRule) => {
    setDuplicatingId(rule.id);
    try {
      await SLAApi.createAlertRule({
        name: `${rule.name} (副本)`,
        slaDefinitionId: rule.slaDefinitionId,
        alertLevel: rule.alertLevel,
        thresholdPercentage: rule.thresholdPercentage,
        notificationChannels: rule.notificationChannels,
        isActive: false,
      });
      message.success('已复制为新规则（默认停用，确认后请手动启用）');
      await loadRules();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '复制规则失败');
    } finally {
      setDuplicatingId(null);
    }
  };

  const handleDelete = async (rule: SLAAlertRule) => {
    try {
      await SLAApi.deleteAlertRule(rule.id);
      message.success(`规则「${rule.name}」已删除`);
      await loadRules();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除规则失败');
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
        message.success('规则已更新');
      } else {
        await SLAApi.createAlertRule({
          name: values.name,
          slaDefinitionId: values.slaDefinitionId,
          alertLevel: values.alertLevel,
          thresholdPercentage: values.thresholdPercentage,
          notificationChannels: values.notificationChannels,
          isActive: values.isActive,
        });
        message.success('规则已创建');
      }
      setShowEditModal(false);
      setSelectedRule(null);
      form.resetFields();
      await loadRules();
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存规则失败');
    } finally {
      setSaving(false);
    }
  };

  const columns: ColumnsType<SLAAlertRule> = [
    {
      title: '规则',
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
      title: '关联 SLA 定义',
      dataIndex: 'slaDefinitionId',
      key: 'slaDefinitionId',
      render: (slaDefinitionId: number) => {
        const def = definitionMap.get(slaDefinitionId);
        if (!def) return <Text type="secondary">SLA 定义 #{slaDefinitionId}（已删除）</Text>;
        return (
          <Space>
            <Text>{def.name}</Text>
            <Tag color={PRIORITY_TAG_COLOR[def.priority] ?? 'default'}>{def.priority}</Tag>
          </Space>
        );
      },
    },
    {
      title: '告警策略',
      key: 'strategy',
      render: (_, record) => {
        const level = ALERT_LEVEL_CONFIG[record.alertLevel] ?? { label: record.alertLevel, color: 'default' };
        return (
          <Space wrap>
            <Tag color={level.color}>{level.label}</Tag>
            <Text style={{ fontSize: 12 }}>阈值 {record.thresholdPercentage}%</Text>
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
      title: '状态',
      dataIndex: 'isActive',
      key: 'isActive',
      width: 90,
      render: (isActive: boolean) =>
        isActive ? <Tag color="success">已启用</Tag> : <Tag color="default">已停用</Tag>,
    },
    {
      title: '操作',
      key: 'actions',
      width: 280,
      render: (_, record) => (
        <Space>
          <Button type="link" size="small" icon={<Eye size={14} />} onClick={() => openDetailDrawer(record)}>
            详情
          </Button>
          <Button type="link" size="small" icon={<Edit size={14} />} onClick={() => openEditModal(record)}>
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            icon={<Copy size={14} />}
            loading={duplicatingId === record.id}
            onClick={() => handleDuplicate(record)}
          >
            复制
          </Button>
          <Button type="link" size="small" loading={togglingId === record.id} onClick={() => handleStatusToggle(record)}>
            {record.isActive ? '停用' : '启用'}
          </Button>
          <Popconfirm
            title="确定删除该告警规则吗？"
            description={`规则「${record.name}」（ID: ${record.id}）删除后不可恢复，对应 SLA 定义上的告警将停止触发。`}
            okText="删除"
            cancelText="取消"
            okButtonProps={{ danger: true }}
            onConfirm={() => handleDelete(record)}
          >
            <Button type="link" size="small" danger icon={<Trash2 size={14} />}>
              删除
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
            SLA 告警规则
          </Space>
        </Title>
        <Text type="secondary">
          为 SLA 定义配置告警阈值：工单耗时达到阈值百分比时按渠道发出预警 / 严重 / 紧急告警
        </Text>
        <Alert
          type="info"
          showIcon
          style={{ marginTop: 12 }}
          message="本页配置的是 SLA 阈值告警，不是升级矩阵"
          description={
            <span>
              升级矩阵（优先级 → 通知角色链，如 15 分钟未响应升级至部门经理）在
              <a href="/admin/escalation-matrices">升级矩阵</a>
              页面配置。两者常被混为一谈，但触发时机与产出完全不同：告警规则产出通知，升级矩阵驱动工单升级。
            </span>
          }
        />
      </div>

      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="告警规则总数" value={stats.total} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="已启用" value={stats.active} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="已停用" value={stats.inactive} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="关联 SLA 定义" value={stats.linkedDefinitions} />
          </Card>
        </Col>
      </Row>

      <Card style={{ marginBottom: 16 }}>
        <Row gutter={[16, 12]} align="middle">
          <Col xs={24} md={8}>
            <Input
              placeholder="搜索规则名称"
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
                { value: 'all', label: '全部状态' },
                { value: 'active', label: '已启用' },
                { value: 'inactive', label: '已停用' },
              ]}
            />
          </Col>
          <Col xs={12} md={4}>
            <Select
              style={{ width: '100%' }}
              allowClear
              placeholder="全部级别"
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
              placeholder="全部 SLA 定义"
              value={definitionFilter}
              onChange={(value) => setDefinitionFilter(value)}
              options={definitions.map((def) => ({ value: def.id, label: def.name }))}
            />
          </Col>
          <Col xs={12} md={3} style={{ textAlign: 'right' }}>
            <Button type="primary" icon={<Plus size={14} />} onClick={openCreateModal}>
              新建规则
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
          pagination={{ showSizeChanger: true, showTotal: (total) => `共 ${total} 条规则` }}
          locale={{ emptyText: '暂无升级规则，点击「新建规则」为 SLA 定义配置告警' }}
        />
      </Card>

      <Modal
        title={selectedRule ? '编辑升级规则' : '新建升级规则'}
        open={showEditModal}
        onCancel={() => {
          setShowEditModal(false);
          setSelectedRule(null);
        }}
        onOk={handleSave}
        confirmLoading={saving}
        width={640}
        okText="保存"
        cancelText="取消"
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
          <Form.Item name="name" label="规则名称" rules={[{ required: true, message: '请输入规则名称' }]}>
            <Input placeholder="例如：P1 响应超时预警" />
          </Form.Item>
          <Form.Item
            name="slaDefinitionId"
            label="关联 SLA 定义"
            rules={[{ required: true, message: '请选择 SLA 定义' }]}
          >
            <Select
              showSearch
              optionFilterProp="label"
              disabled={!!selectedRule}
              placeholder="请选择要监控的 SLA 定义"
              options={definitions.map((def) => ({ value: def.id, label: `${def.name}（${def.priority}）` }))}
            />
          </Form.Item>
          {selectedRule && (
            <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
              关联的 SLA 定义创建后不可更改；如需更换请复制规则后重新关联。
            </Text>
          )}
          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="alertLevel" label="告警级别" rules={[{ required: true, message: '请选择告警级别' }]}>
                <Select
                  options={Object.entries(ALERT_LEVEL_CONFIG).map(([value, cfg]) => ({ value, label: cfg.label }))}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item
                name="thresholdPercentage"
                label="告警阈值（%）"
                rules={[{ required: true, message: '请输入告警阈值' }]}
              >
                <InputNumber min={0} max={100} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="isActive" label="启用状态" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item
            name="notificationChannels"
            label="通知渠道"
            rules={[{ required: true, message: '请至少选择一个通知渠道' }]}
          >
            <Select
              mode="multiple"
              options={Object.entries(CHANNEL_CONFIG).map(([value, cfg]) => ({ value, label: cfg.label }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title="升级规则详情"
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
              { key: 'name', label: '规则名称', children: selectedRule.name },
              { key: 'id', label: '规则 ID', children: selectedRule.id },
              {
                key: 'definition',
                label: '关联 SLA 定义',
                children: detailDefinition
                  ? detailDefinition.name
                  : `SLA 定义 #${selectedRule.slaDefinitionId}（已删除）`,
              },
              {
                key: 'level',
                label: '告警级别',
                children: (
                  <Tag color={ALERT_LEVEL_CONFIG[selectedRule.alertLevel]?.color ?? 'default'}>
                    {ALERT_LEVEL_CONFIG[selectedRule.alertLevel]?.label ?? selectedRule.alertLevel}
                  </Tag>
                ),
              },
              { key: 'threshold', label: '告警阈值', children: `${selectedRule.thresholdPercentage}%` },
              {
                key: 'channels',
                label: '通知渠道',
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
                label: '状态',
                children: selectedRule.isActive ? <Tag color="success">已启用</Tag> : <Tag color="default">已停用</Tag>,
              },
            ]}
          />
        )}
      </Drawer>
    </div>
  );
}
