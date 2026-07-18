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
      const endDate = dayjs().format('YYYY-MM-DD');
      const startDate = dayjs().subtract(DEFAULT_RANGE_DAYS, 'day').format('YYYY-MM-DD');

      // 并行获取 SLA 总体合规率和按类型分布
      const [complianceReport, stats, definitions] = await Promise.allSettled([
        SLAApi.getSLAComplianceReport({ startDate, endDate }),
        SLAApi.getSLAStats(),
        SLAApi.getSLADefinitions({ page: 1, size: 50 }),
      ]);

      const report =
        complianceReport.status === 'fulfilled' ? complianceReport.value : null;
      const statsValue = stats.status === 'fulfilled' ? stats.value : null;

      const totalMetCount = report?.metSla ?? statsValue?.totalDefinitions ?? 0;
      const totalBreachedCount = report?.violatedSla ?? statsValue?.openViolations ?? 0;
      const compliance = report?.complianceRate ?? statsValue?.overallComplianceRate ?? 0;

      setTotalMet(totalMetCount);
      setTotalBreached(totalBreachedCount);
      setOverallRate(compliance);

      // 按 SLA 定义构建分布数据；没有定义时回退到总体数据
      if (
        definitions.status === 'fulfilled' &&
        Array.isArray(definitions.value?.items) &&
        definitions.value.items.length > 0
      ) {
        const items = definitions.value.items;
        const perDefinitionRate = totalMetCount + totalBreachedCount > 0
          ? compliance
          : 0;
        const list: SLAData[] = items.slice(0, 6).map(def => {
          const rate = def.complianceRate ?? perDefinitionRate;
          const breached = 100 - rate;
          return {
            name: def.name || `SLA-${def.id}`,
            met: Math.round(rate),
            breached: Math.round(breached),
            compliance: Math.round(rate),
          };
        });
        setSLAData(list);
        setHasData(true);
      } else if (totalMetCount + totalBreachedCount > 0 || compliance > 0) {
        // 没有 SLA 定义列表时使用总体数据单条记录
        const met = Math.round(compliance);
        setSLAData([
          {
            name: 'SLA 总体合规',
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
      console.error('加载SLA性能数据失败:', error);
      message.error('加载数据失败');
      setSLAData([]);
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const pieData = [
    { name: '达标', value: totalMet, color: COLORS.met },
    { name: '违规', value: totalBreached, color: COLORS.breached },
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
          <p className="font-semibold text-gray-800 mb-2">{`SLA类型: ${label ?? ''}`}</p>
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
          <Empty description="暂无SLA性能数据" />
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
                title="平均合规率"
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
                title="达标次数"
                value={totalMet}
                styles={{ content: { color: COLORS.met } }}
                prefix={<CheckCircle />}
              />
            </Card>
          </Col>
          <Col xs={24} sm={8}>
            <Card>
              <Statistic
                title="违规次数"
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
            <Card title="各类型SLA达成情况">
              <ResponsiveContainer width="100%" height={350}>
                <BarChart data={slaData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis />
                  <Tooltip content={<CustomTooltip />} />
                  <Legend />
                  <Bar dataKey="met" name="达标" stackId="a" fill={COLORS.met} />
                  <Bar dataKey="breached" name="违规" stackId="a" fill={COLORS.breached} />
                </BarChart>
              </ResponsiveContainer>
            </Card>
          </Col>
          <Col xs={24} lg={8}>
            <Card title="总体达标率">
              <div className="text-center py-8">
                <div className="text-5xl font-bold mb-4" style={{ color: COLORS.met }}>
                  {totalForPie > 0 ? ((totalMet / totalForPie) * 100).toFixed(1) : '0.0'}%
                </div>
                <Text type="secondary">SLA总体达标率</Text>
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
        <Card title="各SLA合规率详情">
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
        <Title level={2}>SLA性能报表</Title>
        <p className="text-gray-500 mt-1">展示服务级别协议的达成情况和性能指标</p>
      </header>

      {/* 控制栏 */}
      <Card className="mb-6">
        <Row justify="space-between" align="middle">
          <Col>
            <Text className="text-gray-600">SLA合规率监控</Text>
          </Col>
          <Col>
            <Button icon={<RotateCcw />} onClick={loadData} loading={loading}>
              刷新数据
            </Button>
          </Col>
        </Row>
      </Card>

      {renderContent()}
    </div>
  );
};

export default SLAPerformanceReport;