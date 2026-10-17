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
  if (entries.length === 0) return '无';
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
      setLoadError(errorReason(error, '加载 SLA 模板失败'));
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
        message.success(`模板「${result.templateKey}」已安装到当前租户`);
      } else {
        message.info(`模板「${result.templateKey}」此前已安装，未重复创建`);
      }
    } catch (error) {
      const reason = errorReason(error, '安装失败');
      setFailures([{ templateKey: key, reason }]);
      message.error(`安装模板「${key}」失败：${reason}`);
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
          `批量安装结束：新增 ${created} 个、已存在 ${existed} 个、失败 ${summary.failures.length} 个`
        );
      } else {
        message.success(`批量安装完成：新增 ${created} 个，已存在 ${existed} 个`);
      }
    } catch (error) {
      message.error(errorReason(error, '批量安装失败'));
    } finally {
      setInstallingKey(null);
    }
  };

  const columns: ColumnsType<SLATemplate> = [
    {
      title: '模板',
      key: 'template',
      render: (_, t) => (
        <Space orientation="vertical" size={2}>
          <Space size="small">
            <Text strong>{t.name}</Text>
            <Tag color="blue">{t.key}</Tag>
            {t.recommended && <Tag color="green">推荐</Tag>}
          </Space>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {t.description}
          </Text>
        </Space>
      ),
    },
    {
      title: '行业',
      dataIndex: 'industry',
      key: 'industry',
      width: 100,
      render: (value: string) => <Tag color="geekblue">{getIndustryLabel(value)}</Tag>,
    },
    {
      title: '服务类型',
      dataIndex: 'serviceType',
      key: 'serviceType',
      width: 120,
      render: (value: string) => getServiceTypeLabel(value),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (value: string) => (
        <Tag color={getPriorityColor(value)}>{getPriorityLabel(value)}</Tag>
      ),
    },
    {
      title: '响应时限',
      dataIndex: 'responseTime',
      key: 'responseTime',
      width: 120,
      render: (value: number) => formatDuration(value),
    },
    {
      title: '解决时限',
      dataIndex: 'resolutionTime',
      key: 'resolutionTime',
      width: 120,
      render: (value: number) => formatDuration(value),
    },
    {
      title: '服务窗口',
      key: 'businessHours',
      width: 200,
      render: (_, t) => formatBusinessHours(t.businessHours),
    },
    {
      title: '安装状态',
      key: 'installStatus',
      width: 130,
      render: (_, t) =>
        t.installed ? (
          <Tooltip
            title={`安装时间 ${displayDateTime(t.installedAt)} · SLA 定义 #${t.slaDefinitionId ?? '-'}`}
          >
            <Tag color="green" icon={<CheckCircle size={12} />}>
              已安装
            </Tag>
          </Tooltip>
        ) : (
          <Text type="secondary">未安装</Text>
        ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 200,
      render: (_, t) => (
        <Space>
          <Button size="small" icon={<Eye size={14} />} onClick={() => setDetailKey(t.key)}>
            查看
          </Button>
          <Popconfirm
            title={`确认安装模板「${t.name}」？`}
            description={
              t.installed
                ? '当前租户已安装，重复安装会直接跳过'
                : '安装后当前租户生成对应 SLA 定义并参与计时'
            }
            okText="确认安装"
            cancelText="取消"
            onConfirm={() => handleInstall(t.key)}
          >
            <Button
              type="primary"
              size="small"
              icon={<Download size={14} />}
              loading={installingKey === t.key}
            >
              安装
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const descriptionItems = detail
    ? [
        { key: 'name', label: '名称', children: detail.name, span: 2 },
        {
          key: 'key',
          label: '模板 Key',
          children: <Tag color="blue">{detail.key}</Tag>,
          span: 2,
        },
        {
          key: 'industry',
          label: '行业',
          children: <Tag color="geekblue">{getIndustryLabel(detail.industry)}</Tag>,
        },
        {
          key: 'serviceType',
          label: '服务类型',
          children: getServiceTypeLabel(detail.serviceType),
        },
        {
          key: 'priority',
          label: '优先级',
          children: (
            <Tag color={getPriorityColor(detail.priority)}>{getPriorityLabel(detail.priority)}</Tag>
          ),
        },
        {
          key: 'recommended',
          label: '推荐模板',
          children: detail.recommended ? <Tag color="green">是</Tag> : <Tag>否</Tag>,
        },
        {
          key: 'response',
          label: '响应时限',
          children: formatDuration(detail.responseTime),
        },
        {
          key: 'resolution',
          label: '解决时限',
          children: formatDuration(detail.resolutionTime),
        },
        {
          key: 'window',
          label: '服务窗口',
          children: formatBusinessHours(detail.businessHours),
          span: 2,
        },
        {
          key: 'escalation',
          label: '升级级别',
          span: 2,
          children: detail.escalationLevels?.length ? (
            <Space size={[4, 4]} wrap>
              {detail.escalationLevels.map(level => (
                <Tag key={level.level}>{formatEscalationLevel(level)}</Tag>
              ))}
            </Space>
          ) : (
            '未配置（升级动作需在「升级规则」「升级矩阵」页面配置）'
          ),
        },
        {
          key: 'installStatus',
          label: '安装状态',
          span: 2,
          children: detail.installed
            ? `已安装于 ${displayDateTime(detail.installedAt)}（SLA 定义 #${detail.slaDefinitionId ?? '-'}）`
            : '未安装',
        },
        {
          key: 'conditions',
          label: '适用条件',
          span: 2,
          children: renderConditions(detail.conditions),
        },
        { key: 'description', label: '说明', children: detail.description, span: 2 },
      ]
    : [];

  return (
    <div className="space-y-6">
      <div>
        <Title level={3} style={{ marginBottom: 4 }}>
          <Rocket style={{ marginRight: 8 }} />
          SLA 模板
        </Title>
        <Paragraph type="secondary">
          按业务类型与优先级预置的 SLA 时限套餐。安装到当前租户后即生成对应 SLA 定义；安装状态记录在服务端，刷新页面不会丢失。
        </Paragraph>
      </div>

      <UsageGuideCard
        style={{ marginBottom: 16 }}
        title="SLA 模板怎么用"
        intro="模板是预置的 SLA 时限套餐。安装到当前租户后生成一条 SLA 定义并参与工单计时，无需逐项手工配置。"
        steps={[
          '先看模板行或「查看」详情，确认各优先级的响应 / 解决时限与服务窗口符合组织约定。',
          '点「安装」（或顶部「批量安装全部推荐模板」）安装到当前租户。安装按租户幂等：重复点击只会跳过，不会产生第二份相同配置。',
          '「安装状态」列显示服务端记录的安装事实（含安装时间与生成的 SLA 定义编号），刷新页面或改掉定义名称都不会退回「未安装」。',
          '安装出的定义可在「SLA 定义」页继续调整或停用；模板目录本身是内置的，不会被修改。',
          '注意：模板里的升级级别只是随定义写入的参考配置，当前没有运行时消费者。超时告警与逐级升级请在「升级规则」「升级矩阵」页面配置，那才是实际生效的入口。',
        ]}
      />

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={6}>
          <Card>
            <Statistic title="模板总数" value={stats.total} suffix="个" />
          </Card>
        </Col>
        <Col xs={24} sm={6}>
          <Card>
            <Statistic
              title="推荐模板"
              value={stats.recommended}
              suffix="个"
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={6}>
          <Card>
            <Statistic
              title="本租户已安装"
              value={stats.installed}
              suffix="个"
              styles={{ content: { color: '#1677ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={6}>
          <Card>
            <Statistic title="覆盖行业" value={stats.industries} suffix="个" />
          </Card>
        </Col>
      </Row>

      {loadError && (
        <Alert
          type="error"
          showIcon
          title="SLA 模板加载失败"
          description={loadError}
          action={
            <Button size="small" onClick={() => void loadTemplates()} loading={loading}>
              重试
            </Button>
          }
        />
      )}

      {failures.length > 0 && (
        <Alert
          type="error"
          showIcon
          title={`${failures.length} 个模板安装失败`}
          description={
            <ul style={{ margin: 0, paddingLeft: 18 }}>
              {failures.map(failure => (
                <li key={failure.templateKey}>{`${failure.templateKey}：${failure.reason}`}</li>
              ))}
            </ul>
          }
          action={
            <Button size="small" onClick={() => setFailures([])}>
              忽略
            </Button>
          }
        />
      )}

      <Card>
        <Alert
          title="安装模板会为当前租户创建 SLA 定义；同一模板重复安装会自动跳过"
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Space style={{ marginBottom: 12 }}>
          <Popconfirm
            title="批量安装全部推荐模板？"
            description="仅安装标记为推荐的模板，已安装的自动跳过"
            okText="全部安装"
            cancelText="取消"
            onConfirm={handleInstallAll}
          >
            <Button
              type="primary"
              icon={<Rocket size={14} />}
              loading={installingKey === '__ALL__'}
              disabled={stats.recommended === 0 || loading}
            >
              批量安装全部推荐模板
            </Button>
          </Popconfirm>
          <Button onClick={() => void loadTemplates()} loading={loading}>
            刷新
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
            emptyText: <Empty description={loadError ? '模板加载失败' : '暂无可用模板'} />,
          }}
        />
      </Card>

      <Modal
        open={!!detail}
        title={detail ? `模板详情 · ${detail.name}` : '模板详情'}
        onCancel={() => setDetailKey(null)}
        footer={[
          <Button key="close" onClick={() => setDetailKey(null)}>
            关闭
          </Button>,
        ]}
        width={720}
      >
        <Descriptions column={2} bordered size="small" items={descriptionItems} />
      </Modal>
    </div>
  );
}
