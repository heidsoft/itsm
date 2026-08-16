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
  message,
} from 'antd';
import { Search, Plus, Pencil, Eye, Monitor } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import LoadingEmptyError from '@/components/ui/LoadingEmptyError';

import { AssetApi, AssetStatus, AssetType } from '@/lib/api/asset-api';

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

interface AssetListProps {
  showActions?: boolean;
}

const AssetList: React.FC<AssetListProps> = ({ showActions = true }) => {
  const router = useRouter();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState(false);
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
    setLoadError(false);
    try {
      const values = await form.validateFields();
      const resp = await AssetApi.getAssets({
        ...query,
        ...values,
      });
      setData(resp.assets || []);
      setTotal(resp.total || 0);
    } catch (error) {
      setLoadError(true);
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
      dataIndex:'assetNumber',
      key:'assetNumber',
      width: 130,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className="truncate block" style={{ maxWidth: '110px' }}>
            {text || '-'}
          </span>
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
          <span className="truncate block" style={{ maxWidth: '160px' }}>
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
      width: 100,
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>
          {status === 'in-use'
            ? '使用中'
            : status === 'available'
              ? '可用'
              : status === 'maintenance'
                ? '维护中'
                : status === 'retired'
                  ? '已退役'
                  : status === 'disposed'
                    ? '已处置'
                    : status}
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
          <span className="truncate block" style={{ maxWidth: '100px' }}>
            {text || '-'}
          </span>
        </Tooltip>
      ),
    },
    {
      title: '分配给',
      dataIndex:'assignedToName',
      key:'assignedToName',
      width: 120,
      render: (name: string) => (
        <Tooltip title={name}>
          <span className="truncate block" style={{ maxWidth: '100px' }}>
            {name || '-'}
          </span>
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
          <span className="truncate block" style={{ maxWidth: '130px' }}>
            {text || '-'}
          </span>
        </Tooltip>
      ),
    },
    {
      title: '采购日期',
      dataIndex:'purchaseDate',
      key:'purchaseDate',
      width: 120,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
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
              icon={<Eye />}
              onClick={() => router.push(`/assets/${record.id}`)}
              aria-label={`查看资产 ${record.name || record.assetNumber}`}
            />
          </Tooltip>
          <Tooltip title="编辑资产信息">
            <Button
              type="text"
              icon={<Pencil />}
              onClick={() => router.push(`/assets/${record.id}/edit`)}
              aria-label={`编辑资产 ${record.name || record.assetNumber}`}
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
            <Statistic title="总资产数" value={stats.total || 0} prefix={<Monitor />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="可用"
              value={stats.available || 0}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic title="使用中" value={stats.inUse || 0} styles={{ content: { color: '#1890ff' } }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="维护中"
              value={stats.maintenance || 0}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic title="已退役" value={stats.retired || 0} />
          </Card>
        </Col>
      </Row>

      <Card>
        <Form form={form} layout="inline" style={{ marginBottom: 16 }}>
          <Form.Item name="status" label="状态">
            <Select
              placeholder="选择状态"
              allowClear
              style={{ width: 150 }}
              onChange={handleSearch}
              options={[
                { value: 'available', label: '可用' },
                { value: 'in-use', label: '使用中' },
                { value: 'maintenance', label: '维护中' },
                { value: 'retired', label: '已退役' },
                { value: 'disposed', label: '已处置' },
              ]}
            />
          </Form.Item>
          <Form.Item name="type" label="类型">
            <Select
              placeholder="选择类型"
              allowClear
              style={{ width: 150 }}
              onChange={handleSearch}
              options={[
                { value: 'hardware', label: '硬件' },
                { value: 'software', label: '软件' },
                { value: 'cloud', label: '云资源' },
                { value: 'license', label: '许可证' },
              ]}
            />
          </Form.Item>
          <Form.Item name="category" label="分类">
            <Input placeholder="分类" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" icon={<Search />} onClick={handleSearch}>
                搜索
              </Button>
              <Button onClick={handleReset}>重置</Button>
              {showActions && (
                <Button
                  type="primary"
                  icon={<Plus />}
                  onClick={() => router.push('/assets/new')}
                >
                  创建资产
                </Button>
              )}
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
              <LoadingEmptyError
                state="empty"
                empty={{
                  title: '暂无资产数据',
                  description: '当前没有资产记录，点击下方按钮创建第一个资产',
                  actionText: '新增资产',
                  onAction: () => router.push('/assets/new'),
                  showAction: true,
                  icon: <Monitor size={48} />,
                }}
              />
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
          }}
        />
      </Card>
    </div>
  );
};

export default AssetList;
