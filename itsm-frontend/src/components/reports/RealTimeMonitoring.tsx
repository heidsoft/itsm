'use client';

import React, { useState, useEffect, useRef } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Progress,
  Table,
  Tag,
  Alert,
  Badge,
  Space,
  Typography,
  Switch,
  Tooltip,
} from 'antd';
import {
  Activity,
  TrendingUp,
  AlertTriangle,
  CheckCircle,
  Clock,
  Users,
  Server,
  Database,
  Globe,
  Zap,
  RefreshCw,
  Bell,
} from 'lucide-react';

const { Title, Text } = Typography;

interface RealTimeMonitoringProps {
  autoRefresh?: boolean;
  refreshInterval?: number;
}

interface SystemMetric {
  name: string;
  value: number;
  unit: string;
  status: 'normal' | 'warning' | 'critical';
  trend?: 'up' | 'down' | 'stable';
  icon: React.ReactNode;
}

interface ActiveAlert {
  id: string;
  type: 'performance' | 'availability' | 'capacity' | 'security';
  title: string;
  description: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  timestamp: Date;
  acknowledged: boolean;
}

interface RecentActivity {
  id: string;
  type: 'ticket_created' | 'ticket_resolved' | 'user_login' | 'system_event';
  description: string;
  user?: string;
  timestamp: Date;
  details?: any;
}

