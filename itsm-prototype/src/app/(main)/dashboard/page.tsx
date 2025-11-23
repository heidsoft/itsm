'use client';

import React, { useCallback, useState } from 'react';
import {
  Card,
  Button,
  Switch,
  Tooltip,
  message,
  Space,
  Badge,
  Dropdown,
  Tabs,
  Divider,
  Row,
  Col,
  Skeleton,
} from 'antd';
import {
  RefreshCw,
  Settings,
  LayoutDashboard,
  Zap,
  LineChart,
  TrendingUp,
} from 'lucide-react';
import { KPICards } from './components/KPICards';
import { ChartsSection } from './components/ChartsSection';
import { QuickActions } from './components/QuickActions';
import { useDashboardData } from './hooks/useDashboardData';
import { QuickAction } from './types/dashboard.types';
import TicketTrendChart from './components/TicketTrendChart';
import IncidentDistributionChart from './components/IncidentDistributionChart';
import SLAComplianceChart from './components/SLAComplianceChart';
import UserSatisfactionChart from './components/UserSatisfactionChart';
import ResponseTimeChart from './components/ResponseTimeChart';
import TeamWorkloadChart from './components/TeamWorkloadChart';
import PeakHoursChart from './components/PeakHoursChart';

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
  } = useDashboardData();

  const [activeChartTab, setActiveChartTab] = useState('all');

  // 处理快速操作点击
  const handleQuickActionClick = useCallback((action: QuickAction) => {
    message.info(`正在跳转到 ${action.title}...`);
    if (action.permission) {
      console.log(`检查权限: ${action.permission}`);
    }
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

  // 控制栏下拉菜单
  const controlMenuItems = [
    {
      key: 'auto-refresh',
      label: (
        <div
          className='flex items-center justify-between min-w-[200px]'
          onClick={e => e.stopPropagation()}
        >
          <span className='text-sm font-medium'>自动刷新</span>
          <Switch checked={autoRefresh} onChange={handleAutoRefreshToggle} size='small' />
        </div>
      ),
    },
    {
      key: 'interval',
      label: (
        <div
          className='flex items-center justify-between min-w-[200px]'
          onClick={e => e.stopPropagation()}
        >
          <span className='text-sm font-medium'>刷新间隔</span>
          <select
            value={refreshInterval}
            onChange={e => handleRefreshIntervalChange(Number(e.target.value))}
            className='px-2 py-1 border border-gray-300 rounded text-xs font-medium'
            disabled={!autoRefresh}
            onClick={e => e.stopPropagation()}
          >
            <option value={10000}>10秒</option>
            <option value={30000}>30秒</option>
            <option value={60000}>1分钟</option>
            <option value={300000}>5分钟</option>
          </select>
        </div>
      ),
    },
    { type: 'divider' as const },
    {
      key: 'connection',
      label: (
        <div className='flex items-center justify-between min-w-[200px]'>
          <span className='text-sm font-medium'>连接状态</span>
          <Badge
            status={isConnected ? 'success' : 'error'}
            text={isConnected ? '已连接' : '未连接'}
          />
        </div>
      ),
    },
    {
      key: 'last-updated',
      label: (
        <div className='flex items-center justify-between min-w-[200px]'>
          <span className='text-sm font-medium'>最后更新</span>
          <span className='text-xs text-gray-500 font-medium'>
            {lastUpdated ? new Date(lastUpdated).toLocaleTimeString() : '从未'}
          </span>
        </div>
      ),
    },
  ];

  // 错误状态
  if (error) {
    return (
      <Card
        className='text-center py-16 border-0'
        style={{
          borderRadius: '12px',
          boxShadow: '0 1px 3px rgba(0, 0, 0, 0.12)',
        }}
      >
        <div className='text-red-500 mb-4'>
          <LayoutDashboard style={{ fontSize: 64 }} />
        </div>
        <h3 className='text-xl font-bold text-gray-900 mb-2'>仪表盘加载失败</h3>
        <p className='text-gray-600 mb-6'>{error}</p>
        <Button
          type='primary'
          size='large'
          onClick={refresh}
          icon={<RefreshCw />}
          style={{
            height: '44px',
            borderRadius: '8px',
          }}
        >
          重新加载
        </Button>
      </Card>
    );
  }

  const renderSkeletons = () => (
    <>
      <Row gutter={[16, 16]}>
        {Array.from({ length: 4 }).map((_, index) => (
          <Col key={index} xs={24} sm={12} md={12} lg={6}>
            <Skeleton active paragraph={{ rows: 4 }} />
          </Col>
        ))}
      </Row>
      <Divider />
      <Skeleton active paragraph={{ rows: 2 }} />
      <Divider />
      <Skeleton active paragraph={{ rows: 8 }} />
    </>
  );

  return (
    <div className='enterprise-dashboard-container'>
      {/* 简化顶部工具栏 */}
      <div className='flex items-center justify-between mb-6 pb-4 border-b border-gray-200'>
        <div>
          <h1 className='text-2xl font-bold text-gray-900 flex items-center gap-2'>
            ITSM 运营仪表盘
            {isConnected && <Badge status='success' text='在线' />}
          </h1>
          <p className='text-sm text-gray-600 mt-1'>实时监控系统运行状态和关键业务指标</p>
        </div>

        <Space size='middle'>
          {lastUpdated && (
            <span className='text-sm text-gray-500'>
              更新于 {new Date(lastUpdated).toLocaleTimeString()}
            </span>
          )}

          <Button
            type='default'
            icon={<RefreshCw spin={loading} />}
            onClick={refresh}
          >
            刷新
          </Button>

          <Dropdown menu={{ items: controlMenuItems }} trigger={['click']} placement='bottomRight'>
            <Button icon={<Settings />}>设置</Button>
          </Dropdown>
        </Space>
      </div>

      {loading ? (
        renderSkeletons()
      ) : (
        <>
          {/* KPI指标卡片区域 */}
          <div className='mb-6'>
            <KPICards metrics={data?.kpiMetrics || []} loading={loading} />
          </div>

          <Divider style={{ margin: '32px 0' }} />

          {/* 快速操作区域 */}
          <div className='mb-6'>
            <div className='mb-4'>
              <h2 className='text-lg font-bold text-gray-900 mb-1 flex items-center gap-2'>
                <Zap style={{ fontSize: 22, color: '#f97316' }} />
                快速操作
              </h2>
              <p className='text-sm text-gray-600'>常用功能快捷入口，提升工作效率</p>
            </div>
            <QuickActions
              actions={data?.quickActions || []}
              loading={loading}
              onActionClick={handleQuickActionClick}
              showTitle={false}
              compact={false}
            />
          </div>

          <Divider style={{ margin: '32px 0' }} />

          {/* 图表分析区域 */}
          <div className='mb-6'>
            <Card
              className='border-0 shadow-sm'
              style={{
                borderRadius: '12px',
              }}
              styles={{ body: { padding: '24px' } }}
            >
              <div className='mb-5'>
                <h2 className='text-lg font-bold text-gray-900 mb-1 flex items-center gap-2'>
                  <LineChart style={{ fontSize: 22, color: '#3b82f6' }} />
                  数据分析与趋势
                </h2>
                <p className='text-sm text-gray-600'>系统性能和业务趋势的可视化分析</p>
              </div>

              <Tabs
                activeKey={activeChartTab}
                onChange={setActiveChartTab}
                size='large'
                items={[
                  {
                    key: 'all',
                    label: (
                      <span className='flex items-center gap-2'>
                        <LayoutDashboard />
                        全部图表
                      </span>
                    ),
                    children: (
                      <div className='pt-4'>
                        <ChartsSection loading={loading}>
                          <Col xs={24} lg={12}>
                            <TicketTrendChart data={data?.ticketTrend || []} />
                          </Col>
                          <Col xs={24} lg={12}>
                            <IncidentDistributionChart data={data?.incidentDistribution || []} />
                          </Col>
                          <Col xs={24} lg={12}>
                            <ResponseTimeChart data={data?.responseTimeDistribution || []} />
                          </Col>
                          <Col xs={24} lg={12}>
                            <TeamWorkloadChart data={data?.teamWorkload || []} />
                          </Col>
                          <Col xs={24} lg={12}>
                            <SLAComplianceChart data={data?.slaData || []} />
                          </Col>
                          <Col xs={24} lg={12}>
                            <PeakHoursChart data={data?.peakHours || []} />
                          </Col>
                          <Col xs={24} lg={12}>
                            <UserSatisfactionChart data={data?.satisfactionData || []} />
                          </Col>
                        </ChartsSection>
                      </div>
                    ),
                  },
                  {
                    key: 'tickets',
                    label: (
                      <span className='flex items-center gap-2'>
                        <LineChart />
                        工单与事件
                      </span>
                    ),
                    children: (
                      <div className='pt-4'>
                        <ChartsSection loading={loading}>
                          <Col xs={24} lg={12}>
                            <TicketTrendChart data={data?.ticketTrend || []} />
                          </Col>
                          <Col xs={24} lg={12}>
                            <IncidentDistributionChart data={data?.incidentDistribution || []} />
                          </Col>
                          <Col xs={24} lg={12}>
                            <ResponseTimeChart data={data?.responseTimeDistribution || []} />
                          </Col>
                          <Col xs={24} lg={12}>
                            <PeakHoursChart data={data?.peakHours || []} />
                          </Col>
                        </ChartsSection>
                      </div>
                    ),
                  },
                  {
                    key: 'performance',
                    label: (
                      <span className='flex items-center gap-2'>
                        <TrendingUp />
                        性能与满意度
                      </span>
                    ),
                    children: (
                      <div className='pt-4'>
                        <ChartsSection loading={loading}>
                          <Col xs={24} lg={12}>
                            <SLAComplianceChart data={data?.slaData || []} />
                          </Col>
                          <Col xs={24} lg={12}>
                            <UserSatisfactionChart data={data?.satisfactionData || []} />
                          </Col>
                          <Col xs={24} lg={12}>
                            <TeamWorkloadChart data={data?.teamWorkload || []} />
                          </Col>
                        </ChartsSection>
                      </div>
                    ),
                  },
                ]}
              />
            </Card>
          </div>
        </>
      )}
    </div>
  );
}
