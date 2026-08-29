'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { Card, Row, Col, DatePicker, Space, Typography, Spin, App, Empty, Button, Alert } from 'antd';
import dayjs from 'dayjs';
import SLAApi, { type SLAMonitoringData } from '@/lib/api/sla-api';

const { Text } = Typography;
const { RangePicker } = DatePicker;

interface SLADashboardChartsProps {
  slaDefinitionId?: number;
  timeRange?: [dayjs.Dayjs, dayjs.Dayjs];
  refreshInterval?: number;
}

/**
 * SLA 图表面板（当前无页面引用，保留为趋势图占位实现）。
 *
 * 诚实口径：后端尚未提供按日时间序列接口（/api/v1/sla/metrics 只有指标记录，
 * 没有 trendData），因此趋势图区域渲染空态而不是用聚合值伪造曲线点。
 * 摘要卡片使用 POST /api/v1/sla/monitoring 的真实窗口聚合值。
 */
export const SLADashboardCharts: React.FC<SLADashboardChartsProps> = ({
  timeRange,
  refreshInterval = 300000, // 5分钟
}) => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const [monitoring, setMonitoring] = useState<SLAMonitoringData | null>(null);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs]>(
    timeRange || [dayjs().subtract(30, 'day'), dayjs()]
  );

  const loadData = useCallback(async () => {
    try {
      setLoading(true);
      setLoadError(false);
      const result = await SLAApi.getSLAMonitoring({
        startTime: dateRange[0].toISOString(),
        endTime: dateRange[1].toISOString(),
      });
      setMonitoring(result);
    } catch (error) {
      console.error('加载 SLA 统计失败:', error);
      message.error('加载 SLA 统计失败');
      setLoadError(true);
    } finally {
      setLoading(false);
    }
  }, [dateRange, message]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  useEffect(() => {
    const timer = setInterval(() => void loadData(), refreshInterval);
    return () => clearInterval(timer);
  }, [loadData, refreshInterval]);

  // 无样本时后端返回 0，界面必须显示「暂无样本」而不是 0%。
  const rateText = (samples: number, rate: number) =>
    samples > 0 ? `${rate.toFixed(1)}%` : '暂无样本';

  if (loading && !monitoring) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div className="sla-dashboard-charts space-y-6">
      {/* 控制面板 */}
      <Card>
        <Space wrap>
          <RangePicker
            value={dateRange}
            onChange={dates => {
              if (dates && dates[0] && dates[1]) {
                setDateRange([dates[0], dates[1]]);
              }
            }}
            aria-label="SLA 报表日期范围"
          />
          <Button onClick={() => void loadData()} loading={loading} aria-label="刷新 SLA 统计">
            刷新
          </Button>
        </Space>
      </Card>

      {loadError && !monitoring ? (
        <Card>
          <Empty description="SLA 统计加载失败">
            <Button type="primary" onClick={() => void loadData()}>
              重新加载
            </Button>
          </Empty>
        </Card>
      ) : (
        <>
          {/* 趋势图占位：后端暂无按日时间序列接口，禁止用聚合值伪造曲线 */}
          <Card title="SLA 趋势">
            <Empty description="趋势数据接口建设中：当前仅有窗口聚合指标，暂无按日时间序列" />
          </Card>

          {monitoring?.truncated && (
            <Alert
              type="warning"
              showIcon
              message="统计窗口内工单数超过扫描上限，以下指标基于截断后的样本计算"
            />
          )}

          {/* 统计摘要：全部来自 /api/v1/sla/monitoring 真实聚合 */}
          <Card title="数据统计摘要">
            <Row gutter={16}>
              <Col xs={24} sm={6}>
                <div className="text-center">
                  <div className="text-2xl font-bold text-green-600">
                    {monitoring ? rateText(monitoring.totalTickets, monitoring.complianceRate) : '-'}
                  </div>
                  <div className="text-gray-600">SLA 合规率</div>
                </div>
              </Col>
              <Col xs={24} sm={6}>
                <div className="text-center">
                  <div className="text-2xl font-bold text-blue-600">
                    {monitoring
                      ? monitoring.responseTimeSamples > 0
                        ? `${monitoring.averageResponseMinutes.toFixed(1)} 分钟`
                        : '暂无样本'
                      : '-'}
                  </div>
                  <div className="text-gray-600">平均响应时间</div>
                </div>
              </Col>
              <Col xs={24} sm={6}>
                <div className="text-center">
                  <div className="text-2xl font-bold text-orange-600">
                    {monitoring
                      ? monitoring.resolutionTimeSamples > 0
                        ? `${monitoring.averageResolutionMinutes.toFixed(1)} 分钟`
                        : '暂无样本'
                      : '-'}
                  </div>
                  <div className="text-gray-600">平均解决时间</div>
                </div>
              </Col>
              <Col xs={24} sm={6}>
                <div className="text-center">
                  <div className="text-2xl font-bold text-red-600">
                    {monitoring?.totalViolations ?? 0}
                  </div>
                  <div className="text-gray-600">违规记录数</div>
                </div>
              </Col>
            </Row>
            {monitoring && (
              <div className="mt-4">
                <Text type="secondary">
                  统计窗口：{dayjs(monitoring.startTime).format('YYYY-MM-DD HH:mm')} ~{' '}
                  {dayjs(monitoring.endTime).format('YYYY-MM-DD HH:mm')} · 工单{' '}
                  {monitoring.totalTickets} 条 · 已解决 {monitoring.resolvedTickets} 条（解决率{' '}
                  {rateText(monitoring.totalTickets, monitoring.resolutionRate)}）
                </Text>
              </div>
            )}
          </Card>
        </>
      )}
    </div>
  );
};

export default SLADashboardCharts;
