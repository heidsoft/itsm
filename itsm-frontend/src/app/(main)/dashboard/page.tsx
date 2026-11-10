'use client';

import React, { useCallback, useState } from 'react';
import dynamic from 'next/dynamic';
import {
  Card,
  Button,
  Switch,
  Tooltip,
  App,
  Space,
  Badge,
  Dropdown,
  Tabs,
  Divider,
  Drawer,
  Form,
  Select as AntSelect,
  Row,
  Col,
  Skeleton,
  Select,
  Radio,
  ColorPicker,
  Checkbox,
} from 'antd';
import { useRouter } from 'next/navigation';
import {
  RefreshCw,
  Settings,
  LayoutDashboard,
  Zap,
  LineChart,
  TrendingUp,
  Ticket,
  BookOpen,
  Database,
  Compass,
} from 'lucide-react';
import { KPICards } from './components/KPICards';
import { ChartsSection } from './components/ChartsSection';
import { QuickActions } from './components/QuickActions';
import { useDashboardData } from './hooks/useDashboardData';
import type { QuickAction } from './types/dashboard.types';
import { useTheme } from '@/lib/design-system/theme';
import { userPreferences } from '@/lib/user-preferences';
import { formatDateTime } from '@/lib/formatters';

type DashboardUserPreferences = ReturnType<typeof userPreferences.get>;

// 动态导入图表组件 - 按需加载，减少初始 bundle 大小
const TicketTrendChart = dynamic(() => import('./components/TicketTrendChart'), {
  ssr: false,
  loading: () => <Skeleton active />,
});

const IncidentDistributionChart = dynamic(() => import('./components/IncidentDistributionChart'), {
  ssr: false,
  loading: () => <Skeleton active />,
});

const SLAComplianceChart = dynamic(() => import('./components/SLAComplianceChart'), {
  ssr: false,
  loading: () => <Skeleton active />,
});

const UserSatisfactionChart = dynamic(() => import('./components/UserSatisfactionChart'), {
  ssr: false,
  loading: () => <Skeleton active />,
});

const ResponseTimeChart = dynamic(() => import('./components/ResponseTimeChart'), {
  ssr: false,
  loading: () => <Skeleton active />,
});

const TeamWorkloadChart = dynamic(() => import('./components/TeamWorkloadChart'), {
  ssr: false,
  loading: () => <Skeleton active />,
});

const PeakHoursChart = dynamic(() => import('./components/PeakHoursChart'), {
  ssr: false,
  loading: () => <Skeleton active />,
});

