'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Col,
  Empty,
  Progress,
  Row,
  Segmented,
  Skeleton,
  Statistic,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip as ChartTooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { ReloadOutlined } from '@ant-design/icons';
import { DashboardAPI } from '@/lib/api/dashboard-api';
import type { ServiceQualityReport } from '@/app/(main)/dashboard/types/dashboard.types';

const { Text } = Typography;

const WINDOW_OPTIONS = [
  { label: '近 30 天', value: 30 },
  { label: '近 60 天', value: 60 },
  { label: '近 90 天', value: 90 },
];

// 块缺省（后端对应域未装配）时如实标注「未接入」，不用全零冒充统计结果。
function MissingBlock({ label }: { label: string }) {
  return (
    <Card size="small" className="h-full">
      <Text type="secondary">{label}</Text>
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={
          <Text type="secondary">
            该统计块未接入
            <br />
            <Text type="secondary" style={{ fontSize: 12 }}>
              后端对应域未装配时此块缺省，属于显式降级
            </Text>
          </Text>
        }
        style={{ margin: '12px 0' }}
      />
    </Card>
  );
}

function StatusTable({
  rows,
  nameKey,
}: {
  rows: Array<{ name: string; count: number }>;
  nameKey: string;
}) {
  return (
    <Table
      size="small"
      pagination={false}
      rowKey={row => row.name}
      dataSource={rows}
      columns={[
        { title: nameKey, dataIndex: 'name', key: 'name' },
        { title: '数量', dataIndex: 'count', key: 'count', width: 90, align: 'right' },
      ]}
    />
  );
}

