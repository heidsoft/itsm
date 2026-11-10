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
  { label: "Last 30 Days", value: 30 },
  { label: "Last 60 Days", value: 60 },
  { label: "Last 90 Days", value: 90 },
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
            This statistics block is not connected
            <br />
            <Text type="secondary" style={{ fontSize: 12 }}>
              This block is unavailable because its backend domain is not configured.
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
        { title: "Count", dataIndex: 'count', key: 'count', width: 90, align: 'right' },
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
      title="Service Quality Overview"
      extra={
        <div className="flex items-center gap-3">
          {report && (
            <Tooltip title={`Metric version ${report.metricVersion}: calculation changes increment this version; use it when comparing periods`}>
              <Tag color="blue">Metric v{report.metricVersion}</Tag>
            </Tooltip>
          )}
          {report && (
            <Tag color={report.scope === 'tenant' ? 'geekblue' : 'green'}>
              {report.scope === 'tenant' ? "All tenant records" : "Own records"}
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
          title="Failed to load service quality overview"
          description="Statistics are temporarily unavailable. Please try again later."
          action={
            <Button size="small" icon={<ReloadOutlined />} onClick={() => qualityQuery.refetch()}>
              Retry
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
                      Compared with previous period {metric.trend === 'down' ? '↓' : '↑'} {Math.abs(metric.change)}
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
                  <Statistic title="Overall SLA Compliance" value={report.sla.complianceRate} suffix="%" precision={1} />
                </Card>
              </Col>
              <Col xs={12} md={6}>
                <Card size="small">
                  <Statistic
                    title="Response Deadline Compliance"
                    value={report.sla.responseTimeCompliance}
                    suffix="%"
                    precision={1}
                  />
                </Card>
              </Col>
              <Col xs={12} md={6}>
                <Card size="small">
                  <Statistic
                    title="Resolution Deadline Compliance"
                    value={report.sla.resolutionTimeCompliance}
                    suffix="%"
                    precision={1}
                  />
                </Card>
              </Col>
              <Col xs={12} md={6}>
                <Card size="small">
                  <Statistic
                    title="Due Within 24 Hours"
                    value={report.sla.atRiskTickets}
                    suffix="tickets"
                    styles={{ content: report.sla.atRiskTickets > 0 ? { color: '#faad14' } : undefined }}
                  />
                </Card>
              </Col>
            </Row>
          )}

          <Row gutter={[12, 12]}>
            <Col xs={24} lg={14}>
              <Card size="small" title="Ticket Trends (Created / Resolved)">
                <ResponsiveContainer width="100%" height={220}>
                  <LineChart data={report.ticketTrend}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" tick={{ fontSize: 11 }} />
                    <YAxis allowDecimals={false} tick={{ fontSize: 11 }} />
                    <ChartTooltip />
                    <Legend />
                    <Line type="monotone" dataKey="newTickets" name="Created" stroke="#1677ff" dot={false} />
                    <Line type="monotone" dataKey="completedTickets" name="Resolved" stroke="#52c41a" dot={false} />
                  </LineChart>
                </ResponsiveContainer>
              </Card>
            </Col>
            <Col xs={24} lg={10}>
              <Card size="small" title="Incident Distribution" className="h-full">
                {report.incidentDistribution.length > 0 ? (
                  <StatusTable
                    nameKey="Incident Category"
                    rows={report.incidentDistribution.map(item => ({ name: item.category, count: item.count }))}
                  />
                ) : (
                  <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No incidents in this period" />
                )}
              </Card>
            </Col>
          </Row>

          <Row gutter={[12, 12]}>
            <Col xs={24} lg={8}>
              {report.ticketStats ? (
                <Card size="small" title="Ticket Statistics">
                  <Statistic title="Total Tickets" value={report.ticketStats.total} />
                  <div className="mt-2">
                    <StatusTable
                      nameKey="Status"
                      rows={report.ticketStats.statusGroups.map(g => ({ name: g.status, count: g.count }))}
                    />
                  </div>
                </Card>
              ) : (
                <MissingBlock label="Ticket Statistics" />
              )}
            </Col>
            <Col xs={24} lg={8}>
              {report.problemStats ? (
                <Card size="small" title="Problem Statistics">
                  <Row gutter={8}>
                    <Col span={8}>
                      <Statistic title="Total" value={report.problemStats.total} />
                    </Col>
                    <Col span={8}>
                      <Statistic
                        title="Open"
                        value={report.problemStats.open + report.problemStats.inProgress}
                        styles={{ content: { color: '#faad14' } }}
                      />
                    </Col>
                    <Col span={8}>
                      <Statistic
                        title="High Priority"
                        value={report.problemStats.highPriority}
                        styles={{ content: { color: '#ff4d4f' } }}
                      />
                    </Col>
                  </Row>
                  <div className="mt-2">
                    <StatusTable
                      nameKey="Status"
                      rows={report.problemStats.byStatus.map(g => ({ name: g.status, count: g.count }))}
                    />
                  </div>
                </Card>
              ) : (
                <MissingBlock label="Problem Statistics" />
              )}
            </Col>
            <Col xs={24} lg={8}>
              {report.changeStats ? (
                <Card size="small" title="Change Statistics">
                  <Row gutter={8}>
                    <Col span={8}>
                      <Statistic title="Total" value={report.changeStats.total} />
                    </Col>
                    <Col span={8}>
                      <Statistic
                        title="Awaiting Approval"
                        value={report.changeStats.pending}
                        styles={{ content: { color: '#faad14' } }}
                      />
                    </Col>
                    <Col span={8}>
                      <Statistic
                        title="Implementation Failed"
                        value={report.changeStats.failed}
                        styles={{ content: { color: '#ff4d4f' } }}
                      />
                    </Col>
                  </Row>
                  <div className="mt-2">
                    <StatusTable
                      nameKey="Type"
                      rows={report.changeStats.byType.map(g => ({ name: String(g.type), count: g.count }))}
                    />
                  </div>
                </Card>
              ) : (
                <MissingBlock label="Change Statistics" />
              )}
            </Col>
          </Row>

          {report.satisfaction.length > 0 && (
            <Card size="small" title="Satisfaction by Month">
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
                          {item.responses} ratings
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
