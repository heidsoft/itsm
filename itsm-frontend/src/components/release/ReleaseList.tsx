'use client';

/**
 * 发布列表组件
 */

import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import type { TableColumnsType } from 'antd';
import {
  Table,
  Tag,
  Button,
  Card,
  Space,
  Tooltip,
  Input,
  Select,
  Form,
  App,
  Statistic,
  Row,
  Col,
  Empty,
  Grid,
} from 'antd';
import { Search, Plus, Pencil, Eye, Rocket } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import {
  ReleaseApi,
  type Release,
  type ReleaseStatsResponse,
  type ReleaseStatus,
  type ReleaseType,
} from '@/lib/api/release-api';
import { ManagementPageHeader } from '@/components/ui/ManagementPageHeader';



// 状态颜色映射
const statusColors: Record<string, string> = {
  draft: 'default',
  scheduled: 'blue',
  'in-progress': 'processing',
  completed: 'success',
  cancelled: 'default',
  failed: 'error',
  rolledBack: 'warning',
};

// 类型颜色映射
const typeColors: Record<string, string> = {
  major: 'red',
  minor: 'blue',
  patch: 'green',
  hotfix: 'orange',
};

const ReleaseList: React.FC = () => {
  const router = useRouter();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<Release[]>([]);
  const [total, setTotal] = useState(0);
  const [stats, setStats] = useState<ReleaseStatsResponse | null>(null);
  const [form] = Form.useForm();
  const screens = Grid.useBreakpoint();
  const requestIdRef = useRef(0);
  const hasLoadedDataRef = useRef(false);
  const compact = !screens.md;

  const [query, setQuery] = useState({
    page: 1,
    pageSize: 10,
  });

  const loadData = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setLoading(true);
    try {
      const values = await form.validateFields();
      const resp = await ReleaseApi.getReleases({
        ...query,
        ...values,
      });
      if (requestId === requestIdRef.current) {
        setData(resp.items || []);
        setTotal(resp.total || 0);
        hasLoadedDataRef.current = true;
      }
    } catch (error) {
      if (requestId === requestIdRef.current) {
        if (hasLoadedDataRef.current) {
          message.warning('Failed to refresh publication list, current data has been retained');
        } else {
          message.error('Failed to load release list');
        }
      }
    } finally {
      if (requestId === requestIdRef.current) {
        setLoading(false);
      }
    }
  }, [form, message, query]);

  const loadStats = useCallback(async () => {
    try {
      const resp = await ReleaseApi.getReleaseStats();
      setStats(resp);
    } catch (error) {
      message.error('Failed to load statistics');
    }
  }, [message]);

  useEffect(() => {
    void loadData();
    void loadStats();
  }, [loadData, loadStats]);

  const handleSearch = () => {
    setQuery(prev => ({ ...prev, page: 1 }));
  };

  const handleReset = () => {
    form.resetFields();
    setQuery({ page: 1, pageSize: 10 });
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setQuery({ ...query, page, pageSize });
  };

  const columns = useMemo<TableColumnsType<Release>>(() => [
    {
      title: 'Release number',
      dataIndex:'releaseNumber',
      key:'releaseNumber',
      width: 140,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className="truncate block" style={{ maxWidth: '120px' }}>
            {text || '-'}
          </span>
        </Tooltip>
      ),
    },
    {
      title: 'Title',
      dataIndex: 'title',
      key: 'title',
      width: 200,
      ellipsis: true,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className="truncate block" style={{ maxWidth: '180px' }}>
            {text || '-'}
          </span>
        </Tooltip>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color={typeColors[type] || 'default'}>{type?.toUpperCase()}</Tag>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>
          {status?.replace('-', ' ').toUpperCase()}
        </Tag>
      ),
    },
    {
      title: 'Target environment',
      dataIndex: 'environment',
      key: 'environment',
      width: 100,
      render: (env: string) => (
        <Tag color={env === 'production' ? 'red' : env === 'staging' ? 'orange' : 'default'}>
          {env?.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: 'Planned release date',
      dataIndex:'plannedReleaseDate',
      key:'plannedReleaseDate',
      width: 150,
      render: (date?: string) => (date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: 'Creator',
      dataIndex:'createdByName',
      key:'createdByName',
      width: 120,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className="truncate block" style={{ maxWidth: '100px' }}>
            {text || '-'}
          </span>
        </Tooltip>
      ),
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (date?: string) => (date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 120,
      fixed: 'right' as const,
      render: (_, record) => (
        <Space aria-label="Action button">
          <Tooltip title="View release details">
            <Button
              type="text"
              icon={<Eye />}
              href={`/releases/${record.id}`}
              aria-label={`View releases ${record.title || record.releaseNumber || 'Details'}`}
            />
          </Tooltip>
          <Tooltip title="Edit release information">
            <Button
              type="text"
              icon={<Pencil />}
              href={`/releases/${record.id}/edit`}
              aria-label={`Edit and publish ${record.title || record.releaseNumber || 'Details'}`}
            />
          </Tooltip>
        </Space>
      ),
    },
  ], []);

  return (
    <div className="p-3 md:p-6">
      <ManagementPageHeader
        title="Release Management"
        description="Plan, track, and audit software and infrastructure release activities"
        actions={
          <Button
            type="primary"
            icon={<Plus />}
            onClick={() => router.push('/releases/new')}
          >

            Create publication
          </Button>
        }
        className="mb-4"
      />

      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic title="Total number of releases" value={stats?.total ?? 0} prefix={<Rocket />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="In Progress"
              value={stats?.inProgress ?? 0}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="Completed"
              value={stats?.completed ?? 0}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic title="Cancelled" value={stats?.cancelled ?? 0} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic title="Failed" value={stats?.failed ?? 0} styles={{ content: { color: '#ff4d4f' } }} />
          </Card>
        </Col>
      </Row>

      <Card>
        <Form
          form={form}
          layout={compact ? 'vertical' : 'inline'}
          style={{ marginBottom: 16 }}
        >
          <Form.Item name="status" label="Status">
            <Select
              placeholder="Select status"
              allowClear
              style={{ width: compact ? '100%' : 150 }}
              onChange={handleSearch}
              options={[
                { value: 'draft', label: 'Draft' },
                { value: 'scheduled', label: 'Planned' },
                { value: 'in-progress', label: 'In Progress' },
                { value: 'completed', label: 'Completed' },
                { value: 'cancelled', label: 'Cancelled' },
                { value: 'failed', label: 'Failed' },
              ]}
            />
          </Form.Item>
          <Form.Item name="type" label="Type">
            <Select
              placeholder="Select"
              allowClear
              style={{ width: compact ? '100%' : 150 }}
              onChange={handleSearch}
              options={[
                { value: 'major', label: 'Major version' },
                { value: 'minor', label: 'minor version' },
                { value: 'patch', label: 'Patch' },
                { value: 'hotfix', label: 'Emergency fix' },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Space wrap>
              <Button type="primary" icon={<Search />} onClick={handleSearch}>

                Search
              </Button>
              <Button onClick={handleReset}>Reset</Button>
            </Space>
          </Form.Item>
        </Form>

        <Table
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          scroll={{ x: 'max-content' }}
          locale={{
            emptyText: (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No data released yet">
                <Button type="primary" onClick={() => router.push('/releases/new')}>

                  Create publication
                </Button>
              </Empty>
            ),
          }}
          pagination={{
            current: query.page,
            pageSize: query.pageSize,
            total,
            onChange: handlePageChange,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: total => `Total ${total} items`,
            pageSizeOptions: ['10', '20', '50', '100'],
          }}
        />
      </Card>
    </div>
  );
};

export default ReleaseList;
