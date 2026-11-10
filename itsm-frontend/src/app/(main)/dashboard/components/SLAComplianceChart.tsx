'use client';

import React from 'react';
import { Progress, Tag, Statistic } from 'antd';
import { GaugeIcon } from 'lucide-react';
import type { SLAComplianceData } from '../types/dashboard.types';
import { DashboardChartCard } from './DashboardChartCard';

/**
 * SLA compliance monitoring。
 *
 * 数据源是 /dashboard/overview 的 slaData —— **一个聚合对象**，不是按服务拆分
 * 的数组。此前组件按 `SLAData[]` 消费并先做 `Array.isArray(data)` 过滤，
 * 后端返回对象时恒为空数组，面板一直显示"No data"（契约与实现同时错，
 * 类型检查却发现不了）。现在直接读对象字段，三个达成率分别来自后端按
 * violation_type 分桶的真实结果，不再互相抄。
 */
const SLAComplianceChart: React.FC<{
  data?: SLAComplianceData | null;
  overallValue?: number;
}> = React.memo(({ data, overallValue }) => {
  const num = (v: unknown): number | undefined =>
    typeof v === 'number' && !isNaN(v) && isFinite(v) ? v : undefined;

  // KPI 卡片的 sla-compliance 与 slaData.complianceRate 同源，但任一缺失时用另一个兜底：
  // 只要求「至少有一个真数字」，而不是要求数组非空。
  const compliance = num(data?.complianceRate) ?? num(overallValue);
  const totalTickets = num(data?.totalTickets) ?? 0;

  const getProgressColor = (value: number) => {
    if (value >= 95) return '#22c55e'; // 绿色
    if (value >= 90) return '#a3e635'; // 浅绿
    if (value >= 85) return '#fbbf24'; // 黄色
    if (value >= 80) return '#f59e0b'; // 橙色
    return '#ef4444'; // 红色
  };

  const getSLAStatus = (value: number) => {
    if (value >= 95) return { text: 'Excellent', color: 'success' };
    if (value >= 90) return { text: 'Good', color: 'processing' };
    if (value >= 85) return { text: 'Average', color: 'warning' };
    return { text: 'Needs improvement', color: 'error' };
  };

  const hasValue = compliance !== undefined;
  const status = hasValue ? getSLAStatus(compliance) : null;
  const percentValue = hasValue ? Math.max(0, Math.min(100, compliance)) : 0;

  // 三条分项：分母都是近 30 天工单总数，所以「无工单」时一律不展示百分比。
  const breakdown = [
    {
      label: 'Response time compliance',
      value: num(data?.responseTimeCompliance),
      hint: 'Percentage of tickets with an on-time first response',
    },
    {
      label: 'Resolution time compliance',
      value: num(data?.resolutionTimeCompliance),
      hint: 'Percentage of tickets resolved on time',
    },
  ];

  return (
    <DashboardChartCard
      title="SLA compliance monitoring"
      subtitle={
        totalTickets > 0
          ? `Last 30 days: ${totalTickets} tickets, ${data?.breachedTickets ?? 0} breached, ${data?.atRiskTickets ?? 0} at risk`
          : 'No tickets in the last 30 days; compliance is not applicable'
      }
      icon={<GaugeIcon style={{ width: 20, height: 20 }} />}
      iconColor="#8b5cf6"
      extra={status ? <Tag color={status.color as any}>{status.text}</Tag> : undefined}
    >
      <div
        style={{
          height: '280px',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          alignItems: 'center',
        }}
      >
        {hasValue ? (
          <>
            <Statistic
              title="SLA compliance"
              value={compliance}
              precision={1}
              suffix="%"
              styles={{
                content: {
                  fontSize: '48px',
                  fontWeight: 'bold',
                  color: getProgressColor(compliance),
                  marginBottom: '24px',
                },
              }}
            />
            <Progress
              type="dashboard"
              percent={percentValue}
              strokeColor={getProgressColor(compliance)}
              size={200}
              format={percent => `${percent?.toFixed(1) || 0}%`}
              strokeWidth={12}
            />
          </>
        ) : (
          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%',
              color: '#999',
            }}
          >

            No data
          </div>
        )}
      </div>

      {hasValue && totalTickets > 0 && (
        <div className="mt-4 space-y-3">
          {breakdown.map(item => (
            <div key={item.label} className="flex items-center justify-between">
              <div className="flex flex-col">
                <span className="text-sm text-gray-600 font-medium">{item.label}</span>
                <span className="text-xs text-gray-400">{item.hint}</span>
              </div>
              {item.value === undefined ? (
                <span className="text-xs text-gray-400">No data</span>
              ) : (
                <Progress
                  percent={Math.max(0, Math.min(100, item.value))}
                  size="small"
                  strokeColor={getProgressColor(item.value)}
                  style={{ width: 140 }}
                  format={percent => `${percent?.toFixed(1) || 0}%`}
                />
              )}
            </div>
          ))}
        </div>
      )}
    </DashboardChartCard>
  );
});

SLAComplianceChart.displayName = 'SLAComplianceChart';

export default SLAComplianceChart;
