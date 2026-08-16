'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Space,
  Tag,
  Typography,
  App,
} from 'antd';
import { Search, RefreshCw, ShieldCheck } from 'lucide-react';

import {
  listAuditLogs,
  type AuditLog,
  type ListAuditLogsParams,
} from '@/lib/api/auditlog-api';
import { useAuthStoreHydration } from '@/lib/store/auth-store';

const { Title, Text } = Typography;

const METHOD_COLORS: Record<string, string> = {
  GET: 'blue',
  POST: 'green',
  PUT: 'orange',
  PATCH: 'cyan',
  DELETE: 'red',
};

const statusColor = (code: number): string => {
  if (code >= 500) return 'red';
  if (code >= 400) return 'orange';
  if (code >= 300) return 'gold';
  return 'green';
};

const AuditLogsPage: React.FC = () => {
  const { message } = App.useApp();
  useAuthStoreHydration();

  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0,
  });

  // 筛选条件（与后端 ListAuditLogsRequest 对齐）
  const [filters, setFilters] = useState({
    action: '',
    resource: '',
    method: undefined as string | undefined,
    statusCode: undefined as number | undefined,
    path: '',
  });
  // 已生效的查询条件（点击“查询”后提交）
  const [appliedFilters, setAppliedFilters] = useState(filters);

  const fetchLogs = useCallback(
    async (page: number, pageSize: number, query: typeof appliedFilters) => {
      setLoading(true);
      try {
        const params: ListAuditLogsParams = {
          page,
          pageSize,
          resource: query.resource || undefined,
          action: query.action || undefined,
          method: query.method,
          statusCode: query.statusCode,
          path: query.path || undefined,
        };
        const res = await listAuditLogs(params);
        setLogs(res.logs ?? []);
        setPagination({
          current: res.page || page,
          pageSize: res.pageSize || pageSize,
          total: res.total ?? 0,
        });
      } catch (err) {
        message.error(
          `加载审计日志失败：${err instanceof Error ? err.message : '未知错误'}`
        );
      } finally {
        setLoading(false);
      }
    },
    [message]
  );

  useEffect(() => {
    void fetchLogs(1, pagination.pageSize, appliedFilters);
     
  }, [appliedFilters]);

  const handleSearch = () => {
    setAppliedFilters({ ...filters });
  };

  const handleReset = () => {
    const empty = {
      action: '',
      resource: '',
      method: undefined,
      statusCode: undefined,
      path: '',
    };
    setFilters(empty);
    setAppliedFilters(empty);
  };

  const columns = [
    {
      title: '时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v: string) => (
        <Text style={{ whiteSpace: 'nowrap' }}>{v}</Text>
      ),
    },
    {
      title: '用户',
      dataIndex: 'userId',
      key: 'userId',
      width: 90,
      render: (v: number) => (v ? `#${v}` : '-'),
    },
    {
      title: '操作',
      dataIndex: 'action',
      key: 'action',
      width: 160,
      ellipsis: true,
    },
    {
      title: '资源',
      dataIndex: 'resource',
      key: 'resource',
      width: 160,
      ellipsis: true,
    },
    {
      title: '路径',
      dataIndex: 'path',
      key: 'path',
      ellipsis: true,
      render: (v: string) => <Text code>{v || '-'}</Text>,
    },
    {
      title: '方法',
      dataIndex: 'method',
      key: 'method',
      width: 90,
      render: (v: string) => (
        <Tag color={METHOD_COLORS[v] ?? 'default'}>{v}</Tag>
      ),
    },
    {
      title: '状态码',
      dataIndex: 'statusCode',
      key: 'statusCode',
      width: 90,
      render: (v: number) => <Tag color={statusColor(v)}>{v}</Tag>,
    },
    {
      title: 'IP',
      dataIndex: 'ip',
      key: 'ip',
      width: 130,
      ellipsis: true,
    },
  ];

  return (
    <div className="p-4 space-y-4">
      <div className="flex items-center gap-2">
        <ShieldCheck className="w-5 h-5 text-blue-500" />
        <Title level={4} style={{ margin: 0 }}>
          审计日志
        </Title>
      </div>

      <Card size="small">
        <Space wrap>
          <Input
            placeholder="操作类型"
            allowClear
            style={{ width: 160 }}
            value={filters.action}
            onChange={(e) => setFilters({ ...filters, action: e.target.value })}
            onPressEnter={handleSearch}
          />
          <Input
            placeholder="资源名称"
            allowClear
            style={{ width: 160 }}
            value={filters.resource}
            onChange={(e) => setFilters({ ...filters, resource: e.target.value })}
            onPressEnter={handleSearch}
          />
          <Input
            placeholder="操作路径"
            allowClear
            style={{ width: 220 }}
            value={filters.path}
            onChange={(e) => setFilters({ ...filters, path: e.target.value })}
            onPressEnter={handleSearch}
          />
          <Select
            placeholder="请求方式"
            allowClear
            style={{ width: 120 }}
            value={filters.method}
            onChange={(v) => setFilters({ ...filters, method: v })}
            options={['GET', 'POST', 'PUT', 'PATCH', 'DELETE'].map((m) => ({
              value: m,
              label: m,
            }))}
          />
          <Select
            placeholder="结果状态"
            allowClear
            style={{ width: 110 }}
            value={filters.statusCode}
            onChange={(v) => setFilters({ ...filters, statusCode: v })}
            options={[200, 400, 401, 403, 404, 500].map((c) => ({
              value: c,
              label: String(c),
            }))}
          />
          <Button type="primary" icon={<Search className="w-4 h-4" />} onClick={handleSearch}>
            查询
          </Button>
          <Button icon={<RefreshCw className="w-4 h-4" />} onClick={handleReset}>
            重置
          </Button>
        </Space>
      </Card>

      <Card size="small">
        <Table<AuditLog>
          rowKey="id"
          columns={columns}
          dataSource={logs}
          loading={loading}
          scroll={{ x: 1100 }}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) => {
              void fetchLogs(page, pageSize, appliedFilters);
            },
          }}
          locale={{ emptyText: '暂无审计日志' }}
        />
      </Card>
    </div>
  );
};

export default AuditLogsPage;
