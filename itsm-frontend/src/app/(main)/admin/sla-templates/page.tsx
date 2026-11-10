'use client';

/**
 * SLA 模板管理页面
 *
 * 浏览预置模板、查看详情、安装到当前租户
 */

import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Modal,
  Popconfirm,
  Row,
  Space,
  Statistic,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { CheckCircle, Download, Eye, Rocket } from 'lucide-react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  SLATemplateApi,
  type SLATemplate,
  type TemplateInstallFailure,
} from '@/lib/api/sla-template-api';
import { UsageGuideCard } from '@/components/common/UsageGuideCard';
import { formatDateTime } from '@/lib/formatters';
import {
  formatBusinessHours,
  formatDuration,
  formatEscalationLevel,
  getIndustryLabel,
  getPriorityColor,
  getPriorityLabel,
  getServiceTypeLabel,
} from '@/lib/utils/sla-format';

const { Title, Text, Paragraph } = Typography;

const displayDateTime = (value?: string) => formatDateTime(value) || '-';

const errorReason = (error: unknown, fallback: string) =>
  error instanceof Error && error.message ? error.message : fallback;

// conditions 是安装时原样写入定义的适配条件，键名保留存储层写法
const renderConditions = (conditions: Record<string, unknown>) => {
  const entries = Object.entries(conditions ?? {});
  if (entries.length === 0) return 'None';
  return (
    <Space size={[4, 4]} wrap>
      {entries.map(([key, value]) => (
        <Tag key={key}>
          {key}: {Array.isArray(value) ? value.join('、') : String(value)}
        </Tag>
      ))}
    </Space>
  );
};

