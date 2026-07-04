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
  message,
} from 'antd';
import { useRouter } from 'next/navigation';
import { PageContainer } from '@/components/layout/PageContainer';
import { useI18n } from '@/lib/i18n/useI18n';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { Minus, BarChart3, LineChart, PieChart } from 'lucide-react';
import { ProblemApi, type ProblemTrendData } from '@/lib/api/problem-api';

const { RangePicker } = DatePicker;
const { Text, Title } = Typography;

// 趋势方向映射
const trendDirectionConfig: Record<string, { color: string; icon: React.ReactNode; text: string }> =
  {
    increasing: { color: '#ff4d4f', icon: <RiseOutlined />, text: '上升' },
    decreasing: { color: '#52c41a', icon: <FallOutlined />, text: '下降' },
    stable: { color: '#1890ff', icon: <Minus />, text: '稳定' },
  };

export default function ProblemTrendsPage() {
  const router = useRouter();
  const { t } = useI18n();

  const [loading, setLoading] = useState(false);
  const [trendData, setTrendData] = useState<ProblemTrendData | null>(null);
  const [dateRange, setDateRange] = useState<[Dayjs, Dayjs]>([
    dayjs().subtract(6, 'month'),
    dayjs(),
  ]);
  const [categoryFilter, setCategoryFilter] = useState<string | undefined>(undefined);

  const fetchTrendData = async () => {
    setLoading(true);
    try {
      const response = await ProblemApi.getTrends({
        start_date: dateRange[0].format('YYYY-MM-DD'),
        end_date: dateRange[1].format('YYYY-MM-DD'),
      });
      setTrendData(response);
    } catch (error) {
      console.error('Failed to fetch trend data:', error);
      message.error('获取趋势数据失败');
    } finally {
      setLoading(false);
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

  const trendConfig = trendData?.trend_direction
    ? trendDirectionConfig[trendData.trend_direction] || trendDirectionConfig.stable
    : trendDirectionConfig.stable;

  // 获取分类选项
  const categoryOptions = trendData?.category_breakdown
    ? Object.keys(trendData.category_breakdown).map(cat => ({
        value: cat,
        label: `${cat} (${trendData.category_breakdown[cat]})`,
      }))
    : [];

  // 月度趋势表格数据
  const monthlyTrendColumns = [
    {
      title: '月份',
      dataIndex: 'month',
      key: 'month',
      width: 120,
    },
    {
      title: '问题数量',
      dataIndex: 'count',
      key: 'count',
      width: 120,
      render: (count: number) => <Text strong>{count}</Text>,
    },
    {
      title: '已解决',
      dataIndex: 'resolved',
      key: 'resolved',
      width: 100,
      render: (resolved: number) => <Tag color="green">{resolved}</Tag>,
    },
    {
      title: '待处理',
      dataIndex: 'open',
      key: 'open',
      width: 100,
      render: (open: number) => <Tag color="orange">{open}</Tag>,
    },
    {
      title: '解决率',
      key: 'rate',
      render: (_: any, record: { count: number; resolved: number }) => {
        const rate = record.count > 0 ? (record.resolved / record.count) * 100 : 0;
        return <Progress percent={Math.round(rate)} size="small" strokeColor="#52c41a" />;
      },
    },
  ];

  return (
    <PageContainer title="问题趋势分析" description="分析问题的变化趋势和分布情况">
      <div className="space-y-6">
        {/* 筛选条件 */}
        <Card className="shadow-sm rounded-lg">
          <Space wrap size="large">
            <div>
              <Text type="secondary" className="mb-2 block">
                时间范围
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
                分类筛选
              </Text>
              <Select
                placeholder="全部分类"
                value={categoryFilter}
                onChange={setCategoryFilter}
                allowClear
                options={categoryOptions}
                style={{ width: 200 }}
              />
            </div>
            <div className="self-end">
              <Button type="primary" onClick={handleSearch} loading={loading}>
                查询
              </Button>
            </div>
          </Space>
        </Card>

        {loading ? (
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
                    title="问题总数"
                    value={trendData.total_problems || 0}
                    prefix={<BarChart3 className="text-blue-500" />}
                    valueStyle={{ color: '#1890ff' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="shadow-sm rounded-lg">
                  <Statistic
                    title="已解决"
                    value={trendData.resolved_problems || 0}
                    prefix={<LineChart className="text-green-500" />}
                    valueStyle={{ color: '#52c41a' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="shadow-sm rounded-lg">
                  <Statistic
                    title="待处理"
                    value={trendData.open_problems || 0}
                    prefix={<PieChart className="text-orange-500" />}
                    valueStyle={{ color: '#fa8c16' }}
                  />
                </Card>
              </Col>
              <Col xs={24} sm={12} lg={6}>
                <Card className="shadow-sm rounded-lg">
                  <Statistic
                    title="解决率"
                    value={(trendData.resolution_rate || 0) * 100}
                    precision={1}
                    suffix="%"
                    prefix={<LineChart className="text-purple-500" />}
                    valueStyle={{ color: trendData.resolution_rate >= 0.7 ? '#52c41a' : '#fa8c16' }}
                  />
                </Card>
              </Col>
            </Row>

            {/* 趋势指示 */}
            <Card className="shadow-sm rounded-lg">
              <div className="flex items-center justify-between">
                <div>
                  <Title level={5} className="mb-2">
                    趋势方向
                  </Title>
                  <Space>
                    <Text type="secondary">时间范围内的问题趋势：</Text>
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
                  <Text type="secondary">平均解决时间</Text>
                  <Title level={3} className="mt-1 mb-0">
                    {trendData.avg_resolution_time_hours?.toFixed(1) || 0}{' '}
                    <span className="text-sm">小时</span>
                  </Title>
                </div>
              </div>
            </Card>

            {/* 分类分布 */}
            <Row gutter={[16, 16]}>
              <Col xs={24} lg={12}>
                <Card title="分类分布" className="shadow-sm rounded-lg">
                  {trendData.category_breakdown &&
                  Object.keys(trendData.category_breakdown).length > 0 ? (
                    <div className="space-y-3">
                      {Object.entries(trendData.category_breakdown)
                        .sort(([, a], [, b]) => b - a)
                        .slice(0, 10)
                        .map(([category, count]) => {
                          const total = Object.values(trendData.category_breakdown).reduce(
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
                    <Empty description="暂无分类数据" />
                  )}
                </Card>
              </Col>
              <Col xs={24} lg={12}>
                <Card title="优先级分布" className="shadow-sm rounded-lg">
                  {trendData.priority_breakdown &&
                  Object.keys(trendData.priority_breakdown).length > 0 ? (
                    <div className="space-y-3">
                      {Object.entries(trendData.priority_breakdown)
                        .sort(([, a], [, b]) => b - a)
                        .map(([priority, count]) => {
                          const total = Object.values(trendData.priority_breakdown).reduce(
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
                                    ? '紧急'
                                    : priority === 'high'
                                      ? '高'
                                      : priority === 'medium'
                                        ? '中'
                                        : '低'}
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
                    <Empty description="暂无优先级数据" />
                  )}
                </Card>
              </Col>
            </Row>

            {/* 月度趋势 */}
            <Card title="月度趋势" className="shadow-sm rounded-lg">
              {trendData.monthly_trend && trendData.monthly_trend.length > 0 ? (
                <Table
                  columns={monthlyTrendColumns}
                  dataSource={trendData.monthly_trend.map((item, idx) => ({
                    ...item,
                    key: idx,
                  }))}
                  pagination={false}
                  size="small"
                />
              ) : (
                <Empty description="暂无月度趋势数据" />
              )}
            </Card>
          </>
        ) : (
          <Card className="shadow-sm rounded-lg">
            <Empty description="暂无趋势数据，请选择时间范围后查询" />
          </Card>
        )}
      </div>
    </PageContainer>
  );
}
