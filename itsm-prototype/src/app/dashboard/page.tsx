'use client';

import React, { useCallback } from 'react';
import { Row, Col, Card, Button, Switch, Tooltip, message, Space, Badge } from 'antd';
import { RefreshCw, Wifi, WifiOff, Settings, Activity } from 'lucide-react';
import { KPICards } from './components/KPICards';
import { ChartsSection } from './components/ChartsSection';
import { RecentActivity } from './components/RecentActivity';
import { QuickActions } from './components/QuickActions';
import { useDashboardData } from './hooks/useDashboardData';
import { QuickAction } from './types/dashboard.types';

export default function DashboardPage() {
  const {
    data,
    loading,
    error,
    lastUpdated,
    autoRefresh,
    refreshInterval,
    refresh,
    setAutoRefresh,
    setRefreshInterval,
    isConnected,
    connectionStatus,
  } = useDashboardData();

  // 处理快速操作点击
  const handleQuickActionClick = useCallback((action: QuickAction) => {
    message.info(`正在跳转到 ${action.title}...`);
    // 这里可以添加权限检查逻辑
    if (action.permission) {
      // 检查用户权限
      console.log(`检查权限: ${action.permission}`);
    }
  }, []);

  // 处理查看所有活动
  const handleViewAllActivities = useCallback(() => {
    message.info('正在打开活动日志...');
    // 导航到活动日志页面
  }, []);

  // 处理刷新间隔变化
  const handleRefreshIntervalChange = useCallback(
    (interval: number) => {
      setRefreshInterval(interval);
      message.success(`刷新间隔已更新为 ${interval / 1000}秒`);
    },
    [setRefreshInterval]
  );

  // 处理自动刷新切换
  const handleAutoRefreshToggle = useCallback(
    (enabled: boolean) => {
      setAutoRefresh(enabled);
      message.success(`自动刷新已${enabled ? '启用' : '禁用'}`);
    },
    [setAutoRefresh]
  );

  // 错误状态
  if (error) {
    return (
      <Card className='text-center py-12'>
        <div className='text-red-500 mb-4'>
          <Activity size={48} className='mx-auto' />
        </div>
        <h3 className='text-lg font-semibold text-gray-900 mb-2'>仪表盘加载失败</h3>
        <p className='text-gray-600 mb-4'>{error}</p>
        <Button type='primary' onClick={refresh} icon={<RefreshCw size={16} />}>
          重试
        </Button>
      </Card>
    );
  }

  return (
    <div className='space-y-6'>
      {/* 仪表盘控制栏 */}
      <Card className='mb-6 shadow-sm border-0'>
        <div className='flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4'>
          <div className='flex items-center gap-4'>
            <div className='flex items-center gap-2'>
              <span className='text-sm font-medium text-gray-700'>自动刷新:</span>
              <Switch checked={autoRefresh} onChange={handleAutoRefreshToggle} size='small' />
            </div>

            <div className='flex items-center gap-2'>
              <span className='text-sm font-medium text-gray-700'>间隔:</span>
              <select
                value={refreshInterval}
                onChange={e => handleRefreshIntervalChange(Number(e.target.value))}
                className='px-2 py-1 border border-gray-300 rounded text-sm'
                disabled={!autoRefresh}
              >
                <option value={10000}>10秒</option>
                <option value={30000}>30秒</option>
                <option value={60000}>1分钟</option>
                <option value={300000}>5分钟</option>
              </select>
            </div>

            <div className='flex items-center gap-2'>
              <Tooltip title={`连接状态: ${connectionStatus}`}>
                {isConnected ? (
                  <Badge status='success' text='已连接' />
                ) : (
                  <Badge status='error' text='未连接' />
                )}
              </Tooltip>
            </div>
          </div>

          <div className='flex items-center gap-2'>
            {lastUpdated && (
              <span className='text-xs text-gray-500'>
                最后更新: {new Date(lastUpdated).toLocaleTimeString()}
              </span>
            )}
            <Button
              type='default'
              size='small'
              onClick={refresh}
              loading={loading}
              icon={<RefreshCw size={14} />}
            >
              刷新
            </Button>
          </div>
        </div>
      </Card>

      {/* KPI指标卡片 */}
      <KPICards metrics={data?.kpiMetrics || []} loading={loading} />

      {/* 图表区域 */}
      <ChartsSection
        ticketTrend={data?.ticketTrend || []}
        incidentDistribution={data?.incidentDistribution || []}
        slaData={data?.slaData || []}
        satisfactionData={data?.satisfactionData || []}
        loading={loading}
      />

      {/* 底部区域 - 最近活动和快速操作 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <RecentActivity
            activities={data?.recentActivities || []}
            loading={loading}
            onViewAll={handleViewAllActivities}
          />
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title={
              <div className='flex items-center gap-2'>
                <Settings size={18} className='text-blue-500' />
                <span>系统状态</span>
              </div>
            }
            className='h-full shadow-sm border-0'
          >
            <div className='space-y-4'>
              <div className='flex items-center justify-between'>
                <span className='text-sm text-gray-600'>数据连接</span>
                <Badge
                  status={isConnected ? 'success' : 'error'}
                  text={isConnected ? '已连接' : '未连接'}
                />
              </div>

              <div className='flex items-center justify-between'>
                <span className='text-sm text-gray-600'>自动刷新</span>
                <Badge
                  status={autoRefresh ? 'success' : 'default'}
                  text={autoRefresh ? '已启用' : '已禁用'}
                />
              </div>

              <div className='flex items-center justify-between'>
                <span className='text-sm text-gray-600'>刷新间隔</span>
                <span className='text-sm font-medium'>{refreshInterval / 1000}秒</span>
              </div>

              <div className='flex items-center justify-between'>
                <span className='text-sm text-gray-600'>最后更新</span>
                <span className='text-sm font-medium'>
                  {lastUpdated ? new Date(lastUpdated).toLocaleTimeString() : '从未'}
                </span>
              </div>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 快速操作 */}
      <QuickActions
        actions={data?.quickActions || []}
        loading={loading}
        onActionClick={handleQuickActionClick}
      />
    </div>
  );
}