export default function DashboardPage() {
  const router = useRouter();
  const { message } = App.useApp();
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

  const [activeChartTab, setActiveChartTab] = useState('tickets');
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [prefs, setPrefs] = useState<DashboardUserPreferences | null>(null);
  const { mode: themeMode, setMode: setThemeMode } = useTheme();

  // 后端在工单类 KPI 上下发行级范围；任一指标为 own 即整组为个人范围。
  const ticketKpiScope = React.useMemo(() => {
    const scopes = (data?.kpiMetrics || [])
      .map(metric => metric.scope)
      .filter((scope): scope is 'tenant' | 'own' => Boolean(scope));
    if (scopes.length === 0) return null;
    return scopes.some(scope => scope === 'own') ? 'own' : 'tenant';
  }, [data?.kpiMetrics]);

  // 同步用户偏好
  React.useEffect(() => {
    setPrefs(userPreferences.get());
    const unsub = userPreferences.subscribe(p => setPrefs(p));
    return () => unsub();
  }, []);

  const updatePref = (patch: Partial<DashboardUserPreferences>) => {
    userPreferences.update(patch);
    setPrefs(userPreferences.get());
  };

  // 处理快速操作点击
  const handleQuickActionClick = useCallback(
    (action: QuickAction) => {
      if (action.path) {
        try {
          router.push(action.path);
        } catch (error) {
          console.error('Navigation error:', error);
          message.error('Navigation failed, please try again later');
        }
      } else {
        console.warn('Quick action missing path:', action);
      }
    },
    [router, message]
  );

  // 处理刷新间隔变化
  const handleRefreshIntervalChange = useCallback(
    (interval: number) => {
      setRefreshInterval(interval);
      message.success(`The refresh interval has been updated to ${interval / 1000} seconds`);
    },
    [setRefreshInterval]
  );

  // 处理自动刷新切换
  const handleAutoRefreshToggle = useCallback(
    (enabled: boolean) => {
      setAutoRefresh(enabled);
      message.success(`Auto refresh failed ${enabled ? 'Enabled' : 'Disabled'}`);
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
          <span className='text-sm font-medium'>Auto refresh</span>
          <Switch aria-label="Auto refresh" checked={autoRefresh} onChange={handleAutoRefreshToggle} size='small' />
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
          <span className='text-sm font-medium'>Refresh interval</span>
          <Select
            value={refreshInterval}
            onChange={value => handleRefreshIntervalChange(Number(value))}
            className='w-[100px]'
            size='small'
            disabled={!autoRefresh}
            onClick={e => e.stopPropagation()}
            options={[
              { value: 10000, label: '10 seconds' },
              { value: 30000, label: '30 seconds' },
              { value: 60000, label: '1 minute' },
              { value: 300000, label: '5 minutes' },
            ]}
          />
        </div>
      ),
    },
    { type: 'divider' as const },
    {
      key: 'connection',
      label: (
        <div className='flex items-center justify-between min-w-[200px]'>
          <span className='text-sm font-medium'>Connection status</span>
          <Badge
            status={isConnected ? 'success' : 'error'}
            text={isConnected ? 'Connected' : 'Disconnected'}
          />
        </div>
      ),
    },
    {
      key: 'last-updated',
      label: (
        <div className='flex items-center justify-between min-w-[200px]'>
          <span className='text-sm font-medium'>Last Update</span>
          <span className='text-xs text-gray-500 font-medium'>
            {formatDateTime(lastUpdated) || 'Never'}
          </span>
        </div>
      ),
    },
  ];

  // 错误状态
  if (error) {
    return (
      <Card className='text-center py-16 rounded-xl border-0 shadow-sm'>
        <div className='text-red-500 mb-4 flex justify-center'>
          <LayoutDashboard size={64} />
        </div>
        <h3 className='text-xl font-bold text-gray-900 mb-2'>Dashboard loading failed</h3>
        <p className='text-gray-600 mb-6'>{error}</p>
        <Button
          type='primary'
          size='large'
          onClick={() => refresh()}
          icon={<RefreshCw />}
          className='h-11 rounded-lg'
        >

          Reload
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
    <div
      className='p-6 min-h-screen'
      style={{ backgroundColor: 'var(--color-bg-secondary, #f9fafb)' }}
    >
      {/* 简化顶部工具栏 */}
      <div className='flex items-center justify-between mb-6 pb-4 border-b border-gray-200'>
        <div>
          <h1 className='text-2xl font-bold text-gray-900 flex items-center gap-2'>

            AI-Native ITSM Operations Dashboard
            {isConnected && <Badge status='success' text="Online" />}
          </h1>
          <p className='text-sm text-gray-600 mt-1'>Real-time monitoring of system operating status and key business indicators</p>
        </div>

        <Space size='middle'>
          {lastUpdated && (
            <span className='text-sm text-gray-500'>

              updated on {formatDateTime(lastUpdated)}
            </span>
          )}

          <Button
            type='default'
            icon={<RefreshCw className={loading ? 'animate-spin' : ''} />}
            onClick={() => refresh()}
            loading={loading}
          >

            Refresh
          </Button>

          <Button
            type='default'
            icon={<Settings />}
            onClick={() => setSettingsOpen(true)}
          >

            Settings
          </Button>
        </Space>
      </div>

      {loading ? (
        renderSkeletons()
      ) : (
        <>
          {/* KPI指标卡片区域 */}
          <div className='mb-6'>
            {/* 行级范围由后端按登录者角色下发；坐席与管理员数字口径不同，不说明会被当成 bug */}
            <div className='mb-2 flex flex-wrap items-center gap-x-2 text-xs text-gray-500'>
              {ticketKpiScope === 'own' ? (
                <span className='px-2 py-0.5 rounded-full bg-blue-50 text-blue-700 font-medium'>

                  Work order indicator: Only I can create or accept it
                </span>
              ) : (
                ticketKpiScope === 'tenant' && (
                  <span className='px-2 py-0.5 rounded-full bg-gray-100 text-gray-600 font-medium'>

                    Work order indicator: all work orders of this tenant
                  </span>
                )
              )}
              <span>Trend, load and distribution charts provide full statistics for tenants</span>
            </div>
            <KPICards metrics={data?.kpiMetrics || []} loading={loading} />
          </div>

          <Divider className='my-8' />

          {/* 快速操作区域 */}
          <div className='mb-6'>
            <div className='mb-4'>
              <h2 className='text-lg font-bold text-gray-900 mb-1 flex items-center gap-2'>
                <Zap className='text-orange-500 w-5 h-5' />

                Quick Actions
              </h2>
              <p className='text-sm text-gray-600'>Quick access to commonly used functions to improve work efficiency</p>
            </div>
            {(!data?.quickActions || data.quickActions.length === 0) && !loading ? (
              <Row gutter={[16, 16]}>
                <Col xs={24} sm={12} lg={6}>
                  <Card className='rounded-xl border border-gray-100 shadow-sm hover:shadow-md transition-all h-full'>
                    <div className='flex flex-col items-center text-center py-4'>
                      <div className='w-12 h-12 bg-blue-50 rounded-lg flex items-center justify-center text-blue-600 mb-3'>
                        <Ticket className='w-6 h-6' />
                      </div>
                      <h3 className='text-base font-semibold text-gray-900 mb-1'>Create a work order</h3>
                      <p className='text-sm text-gray-500 mb-4'>Submit an IT service request or report an issue</p>
                      <Button type='primary' onClick={() => router.push('/tickets/create')}>

                        Go create
                      </Button>
                    </div>
                  </Card>
                </Col>
                <Col xs={24} sm={12} lg={6}>
                  <Card className='rounded-xl border border-gray-100 shadow-sm hover:shadow-md transition-all h-full'>
                    <div className='flex flex-col items-center text-center py-4'>
                      <div className='w-12 h-12 bg-green-50 rounded-lg flex items-center justify-center text-green-600 mb-3'>
                        <Compass className='w-6 h-6' />
                      </div>
                      <h3 className='text-base font-semibold text-gray-900 mb-1'>Browse service catalog</h3>
                      <p className='text-sm text-gray-500 mb-4'>View available IT services catalog</p>
                      <Button type='primary' onClick={() => router.push('/service-catalog')}>

                        Go to browse
                      </Button>
                    </div>
                  </Card>
                </Col>
                <Col xs={24} sm={12} lg={6}>
                  <Card className='rounded-xl border border-gray-100 shadow-sm hover:shadow-md transition-all h-full'>
                    <div className='flex flex-col items-center text-center py-4'>
                      <div className='w-12 h-12 bg-purple-50 rounded-lg flex items-center justify-center text-purple-600 mb-3'>
                        <Database className='w-6 h-6' />
                      </div>
                      <h3 className='text-base font-semibold text-gray-900 mb-1'>Configure CMDB</h3>
                      <p className='text-sm text-gray-500 mb-4'>Managing Configuration Items and Asset Relationship Topology</p>
                      <Button type='primary' onClick={() => router.push('/cmdb')}>

                        Go to configuration
                      </Button>
                    </div>
                  </Card>
                </Col>
                <Col xs={24} sm={12} lg={6}>
                  <Card className='rounded-xl border border-gray-100 shadow-sm hover:shadow-md transition-all h-full'>
                    <div className='flex flex-col items-center text-center py-4'>
                      <div className='w-12 h-12 bg-amber-50 rounded-lg flex items-center justify-center text-amber-600 mb-3'>
                        <BookOpen className='w-6 h-6' />
                      </div>
                      <h3 className='text-base font-semibold text-gray-900 mb-1'>Check the knowledge base</h3>
                      <p className='text-sm text-gray-500 mb-4'>Search solutions and IT knowledge documents</p>
                      <Button type='primary' onClick={() => router.push('/knowledge')}>

                        Go check
                      </Button>
                    </div>
                  </Card>
                </Col>
              </Row>
            ) : (
              <QuickActions
                actions={data?.quickActions || []}
                loading={loading}
                onActionClick={handleQuickActionClick}
                showTitle={false}
                compact={false}
              />
            )}
          </div>

          <Divider className='my-8' />

          {/* 图表分析区域 */}
          <div className='mb-6'>
            <Card className='rounded-xl border-0 shadow-sm' styles={{ body: { padding: '24px' } }}>
              <div className='mb-5'>
                <h2 className='text-lg font-bold text-gray-900 mb-1 flex items-center gap-2'>
                  <LineChart className='text-blue-500 w-5 h-5' />

                  Data Analysis and Trends
                </h2>
                <p className='text-sm text-gray-600'>Visual analysis of system performance and business trends</p>
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

                        All charts
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
                            {(() => {
                              const slaMetric = data?.kpiMetrics?.find(
                                m => m.id === 'sla-compliance'
                              );
                              const slaOverall =
                                slaMetric?.value !== undefined
                                  ? Number(slaMetric.value)
                                  : undefined;
                              return (
                                <SLAComplianceChart
                                  data={data?.slaData || []}
                                  overallValue={slaOverall}
                                />
                              );
                            })()}
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

                        Tickets and Incidents
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

                        Performance and Satisfaction
                      </span>
                    ),
                    children: (
                      <div className='pt-4'>
                        <ChartsSection loading={loading}>
                          <Col xs={24} lg={12}>
                            {(() => {
                              const slaMetric = data?.kpiMetrics?.find(
                                m => m.id === 'sla-compliance'
                              );
                              const slaOverall =
                                slaMetric?.value !== undefined
                                  ? Number(slaMetric.value)
                                  : undefined;
                              return (
                                <SLAComplianceChart
                                  data={data?.slaData || []}
                                  overallValue={slaOverall}
                                />
                              );
                            })()}
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

      {/* 仪表盘设置抽屉 */}
      <Drawer
        title="Dashboard settings"
        placement='right'
        width={420}
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        destroyOnHidden
        extra={
          <Space>
            <Button
              onClick={() => {
                userPreferences.reset();
                setPrefs(userPreferences.get());
                message.success('Settings reset to default');
              }}
            >

              Reset
            </Button>
            <Button type='primary' onClick={() => setSettingsOpen(false)}>

              Complete
            </Button>
          </Space>
        }
      >
        {prefs ? (
          <Form layout='vertical'>
            <Form.Item label="Theme mode">
              <Radio.Group
                value={themeMode}
                onChange={e => {
                  setThemeMode(e.target.value);
                  updatePref({ theme: e.target.value });
                }}
              >
                <Radio.Button value='light'>light color</Radio.Button>
                <Radio.Button value='dark'>Dark</Radio.Button>
                <Radio.Button value='system'>Follow the system</Radio.Button>
              </Radio.Group>
            </Form.Item>

            <Form.Item label="Language">
              <Select
                value={prefs.language}
                onChange={v => updatePref({ language: v })}
                options={[
                  { value: 'en-US', label: 'English' },
                ]}
              />
            </Form.Item>

            <Form.Item label="Time zone">
              <Select
                value={prefs.timezone}
                onChange={v => updatePref({ timezone: v })}
                options={[
                  { value: 'Asia/Shanghai', label: '(UTC+8) Shanghai' },
                  { value: 'Asia/Tokyo', label: '(UTC+9) Tokyo' },
                  { value: 'UTC', label: '(UTC+0) Coordinated Universal Time' },
                  { value: 'America/New_York', label: '(UTC-5) New York' },
                ]}
              />
            </Form.Item>

            <Form.Item label="date format">
              <Select
                value={prefs.dateFormat}
                onChange={v => updatePref({ dateFormat: v })}
                options={[
                  { value: 'YYYY-MM-DD HH:mm:ss', label: '2026-08-02 14:30:00' },
                  { value: 'MM/DD/YYYY HH:mm', label: '08/02/2026 14:30' },
                  { value: 'DD/MM/YYYY HH:mm', label: '02/08/2026 14:30' },
                ]}
              />
            </Form.Item>

            <Form.Item label="Number of entries per table page">
              <Select
                value={prefs.pageSize}
                onChange={v => updatePref({ pageSize: Number(v) })}
                options={[10, 20, 50, 100].map(n => ({ value: n, label: `${n} Article/page` }))}
              />
            </Form.Item>

            <Divider>Data refresh</Divider>
            <Form.Item label="Auto refresh">
              <Switch
                checked={autoRefresh}
                onChange={checked => {
                  setAutoRefresh(checked);
                  message.success(`Auto refresh failed ${checked ? 'Enabled' : 'Close'}`);
                }}
              />
            </Form.Item>
            <Form.Item label="Refresh interval">
              <Select
                value={refreshInterval}
                disabled={!autoRefresh}
                onChange={v => {
                  setRefreshInterval(Number(v));
                  message.success(`The refresh interval has been updated to ${Number(v) / 1000} seconds`);
                }}
                options={[
                  { value: 10000, label: '10 seconds' },
                  { value: 30000, label: '30 seconds' },
                  { value: 60000, label: '1 minute' },
                  { value: 300000, label: '5 minutes' },
                ]}
              />
            </Form.Item>

            <Divider>Notification</Divider>
            <Form.Item label="Notification Channels">
              <Checkbox.Group
                value={Object.entries(prefs.notifications || {})
                  .filter(([k, v]) => k !== 'types' && v)
                  .map(([k]) => k)}
                onChange={vals => {
                  updatePref({
                    notifications: {
                      ...prefs.notifications,
                      email: (vals as string[]).includes('email'),
                      browser: (vals as string[]).includes('browser'),
                      mobile: (vals as string[]).includes('mobile'),
                    },
                  });
                }}
              >
                <Checkbox value='email'>Email</Checkbox>
                <Checkbox value='browser'>Browser</Checkbox>
                <Checkbox value='mobile'>Mobile version</Checkbox>
              </Checkbox.Group>
            </Form.Item>
          </Form>
        ) : (
          <Skeleton active />
        )}
      </Drawer>
    </div>
  );
}
