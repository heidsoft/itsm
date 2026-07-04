'use client';

/**
 * SLA 模板管理页面
 *
 * 对应后端 /api/v1/sla/templates：
 *   - GET  /                 列出预置模板
 *   - GET  /:key             模板详情
 *   - POST /:key/install     安装到当前租户（幂等）
 *
 * 主要功能：
 *   1. 浏览 6 个开箱即用预置模板
 *   2. 查看模板详情（响应时间/解决时间/升级规则等）
 *   3. 一键安装到当前租户
 *   4. 批量安装全部推荐模板
 */

import {
  Alert,
  App,
  Button,
  Card,
  Col,
  Descriptions,
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
import { Download, Eye, CheckCircle, Rocket } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { SLATemplateApi, type SLATemplate, type TemplateInstallResult } from '@/lib/api/sla-template-api';

const { Title, Text, Paragraph } = Typography;

const priorityColorMap: Record<string, string> = {
  P1: 'red',
  P2: 'orange',
  P3: 'gold',
  P4: 'blue',
};

const industryLabelMap: Record<string, string> = {
  incident: '事件',
  change: '变更',
  service_request: '服务请求',
};

function formatMinutes(mins?: number): string {
  if (!mins || mins <= 0) return '-';
  if (mins < 60) return `${mins} 分钟`;
  if (mins < 1440) return `${(mins / 60).toFixed(1)} 小时`;
  return `${(mins / 1440).toFixed(1)} 天`;
}

export default function SLATemplatesPage() {
  const { message } = App.useApp();
  const [templates, setTemplates] = useState<SLATemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [installingKey, setInstallingKey] = useState<string | null>(null);
  const [detail, setDetail] = useState<SLATemplate | null>(null);
  const [installResults, setInstallResults] = useState<Record<string, TemplateInstallResult>>({});

  const loadTemplates = async () => {
    setLoading(true);
    try {
      const data = await SLATemplateApi.listTemplates();
      setTemplates(data);
    } catch (err) {
       
      console.error(err);
      message.error('加载 SLA 模板失败');
      setTemplates([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadTemplates();
  }, []);

  const stats = useMemo(() => {
    const recommended = templates.filter(t => t.recommended).length;
    const byIndustry = templates.reduce<Record<string, number>>((acc, t) => {
      acc[t.industry] = (acc[t.industry] ?? 0) + 1;
      return acc;
    }, {});
    return { total: templates.length, recommended, byIndustry };
  }, [templates]);

  const handleInstall = async (key: string) => {
    setInstallingKey(key);
    try {
      const result = await SLATemplateApi.installTemplate(key);
      setInstallResults(prev => ({ ...prev, [key]: result }));
      if (result.created) {
        message.success(`模板「${key}」已成功安装到当前租户`);
      } else {
        message.info(`模板「${key}」已存在，未重复安装`);
      }
    } catch (err) {
       
      console.error(err);
      message.error(`安装模板「${key}」失败`);
    } finally {
      setInstallingKey(null);
    }
  };

  const handleInstallAll = async () => {
    setInstallingKey('__ALL__');
    try {
      const results = await SLATemplateApi.installAllRecommended();
      const map: Record<string, TemplateInstallResult> = {};
      let createdCount = 0;
      let existingCount = 0;
      results.forEach(r => {
        map[r.templateKey] = r;
        if (r.created) createdCount += 1;
        else existingCount += 1;
      });
      setInstallResults(prev => ({ ...prev, ...map }));
      message.success(`批量安装完成：新增 ${createdCount} 个，已存在 ${existingCount} 个`);
    } catch (err) {
       
      console.error(err);
      message.error('批量安装失败');
    } finally {
      setInstallingKey(null);
    }
  };

  const columns = [
    {
      title: '模板',
      key: 'template',
      render: (_: unknown, t: SLATemplate) => (
        <div>
          <Space size="small">
            <Text strong>{t.name}</Text>
            <Tag color="blue">{t.key}</Tag>
            {t.recommended && <Tag color="green">推荐</Tag>}
          </Space>
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>
              {t.description}
            </Text>
          </div>
        </div>
      ),
    },
    {
      title: '行业',
      dataIndex: 'industry',
      key: 'industry',
      width: 110,
      render: (v: string) => <Tag color="geekblue">{industryLabelMap[v] ?? v}</Tag>,
    },
    {
      title: '服务类型',
      dataIndex: 'serviceType',
      key: 'serviceType',
      width: 130,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 90,
      render: (v: string) => <Tag color={priorityColorMap[v] ?? 'default'}>{v}</Tag>,
    },
    {
      title: '响应时间',
      dataIndex: 'responseTime',
      key: 'responseTime',
      width: 110,
      render: (v: number) => formatMinutes(v),
    },
    {
      title: '解决时间',
      dataIndex: 'resolutionTime',
      key: 'resolutionTime',
      width: 110,
      render: (v: number) => formatMinutes(v),
    },
    {
      title: '安装状态',
      key: 'installStatus',
      width: 140,
      render: (_: unknown, t: SLATemplate) => {
        const r = installResults[t.key];
        if (!r) return <Text type="secondary">未安装</Text>;
        if (r.created) return <Tag color="green" icon={<CheckCircle />}>已安装（新建）</Tag>;
        if (r.wasAlreadyExist) return <Tag color="blue">已存在</Tag>;
        return <Tag color="default">-</Tag>;
      },
    },
    {
      title: '操作',
      key: 'actions',
      width: 220,
      render: (_: unknown, t: SLATemplate) => (
        <Space>
          <Tooltip title="查看模板详情">
            <Button type="text" icon={<Eye />} onClick={() => setDetail(t)} />
          </Tooltip>
          <Popconfirm
            title={`确认将模板「${t.name}」安装到当前租户？`}
            description="幂等操作：已存在则跳过"
            okText="确认安装"
            cancelText="取消"
            onConfirm={() => handleInstall(t.key)}
          >
            <Button
              type="primary"
              size="small"
              icon={<Download />}
              loading={installingKey === t.key}
            >
              安装
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title level={3} style={{ marginBottom: 4 }}>
          <Rocket style={{ marginRight: 8 }} />
          SLA 模板
        </Title>
        <Paragraph type="secondary">
          开箱即用的 SLA 模板，按行业（事件 / 变更 / 服务请求）和优先级预置。一键安装到当前租户即可生效，无需手工配置。
        </Paragraph>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic title="模板总数" value={stats.total} suffix="个" />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic title="推荐模板" value={stats.recommended} suffix="个" valueStyle={{ color: '#52c41a' }} />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card>
            <Statistic
              title="覆盖行业"
              value={Object.keys(stats.byIndustry).length}
              suffix="个"
              valueStyle={{ color: '#1677ff' }}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <Alert
          message="安装说明"
          description={
            <ul style={{ marginBottom: 0, paddingLeft: 20 }}>
              <li>每个模板对应一张 SLA 定义表（sla_definitions）。</li>
              <li>安装操作是幂等的：相同 key 重复安装不会产生重复记录。</li>
              <li>已安装的 SLA 定义可在「SLA 定义管理」页面查看和编辑。</li>
            </ul>
          }
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />
        <Space style={{ marginBottom: 12 }}>
          <Popconfirm
            title="批量安装全部推荐模板？"
            description="仅会安装 recommended=true 的模板，跳过已存在的"
            okText="全部安装"
            cancelText="取消"
            onConfirm={handleInstallAll}
          >
            <Button
              type="primary"
              icon={<Rocket />}
              loading={installingKey === '__ALL__'}
              disabled={templates.filter(t => t.recommended).length === 0}
            >
              批量安装全部推荐模板
            </Button>
          </Popconfirm>
        </Space>
        <Table
          rowKey="key"
          loading={loading}
          dataSource={templates}
          columns={columns}
          pagination={false}
          scroll={{ x: 'max-content' }}
        />
      </Card>

      <Modal
        open={!!detail}
        title={detail ? `模板详情 · ${detail.name}` : '模板详情'}
        onCancel={() => setDetail(null)}
        footer={[
          <Button key="close" onClick={() => setDetail(null)}>
            关闭
          </Button>,
        ]}
        width={720}
      >
        {detail && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="模板 Key" span={2}>
              <Tag color="blue">{detail.key}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="名称" span={2}>
              {detail.name}
            </Descriptions.Item>
            <Descriptions.Item label="行业" span={2}>
              <Tag color="geekblue">{industryLabelMap[detail.industry] ?? detail.industry}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="服务类型">{detail.serviceType}</Descriptions.Item>
            <Descriptions.Item label="优先级">
              <Tag color={priorityColorMap[detail.priority] ?? 'default'}>{detail.priority}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="响应时间">{formatMinutes(detail.responseTime)}</Descriptions.Item>
            <Descriptions.Item label="解决时间">{formatMinutes(detail.resolutionTime)}</Descriptions.Item>
            <Descriptions.Item label="推荐模板" span={2}>
              {detail.recommended ? <Tag color="green">是</Tag> : <Tag>否</Tag>}
            </Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>
              {detail.description}
            </Descriptions.Item>
            <Descriptions.Item label="升级规则" span={2}>
              <pre style={{ background: '#f5f5f5', padding: 8, borderRadius: 4, margin: 0 }}>
                {JSON.stringify(detail.escalationRules, null, 2)}
              </pre>
            </Descriptions.Item>
            <Descriptions.Item label="适用条件" span={2}>
              <pre style={{ background: '#f5f5f5', padding: 8, borderRadius: 4, margin: 0 }}>
                {JSON.stringify(detail.conditions, null, 2)}
              </pre>
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
}