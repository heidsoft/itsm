'use client';

/**
 * 发布列表组件
 */

import React, { useState, useEffect } from 'react';
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
} from 'antd';
import {
  SearchOutlined,
  EyeOutlined,
  EditOutlined,
  PlusOutlined,
  RocketOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { ReleaseApi, ReleaseStatus, ReleaseType } from '@/lib/api/release-api';

const { Option } = Select;

// 状态颜色映射
const statusColors: Record<string, string> = {
  draft: 'default',
  scheduled: 'blue',
  'in-progress': 'processing',
  completed: 'success',
  cancelled: 'default',
  failed: 'error',
  rolled_back: 'warning',
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

  const [query, setQuery] = useState({
    page: 1,
    pageSize: 10,
  });

  const loadData = async () => {
    setLoading(true);
    try {
      const values = await form.validateFields();
      const resp = await ReleaseApi.getReleases({
        ...query,
        ...values,
      });
      setData(resp.releases || []);
      setTotal(resp.total || 0);
    } catch (error) {
      message.error('加载发布列表失败');
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      const resp = await ReleaseApi.getReleaseStats();
      setStats(resp);
    } catch (error) {
      console.error('Failed to load stats:', error);
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
      dataIndex: 'release_number',
      key: 'release_number',
      width: 140,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className="truncate block" style={{ maxWidth: '120px' }}>{text || '-'}</span>
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
          <span className="truncate block" style={{ maxWidth: '180px' }}>{text || '-'}</span>
        </Tooltip>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color={typeColors[type] || 'default'}>
          {type?.toUpperCase()}
        </Tag>
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
      dataIndex: 'planned_release_date',
      key: 'planned_release_date',
      width: 150,
      render: (date: string) => date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-',
    },
    {
      title: '创建人',
      dataIndex: 'created_by_name',
      key: 'created_by_name',
      width: 120,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className="truncate block" style={{ maxWidth: '100px' }}>{text || '-'}</span>
        </Tooltip>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: any, record: any) => (
        <Space aria-label="操作按钮">
          <Tooltip title="查看发布详情">
            <Button
              type="text"
              icon={<EyeOutlined />}
              onClick={() => router.push(`/releases/${record.id}`)}
              aria-label={`查看发布 ${record.title || record.release_number || '详情'}`}
            />
          </Tooltip>
          <Tooltip title="编辑发布信息">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => router.push(`/releases/${record.id}`)}
              aria-label={`编辑发布 ${record.title || record.release_number || '详情'}`}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="总发布数"
              value={stats.total || 0}
              prefix={<RocketOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="进行中"
              value={stats.in_progress || 0}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="已完成"
              value={stats.completed || 0}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="已取消"
              value={stats.cancelled || 0}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="失败"
              value={stats.failed || 0}
              valueStyle={{ color: '#ff4d4f' }}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <Form
          form={form}
          layout="inline"
          style={{ marginBottom: 16 }}
        >
          <Form.Item name="status" label="状态">
            <Select
              placeholder="选择状态"
              allowClear
              style={{ width: 150 }}
              onChange={handleSearch}
            >
              <Option value="draft">草稿</Option>
              <Option value="scheduled">已计划</Option>
              <Option value="in-progress">进行中</Option>
              <Option value="completed">已完成</Option>
              <Option value="cancelled">已取消</Option>
              <Option value="failed">失败</Option>
            </Select>
          </Form.Item>
          <Form.Item name="type" label="类型">
            <Select
              placeholder="选择类型"
              allowClear
              style={{ width: 150 }}
              onChange={handleSearch}
            >
              <Option value="major">主版本</Option>
              <Option value="minor">次版本</Option>
              <Option value="patch">补丁</Option>
              <Option value="hotfix">紧急修复</Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button
                type="primary"
                icon={<SearchOutlined />}
                onClick={handleSearch}
              >
                搜索
              </Button>
              <Button onClick={handleReset}>重置</Button>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => router.push('/releases/new')}
              >
                创建发布
              </Button>
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
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description="暂无发布数据"
              >
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
            showTotal: (total) => `共 ${total} 条`,
          }}
        />
      </Card>
    </div>
  );
};

export default ReleaseList;