export default function SLATemplatesPage() {
  const { message } = App.useApp();
  const [templates, setTemplates] = useState<SLATemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [installingKey, setInstallingKey] = useState<string | null>(null);
  const [detailKey, setDetailKey] = useState<string | null>(null);
  const [failures, setFailures] = useState<TemplateInstallFailure[]>([]);

  const loadTemplates = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      setTemplates(await SLATemplateApi.listTemplates());
    } catch (error) {
      setTemplates([]);
      setLoadError(errorReason(error, 'Failed to load SLA template'));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadTemplates();
  }, [loadTemplates]);

  // 详情取列表里的最新一条，安装完再打开弹窗不会显示旧状态
  const detail = useMemo(
    () => templates.find(t => t.key === detailKey) ?? null,
    [templates, detailKey]
  );

  const stats = useMemo(() => {
    const industries = new Set(templates.map(t => t.industry));
    return {
      total: templates.length,
      recommended: templates.filter(t => t.recommended).length,
      installed: templates.filter(t => t.installed).length,
      industries: industries.size,
    };
  }, [templates]);

  const handleInstall = async (key: string) => {
    setInstallingKey(key);
    setFailures([]);
    try {
      const result = await SLATemplateApi.installTemplate(key);
      // 安装状态以服务端为准：重新拉列表，刷新后不会退回「未安装」
      await loadTemplates();
      if (result.created) {
        message.success(`Template "${result.templateKey}"Already installed in the current tenant"`);
      } else {
        message.info(`Template "${result.templateKey}"It has been installed before and has not been re-created.`);
      }
    } catch (error) {
      const reason = errorReason(error, 'Installation failed');
      setFailures([{ templateKey: key, reason }]);
      message.error(`Installation template "${key}"Failed:${reason}`);
    } finally {
      setInstallingKey(null);
    }
  };

  const handleInstallAll = async () => {
    setInstallingKey('__ALL__');
    setFailures([]);
    try {
      const summary = await SLATemplateApi.installAllRecommended();
      await loadTemplates();
      setFailures(summary.failures);
      const created = summary.results.filter(r => r.created).length;
      const existed = summary.results.filter(r => !r.created).length;
      if (summary.failures.length > 0) {
        message.error(
          `Batch installation completed: new ${created} , already exists ${existed} , failed ${summary.failures.length} items`
        );
      } else {
        message.success(`Batch installation completed: New ${created} already exists ${existed} items`);
      }
    } catch (error) {
      message.error(errorReason(error, 'Batch installation failed'));
    } finally {
      setInstallingKey(null);
    }
  };

  const columns: ColumnsType<SLATemplate> = [
    {
      title: 'Templates',
      key: 'template',
      render: (_, t) => (
        <Space orientation="vertical" size={2}>
          <Space size="small">
            <Text strong>{t.name}</Text>
            <Tag color="blue">{t.key}</Tag>
            {t.recommended && <Tag color="green">Recommended</Tag>}
          </Space>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {t.description}
          </Text>
        </Space>
      ),
    },
    {
      title: 'Industry',
      dataIndex: 'industry',
      key: 'industry',
      width: 100,
      render: (value: string) => <Tag color="geekblue">{getIndustryLabel(value)}</Tag>,
    },
    {
      title: 'Service Type',
      dataIndex: 'serviceType',
      key: 'serviceType',
      width: 120,
      render: (value: string) => getServiceTypeLabel(value),
    },
    {
      title: 'Priority',
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (value: string) => (
        <Tag color={getPriorityColor(value)}>{getPriorityLabel(value)}</Tag>
      ),
    },
    {
      title: 'Response time limit',
      dataIndex: 'responseTime',
      key: 'responseTime',
      width: 120,
      render: (value: number) => formatDuration(value),
    },
    {
      title: 'Resolution time limit',
      dataIndex: 'resolutionTime',
      key: 'resolutionTime',
      width: 120,
      render: (value: number) => formatDuration(value),
    },
    {
      title: 'Service window',
      key: 'businessHours',
      width: 200,
      render: (_, t) => formatBusinessHours(t.businessHours),
    },
    {
      title: 'InstallationStatus',
      key: 'installStatus',
      width: 130,
      render: (_, t) =>
        t.installed ? (
          <Tooltip
            title={`Installation time ${displayDateTime(t.installedAt)} · SLA Definition #${t.slaDefinitionId ?? '-'}`}
          >
            <Tag color="green" icon={<CheckCircle size={12} />}>

              Already installed
            </Tag>
          </Tooltip>
        ) : (
          <Text type="secondary">Not installed</Text>
        ),
    },
    {
      title: 'Actions',
      key: 'actions',
      width: 200,
      render: (_, t) => (
        <Space>
          <Button size="small" icon={<Eye size={14} />} onClick={() => setDetailKey(t.key)}>

            View
          </Button>
          <Popconfirm
            title={`Confirm installation template "${t.name}」？`}
            description={
              t.installed
                ? 'The current tenant has already been installed. Repeated installation will be skipped directly.'
                : 'After installation, the current tenant generates the corresponding SLA definition and participates in timing'
            }
            okText="Confirm installation"
            cancelText="Cancel"
            onConfirm={() => handleInstall(t.key)}
          >
            <Button
              type="primary"
              size="small"
              icon={<Download size={14} />}
              loading={installingKey === t.key}
            >

              Installation
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const descriptionItems = detail
    ? [
        { key: 'name', label: 'Name', children: detail.name, span: 2 },
        {
          key: 'key',
          label: 'Template Key',
          children: <Tag color="blue">{detail.key}</Tag>,
          span: 2,
        },
        {
          key: 'industry',
          label: 'Industry',
          children: <Tag color="geekblue">{getIndustryLabel(detail.industry)}</Tag>,
        },
        {
          key: 'serviceType',
          label: 'Service Type',
          children: getServiceTypeLabel(detail.serviceType),
        },
        {
          key: 'priority',
          label: 'Priority',
          children: (
            <Tag color={getPriorityColor(detail.priority)}>{getPriorityLabel(detail.priority)}</Tag>
          ),
        },
        {
          key: 'recommended',
          label: 'Recommended Templates',
          children: detail.recommended ? <Tag color="green">Yes</Tag> : <Tag>No</Tag>,
        },
        {
          key: 'response',
          label: 'Response time limit',
          children: formatDuration(detail.responseTime),
        },
        {
          key: 'resolution',
          label: 'Resolution time limit',
          children: formatDuration(detail.resolutionTime),
        },
        {
          key: 'window',
          label: 'Service window',
          children: formatBusinessHours(detail.businessHours),
          span: 2,
        },
        {
          key: 'escalation',
          label: 'Escalation Level',
          span: 2,
          children: detail.escalationLevels?.length ? (
            <Space size={[4, 4]} wrap>
              {detail.escalationLevels.map(level => (
                <Tag key={level.level}>{formatEscalationLevel(level)}</Tag>
              ))}
            </Space>
          ) : (
            'Not configured (upgrade action needs to be configured on the "Upgrade Rules" "Upgrade Matrix" page)'
          ),
        },
        {
          key: 'installStatus',
          label: 'InstallationStatus',
          span: 2,
          children: detail.installed
            ? `Installed in ${displayDateTime(detail.installedAt)}(SLA Definition #${detail.slaDefinitionId ?? '-'}）`
            : 'Not installed',
        },
        {
          key: 'conditions',
          label: 'Applicable conditions',
          span: 2,
          children: renderConditions(detail.conditions),
        },
        { key: 'description', label: 'Description', children: detail.description, span: 2 },
      ]
    : [];

  return (
    <div className="space-y-6">
      <div>
        <Title level={3} style={{ marginBottom: 4 }}>
          <Rocket style={{ marginRight: 8 }} />

          SLA Template
        </Title>
        <Paragraph type="secondary">

          SLA time limit packages preset according to business type and priority. After being installed into the current tenant, the corresponding SLA definition is generated; the installation status is recorded on the server and will not be lost when the page is refreshed.
        </Paragraph>
      </div>

      <UsageGuideCard
        style={{ marginBottom: 16 }}
        title="How to use SLA template"
        intro="The template is a preset SLA time-limited package. After being installed into the current tenant, an SLA definition is generated and participates in ticket timing, eliminating the need for manual configuration one by one."
        steps={[
          'First read the template line or "View" details to confirm that the response/resolution time limit and service window of each priority level comply with the organization\'s agreement.',
          'Click "Install" (or "Batch Install All Recommended Templates" at the top) to install it to the current tenant. The installation is idempotent by tenant: repeated clicks will only skip and will not generate a second copy of the same configuration.',
          'The "Installation Status" column displays the installation facts recorded by the server (including the installation time and the generated SLA definition number). Refreshing the page or changing the definition name will not return "Not Installed".',
          'The installed definition can be adjusted or deactivated on the "SLA Definition" page; the template directory itself is built-in and will not be modified.',
          'Note: The upgrade level in the template is only a reference configuration written with the definition, and there is currently no runtime consumer. Please configure the timeout alarm and step-by-step upgrade on the "Upgrade Rules" and "Upgrade Matrix" pages. That is the actual entrance.',
        ]}
      />

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={6}>
          <Card>
            <Statistic title="Total number of templates" value={stats.total} suffix="items" />
          </Card>
        </Col>
        <Col xs={24} sm={6}>
          <Card>
            <Statistic
              title="Recommended Templates"
              value={stats.recommended}
              suffix="items"
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={6}>
          <Card>
            <Statistic
              title="This tenant has already been installed"
              value={stats.installed}
              suffix="items"
              styles={{ content: { color: '#1677ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={6}>
          <Card>
            <Statistic title="Covered industries" value={stats.industries} suffix="items" />
          </Card>
        </Col>
      </Row>

      {loadError && (
        <Alert
          type="error"
          showIcon
          title="SLA template loading failed"
          description={loadError}
          action={
            <Button size="small" onClick={() => void loadTemplates()} loading={loading}>

              Retry
            </Button>
          }
        />
      )}

      {failures.length > 0 && (
        <Alert
          type="error"
          showIcon
          title={`${failures.length} Template installation failed`}
          description={
            <ul style={{ margin: 0, paddingLeft: 18 }}>
              {failures.map(failure => (
                <li key={failure.templateKey}>{`${failure.templateKey}：${failure.reason}`}</li>
              ))}
            </ul>
          }
          action={
            <Button size="small" onClick={() => setFailures([])}>

              ignore
            </Button>
          }
        />
      )}

      <Card>
        <Alert
          title="Installing a template will create an SLA definition for the current tenant; repeated installations of the same template will be automatically skipped"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Space style={{ marginBottom: 12 }}>
          <Popconfirm
            title="Install all recommended templates in batches?"
            description="Only install templates marked as recommended, and installed ones will be automatically skipped."
            okText="Install all"
            cancelText="Cancel"
            onConfirm={handleInstallAll}
          >
            <Button
              type="primary"
              icon={<Rocket size={14} />}
              loading={installingKey === '__ALL__'}
              disabled={stats.recommended === 0 || loading}
            >

              Install all recommended templates in batches
            </Button>
          </Popconfirm>
          <Button onClick={() => void loadTemplates()} loading={loading}>

            Refresh
          </Button>
        </Space>
        <Table<SLATemplate>
          rowKey="key"
          loading={loading}
          dataSource={templates}
          columns={columns}
          pagination={false}
          scroll={{ x: 'max-content' }}
          locale={{
            emptyText: <Empty description={loadError ? 'Template loading failed' : 'No template available yet'} />,
          }}
        />
      </Card>

      <Modal
        open={!!detail}
        title={detail ? `Template details · ${detail.name}` : 'Template details'}
        onCancel={() => setDetailKey(null)}
        footer={[
          <Button key="close" onClick={() => setDetailKey(null)}>

            Close
          </Button>,
        ]}
        width={720}
      >
        <Descriptions column={2} bordered size="small" items={descriptionItems} />
      </Modal>
    </div>
  );
}
