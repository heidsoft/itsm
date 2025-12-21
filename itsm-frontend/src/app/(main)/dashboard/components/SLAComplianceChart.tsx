'use client';

import React from 'react';
import { Progress, Tag, Statistic } from 'antd';
import { GaugeIcon } from 'lucide-react';
import { SLAData } from '../types/dashboard.types';
import { DashboardChartCard } from './DashboardChartCard';

const SLAComplianceChart: React.FC<{ data: SLAData[] }> = React.memo(({ data }) => {
  // 确保数据有效性
  const validData = Array.isArray(data)
    ? data.filter(
        item => item && typeof item.actual === 'number' && !isNaN(item.actual) && item.actual >= 0
      )
    : [];

  const averageSLA =
    validData.length > 0
      ? validData.reduce((sum, item) => sum + (item.actual ?? 0), 0) / validData.length
      : 0;

  // 确保 percent 在 0-1 之间，且是有效数字
  const safePercent = (() => {
    if (!validData || validData.length === 0) {
      console.warn('SLAComplianceChart: No valid data, using 0');
      return 0;
    }
    if (typeof averageSLA !== 'number' || isNaN(averageSLA) || averageSLA < 0) {
      console.warn('SLAComplianceChart: Invalid averageSLA:', averageSLA);
      return 0;
    }
    const percent = averageSLA / 100;
    if (isNaN(percent) || !isFinite(percent)) {
      console.warn('SLAComplianceChart: Invalid percent:', percent);
      return 0;
    }
    const clampedPercent = Math.max(0, Math.min(1, percent));
    // 仅在开发环境输出调试日志
    if (
      process.env.NODE_ENV === 'development' &&
      process.env.NEXT_PUBLIC_DEBUG_DASHBOARD === 'true'
    ) {
      console.log('SLAComplianceChart: safePercent calculated:', {
        averageSLA,
        percent,
        clampedPercent,
        validData,
      });
    }
    return clampedPercent;
  })();

  // 确保percent是数字类型，且是有效的0-100之间的值（用于Progress组件）
  const percentValue = (() => {
    if (typeof averageSLA !== 'number' || isNaN(averageSLA) || !isFinite(averageSLA)) {
      return 0;
    }
    // Progress组件使用0-100的值
    return Math.max(0, Math.min(100, Number(averageSLA)));
  })();

  // 根据SLA值确定颜色
  const getProgressColor = (value: number) => {
    if (value >= 95) return '#22c55e'; // 绿色
    if (value >= 90) return '#a3e635'; // 浅绿
    if (value >= 85) return '#fbbf24'; // 黄色
    if (value >= 80) return '#f59e0b'; // 橙色
    return '#ef4444'; // 红色
  };

  const getSLAStatus = (value: number) => {
    if (value >= 95) return { text: '优秀', color: 'success' };
    if (value >= 90) return { text: '良好', color: 'processing' };
    if (value >= 85) return { text: '一般', color: 'warning' };
    return { text: '需改进', color: 'error' };
  };

  const status = getSLAStatus(averageSLA);

  return (
    <DashboardChartCard
      title='SLA 达成率监控'
      subtitle='服务水平协议整体达成情况'
      icon={<GaugeIcon style={{ width: 20, height: 20 }} />}
      iconColor='#8b5cf6'
      extra={<Tag color={status.color as any}>{status.text}</Tag>}
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
        {validData.length > 0 && typeof percentValue === 'number' && !isNaN(percentValue) ? (
          <>
            <Statistic
              title='SLA 达成率'
              value={averageSLA}
              precision={1}
              suffix='%'
              valueStyle={{
                fontSize: '48px',
                fontWeight: 'bold',
                color: getProgressColor(averageSLA),
                marginBottom: '24px',
              }}
            />
            <Progress
              type='dashboard'
              percent={percentValue}
              strokeColor={getProgressColor(averageSLA)}
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
            暂无数据
          </div>
        )}
      </div>

      {/* SLA服务详情 */}
      {validData.length > 0 && (
        <div className='mt-4 space-y-3'>
          {validData.map((item, index) => {
            const actual = typeof item.actual === 'number' && !isNaN(item.actual) ? item.actual : 0;
            const target = typeof item.target === 'number' && !isNaN(item.target) ? item.target : 0;
            return (
              <div key={index} className='flex items-center justify-between'>
                <span className='text-sm text-gray-600 font-medium'>
                  {item.service || '未知服务'}
                </span>
                <div className='flex items-center gap-3'>
                  <Progress
                    percent={actual}
                    size='small'
                    strokeColor={actual >= target ? '#22c55e' : '#f59e0b'}
                    style={{ width: 120 }}
                    format={percent => `${percent?.toFixed(1) || 0}%`}
                  />
                  <span className='text-xs text-gray-500 w-12 text-right'>目标{target}%</span>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </DashboardChartCard>
  );
});

SLAComplianceChart.displayName = 'SLAComplianceChart';

export default SLAComplianceChart;