const RealTimeMonitoring: React.FC<RealTimeMonitoringProps> = ({
  autoRefresh = true,
  refreshInterval = 30,
}) => {
  const [systemMetrics, setSystemMetrics] = useState<SystemMetric[]>([]);
  const [activeAlerts, setActiveAlerts] = useState<ActiveAlert[]>([]);
  const [recentActivities, setRecentActivities] = useState<RecentActivity[]>([]);
  const [lastRefresh, setLastRefresh] = useState(new Date());
  const [refreshEnabled, setRefreshEnabled] = useState(autoRefresh);
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  // 系统指标数据
  useEffect(() => {
    const mockMetrics: SystemMetric[] = [
      {
        name: 'CPU使用率',
        value: 68,
        unit: '%',
        status: 'normal',
        trend: 'up',
        icon: <Server className="w-5 h-5 text-blue-500" />,
      },
      {
        name: '内存使用率',
        value: 72,
        unit: '%',
        status: 'warning',
        trend: 'stable',
        icon: <Database className="w-5 h-5 text-green-500" />,
      },
      {
        name: '响应时间',
        value: 245,
        unit: 'ms',
        status: 'normal',
        trend: 'down',
        icon: <Clock className="w-5 h-5 text-orange-500" />,
      },
      {
        name: '在线用户',
        value: 1247,
        unit: '人',
        status: 'normal',
        trend: 'up',
        icon: <Users className="w-5 h-5 text-purple-500" />,
      },
      {
        name: 'API调用',
        value: 15847,
        unit: '次/分',
        status: 'normal',
        trend: 'up',
        icon: <Activity className="w-5 h-5 text-cyan-500" />,
      },
      {
        name: '网络流量',
        value: 892,
        unit: 'Mbps',
        status: 'warning',
        trend: 'stable',
        icon: <Globe className="w-5 h-5 text-teal-500" />,
      },
    ];

    setSystemMetrics(mockMetrics);
  }, []);

  // 活跃告警数据
  useEffect(() => {
    const mockAlerts: ActiveAlert[] = [
      {
        id: 'alert1',
        type: 'performance',
        title: '响应时间异常',
        description: 'API平均响应时间超过300ms阈值',
        severity: 'medium',
        timestamp: new Date(Date.now() - 300000),
        acknowledged: false,
      },
      {
        id: 'alert2',
        type: 'capacity',
        title: '内存使用率过高',
        description: '服务器内存使用率达到72%，接近预警阈值',
        severity: 'low',
        timestamp: new Date(Date.now() - 600000),
        acknowledged: true,
      },
      {
        id: 'alert3',
        type: 'availability',
        title: '数据库连接异常',
        description: '数据库连接池使用率过高',
        severity: 'high',
        timestamp: new Date(Date.now() - 900000),
        acknowledged: false,
      },
    ];

    setActiveAlerts(mockAlerts);
  }, []);

  // 最近活动数据
  useEffect(() => {
    const mockActivities: RecentActivity[] = [
      {
        id: 'activity1',
        type: 'ticket_created',
        description: '新工单创建',
        user: '张三',
        timestamp: new Date(Date.now() - 120000),
        details: { ticketId: 'TICK-2024-001', title: '系统登录异常' },
      },
      {
        id: 'activity2',
        type: 'ticket_resolved',
        description: '工单已解决',
        user: '李四',
        timestamp: new Date(Date.now() - 300000),
        details: { ticketId: 'TICK-2024-002', resolutionTime: '2.5h' },
      },
      {
        id: 'activity3',
        type: 'system_event',
        description: '系统备份完成',
        timestamp: new Date(Date.now() - 1800000),
        details: { backupSize: '2.3GB', duration: '45min' },
      },
      {
        id: 'activity4',
        type: 'user_login',
        description: '用户登录',
        user: '王五',
        timestamp: new Date(Date.now() - 600000),
        details: { ip: '192.168.1.100', location: '北京' },
      },
    ];

    setRecentActivities(mockActivities);
  }, []);

  // 自动刷新逻辑
  useEffect(() => {
    if (refreshEnabled && refreshInterval > 0) {
      intervalRef.current = setInterval(() => {
        refreshData();
      }, refreshInterval * 1000);
    }

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [refreshEnabled, refreshInterval]);

  const refreshData = () => {
    // 模拟数据刷新
    setLastRefresh(new Date());
    console.log('Real-time data refreshed');
  };

  // 手动刷新
  const handleManualRefresh = () => {
    refreshData();
  };

  // 确认告警
  const acknowledgeAlert = (alertId: string) => {
    setActiveAlerts(prev =>
      prev.map(alert =>
        alert.id === alertId ? { ...alert, acknowledged: true } : alert
      )
    );
  };

  // 获取状态颜色
  const getStatusColor = (status: string) => {
    switch (status) {
      case 'normal':
        return '#52c41a';
      case 'warning':
        return '#faad14';
      case 'critical':
        return '#ff4d4f';
      default:
        return '#d9d9d9';
    }
  };

  // 获取趋势图标
  const getTrendIcon = (trend?: string) => {
    switch (trend) {
      case 'up':
        return <TrendingUp className="w-4 h-4 text-green-500" />;
      case 'down':
        return <TrendingUp className="w-4 h-4 text-red-500 rotate-180" />;
      default:
        return <div className="w-4 h-4 bg-gray-400 rounded-full" />;
    }
  };

  // 获取告警严重程度颜色
  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'red';
      case 'high':
        return 'orange';
      case 'medium':
        return 'yellow';
      case 'low':
        return 'blue';
      default:
        return 'default';
    }
  };

  // 活动类型图标
  const getActivityIcon = (type: string) => {
    switch (type) {
      case 'ticket_created':
        return <Bell className="w-4 h-4 text-blue-500" />;
      case 'ticket_resolved':
        return <CheckCircle className="w-4 h-4 text-green-500" />;
      case 'user_login':
        return <Users className="w-4 h-4 text-purple-500" />;
      default:
        return <Activity className="w-4 h-4 text-gray-500" />;
    }
  };

  // 告警表格列
  const alertColumns = [
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color={type === 'performance' ? 'orange' : type === 'availability' ? 'red' : 'blue'}>
          {type === 'performance' ? '性能' : type === 'availability' ? '可用性' : '其他'}
        </Tag>
      ),
    },
    {
      title: '告警信息',
      dataIndex: 'title',
      key: 'title',
      render: (title: string, record: ActiveAlert) => (
        <div>
          <div className="font-medium">{title}</div>
          <Text type="secondary" className="text-xs">{record.description}</Text>
        </div>
      ),
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 80,
      render: (severity: string) => (
        <Tag color={getSeverityColor(severity)}>
          {severity === 'critical' ? '严重' :
           severity === 'high' ? '高' :
           severity === 'medium' ? '中' : '低'}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'acknowledged',
      key: 'acknowledged',
      width: 80,
      render: (acknowledged: boolean) => (
        <Badge
          status={acknowledged ? 'success' : 'error'}
          text={acknowledged ? '已确认' : '待确认'}
        />
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* 控制面板 */}
      <Card title="实时监控控制台" extra={
        <Space>
          <Text type="secondary">
            最后更新: {lastRefresh.toLocaleTimeString()}
          </Text>
          <Tooltip title={refreshEnabled ? '关闭自动刷新' : '开启自动刷新'}>
            <Switch
              checked={refreshEnabled}
              onChange={setRefreshEnabled}
              checkedChildren="自动"
              unCheckedChildren="手动"
            />
          </Tooltip>
          <Button
            icon={<RefreshCw className="w-4 h-4" />}
            onClick={handleManualRefresh}
          >
            刷新
          </Button>
        </Space>
      }>
        <div className="flex items-center space-x-4 text-sm text-gray-600">
          <Activity className="w-4 h-4" />
          <span>实时监控已启用</span>
          <div className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
          <span>刷新间隔: {refreshInterval}秒</span>
        </div>
      </Card>

      {/* 系统指标 */}
      <Row gutter={[16, 16]}>
        {systemMetrics.map((metric, index) => (
          <Col xs={24} sm={12} md={8} lg={6} key={index}>
            <Card size="small" className="system-metric-card">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center space-x-2">
                  {metric.icon}
                  <Text type="secondary">{metric.name}</Text>
                </div>
                <div className="flex items-center space-x-1">
                  {getTrendIcon(metric.trend)}
                </div>
              </div>
              <div className="flex items-baseline space-x-2">
                <Statistic
                  value={metric.value}
                  suffix={metric.unit}
                  valueStyle={{
                    color: getStatusColor(metric.status),
                    fontSize: '20px',
                  }}
                />
              </div>
              <Progress
                percent={metric.unit === '%' ? metric.value : Math.min((metric.value / 100) * 100, 100)}
                strokeColor={getStatusColor(metric.status)}
                showInfo={false}
                size="small"
                className="mt-2"
              />
            </Card>
          </Col>
        ))}
      </Row>

      {/* 告警和活动 */}
      <Row gutter={[16, 16]}>
        {/* 活跃告警 */}
        <Col xs={24} lg={14}>
          <Card
            title={
              <Space>
                <AlertTriangle className="w-5 h-5 text-orange-600" />
                <span>活跃告警</span>
                <Badge count={activeAlerts.filter(a => !a.acknowledged).length} />
              </Space>
            }
            extra={
              <Text type="secondary">
                总计: {activeAlerts.length}个
              </Text>
            }
          >
            <Table
              columns={alertColumns}
              dataSource={activeAlerts}
              rowKey="id"
              pagination={false}
              size="small"
              onRow={(record) => ({
                onClick: () => !record.acknowledged && acknowledgeAlert(record.id),
                style: { cursor: !record.acknowledged ? 'pointer' : 'default' },
              })}
            />
          </Card>
        </Col>

        {/* 最近活动 */}
        <Col xs={24} lg={10}>
          <Card
            title={
              <Space>
                <Zap className="w-5 h-5 text-blue-600" />
                <span>最近活动</span>
              </Space>
            }
            >
            <div className="space-y-3" style={{ maxHeight: '400px', overflowY: 'auto' }}>
              {recentActivities.map((activity) => (
                <div key={activity.id} className="flex items-start space-x-3 pb-3 border-b border-gray-100 last:border-0">
                  <div className="mt-1">
                    {getActivityIcon(activity.type)}
                  </div>
                  <div className="flex-1">
                    <div className="flex items-center justify-between">
                      <Text className="font-medium">{activity.description}</Text>
                      <Text type="secondary" className="text-xs">
                        {activity.timestamp.toLocaleTimeString()}
                      </Text>
                    </div>
                    {activity.user && (
                      <Text type="secondary" className="text-xs">
                        操作者: {activity.user}
                      </Text>
                    )}
                    {activity.details && (
                      <div className="text-xs text-gray-500 mt-1">
                        {Object.entries(activity.details).map(([key, value]) => (
                          <div key={key}>
                            {key}: {JSON.stringify(value)}
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>

      {/* 系统状态总览 */}
      <Alert
        message="系统状态总览"
        description={
          <div className="space-y-1">
            <div>• 3个警告需要关注，1个严重告警待处理</div>
            <div>• 系统整体运行正常，各项指标在可接受范围内</div>
            <div>• 建议关注内存使用率趋势，必要时进行扩容</div>
          </div>
        }
        type="info"
        showIcon
        className="status-overview"
      />
    </div>
  );
};

export default RealTimeMonitoring;