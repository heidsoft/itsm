'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { Card, Col, Empty, Row, Select, Space, Spin, Statistic, Table, Tag, Typography, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { Activity, AlertTriangle, CheckCircle, Zap } from 'lucide-react';
import { IncidentAPI } from '@/lib/api/incident-api';
import type { Incident, ListIncidentsResponse } from '@/lib/api/incident-api';

const { Title, Text } = Typography;

const PRIORITY_META: Record<string, { color: string; label: string }> = {
  critical: { color: 'red', label: '紧急' },
  urgent: { color: 'red', label: '紧急' },
  high: { color: 'orange', label: '高' },
  medium: { color: 'blue', label: '中' },
  low: { color: 'default', label: '低' },
};

const STATUS_META: Record<string, { color: string; label: string }> = {
  new: { color: 'blue', label: '新建' },
  open: { color: 'orange', label: '打开' },
  in_progress: { color: 'cyan', label: '处理中' },
  resolved: { color: 'green', label: '已解决' },
  closed: { color: 'default', label: '已关闭' },
};

const PAGE_SIZE = 20;

export default function NOCPage() {
  const [loading, setLoading] = useState(false);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined);

  const loadData = useCallback(async (p: number, status?: string) => {
    setLoading(true);
    try {
      const resp: ListIncidentsResponse = await IncidentAPI.getIncidents({
        isMajorIncident: true,
        page: p,
        pageSize: PAGE_SIZE,
        ...(status ? { status } : {}),
      });
      setIncidents(resp.incidents || resp.items || []);
      setTotal(resp.total ?? 0);
    } catch {
      message.error('加载重大事件失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadData(page, statusFilter);
  }, [loadData, page, statusFilter]);

  const columns: ColumnsType<Incident> = [
    {
      title: '编号',
      dataIndex: 'incidentNumber',
      key: 'incidentNumber',
      width: 130,
      render: (num: string | undefined, record) => num || `#${record.id}`,
    },
    {
      title: '标题',
      dataIndex: 'title',
      key: 'title',
      ellipsis: true,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      render: (v: string) => {
        const m = PRIORITY_META[(v || '').toLowerCase()];
        return <Tag color={m?.color || 'default'}>{m?.label || v || '-'}</Tag>;
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (v: string) => {
        const m = STATUS_META[(v || '').toLowerCase()];
        return <Tag color={m?.color || 'default'}>{m?.label || v || '-'}</Tag>;
      },
    },
    {
      title: '处理人',
      key: 'assignee',
      width: 120,
      render: (_, r) => r.assignee?.name || '-',
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (v: string) => (v ? new Date(v).toLocaleString('zh-CN', { hour12: false }) : '-'),
    },
  ];

  // 统计按优先级分组
  const priorityCounts = incidents.reduce<Record<string, number>>((acc, inc) => {
    const k = (inc.priority || 'unknown').toLowerCase();
    acc[k] = (acc[k] || 0) + 1;
    return acc;
  }, {});

  const cardStyle: React.CSSProperties = { height: '100%' };

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      <div className="mb-6">
        <Title level={2} style={{ marginBottom: 4 }}>NOC 工作台</Title>
        <Text type="secondary">重大事件作战室：实时跟踪企业级重大事件处置进展</Text>
      </div>

      {/* KPI 卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={8}>
          <Card style={cardStyle}>
            <Statistic
              title="重大事件总数"
              value={total}
              prefix={<Zap size={18} className="text-red-500 mr-2" />}
              valueStyle={{ color: '#cf1322' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card style={cardStyle}>
            <Statistic
              title="高优事件"
              value={(priorityCounts['critical'] || 0) + (priorityCounts['urgent'] || 0) + (priorityCounts['high'] || 0)}
              prefix={<AlertTriangle size={18} className="text-orange-500 mr-2" />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={8}>
          <Card style={cardStyle}>
            <Statistic
              title="正在处理"
              value={incidents.filter(i => ['in_progress', 'open'].includes((i.status || '').toLowerCase())).length}
              prefix={<Activity size={18} className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 筛选 + 列表 */}
      <Card>
        <div className="mb-4 flex flex-wrap items-center gap-3">
          <Text strong>状态筛选：</Text>
          <Select
            allowClear
            placeholder="全部状态"
            style={{ width: 160 }}
            value={statusFilter}
            onChange={(v) => { setStatusFilter(v); setPage(1); }}
            options={[
              { value: 'new', label: '新建' },
              { value: 'open', label: '打开' },
              { value: 'in_progress', label: '处理中' },
              { value: 'resolved', label: '已解决' },
            ]}
          />
          <CheckCircle size={16} className="text-gray-400" />
          <Text type="secondary">共 {total} 条重大事件</Text>
        </div>
        <Table<Incident>
          columns={columns}
          dataSource={incidents}
          rowKey="id"
          loading={loading}
          pagination={{
            current: page,
            pageSize: PAGE_SIZE,
            total,
            onChange: (p) => setPage(p),
            showSizeChanger: false,
          }}
          locale={{ emptyText: <Empty description="暂无重大事件" /> }}
          scroll={{ x: 900 }}
          size="middle"
        />
      </Card>
    </div>
  );
}
