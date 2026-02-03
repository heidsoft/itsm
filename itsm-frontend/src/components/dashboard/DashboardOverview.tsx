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
  TrendingUp,
  User as UserIcon,
  Clock,
  CheckCircle,
  AlertTriangle,
  Trophy,
  Users,
  BarChart3,
  RefreshCw,
  FileText,
} from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(relativeTime);

import type { TicketStats, UserStats, SystemStats } from '@/types/dashboard';
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

      const { TicketApi } = await import('@/lib/api/ticket-api');
      const { UserApi } = await import('@/lib/api/user-api');
      const { SystemConfigAPI } = await import('@/lib/api/system-config-api');

      // 并行请求所有数据
      const [ticketStatsData, userStatsData, systemStatus, recentTicketsData, activeUsersData] =
        await Promise.all([
          TicketApi.getTicketStats(),
          UserApi.getUserStats(),
          SystemConfigAPI.getSystemStatus(),
          TicketApi.getTickets({ page: 1, size: 5, sort: 'created_at,desc' }),
          UserApi.getUsers({ page: 1, page_size: 5, status: 'active' }),
        ]);

      // 转换工单统计数据
      const mappedTicketStats: TicketStats = {
        total: ticketStatsData.total,
        open: ticketStatsData.open,
        inProgress: ticketStatsData.in_progress,
        resolved: ticketStatsData.resolved,
        closed: 0, // API可能没返回closed
        byPriority: {
          low: 0,
          medium: 0,
          high: ticketStatsData.high_priority,
          urgent: 0,
          critical: 0,
        },
        byStatus: {
          open: ticketStatsData.open,
          in_progress: ticketStatsData.in_progress,
          resolved: ticketStatsData.resolved,
          closed: 0,
        },
        byType: { incident: 0, request: 0, problem: 0, change: 0 },
        byAssignee: {},
        byDepartment: {},
        avgResolutionTime: 0,
        avgResponseTime: 0,
        slaCompliance: 0,
        trend: [],
      };

      // 转换用户统计数据
      const mappedUserStats: UserStats = {
        total: userStatsData.total,
        active: userStatsData.active,
        online: 0, // 需要在线状态API
        byRole: { admin: 0, manager: 0, agent: 0, technician: 0, end_user: 0 },
        byDepartment: {},
        loginToday: 0,
        activeThisWeek: userStatsData.active,
        newThisMonth: 0,
      };

      // 转换系统状态数据
      const mappedSystemStats: SystemStats = {
        uptime: 100,
        cpuUsage: systemStatus?.cpu?.usage || 0,
        memoryUsage: systemStatus?.memory?.usage || 0,
        diskUsage: systemStatus?.disk?.usage || 0,
        avgResponseTime: systemStatus?.response?.avgResponseTime || 0,
        requestsPerSecond: 0,
        errorRate: systemStatus?.response?.errorRate || 0,
        dbConnections: systemStatus?.database?.connections || 0,
        dbSize: 0,
        cacheHitRate: 0,
        cacheSize: 0,
      };

      setTicketStats(mappedTicketStats);
      setUserStats(mappedUserStats);
      setSystemStats(mappedSystemStats);

      // 转换最近工单
      setRecentTickets(
        recentTicketsData.tickets.map((t: any) => ({
          ...t,
          requester: t.requester
            ? {
                id: t.requester.id,
                fullName: t.requester.name,
                username: t.requester.username,
                email: t.requester.email,
                role: t.requester.role || 'user',
              }
            : undefined,
          assignee: t.assignee
            ? {
                id: t.assignee.id,
                fullName: t.assignee.name,
                username: t.assignee.username,
                email: t.assignee.email,
                role: t.assignee.role || 'agent',
              }
            : undefined,
        }))
      );

      // 转换活跃用户
      setActiveUsers(
        activeUsersData.users.map((u: any) => ({
          id: u.id,
          username: u.username,
          email: u.email,
          fullName: u.name,
          role: u.role || 'user',
          status: u.active ? 'active' : 'inactive',
          lastLoginAt: u.updated_at, // 暂用更新时间代替登录时间
        }))
      );
    } catch (error) {
      console.error('Failed to fetch stats:', error);
      // 失败时不显示mock数据
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
      render: text => <span style={{ fontFamily: 'monospace' }}>#{text}</span>,
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
      render: assignee => assignee?.fullName || '未分配',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 120,
      render: text => dayjs(text).format('MM-DD HH:mm'),
    },
  ];

  if (loading && !ticketStats) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size='large' />
      </div>
    );
  }

  return (
    <div>
      {/* 头部操作栏 */}
      <Row justify='space-between' align='middle' style={{ marginBottom: 24 }}>
        <Col>
          <h2 style={{ margin: 0 }}>仪表盘概览</h2>
        </Col>
        <Col>
          <Button icon={<RefreshCw />} onClick={fetchStats} loading={loading}>
            刷新数据
          </Button>
        </Col>
      </Row>

      {/* 核心指标卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title='总工单数'
              value={ticketStats?.total || 0}
              prefix={<FileText />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title='待处理'
              value={ticketStats?.open || 0}
              prefix={<AlertTriangle />}
              styles={{ content: { color: '#faad14' } }}
              suffix={
                <Tooltip title='较昨日'>
                  <TrendingUp style={{ color: '#cf1322', fontSize: '12px' }} />
                </Tooltip>
              }
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title='处理中'
              value={ticketStats?.inProgress || 0}
              prefix={<Clock />}
              styles={{ content: { color: '#fa8c16' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title='已解决'
              value={ticketStats?.resolved || 0}
              prefix={<CheckCircle />}
              styles={{ content: { color: '#52c41a' } }}
              suffix={
                <Tooltip title='较昨日'>
                  <TrendingUp style={{ color: '#389e0d', fontSize: '12px' }} />
                </Tooltip>
              }
            />
          </Card>
        </Col>
      </Row>

      {/* 性能指标 */}
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={8}>
          <Card title='SLA 合规率' extra={<Trophy />}>
            <Progress
              type='circle'
              percent={ticketStats?.slaCompliance || 0}
              format={percent => `${percent}%`}
              strokeColor={{
                '0%': '#108ee9',
                '100%': '#87d068',
              }}
            />
            <div style={{ textAlign: 'center', marginTop: 16 }}>
              <Space orientation='vertical' size='small'>
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
          <Card title='用户统计' extra={<Users />}>
            <Row gutter={16}>
              <Col span={12}>
                <Statistic
                  title='总用户'
                  value={userStats?.total || 0}
                  styles={{ content: { fontSize: '20px' } }}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title='在线用户'
                  value={userStats?.online || 0}
                  styles={{ content: { fontSize: '20px', color: '#52c41a' } }}
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
          <Card title='系统状态' extra={<BarChart3 />}>
            <Space orientation='vertical' style={{ width: '100%' }} size='middle'>
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
          <Card title='最近工单' extra={<Button type='link'>查看全部</Button>}>
            {recentTickets.length > 0 ? (
              <Table
                columns={ticketColumns}
                dataSource={recentTickets}
                rowKey='id'
                pagination={false}
                size='small'
              />
            ) : (
              <Empty description='暂无数据' />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title='活跃用户' extra={<Button type='link'>查看全部</Button>}>
            {activeUsers.length > 0 ? (
              <List
                itemLayout='horizontal'
                dataSource={activeUsers}
                renderItem={user => (
                  <List.Item>
                    <List.Item.Meta
                      avatar={<Avatar icon={<UserIcon />} src={user.avatar} />}
                      title={user.fullName}
                      description={
                        <Space>
                          <Tag>{user.role}</Tag>
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
              <Empty description='暂无数据' />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default DashboardOverview;