export default function ServiceQualityOverview() {
  const [days, setDays] = useState<number>(30);

  const qualityQuery = useQuery({
    queryKey: ['service-quality', days],
    queryFn: () => DashboardAPI.getServiceQuality(days),
  });

  const report = qualityQuery.data;

  return (
    <Card
      title="服务质量总览"
      extra={
        <div className="flex items-center gap-3">
          {report && (
            <Tooltip title={`口径版本 ${report.metricVersion}：任何指标的计算语义变化都会递增版本，跨期对比以此为准`}>
              <Tag color="blue">口径 v{report.metricVersion}</Tag>
            </Tooltip>
          )}
          {report && (
            <Tag color={report.scope === 'tenant' ? 'geekblue' : 'green'}>
              {report.scope === 'tenant' ? '全租户口径' : '本人口径'}
            </Tag>
          )}
          <Segmented options={WINDOW_OPTIONS} value={days} onChange={value => setDays(value as number)} />
        </div>
      }
    >
      {qualityQuery.isPending ? (
        // 首次拉取完成前显示加载骨架，不渲染空数字——空态和「共 0」必须等真实数据
        <Skeleton active paragraph={{ rows: 8 }} />
      ) : qualityQuery.isError ? (
        <Alert
          type="error"
          showIcon
          title="服务质量总览加载失败"
          description="统计服务暂时不可用，稍后可以重试。"
          action={
            <Button size="small" icon={<ReloadOutlined />} onClick={() => qualityQuery.refetch()}>
              重试
            </Button>
          }
        />
      ) : report ? (
        <div className="space-y-4">
          <Row gutter={[12, 12]}>
            {report.kpi.map(metric => (
              <Col key={metric.id} xs={12} md={6}>
                <Card size="small">
                  <Statistic
                    title={metric.title}
                    value={metric.value}
                    suffix={metric.unit}
                    styles={{ content: metric.color ? { color: metric.color } : undefined }}
                  />
                  {metric.change !== undefined && metric.change !== 0 && (
                    <Text
                      type={metric.trend === 'up' ? 'danger' : metric.trend === 'down' ? 'success' : 'secondary'}
                      style={{ fontSize: 12 }}
                    >
                      较上期 {metric.trend === 'down' ? '↓' : '↑'} {Math.abs(metric.change)}
                      {metric.unit}
                    </Text>
                  )}
                </Card>
              </Col>
            ))}
          </Row>

          {report.sla && (
            <Row gutter={[12, 12]}>
              <Col xs={12} md={6}>
                <Card size="small">
                  <Statistic title="SLA 整体合规率" value={report.sla.complianceRate} suffix="%" precision={1} />
                </Card>
              </Col>
              <Col xs={12} md={6}>
                <Card size="small">
                  <Statistic
                    title="响应时限达成率"
                    value={report.sla.responseTimeCompliance}
                    suffix="%"
                    precision={1}
                  />
                </Card>
              </Col>
              <Col xs={12} md={6}>
                <Card size="small">
                  <Statistic
                    title="解决时限达成率"
                    value={report.sla.resolutionTimeCompliance}
                    suffix="%"
                    precision={1}
                  />
                </Card>
              </Col>
              <Col xs={12} md={6}>
                <Card size="small">
                  <Statistic
                    title="24 小时内到期"
                    value={report.sla.atRiskTickets}
                    suffix="张"
                    styles={{ content: report.sla.atRiskTickets > 0 ? { color: '#faad14' } : undefined }}
                  />
                </Card>
              </Col>
            </Row>
          )}

          <Row gutter={[12, 12]}>
            <Col xs={24} lg={14}>
              <Card size="small" title="工单趋势（新建 / 已解决）">
                <ResponsiveContainer width="100%" height={220}>
                  <LineChart data={report.ticketTrend}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" tick={{ fontSize: 11 }} />
                    <YAxis allowDecimals={false} tick={{ fontSize: 11 }} />
                    <ChartTooltip />
                    <Legend />
                    <Line type="monotone" dataKey="newTickets" name="新建" stroke="#1677ff" dot={false} />
                    <Line type="monotone" dataKey="completedTickets" name="已解决" stroke="#52c41a" dot={false} />
                  </LineChart>
                </ResponsiveContainer>
              </Card>
            </Col>
            <Col xs={24} lg={10}>
              <Card size="small" title="事件分布" className="h-full">
                {report.incidentDistribution.length > 0 ? (
                  <StatusTable
                    nameKey="事件分类"
                    rows={report.incidentDistribution.map(item => ({ name: item.category, count: item.count }))}
                  />
                ) : (
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="窗口内没有事件" />
                )}
              </Card>
            </Col>
          </Row>

          <Row gutter={[12, 12]}>
            <Col xs={24} lg={8}>
              {report.ticketStats ? (
                <Card size="small" title="工单统计">
                  <Statistic title="工单总数" value={report.ticketStats.total} />
                  <div className="mt-2">
                    <StatusTable
                      nameKey="状态"
                      rows={report.ticketStats.statusGroups.map(g => ({ name: g.status, count: g.count }))}
                    />
                  </div>
                </Card>
              ) : (
                <MissingBlock label="工单统计" />
              )}
            </Col>
            <Col xs={24} lg={8}>
              {report.problemStats ? (
                <Card size="small" title="问题统计">
                  <Row gutter={8}>
                    <Col span={8}>
                      <Statistic title="总数" value={report.problemStats.total} />
                    </Col>
                    <Col span={8}>
                      <Statistic
                        title="待处理"
                        value={report.problemStats.open + report.problemStats.inProgress}
                        styles={{ content: { color: '#faad14' } }}
                      />
                    </Col>
                    <Col span={8}>
                      <Statistic
                        title="高优先级"
                        value={report.problemStats.highPriority}
                        styles={{ content: { color: '#ff4d4f' } }}
                      />
                    </Col>
                  </Row>
                  <div className="mt-2">
                    <StatusTable
                      nameKey="状态"
                      rows={report.problemStats.byStatus.map(g => ({ name: g.status, count: g.count }))}
                    />
                  </div>
                </Card>
              ) : (
                <MissingBlock label="问题统计" />
              )}
            </Col>
            <Col xs={24} lg={8}>
              {report.changeStats ? (
                <Card size="small" title="变更统计">
                  <Row gutter={8}>
                    <Col span={8}>
                      <Statistic title="总数" value={report.changeStats.total} />
                    </Col>
                    <Col span={8}>
                      <Statistic
                        title="待审批"
                        value={report.changeStats.pending}
                        styles={{ content: { color: '#faad14' } }}
                      />
                    </Col>
                    <Col span={8}>
                      <Statistic
                        title="实施失败"
                        value={report.changeStats.failed}
                        styles={{ content: { color: '#ff4d4f' } }}
                      />
                    </Col>
                  </Row>
                  <div className="mt-2">
                    <StatusTable
                      nameKey="类型"
                      rows={report.changeStats.byType.map(g => ({ name: String(g.type), count: g.count }))}
                    />
                  </div>
                </Card>
              ) : (
                <MissingBlock label="变更统计" />
              )}
            </Col>
          </Row>

          {report.satisfaction.length > 0 && (
            <Card size="small" title="满意度（按月）">
              <Row gutter={[12, 12]}>
                {report.satisfaction.map(item => (
                  <Col key={item.month} xs={12} md={6}>
                    <div className="text-center">
                      <Text type="secondary">{item.month}</Text>
                      <Progress
                        type="circle"
                        size={64}
                        percent={Math.round((item.rating / 5) * 100)}
                        format={() => `${item.rating.toFixed(1)}`}
                      />
                      <div>
                        <Text type="secondary" style={{ fontSize: 12 }}>
                          {item.responses} 份评价
                        </Text>
                      </div>
                    </div>
                  </Col>
                ))}
              </Row>
            </Card>
          )}
        </div>
      ) : null}
    </Card>
  );
}
