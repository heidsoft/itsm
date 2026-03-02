'use client';

/**
 * 资产列表组件
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
  message,
} from 'antd';
import {
  SearchOutlined,
  EyeOutlined,
  EditOutlined,
  PlusOutlined,
  DesktopOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { AssetApi, AssetStatus, AssetType } from '@/lib/api/asset-api';

const { Option } = Select;

// 状态颜色映射
const statusColors: Record<string, string> = {
  available: 'success',
  'in-use': 'processing',
  maintenance: 'warning',
  retired: 'default',
  disposed: 'error',
};

// 类型颜色映射
const typeColors: Record<string, string> = {
  hardware: 'blue',
  software: 'purple',
  cloud: 'cyan',
  license: 'orange',
};

const AssetList: React.FC = () => {
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
      const resp = await AssetApi.getAssets({
        ...query,
        ...values,
      });
      setData(resp.assets || []);
      setTotal(resp.total || 0);
    } catch (error) {
      message.error('加载资产列表失败');
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      const resp = await AssetApi.getAssetStats();
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
      title: '资产编号',
      dataIndex: 'asset_number',
      key: 'asset_number',
      width: 130,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className="truncate block" style={{ maxWidth: '110px' }}>{text || '-'}</span>
        </Tooltip>
      ),
    },
    {
      title: '资产名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
      ellipsis: true,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className="truncate block" style={{ maxWidth: '160px' }}>{text || '-'}</span>
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
      width: 100,
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>
          {status === 'in-use' ? '使用中' : status === 'available' ? '可用' : status === 'maintenance' ? '维护中' : status === 'retired' ? '已退役' : status === 'disposed' ? '已处置' : status}
        </Tag>
      ),
    },
    {
      title: '分类',
      dataIndex: 'category',
      key: 'category',
      width: 120,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className="truncate block" style={{ maxWidth: '100px' }}>{text || '-'}</span>
        </Tooltip>
      ),
    },
    {
      title: '分配给',
      dataIndex: 'assigned_to_name',
      key: 'assigned_to_name',
      width: 120,
      render: (name: string) => (
        <Tooltip title={name}>
          <span className="truncate block" style={{ maxWidth: '100px' }}>{name || '-'}</span>
        </Tooltip>
      ),
    },
    {
      title: '位置',
      dataIndex: 'location',
      key: 'location',
      width: 150,
      ellipsis: true,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className="truncate block" style={{ maxWidth: '130px' }}>{text || '-'}</span>
        </Tooltip>
      ),
    },
    {
      title: '采购日期',
      dataIndex: 'purchase_date',
      key: 'purchase_date',
      width: 120,
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
          <Tooltip title="查看资产详情">
            <Button
              type="text"
              icon={<EyeOutlined />}
              onClick={() => router.push(`/assets/${record.id}`)}
              aria-label={`查看资产 ${record.name || record.asset_number}`}
            />
          </Tooltip>
          <Tooltip title="编辑资产信息">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => router.push(`/assets/${record.id}`)}
              aria-label={`编辑资产 ${record.name || record.asset_number}`}
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
              title="总资产数"
              value={stats.total || 0}
              prefix={<DesktopOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="可用"
              value={stats.available || 0}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="使用中"
              value={stats.in_use || 0}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="维护中"
              value={stats.maintenance || 0}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="已退役"
              value={stats.retired || 0}
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
              <Option value="available">可用</Option>
              <Option value="in-use">使用中</Option>
              <Option value="maintenance">维护中</Option>
              <Option value="retired">已退役</Option>
              <Option value="disposed">已处置</Option>
            </Select>
          </Form.Item>
          <Form.Item name="type" label="类型">
            <Select
              placeholder="选择类型"
              allowClear
              style={{ width: 150 }}
              onChange={handleSearch}
            >
              <Option value="hardware">硬件</Option>
              <Option value="software">软件</Option>
              <Option value="cloud">云资源</Option>
              <Option value="license">许可证</Option>
            </Select>
          </Form.Item>
          <Form.Item name="category" label="分类">
            <Input placeholder="分类" style={{ width: 150 }} />
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
                onClick={() => router.push('/assets/new')}
              >
                创建资产
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
                description="暂无资产数据"
              >
                <Button type="primary" onClick={() => router.push('/assets/new')}>
                  创建第一个资产
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

export default AssetList;
