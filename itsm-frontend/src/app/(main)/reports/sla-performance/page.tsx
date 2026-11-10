'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { App, Button, Card, Col, Empty, Progress, Row, Skeleton, Spin, Statistic, Typography } from 'antd';
import { Clock, RotateCcw, CheckCircle, XCircle } from 'lucide-react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import dayjs from 'dayjs';

import { SLAApi } from '@/lib/api/sla-api';

const { Title, Text } = Typography;

const COLORS = {
  met: '#52c41a',
  breached: '#ff4d4f',
  warning: '#faad14',
};

interface SLAData {
  name: string;
  met: number;
  breached: number;
  compliance: number;
}

const DEFAULT_RANGE_DAYS = 30;

const SLAPerformanceReport = () => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [slaData, setSLAData] = useState<SLAData[]>([]);
  const [overallRate, setOverallRate] = useState<number>(0);
  const [totalMet, setTotalMet] = useState<number>(0);
  const [totalBreached, setTotalBreached] = useState<number>(0);
  const [hasData, setHasData] = useState<boolean>(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setHasData(false);
    try {
      // 后端按 RFC3339 解析日期，只传 YYYY-MM-DD 会 400 并静默回退到统计接口
      const endDate = dayjs().endOf('day').toISOString();
      const startDate = dayjs().subtract(DEFAULT_RANGE_DAYS, 'day').startOf('day').toISOString();

      // 并行获取 SLA 总体合规率 + 按服务类型的真实分维表现
      const [complianceReport, stats, performance] = await Promise.allSettled([
        SLAApi.getSLAComplianceReport({ startDate, endDate }),
        SLAApi.getSLAStats(),
        SLAApi.getSLAPerformance({
          dimension: 'serviceType',
          startDate,
          endDate,
          page: 1,
          pageSize: 20,
        }),
      ]);

      const report =
        complianceReport.status === 'fulfilled' ? complianceReport.value : null;
      const statsValue = stats.status === 'fulfilled' ? stats.value : null;
      const perfRows =
        performance.status === 'fulfilled' && Array.isArray(performance.value?.items)
          ? performance.value.items
          : [];

      const totalMetCount = report?.metSla ?? statsValue?.totalDefinitions ?? 0;
      const totalBreachedCount = report?.violatedSla ?? statsValue?.openViolations ?? 0;
      const compliance = report?.complianceRate ?? statsValue?.overallComplianceRate ?? 0;

      setTotalMet(totalMetCount);
      setTotalBreached(totalBreachedCount);
      setOverallRate(compliance);

      // 按服务类型构建分布数据：每行 complianceRate 是后端按该维度真实聚合的达成率。
      // 历史缺陷：此处曾用总体合规率冒充每个定义的达成率（def.complianceRate 后端不返回），
      // 导致图中每一行显示同一数字；现改走 GET /sla/performance 真实分维端点。
      if (perfRows.length > 0) {
        const list: SLAData[] = perfRows.slice(0, 8).map(row => {
          const rate = Math.round(row.complianceRate);
          return {
            name: row.key === 'unassigned' ? 'SLA not bound' : row.key,
            met: rate,
            breached: 100 - rate,
            compliance: rate,
          };
        });
        setSLAData(list);
        setHasData(true);
      } else if (totalMetCount + totalBreachedCount > 0 || compliance > 0) {
        // 没有 SLA 定义列表时使用总体数据单条记录
        const met = Math.round(compliance);
        setSLAData([
          {
            name: "Overall SLA Compliance",
            met,
            breached: 100 - met,
            compliance: met,
          },
        ]);
        setHasData(true);
      } else {
        setSLAData([]);
        setHasData(false);
      }
    } catch (error) {
      console.error('Failed to load SLA performance data:', error);
      message.error('Failed to load data');
      setSLAData([]);
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const pieData = [
    { name: "Compliance", value: totalMet, color: COLORS.met },
    { name: "Violation", value: totalBreached, color: COLORS.breached },
  ];

  interface TooltipPayloadEntry {
    color?: string;
    name?: string;
    value?: number | string;
  }

  const CustomTooltip = ({
    active,
    payload,
    label,
  }: {
    active?: boolean;
    payload?: TooltipPayloadEntry[];
    label?: string | number;
  }) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white p-3 rounded-lg shadow-lg border border-gray-200">
          <p className="font-semibold text-gray-800 mb-2">{`SLA type: ${label ?? ''}`}</p>
          {payload.map((entry, index) => (
            <p
              key={`${entry.name ?? 'item'}-${index}`}
              className="text-sm"
              style={{ color: entry.color }}
            >
              {`${entry.name ?? ''}: ${entry.value ?? 0}%`}
            </p>
          ))}
        </div>
      );
    }
    return null;
  };

  const getComplianceStatus = (compliance: number): 'success' | 'warning' | 'danger' => {
    if (compliance >= 95) return 'success';
    if (compliance >= 85) return 'warning';
    return 'danger';
  };

  const renderContent = () => {
    if (loading) {
      return (
        <div className="space-y-4">
          <Row gutter={[16, 16]}>
            {[0, 1, 2].map(i => (
              <Col xs={24} sm={8} key={`skeleton-${i}`}>
                <Card>
                  <Skeleton active />
                </Card>
              </Col>
            ))}
          </Row>
          <Skeleton active paragraph={{ rows: 6 }} />
        </div>
      );
    }

    if (!hasData || slaData.length === 0) {
      return (
        <Card>
          <Empty description="No SLA performance data yet" />
        </Card>
      );
    }

    const totalForPie = totalMet + totalBreached;

    return (
      <>
        {/* 统计卡片 */}
        <Row gutter={[16, 16]} className="mb-6">
          <Col xs={24} sm={8}>
            <Card>
              <Statistic
                title="Average Compliance Rate"
                value={overallRate}
                suffix="%"
                styles={{
                  content: {
                    color:
                      overallRate >= 95
                        ? COLORS.met
                        : overallRate >= 85
                          ? COLORS.warning
                          : COLORS.breached,
                  },
                }}
                prefix={<CheckCircle />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card>
              <Statistic
                title="Number of times achieved"
                value={totalMet}
                styles={{ content: { color: COLORS.met } }}
                prefix={<CheckCircle />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card>
              <Statistic
                title="Number of violations"
                value={totalBreached}
                styles={{ content: { color: COLORS.breached } }}
                prefix={<XCircle />}
              />
            </Card>
          </Col>
        </Row>

        {/* 图表区域 */}
        <Row gutter={[16, 16]} className="mb-6">
          <Col xs={24} lg={16}>
            <Card title="Achievement status of various types of SLA">
              <ResponsiveContainer width="100%" height={350}>
                <BarChart data={slaData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis />
                  <Tooltip content={<CustomTooltip />} />
                  <Legend />
                  <Bar dataKey="met" name="Compliance" stackId="a" fill={COLORS.met} />
                  <Bar dataKey="breached" name="Violation" stackId="a" fill={COLORS.breached} />
                </BarChart>
              </ResponsiveContainer>
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card title="Overall compliance rate">
              <div className="text-center py-8">
                <div className="text-5xl font-bold mb-4" style={{ color: COLORS.met }}>
                  {totalForPie > 0 ? ((totalMet / totalForPie) * 100).toFixed(1) : '0.0'}%
                </div>
                <Text type="secondary">Overall SLA compliance rate</Text>
                <Progress
                  percent={totalForPie > 0 ? (totalMet / totalForPie) * 100 : 0}
                  strokeColor={COLORS.met}
                  showInfo={false}
                  className="mt-4"
                />
              </div>
            </Card>
          </Col>
        </Row>

        {/* 合规率详情 */}
        <Card title="Details of each SLA compliance rate">
          <Row gutter={[16, 16]}>
            {slaData.map((sla, index) => (
              <Col xs={24} sm={12} md={8} key={`${sla.name}-${index}`}>
                <Card size="small" className="h-full">
                  <div className="flex items-center justify-between mb-4">
                    <Text strong>{sla.name}</Text>
                    <Text type={getComplianceStatus(sla.compliance)} strong>
                      {sla.compliance}%
                    </Text>
                  </div>
                  <Progress
                    percent={sla.compliance}
                    strokeColor={
                      sla.compliance >= 95
                        ? COLORS.met
                        : sla.compliance >= 85
                          ? COLORS.warning
                          : COLORS.breached
                    }
                    showInfo={false}
                  />
                  <div className="flex justify-between mt-2 text-sm text-gray-500">
                    <span>
                      <CheckCircle
                        className="inline mr-1"
                        style={{ color: COLORS.met }}
                      />
                      {sla.met}%
                    </span>
                    <span>
                      <XCircle
                        className="inline mr-1"
                        style={{ color: COLORS.breached }}
                      />
                      {sla.breached}%
                    </span>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        </Card>
      </>
    );
  };

  return (
    <div className="p-6 bg-gray-50 min-h-full">
      <header className="mb-6">
        <Title level={2}>SLA Performance Report</Title>
        <p className="text-gray-500 mt-1">Display the achievement of service level agreement and performance indicators</p>
      </header>

      {/* 控制栏 */}
      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col>
            <Text className="text-gray-600">SLA compliance rate monitoring</Text>
          </Col>
          <Col>
            <Button icon={<RotateCcw />} onClick={loadData} loading={loading}>

              Refresh Data
            </Button>
          </Col>
        </Row>
      </Card>

      {renderContent()}
    </div>
  );
};

export default SLAPerformanceReport;