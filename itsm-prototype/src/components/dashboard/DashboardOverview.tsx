'use client';

import React, { useState, useEffect } from 'react';
import {
  Row,
  Col,
  Card,
  Statistic,
  Progress,
  Table,
  List,
  Avatar,
  Tag,
  Space,
  Button,
  Tooltip,
  Empty,
  Spin,
} from 'antd';
import {
  ArrowUpOutlined,
  UserOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  TrophyOutlined,
  TeamOutlined,
  BarChartOutlined,
  ReloadOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

import type { 
  TicketStats, 
  UserStats, 
  SystemStats
} from '@/types/dashboard';
import type { Ticket } from '@/types/ticket';
import type { User } from '@/types/user';

interface DashboardOverviewProps {
  timeRange?: '24h' | '7d' | '30d';
  refreshInterval?: number;
}

const DashboardOverview: React.FC<DashboardOverviewProps> = ({
  timeRange = '24h',
  refreshInterval = 30000, // 30秒
}) => {
  const [loading, setLoading] = useState(false);
  const [ticketStats, setTicketStats] = useState<TicketStats | null>(null);
  const [userStats, setUserStats] = useState<UserStats | null>(null);
  const [systemStats, setSystemStats] = useState<SystemStats | null>(null);
  const [recentTickets, setRecentTickets] = useState<Ticket[]>([]);
  const [activeUsers, setActiveUsers] = useState<User[]>([]);

  // 获取统计数据
  const fetchStats = async () => {
    try {
      setLoading(true);
      
      // 模拟API调用
      const mockTicketStats: TicketStats = {
        total: 1234,
        open: 156,
        inProgress: 89,
        resolved: 945,
        closed: 44,
        byPriority: {
          low: 234,
          medium: 567,
          high: 345,
          urgent: 67,
          critical: 21,
        },
        byStatus: {
          open: 156,
          in_progress: 89,
          resolved: 945,
          closed: 44,
        },
        byType: {
          incident: 567,
          request: 345,
          problem: 234,
          change: 88,
        },
        byAssignee: {
          '张三': 45,
          '李四': 38,
          '王五': 32,
        },
        byDepartment: {
          'IT部门': 234,
          '人事部': 123,
          '财务部': 89,
        },
        avgResolutionTime: 4.5,
        avgResponseTime: 0.8,
        slaCompliance: 92.5,
        trend: [
          { period: '00:00', created: 12, resolved: 8, backlog: 4 },
          { period: '04:00', created: 8, resolved: 15, backlog: -7 },
          { period: '08:00', created: 25, resolved: 18, backlog: 7 },
          { period: '12:00', created: 18, resolved: 22, backlog: -4 },
          { period: '16:00', created: 15, resolved: 12, backlog: 3 },
          { period: '20:00', created: 10, resolved: 14, backlog: -4 },
        ],
      };

      const mockUserStats: UserStats = {
        total: 245,
        active: 189,
        online: 67,
        byRole: {
          admin: 5,
          manager: 12,
          agent: 45,
          technician: 78,
          end_user: 105,
        },
        byDepartment: {
          'IT部门': 45,
          '人事部': 23,
          '财务部': 34,
          '市场部': 56,
          '运营部': 87,
        },
        loginToday: 156,
        activeThisWeek: 203,
        newThisMonth: 12,
      };

      const mockSystemStats: SystemStats = {
        uptime: 99.8,
        cpuUsage: 45.2,
        memoryUsage: 67.8,
        diskUsage: 34.5,
        avgResponseTime: 245,
        requestsPerSecond: 1250,
        errorRate: 0.02,
        dbConnections: 45,
        dbSize: 2.3,
        cacheHitRate: 94.5,
        cacheSize: 1.2,
      };

      const mockRecentTickets: Ticket[] = [
        {
          id: 1,
          ticketNumber: 'T-2024-001',
          title: '系统登录异常',
          status: 'open',
          priority: 'high',
          type: 'incident',
          requester: { id: 1, fullName: '张三', avatar: '' },
          assignee: { id: 2, fullName: '李四', avatar: '' },
          createdAt: dayjs().subtract(2, 'hour').toISOString(),
          updatedAt: dayjs().subtract(1, 'hour').toISOString(),
        },
        {
          id: 2,
          ticketNumber: 'T-2024-002',
          title: '新员工账号申请',
          status: 'in_progress',
          priority: 'medium',
          type: 'request',
          requester: { id: 3, fullName: '王五', avatar: '' },
          assignee: { id: 4, fullName: '赵六', avatar: '' },
          createdAt: dayjs().subtract(4, 'hour').toISOString(),
          updatedAt: dayjs().subtract(30, 'minute').toISOString(),
        },
      ];

      const mockActiveUsers: User[] = [
        {
          id: 1,
          username: 'zhangsan',
          email: 'zhangsan@example.com',
          fullName: '张三',
          role: 'agent',
          status: 'active',
          lastLoginAt: dayjs().subtract(10, 'minute').toISOString(),
        },
        {
          id: 2,
          username: 'lisi',
          email: 'lisi@example.com',
          fullName: '李四',
          role: 'technician',
          status: 'active',
          lastLoginAt: dayjs().subtract(5, 'minute').toISOString(),
        },
      ];

      setTicketStats(mockTicketStats);
      setUserStats(mockUserStats);
      setSystemStats(mockSystemStats);
      setRecentTickets(mockRecentTickets);
      setActiveUsers(mockActiveUsers);
    } catch (error) {
      console.error('Failed to fetch stats:', error);
    } finally {
      setLoading(false);
    }
  };

  // 初始化和定时刷新
  useEffect(() => {
    fetchStats();
    
    if (refreshInterval > 0) {
      const interval = setInterval(fetchStats, refreshInterval);
      return () => clearInterval(interval);
    }
  }, [timeRange, refreshInterval]);

  // 状态标签渲染
  const renderStatusTag = (status: string) => {
    const statusConfig: Record<string, { color: string; text: string }> = {
      open: { color: 'blue', text: '待处理' },
      in_progress: { color: 'orange', text: '处理中' },
      resolved: { color: 'green', text: '已解决' },
      closed: { color: 'default', text: '已关闭' },
    };
    
    const config = statusConfig[status] || { color: 'default', text: status };
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  // 优先级标签渲染
  const renderPriorityTag = (priority: string) => {
    const priorityConfig: Record<string, { color: string; text: string }> = {
      low: { color: 'green', text: '低' },
      medium: { color: 'blue', text: '中' },
      high: { color: 'orange', text: '高' },
      urgent: { color: 'red', text: '紧急' },
      critical: { color: 'magenta', text: '严重' },
    };
    
    const config = priorityConfig[priority] || { color: 'default', text: priority };
    return <Tag color={config.color}>{config.text}</Tag>;
  };

  // 最近工单表格列
  const ticketColumns: ColumnsType<Ticket> = [
    {
      title: '工单号',
      dataIndex: 'ticketNumber',
      key: 'ticketNumber',
      width: 120,
      render: (text) => <span style={{ fontFamily: 'monospace' }}>#{text}</span>,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: renderStatusTag,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      render: renderPriorityTag,
    },
    {
      title: '分配人',
      dataIndex: 'assignee',
      key: 'assignee',
      width: 100,
      render: (assignee) => assignee?.fullName || '未分配',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 120,
      render: (text) => dayjs(text).format('MM-DD HH:mm'),
    },
  ];

  if (loading && !ticketStats) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size="large" />
      </div>
    );
  }

  return (
    <div>
      {/* 头部操作栏 */}
      <Row justify="space-between" align="middle" style={{ marginBottom: 24 }}>
        <Col>
          <h2 style={{ margin: 0 }}>仪表盘概览</h2>
        </Col>
        <Col>
          <Button
            icon={<ReloadOutlined />}
            onClick={fetchStats}
            loading={loading}
          >
            刷新数据
          </Button>
        </Col>
      </Row>

      {/* 核心指标卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总工单数"
              value={ticketStats?.total || 0}
              prefix={<FileTextOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="待处理"
              value={ticketStats?.open || 0}
              prefix={<ExclamationCircleOutlined />}
              valueStyle={{ color: '#faad14' }}
              suffix={
                <Tooltip title="较昨日">
                  <ArrowUpOutlined style={{ color: '#cf1322', fontSize: '12px' }} />
                </Tooltip>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="处理中"
              value={ticketStats?.inProgress || 0}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="已解决"
              value={ticketStats?.resolved || 0}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#52c41a' }}
              suffix={
                <Tooltip title="较昨日">
                  <ArrowUpOutlined style={{ color: '#389e0d', fontSize: '12px' }} />
                </Tooltip>
              }
            />
          </Card>
        </Col>
      </Row>

      {/* 性能指标 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={8}>
          <Card title="SLA 合规率" extra={<TrophyOutlined />}>
            <Progress
              type="circle"
              percent={ticketStats?.slaCompliance || 0}
              format={(percent) => `${percent}%`}
              strokeColor={{
                '0%': '#108ee9',
                '100%': '#87d068',
              }}
            />
            <div style={{ textAlign: 'center', marginTop: 16 }}>
              <Space direction="vertical" size="small">
                <div>
                  <span style={{ color: '#999' }}>平均解决时间: </span>
                  <span>{ticketStats?.avgResolutionTime || 0}小时</span>
                </div>
                <div>
                  <span style={{ color: '#999' }}>平均响应时间: </span>
                  <span>{ticketStats?.avgResponseTime || 0}小时</span>
                </div>
              </Space>
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="用户统计" extra={<TeamOutlined />}>
            <Row gutter={16}>
              <Col span={12}>
                <Statistic
                  title="总用户"
                  value={userStats?.total || 0}
                  valueStyle={{ fontSize: '20px' }}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title="在线用户"
                  value={userStats?.online || 0}
                  valueStyle={{ fontSize: '20px', color: '#52c41a' }}
                />
              </Col>
            </Row>
            <div style={{ marginTop: 16 }}>
              <div style={{ marginBottom: 8 }}>
                <span style={{ color: '#999' }}>今日登录: </span>
                <span>{userStats?.loginToday || 0}</span>
              </div>
              <div style={{ marginBottom: 8 }}>
                <span style={{ color: '#999' }}>本周活跃: </span>
                <span>{userStats?.activeThisWeek || 0}</span>
              </div>
              <div>
                <span style={{ color: '#999' }}>本月新增: </span>
                <span>{userStats?.newThisMonth || 0}</span>
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="系统状态" extra={<BarChartOutlined />}>
            <Space direction="vertical" style={{ width: '100%' }} size="middle">
              <div>
                <div style={{ marginBottom: 4 }}>
                  <span>CPU 使用率</span>
                  <span style={{ float: 'right' }}>{systemStats?.cpuUsage || 0}%</span>
                </div>
                <Progress percent={systemStats?.cpuUsage || 0} showInfo={false} />
              </div>
              <div>
                <div style={{ marginBottom: 4 }}>
                  <span>内存使用率</span>
                  <span style={{ float: 'right' }}>{systemStats?.memoryUsage || 0}%</span>
                </div>
                <Progress percent={systemStats?.memoryUsage || 0} showInfo={false} />
              </div>
              <div>
                <div style={{ marginBottom: 4 }}>
                  <span>磁盘使用率</span>
                  <span style={{ float: 'right' }}>{systemStats?.diskUsage || 0}%</span>
                </div>
                <Progress percent={systemStats?.diskUsage || 0} showInfo={false} />
              </div>
            </Space>
          </Card>
        </Col>
      </Row>

      {/* 详细信息 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card title="最近工单" extra={<Button type="link">查看全部</Button>}>
            {recentTickets.length > 0 ? (
              <Table
                columns={ticketColumns}
                dataSource={recentTickets}
                rowKey="id"
                pagination={false}
                size="small"
              />
            ) : (
              <Empty description="暂无数据" />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="活跃用户" extra={<Button type="link">查看全部</Button>}>
            {activeUsers.length > 0 ? (
              <List
                itemLayout="horizontal"
                dataSource={activeUsers}
                renderItem={(user) => (
                  <List.Item>
                    <List.Item.Meta
                      avatar={<Avatar icon={<UserOutlined />} src={user.avatar} />}
                      title={user.fullName}
                      description={
                        <Space>
                          <Tag size="small">{user.role}</Tag>
                          <span style={{ color: '#999', fontSize: '12px' }}>
                            {dayjs(user.lastLoginAt).fromNow()}
                          </span>
                        </Space>
                      }
                    />
                  </List.Item>
                )}
              />
            ) : (
              <Empty description="暂无数据" />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default DashboardOverview;