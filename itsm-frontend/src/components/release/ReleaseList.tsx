'use client';

/**
 * 发布列表组件
 */

import React, { useState, useEffect, useRef } from 'react';
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

import { ReleaseApi, ReleaseStatus, ReleaseType } from '@/lib/api/release-api';
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
  const [data, setData] = useState<any[]>([]);
  const [total, setTotal] = useState(0);
  const [stats, setStats] = useState<any>({});
  const [form] = Form.useForm();
  const screens = Grid.useBreakpoint();
  const requestIdRef = useRef(0);
  const hasLoadedDataRef = useRef(false);
  const compact = !screens.md;

  const [query, setQuery] = useState({
    page: 1,
    pageSize: 10,
  });

  const loadData = async () => {
    const requestId = ++requestIdRef.current;
    setLoading(true);
    try {
      const values = await form.validateFields();
      const resp = await ReleaseApi.getReleases({
        ...query,
        ...values,
      });
      if (requestId === requestIdRef.current) {
        setData(resp.releases || []);
        setTotal(resp.total || 0);
        hasLoadedDataRef.current = true;
      }
    } catch (error) {
      if (requestId === requestIdRef.current) {
        if (hasLoadedDataRef.current) {
          message.warning('刷新发布列表失败，已保留当前数据');
        } else {
          message.error('加载发布列表失败');
        }
      }
    } finally {
      if (requestId === requestIdRef.current) {
        setLoading(false);
      }
    }
  };

  const loadStats = async () => {
    try {
      const resp = await ReleaseApi.getReleaseStats();
      setStats(resp);
    } catch (error) {
      message.error('加载统计数据失败');
    }
  };

  useEffect(() => {
    loadData();
    loadStats();
  }, [query]);

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

  const columns = [
    {
      title: '发布编号',
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
      title: '标题',
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
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color={typeColors[type] || 'default'}>{type?.toUpperCase()}</Tag>
      ),
    },
    {
      title: '状态',
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
      title: '目标环境',
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
      title: '计划发布日期',
      dataIndex:'plannedReleaseDate',
      key:'plannedReleaseDate',
      width: 150,
      render: (date?: string) => (date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '创建人',
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
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (date?: string) => (date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      fixed: 'right' as const,
      render: (_: any, record: any) => (
        <Space aria-label="操作按钮">
          <Tooltip title="查看发布详情">
            <Button
              type="text"
              icon={<Eye />}
              href={`/releases/${record.id}`}
              aria-label={`查看发布 ${record.title || record.releaseNumber || '详情'}`}
            />
          </Tooltip>
          <Tooltip title="编辑发布信息">
            <Button
              type="text"
              icon={<Pencil />}
              href={`/releases/${record.id}/edit`}
              aria-label={`编辑发布 ${record.title || record.releaseNumber || '详情'}`}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-3 md:p-6">
      <ManagementPageHeader
        title="发布管理"
        description="规划、跟踪并审计软件和基础设施发布活动"
        actions={
          <Button
            type="primary"
            icon={<Plus />}
            onClick={() => router.push('/releases/new')}
          >
            创建发布
          </Button>
        }
        className="mb-4"
      />

      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic title="总发布数" value={stats.total || 0} prefix={<Rocket />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="进行中"
              value={stats.inProgress || 0}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="已完成"
              value={stats.completed || 0}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic title="已取消" value={stats.cancelled || 0} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic title="失败" value={stats.failed || 0} styles={{ content: { color: '#ff4d4f' } }} />
          </Card>
        </Col>
      </Row>

      <Card>
        <Form
          form={form}
          layout={compact ? 'vertical' : 'inline'}
          style={{ marginBottom: 16 }}
        >
          <Form.Item name="status" label="状态">
            <Select
              placeholder="选择状态"
              allowClear
              style={{ width: compact ? '100%' : 150 }}
              onChange={handleSearch}
              options={[
                { value: 'draft', label: '草稿' },
                { value: 'scheduled', label: '已计划' },
                { value: 'in-progress', label: '进行中' },
                { value: 'completed', label: '已完成' },
                { value: 'cancelled', label: '已取消' },
                { value: 'failed', label: '失败' },
              ]}
            />
          </Form.Item>
          <Form.Item name="type" label="类型">
            <Select
              placeholder="选择类型"
              allowClear
              style={{ width: compact ? '100%' : 150 }}
              onChange={handleSearch}
              options={[
                { value: 'major', label: '主版本' },
                { value: 'minor', label: '次版本' },
                { value: 'patch', label: '补丁' },
                { value: 'hotfix', label: '紧急修复' },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Space wrap>
              <Button type="primary" icon={<Search />} onClick={handleSearch}>
                搜索
              </Button>
              <Button onClick={handleReset}>重置</Button>
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
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="暂无发布数据">
                <Button type="primary" onClick={() => router.push('/releases/new')}>
                  创建第一个发布
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
            showTotal: total => `共 ${total} 条`,
            pageSizeOptions: ['10', '20', '50', '100'],
          }}
        />
      </Card>
    </div>
  );
};

export default ReleaseList;
