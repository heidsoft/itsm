'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  DatePicker,
  Button,
  Space,
  Row,
  Col,
  Statistic,
  Spin,
  Empty,
  Select,
  Typography,
  Table,
  Tag,
  Progress,
  App,
} from 'antd';
import { useRouter } from 'next/navigation';
import { PageContainer } from '@/components/layout/PageContainer';
import { useI18n } from '@/lib/i18n/useI18n';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { Minus, BarChart3, LineChart, PieChart, TrendingUp, TrendingDown } from 'lucide-react';
import { ProblemApi, type ProblemTrendData } from '@/lib/api/problem-api';

const { RangePicker } = DatePicker;
const { Text, Title } = Typography;

// 趋势方向映射
const trendDirectionConfig: Record<string, { color: string; icon: React.ReactNode; text: string }> =
  {
    increasing: { color: '#ff4d4f', icon: <TrendingUp />, text: 'rising' },
    decreasing: { color: '#52c41a', icon: <TrendingDown />, text: 'Descent' },
    stable: { color: '#1890ff', icon: <Minus />, text: 'stable' },
  };

export default function ProblemTrendsPage() {
  const router = useRouter();
  const { t } = useI18n();
  const { message } = App.useApp();

  // 数据加载初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 「暂无趋势数据」结论，再被真实数据覆盖。
  const [listLoading, setListLoading] = useState(true);
  // 「查询」按钮的动作态，与列表加载态分开。
  const [actionLoading, setActionLoading] = useState(false);
  const [trendData, setTrendData] = useState<ProblemTrendData | null>(null);
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([
    dayjs().subtract(6, 'month'),
    dayjs(),
  ]);
  const [categoryFilter, setCategoryFilter] = useState<string | undefined>(undefined);

  const fetchTrendData = async () => {
    setListLoading(true);
    setActionLoading(true);
    try {
      const response = await ProblemApi.getTrends({
        startDate: dateRange[0].format('YYYY-MM-DD'),
        endDate: dateRange[1].format('YYYY-MM-DD'),
      });
      setTrendData(response);
    } catch (error) {
      console.error('Failed to fetch trend data:', error);
      message.error('Failed to obtain trend data');
    } finally {
      setListLoading(false);
      setActionLoading(false);
    }
  };

  useEffect(() => {
    fetchTrendData();
  }, []);

  const handleDateRangeChange = (dates: [Dayjs | null, Dayjs | null] | null) => {
    if (dates && dates[0] && dates[1]) {
      setDateRange([dates[0], dates[1]]);
    }
  };

  const handleSearch = () => {
    fetchTrendData();
  };

  const trendConfig = trendData?.trendDirection
    ? trendDirectionConfig[trendData.trendDirection] || trendDirectionConfig.stable
    : trendDirectionConfig.stable;

  // 获取分类选项
  const categoryOptions = trendData?.categoryBreakdown
    ? Object.keys(trendData.categoryBreakdown).map(cat => ({
        value: cat,
        label: `${cat} (${trendData.categoryBreakdown[cat]})`,
      }))
    : [];

  // 月度趋势表格数据
  const monthlyTrendColumns = [
    {
      title: 'month',
      dataIndex: 'month',
      key: 'month',
      width: 120,
    },
    {
      title: 'Number of questions',
      dataIndex: 'count',
      key: 'count',
      width: 120,
      render: (count: number) => <Text strong>{count}</Text>,
    },
    {
      title: 'Resolved',
      dataIndex: 'resolved',
      key: 'resolved',
      width: 100,
      render: (resolved: number) => <Tag color="green">{resolved}</Tag>,
    },
    {
      title: 'Pending',
      dataIndex: 'open',
      key: 'open',
      width: 100,
      render: (open: number) => <Tag color="orange">{open}</Tag>,
    },
    {
      title: 'Resolution rate',
      key: 'rate',
      render: (_value: unknown, record: { count: number; resolved: number }) => {
        const rate = record.count > 0 ? (record.resolved / record.count) * 100 : 0;
        return <Progress percent={Math.round(rate)} size="small" strokeColor="#52c41a" />;
      },
    },
  ];

  return (
    <PageContainer title="Problem trend analysis" description="Analyze the changing trends and distribution of problems">
      <div className="space-y-6">
        {/* 筛选条件 */}
        <Card className="shadow-sm rounded-lg">
          <Space wrap size="large">
            <div>
              <Text type="secondary" className="mb-2 block">

                Time Range
              </Text>
              <RangePicker
                value={dateRange}
                onChange={handleDateRangeChange}
                format="YYYY-MM-DD"
                allowClear={false}
              />
            </div>
            <div>
              <Text type="secondary" className="mb-2 block">

                Category Filter
              </Text>
              <Select
                placeholder="All categories"
                value={categoryFilter}
                onChange={setCategoryFilter}
                allowClear
                options={categoryOptions}
                style={{ width: 200 }}
              />
            </div>
            <div className="self-end">
              <Button type="primary" onClick={handleSearch} loading={actionLoading}>
                Query
              </Button>
            </div>
          </Space>
        </Card>

        {listLoading ? (
          <Card className="shadow-sm rounded-lg">
            <div className="flex justify-center py-12">
              <Spin size="large" />
            </div>
          </Card>
        ) : trendData ? (
          <>
            {/* 概览统计 */}
            <Row gutter={[16, 16]}>
              <Col xs={24} sm={12} lg={6}>
                <Card className="shadow-sm rounded-lg">
                  <Statistic
                    title="Total number of questions"
                    value={trendData.totalProblems || 0}
                    prefix={<BarChart3 className="text-blue-500" />}
                    styles={{ content: { color: '#1890ff' } }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="shadow-sm rounded-lg">
                  <Statistic
                    title="Resolved"
                    value={trendData.resolvedProblems || 0}
                    prefix={<LineChart className="text-green-500" />}
                    styles={{ content: { color: '#52c41a' } }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="shadow-sm rounded-lg">
                  <Statistic
                    title="Pending"
                    value={trendData.openProblems || 0}
                    prefix={<PieChart className="text-orange-500" />}
                    styles={{ content: { color: '#fa8c16' } }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="shadow-sm rounded-lg">
                  <Statistic
                    title="Resolution rate"
                    value={(trendData.resolutionRate || 0) * 100}
                    precision={1}
                    suffix="%"
                    prefix={<LineChart className="text-purple-500" />}
                    styles={{ content: { color: trendData.resolutionRate >= 0.7 ? '#52c41a' : '#fa8c16' } }}
                  />
                </Card>
              </Col>
            </Row>

            {/* 趋势指示 */}
            <Card className="shadow-sm rounded-lg">
              <div className="flex items-center justify-between">
                <div>
                  <Title level={5} className="mb-2">

                    Trend direction
                  </Title>
                  <Space>
                    <Text type="secondary">Problem trends within time range:</Text>
                    <Tag
                      icon={trendConfig.icon}
                      color={trendConfig.color}
                      style={{ fontSize: 16, padding: '4px 12px' }}
                    >
                      {trendConfig.text}
                    </Tag>
                  </Space>
                </div>
                <div className="text-right">
                  <Text type="secondary">Avg Resolution Time</Text>
                  <Title level={3} className="mt-1 mb-0">
                    {trendData.avgResolutionTimeHours?.toFixed(1) || 0}{' '}
                    <span className="text-sm">hours</span>
                  </Title>
                </div>
              </div>
            </Card>

            {/* 分类分布 */}
            <Row gutter={[16, 16]}>
              <Col xs={24} lg={12}>
                <Card title="Classification distribution" className="shadow-sm rounded-lg">
                  {trendData.categoryBreakdown &&
                  Object.keys(trendData.categoryBreakdown).length > 0 ? (
                    <div className="space-y-3">
                      {Object.entries(trendData.categoryBreakdown)
                        .sort(([, a], [, b]) => b - a)
                        .slice(0, 10)
                        .map(([category, count]) => {
                          const total = Object.values(trendData.categoryBreakdown).reduce(
                            (a, b) => a + b,
                            0
                          );
                          const percentage = total > 0 ? (count / total) * 100 : 0;
                          return (
                            <div key={category}>
                              <div className="flex justify-between mb-1">
                                <Text>{category}</Text>
                                <Text strong>{count}</Text>
                              </div>
                              <Progress
                                percent={Math.round(percentage)}
                                size="small"
                                strokeColor="#1890ff"
                                showInfo={false}
                              />
                            </div>
                          );
                        })}
                    </div>
                  ) : (
                    <Empty description="No classified data yet" />
                  )}
                </Card>
              </Col>
              <Col xs={24} lg={12}>
                <Card title="Priority distribution" className="shadow-sm rounded-lg">
                  {trendData.priorityBreakdown &&
                  Object.keys(trendData.priorityBreakdown).length > 0 ? (
                    <div className="space-y-3">
                      {Object.entries(trendData.priorityBreakdown)
                        .sort(([, a], [, b]) => b - a)
                        .map(([priority, count]) => {
                          const total = Object.values(trendData.priorityBreakdown).reduce(
                            (a, b) => a + b,
                            0
                          );
                          const percentage = total > 0 ? (count / total) * 100 : 0;
                          const colorMap: Record<string, string> = {
                            critical: '#ff4d4f',
                            high: '#fa8c16',
                            medium: '#1890ff',
                            low: '#52c41a',
                          };
                          return (
                            <div key={priority}>
                              <div className="flex justify-between mb-1">
                                <Tag color={colorMap[priority] || 'default'}>
                                  {priority === 'critical'
                                    ? 'Urgent'
                                    : priority === 'high'
                                      ? 'High'
                                      : priority === 'medium'
                                        ? 'Medium'
                                        : 'Low'}
                                </Tag>
                                <Text strong>{count}</Text>
                              </div>
                              <Progress
                                percent={Math.round(percentage)}
                                size="small"
                                strokeColor={colorMap[priority] || '#1890ff'}
                                showInfo={false}
                              />
                            </div>
                          );
                        })}
                    </div>
                  ) : (
                    <Empty description="No priority data yet" />
                  )}
                </Card>
              </Col>
            </Row>

            {/* 月度趋势 */}
            <Card title="Monthly Trend" className="shadow-sm rounded-lg">
              {trendData.monthlyTrend && trendData.monthlyTrend.length > 0 ? (
                <Table
                  columns={monthlyTrendColumns}
                  dataSource={trendData.monthlyTrend.map((item, idx) => ({
                    ...item,
                    key: idx,
                  }))}
                  pagination={false}
                  size="small"
                />
              ) : (
                <Empty description="No monthly trend data yet" />
              )}
            </Card>
          </>
        ) : (
          <Card className="shadow-sm rounded-lg">
            <Empty description="There is no trend data yet, please select a time range and query" />
          </Card>
        )}
      </div>
    </PageContainer>
  );
}
