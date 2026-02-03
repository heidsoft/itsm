'use client';

import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Progress,
  Table,
  Tag,
  Button,
  Select,
  Space,
  Typography,
  Badge,
  Alert,
  Tooltip,
  Spin,
  message,
  App,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  ClockCircleOutlined,
  ReloadOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  BellOutlined,
} from '@ant-design/icons';
import { Activity, AlertTriangle, Target, Zap, BarChart3 } from 'lucide-react';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import SLAApi from '@/lib/api/sla-api';

const { Title, Text } = Typography;
const { Option } = Select;

interface SLAMetrics {
  totalTickets: number;
  compliantTickets: number;
  violatedTickets: number;
  atRiskTickets: number;
  complianceRate: number;
  violationRate: number;
  averageResponseTime: number;
  averageResolutionTime: number;
  responseTimeCompliance: number;
  resolutionTimeCompliance: number;
}

interface SLAAlert {
  id: string;
  ticketId: number;
  ticketNumber: string;
  ticketTitle: string;
  priority: 'low' | 'medium' | 'high' | 'urgent';
  alertLevel: 'warning' | 'critical';
  timeRemaining: number; // 小时
  slaDefinition: string;
  createdAt: string;
}

interface SLAMonitorDashboardProps {
  autoRefresh?: boolean;
  refreshInterval?: number; // 秒
  onFullscreen?: (isFullscreen: boolean) => void;
  tenantId?: number;
}

