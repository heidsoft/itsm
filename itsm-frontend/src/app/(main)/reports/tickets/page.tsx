'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { App, Card, Col, Empty, Row, Skeleton, Statistic, Typography } from 'antd';
import {
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import { FileText, CheckCircle, Clock, AlertTriangle } from 'lucide-react';
import { ticketService } from '@/lib/services/ticket-service';

const { Title } = Typography;

const COLORS = ['#1890ff', '#52c41a', '#faad14', '#ff4d4f', '#722ed1', '#13c2c2'];

const TicketsReportPage = () => {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState({ total: 0, open: 0, closed: 0, overdue: 0 });
  const [statusData, setStatusData] = useState<{ name: string; value: number }[]>([]);
  const [priorityData, setPriorityData] = useState<{ name: string; value: number }[]>([]);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const response = await ticketService.listTickets({ pageSize: 200 });
      const tickets = response?.tickets ?? [];

      const total = tickets.length;
      const open = tickets.filter((t: any) => t.status === 'open' || t.status === 'in_progress').length;
      const closed = tickets.filter((t: any) => t.status === 'closed' || t.status === 'resolved').length;
      const overdue = tickets.filter((t: any) => t.status === 'overdue').length;
      setStats({ total, open, closed, overdue });

      // Status distribution
      const statusMap: Record<string, number> = {};
      tickets.forEach((t: any) => {
        const s = t.status || 'unknown';
        statusMap[s] = (statusMap[s] || 0) + 1;
      });
      setStatusData(Object.entries(statusMap).map(([name, value]) => ({ name, value })));

      // Priority distribution
      const priorityMap: Record<string, number> = {};
      tickets.forEach((t: any) => {
        const p = t.priority || 'unknown';
        priorityMap[p] = (priorityMap[p] || 0) + 1;
      });
      setPriorityData(Object.entries(priorityMap).map(([name, value]) => ({ name, value })));
    } catch (error) {
      message.error('加载工单数据失败');
    } finally {
      setLoading(false);
    }
  }, [message]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  if (loading) {
    return (
      <div className="p-6 space-y-6">
        <Title level={3}>工单报表</Title>
        <Skeleton active />
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <Title level={3}>
        <FileText className="inline-block w-6 h-6 mr-2" />
        工单报表
      </Title>

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="工单总数" value={stats.total} prefix={<FileText className="w-4 h-4" />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="进行中" value={stats.open} prefix={<Clock className="w-4 h-4" />} styles={{ content: { color: '#1890ff' } }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="已完成" value={stats.closed} prefix={<CheckCircle className="w-4 h-4" />} styles={{ content: { color: '#52c41a' } }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic title="超时" value={stats.overdue} prefix={<AlertTriangle className="w-4 h-4" />} styles={{ content: { color: '#ff4d4f' } }} />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="工单状态分布">
            {statusData.length > 0 ? (
              <ResponsiveContainer width="100%" height={300}>
                <PieChart>
                  <Pie data={statusData} cx="50%" cy="50%" outerRadius={100} dataKey="value" label>
                    {statusData.map((_, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <Empty description="暂无数据" />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="工单优先级分布">
            {priorityData.length > 0 ? (
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={priorityData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" />
                  <YAxis />
                  <Tooltip />
                  <Bar dataKey="value" fill="#1890ff">
                    {priorityData.map((_, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <Empty description="暂无数据" />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default TicketsReportPage;