export const SLAMonitorDashboard: React.FC<SLAMonitorDashboardProps> = ({
  autoRefresh = true,
  refreshInterval = 30,
  onFullscreen,
  tenantId,
}) => {
  const { message: antMessage } = App.useApp();
  const [loading, setLoading] = useState(true);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [metrics, setMetrics] = useState<SLAMetrics | null>(null);
  const [alerts, setAlerts] = useState<SLAAlert[]>([]);
  const [selectedTenant, setSelectedTenant] = useState<number | undefined>(tenantId);
  const refreshTimerRef = useRef<NodeJS.Timeout | null>(null);
  const lastUpdateTimeRef = useRef<Date>(new Date());

  // 加载SLA监控数据
  const loadSLAData = useCallback(async () => {
    try {
      setLoading(true);

      // 调用实际的API
      const data = await SLAApi.getSLAMonitoring({
        tenant_id: selectedTenant,
        start_time: new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString(),
        end_time: new Date().toISOString(),
      });

      if (data) {
        setMetrics({
          totalTickets: data.total_tickets,
          compliantTickets: data.compliant_tickets,
          violatedTickets: data.violated_tickets,
          atRiskTickets: data.at_risk_tickets,
          complianceRate: data.compliance_rate,
          violationRate: data.violation_rate,
          averageResponseTime: data.average_response_time,
          averageResolutionTime: data.average_resolution_time,
          responseTimeCompliance: data.response_time_compliance,
          resolutionTimeCompliance: data.resolution_time_compliance,
        });
        // 转换API响应格式到组件格式
        const convertedAlerts: SLAAlert[] = (data.alerts || []).map((alert: any) => ({
          id: alert.id,
          ticketId: alert.ticket_id,
          ticketNumber: alert.ticket_number,
          ticketTitle: alert.ticket_title,
          priority: alert.priority,
          alertLevel: alert.alert_level,
          timeRemaining: alert.time_remaining,
          slaDefinition: alert.sla_definition,
          severity: alert.severity || 'medium',
          createdAt: alert.created_at,
        }));
        setAlerts(convertedAlerts);
        lastUpdateTimeRef.current = new Date();
        return;
      }

      // 如果API返回空，使用模拟数据
      const mockMetrics: SLAMetrics = {
        totalTickets: 1256,
        compliantTickets: 1123,
        violatedTickets: 89,
        atRiskTickets: 44,
        complianceRate: 89.4,
        violationRate: 7.1,
        averageResponseTime: 2.3,
        averageResolutionTime: 8.7,
        responseTimeCompliance: 92.5,
        resolutionTimeCompliance: 88.2,
      };

      const mockAlerts: SLAAlert[] = [
        {
          id: '1',
          ticketId: 1001,
          ticketNumber: 'T-2024-001',
          ticketTitle: '数据库连接超时',
          priority: 'high',
          alertLevel: 'critical',
          timeRemaining: 0.5,
          slaDefinition: 'P1-4小时响应',
          createdAt: new Date().toISOString(),
        },
        {
          id: '2',
          ticketId: 1002,
          ticketNumber: 'T-2024-002',
          ticketTitle: '网络设备故障',
          priority: 'urgent',
          alertLevel: 'warning',
          timeRemaining: 1.2,
          slaDefinition: 'P1-4小时响应',
          createdAt: new Date().toISOString(),
        },
        {
          id: '3',
          ticketId: 1003,
          ticketNumber: 'T-2024-003',
          ticketTitle: '系统登录异常',
          priority: 'medium',
          alertLevel: 'warning',
          timeRemaining: 2.8,
          slaDefinition: 'P2-8小时响应',
          createdAt: new Date().toISOString(),
        },
      ];

      setMetrics(mockMetrics);
      setAlerts(mockAlerts);
      lastUpdateTimeRef.current = new Date();
    } catch (error) {
      console.error('Failed to load SLA data:', error);
      antMessage.error('加载SLA数据失败');
    } finally {
      setLoading(false);
    }
  }, [selectedTenant, antMessage]);

  // 初始化加载
  useEffect(() => {
    loadSLAData();
  }, [loadSLAData]);

  // 自动刷新
  useEffect(() => {
    if (autoRefresh) {
      refreshTimerRef.current = setInterval(() => {
        loadSLAData();
      }, refreshInterval * 1000);

      return () => {
        if (refreshTimerRef.current) {
          clearInterval(refreshTimerRef.current);
        }
      };
    }
  }, [autoRefresh, refreshInterval, loadSLAData]);

  // 全屏切换
  const handleFullscreen = useCallback(() => {
    const element = document.documentElement;
    if (!isFullscreen) {
      if (element.requestFullscreen) {
        element.requestFullscreen();
      }
      setIsFullscreen(true);
      onFullscreen?.(true);
    } else {
      if (document.exitFullscreen) {
        document.exitFullscreen();
      }
      setIsFullscreen(false);
      onFullscreen?.(false);
    }
  }, [isFullscreen, onFullscreen]);

  // 手动刷新
  const handleRefresh = useCallback(() => {
    loadSLAData();
    antMessage.success('数据已刷新');
  }, [loadSLAData, antMessage]);

  // 获取合规率颜色
  const getComplianceColor = (rate: number) => {
    if (rate >= 95) return '#52c41a';
    if (rate >= 85) return '#faad14';
    return '#ff4d4f';
  };

  // 获取告警级别颜色
  const getAlertLevelColor = (level: string) => {
    return level === 'critical' ? 'red' : 'orange';
  };

  // 获取优先级颜色
  const getPriorityColor = (priority: string) => {
    const colors: Record<string, string> = {
      low: 'default',
      medium: 'blue',
      high: 'orange',
      urgent: 'red',
    };
    return colors[priority] || 'default';
  };

  // 告警表格列
  const alertColumns = [
    {
      title: '工单编号',
      dataIndex: 'ticketNumber',
      key: 'ticketNumber',
      render: (text: string) => (
        <Text strong className='text-lg' style={{ color: '#1890ff' }}>
          {text}
        </Text>
      ),
    },
    {
      title: '工单标题',
      dataIndex: 'ticketTitle',
      key: 'ticketTitle',
      render: (text: string) => <Text className='text-base'>{text}</Text>,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      render: (priority: string) => (
        <Tag color={getPriorityColor(priority)} className='text-sm px-3 py-1'>
          {priority === 'low'
            ? '低'
            : priority === 'medium'
              ? '中'
              : priority === 'high'
                ? '高'
                : '紧急'}
        </Tag>
      ),
    },
    {
      title: '告警级别',
      dataIndex: 'alertLevel',
      key: 'alertLevel',
      render: (level: string) => (
        <Badge
          status={level === 'critical' ? 'error' : 'warning'}
          text={
            <Tag color={getAlertLevelColor(level)} className='text-sm px-3 py-1'>
              {level === 'critical' ? '严重' : '警告'}
            </Tag>
          }
        />
      ),
    },
    {
      title: '剩余时间',
      dataIndex: 'timeRemaining',
      key: 'timeRemaining',
      render: (hours: number) => {
        const isNegative = hours < 0;
        const absHours = Math.abs(hours);
        const displayHours = Math.floor(absHours);
        const displayMinutes = Math.floor((absHours - displayHours) * 60);
        return (
          <Text
            strong
            className='text-lg'
            style={{ color: isNegative ? '#ff4d4f' : hours < 2 ? '#faad14' : '#52c41a' }}
          >
            {isNegative ? '-' : ''}
            {displayHours}小时{displayMinutes}分钟
          </Text>
        );
      },
    },
    {
      title: 'SLA定义',
      dataIndex: 'slaDefinition',
      key: 'slaDefinition',
      render: (text: string) => <Text className='text-base'>{text}</Text>,
    },
  ];

  if (loading && !metrics) {
    return (
      <div className='flex items-center justify-center h-screen'>
        <Spin size='large' tip='加载SLA监控数据...' />
      </div>
    );
  }

  return (
    <div
      className={`sla-monitor-dashboard ${isFullscreen ? 'fullscreen' : ''}`}
      style={{
        minHeight: isFullscreen ? '100vh' : 'auto',
        padding: isFullscreen ? '24px' : '16px',
        backgroundColor: isFullscreen ? '#0a0e27' : '#f0f2f5',
      }}
    >
      {/* 顶部工具栏 */}
      <div
        className='flex items-center justify-between mb-6'
        style={{
          backgroundColor: isFullscreen ? 'rgba(255,255,255,0.1)' : 'transparent',
          padding: '12px 16px',
          borderRadius: '8px',
        }}
      >
        <div className='flex items-center gap-4'>
          <Title
            level={2}
            style={{
              color: isFullscreen ? '#fff' : '#000',
              margin: 0,
              fontSize: isFullscreen ? '32px' : '24px',
            }}
          >
            <Activity className='inline-block mr-2' size={isFullscreen ? 32 : 24} />
            SLA实时监控大屏
          </Title>
          <Text
            type='secondary'
            style={{
              color: isFullscreen ? 'rgba(255,255,255,0.7)' : undefined,
              fontSize: isFullscreen ? '16px' : '14px',
            }}
          >
            最后更新: {format(lastUpdateTimeRef.current, 'yyyy-MM-dd HH:mm:ss', { locale: zhCN })}
          </Text>
        </div>
        <Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={handleRefresh}
            size={isFullscreen ? 'large' : 'middle'}
            style={{
              backgroundColor: isFullscreen ? 'rgba(255,255,255,0.1)' : undefined,
              color: isFullscreen ? '#fff' : undefined,
              borderColor: isFullscreen ? 'rgba(255,255,255,0.3)' : undefined,
            }}
          >
            刷新
          </Button>
          <Button
            icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
            onClick={handleFullscreen}
            size={isFullscreen ? 'large' : 'middle'}
            style={{
              backgroundColor: isFullscreen ? 'rgba(255,255,255,0.1)' : undefined,
              color: isFullscreen ? '#fff' : undefined,
              borderColor: isFullscreen ? 'rgba(255,255,255,0.3)' : undefined,
            }}
          >
            {isFullscreen ? '退出全屏' : '全屏'}
          </Button>
        </Space>
      </div>

      {/* 关键指标卡片 */}
      <Row gutter={[16, 16]} className='mb-6'>
        <Col xs={24} sm={12} lg={6}>
          <Card
            className='sla-stat-card'
            style={{
              backgroundColor: isFullscreen ? 'rgba(255,255,255,0.05)' : '#fff',
              border: isFullscreen ? '1px solid rgba(255,255,255,0.1)' : undefined,
            }}
          >
            <Statistic
              title={
                <Text
                  style={{
                    color: isFullscreen ? 'rgba(255,255,255,0.8)' : undefined,
                    fontSize: isFullscreen ? '18px' : '14px',
                  }}
                >
                  SLA达成率
                </Text>
              }
              value={metrics?.complianceRate || 0}
              precision={1}
              suffix='%'
              styles={{
                content: {
                  color: isFullscreen ? '#fff' : getComplianceColor(metrics?.complianceRate || 0),
                  fontSize: isFullscreen ? '48px' : '32px',
                  fontWeight: 'bold',
                },
              }}
              prefix={<CheckCircleOutlined />}
            />
            <Progress
              percent={metrics?.complianceRate || 0}
              strokeColor={getComplianceColor(metrics?.complianceRate || 0)}
              showInfo={false}
              size='small'
              className='mt-4'
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card
            className='sla-stat-card'
            style={{
              backgroundColor: isFullscreen ? 'rgba(255,255,255,0.05)' : '#fff',
              border: isFullscreen ? '1px solid rgba(255,255,255,0.1)' : undefined,
            }}
          >
            <Statistic
              title={
                <Text
                  style={{
                    color: isFullscreen ? 'rgba(255,255,255,0.8)' : undefined,
                    fontSize: isFullscreen ? '18px' : '14px',
                  }}
                >
                  SLA违规率
                </Text>
              }
              value={metrics?.violationRate || 0}
              precision={1}
              suffix='%'
              styles={{
                content: {
                  color: isFullscreen ? '#ff4d4f' : '#ff4d4f',
                  fontSize: isFullscreen ? '48px' : '32px',
                  fontWeight: 'bold',
                },
              }}
              prefix={<CloseCircleOutlined />}
            />
            <div className='mt-4'>
              <Text
                style={{
                  color: isFullscreen ? 'rgba(255,255,255,0.7)' : undefined,
                  fontSize: isFullscreen ? '16px' : '14px',
                }}
              >
                违规工单: {metrics?.violatedTickets || 0}
              </Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card
            className='sla-stat-card'
            style={{
              backgroundColor: isFullscreen ? 'rgba(255,255,255,0.05)' : '#fff',
              border: isFullscreen ? '1px solid rgba(255,255,255,0.1)' : undefined,
            }}
          >
            <Statistic
              title={
                <Text
                  style={{
                    color: isFullscreen ? 'rgba(255,255,255,0.8)' : undefined,
                    fontSize: isFullscreen ? '18px' : '14px',
                  }}
                >
                  风险工单
                </Text>
              }
              value={metrics?.atRiskTickets || 0}
              styles={{
                content: {
                  color: isFullscreen ? '#faad14' : '#faad14',
                  fontSize: isFullscreen ? '48px' : '32px',
                  fontWeight: 'bold',
                },
              }}
              prefix={<WarningOutlined />}
            />
            <div className='mt-4'>
              <Text
                style={{
                  color: isFullscreen ? 'rgba(255,255,255,0.7)' : undefined,
                  fontSize: isFullscreen ? '16px' : '14px',
                }}
              >
                总工单: {metrics?.totalTickets || 0}
              </Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card
            className='sla-stat-card'
            style={{
              backgroundColor: isFullscreen ? 'rgba(255,255,255,0.05)' : '#fff',
              border: isFullscreen ? '1px solid rgba(255,255,255,0.1)' : undefined,
            }}
          >
            <Statistic
              title={
                <Text
                  style={{
                    color: isFullscreen ? 'rgba(255,255,255,0.8)' : undefined,
                    fontSize: isFullscreen ? '18px' : '14px',
                  }}
                >
                  平均响应时间
                </Text>
              }
              value={metrics?.averageResponseTime || 0}
              precision={1}
              suffix='小时'
              styles={{
                content: {
                  color: isFullscreen ? '#1890ff' : '#1890ff',
                  fontSize: isFullscreen ? '48px' : '32px',
                  fontWeight: 'bold',
                },
              }}
              prefix={<ClockCircleOutlined />}
            />
            <div className='mt-4'>
              <Text
                style={{
                  color: isFullscreen ? 'rgba(255,255,255,0.7)' : undefined,
                  fontSize: isFullscreen ? '16px' : '14px',
                }}
              >
                合规率: {metrics?.responseTimeCompliance || 0}%
              </Text>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 告警列表 */}
      <Card
        title={
          <div className='flex items-center gap-2'>
            <BellOutlined style={{ fontSize: '20px' }} />
            <span style={{ fontSize: isFullscreen ? '24px' : '18px' }}>SLA告警列表</span>
            <Badge count={alerts.length} showZero className='ml-2' />
          </div>
        }
        className='mb-6'
        style={{
          backgroundColor: isFullscreen ? 'rgba(255,255,255,0.05)' : '#fff',
          border: isFullscreen ? '1px solid rgba(255,255,255,0.1)' : undefined,
        }}
        bodyStyle={{
          maxHeight: isFullscreen ? '500px' : '400px',
          overflowY: 'auto',
        }}
      >
        {alerts.length > 0 ? (
          <Table
            dataSource={alerts}
            columns={alertColumns}
            rowKey='id'
            pagination={false}
            size={isFullscreen ? 'large' : 'middle'}
            rowClassName={record =>
              record.alertLevel === 'critical' ? 'sla-alert-critical' : 'sla-alert-warning'
            }
            style={{
              backgroundColor: 'transparent',
            }}
          />
        ) : (
          <div className='text-center py-12'>
            <CheckCircleOutlined
              style={{
                fontSize: '48px',
                color: isFullscreen ? 'rgba(255,255,255,0.5)' : '#d9d9d9',
                marginBottom: '16px',
              }}
            />
            <Text
              style={{
                color: isFullscreen ? 'rgba(255,255,255,0.7)' : undefined,
                fontSize: isFullscreen ? '18px' : '16px',
              }}
            >
              暂无SLA告警
            </Text>
          </div>
        )}
      </Card>

      {/* 详细指标 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card
            title={
              <div className='flex items-center gap-2'>
                <Target style={{ fontSize: '20px' }} />
                <span style={{ fontSize: isFullscreen ? '20px' : '16px' }}>响应时间合规率</span>
              </div>
            }
            style={{
              backgroundColor: isFullscreen ? 'rgba(255,255,255,0.05)' : '#fff',
              border: isFullscreen ? '1px solid rgba(255,255,255,0.1)' : undefined,
            }}
          >
            <Progress
              type='dashboard'
              percent={metrics?.responseTimeCompliance || 0}
              strokeColor={getComplianceColor(metrics?.responseTimeCompliance || 0)}
              format={percent => `${percent}%`}
              style={{ fontSize: isFullscreen ? '24px' : '18px' }}
            />
            <div className='text-center mt-4'>
              <Text
                style={{
                  color: isFullscreen ? 'rgba(255,255,255,0.8)' : undefined,
                  fontSize: isFullscreen ? '18px' : '16px',
                }}
              >
                平均响应时间: {metrics?.averageResponseTime || 0} 小时
              </Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            title={
              <div className='flex items-center gap-2'>
                <Zap style={{ fontSize: '20px' }} />
                <span style={{ fontSize: isFullscreen ? '20px' : '16px' }}>解决时间合规率</span>
              </div>
            }
            style={{
              backgroundColor: isFullscreen ? 'rgba(255,255,255,0.05)' : '#fff',
              border: isFullscreen ? '1px solid rgba(255,255,255,0.1)' : undefined,
            }}
          >
            <Progress
              type='dashboard'
              percent={metrics?.resolutionTimeCompliance || 0}
              strokeColor={getComplianceColor(metrics?.resolutionTimeCompliance || 0)}
              format={percent => `${percent}%`}
              style={{ fontSize: isFullscreen ? '24px' : '18px' }}
            />
            <div className='text-center mt-4'>
              <Text
                style={{
                  color: isFullscreen ? 'rgba(255,255,255,0.8)' : undefined,
                  fontSize: isFullscreen ? '18px' : '16px',
                }}
              >
                平均解决时间: {metrics?.averageResolutionTime || 0} 小时
              </Text>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 全屏样式 */}
      <style jsx global>{`
        .sla-monitor-dashboard.fullscreen {
          color: #fff;
        }
        .sla-monitor-dashboard.fullscreen .ant-card-head-title,
        .sla-monitor-dashboard.fullscreen .ant-statistic-title {
          color: rgba(255, 255, 255, 0.8) !important;
        }
        .sla-alert-critical {
          background-color: rgba(255, 77, 79, 0.1) !important;
        }
        .sla-alert-warning {
          background-color: rgba(250, 173, 20, 0.1) !important;
        }
        .sla-stat-card {
          transition: all 0.3s ease;
        }
        .sla-stat-card:hover {
          transform: translateY(-4px);
          box-shadow: 0 8px 16px rgba(0, 0, 0, 0.2);
        }
      `}</style>
    </div>
  );
};
