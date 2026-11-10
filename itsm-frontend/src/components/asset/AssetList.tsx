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
} from 'antd';
import { Search, Plus, Pencil, Eye, Monitor } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import LoadingEmptyError from '@/components/ui/LoadingEmptyError';

import { AssetApi, AssetStatus, AssetType } from '@/lib/api/asset-api';
import { usePermissions } from '@/lib/hooks/use-permissions';

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

// 类型中文名映射（与 AssetDetail、筛选下拉保持一致）
const typeLabels: Record<string, string> = {
  hardware: 'Hardware',
  software: 'Software',
  cloud: 'Cloud Resources',
  license: 'Licenses',
};

interface AssetListProps {
  showActions?: boolean;
}

const AssetList: React.FC<AssetListProps> = ({ showActions = true }) => {
  const router = useRouter();
  const { message } = App.useApp();
  // 空态里的「新增资产」是独立入口，不经过页面 primaryAction 的门控——
  // 只读角色会看到一个点进去 403 的按钮（2026-10-26 浏览器实测：end_user 仍可见）。
  const { hasPermission } = usePermissions();
  const canCreateAsset = hasPermission('asset', 'write') || hasPermission('asset', 'create');
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const [data, setData] = useState<any[]>([]);
  const [total, setTotal] = useState(0);
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
      setData(resp.items);
      setTotal(resp.total);
    } catch (error) {
      setLoadError(true);
      message.error('Failed to load asset list');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
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
      title: 'Asset Number',
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
      title: 'Asset Name',
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
      title: 'Type',
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: string) => (
        <Tag color={typeColors[type] || 'default'}>{typeLabels[type] || type || '-'}</Tag>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>
          {status === 'in-use'
            ? 'In Use'
            : status === 'available'
              ? 'Available'
              : status === 'maintenance'
                ? 'Maintenance'
                : status === 'retired'
                  ? 'Retired'
                  : status === 'disposed'
                    ? 'Disposed'
                    : status}
        </Tag>
      ),
    },
    {
      title: 'Category',
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
      title: 'Assigned To',
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
      title: 'Location',
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
      title: 'Purchase date',
      dataIndex:'purchaseDate',
      key:'purchaseDate',
      width: 120,
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: 'Actions',
      key: 'action',
      width: 120,
      render: (_: any, record: any) => (
        <Space aria-label="Action button">
          <Tooltip title="View asset details">
            <Button
              type="text"
              icon={<Eye />}
              onClick={() => router.push(`/assets/${record.id}`)}
              aria-label={`View assets ${record.name || record.assetNumber}`}
            />
          </Tooltip>
          <Tooltip title="Edit asset information">
            <Button
              type="text"
              icon={<Pencil />}
              onClick={() => router.push(`/assets/${record.id}/edit`)}
              aria-label={`Edit assets ${record.name || record.assetNumber}`}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '24px' }}>
      <Card>
        <Form form={form} layout="inline" style={{ marginBottom: 16 }}>
          <Form.Item name="status" label="Status">
            <Select
              placeholder="Select status"
              allowClear
              style={{ width: 150 }}
              onChange={handleSearch}
              options={[
                { value: 'available', label: 'Available' },
                { value: 'in-use', label: 'In Use' },
                { value: 'maintenance', label: 'Maintenance' },
                { value: 'retired', label: 'Retired' },
                { value: 'disposed', label: 'Disposed' },
              ]}
            />
          </Form.Item>
          <Form.Item name="type" label="Type">
            <Select
              placeholder="Select"
              allowClear
              style={{ width: 150 }}
              onChange={handleSearch}
              options={[
                { value: 'hardware', label: 'Hardware' },
                { value: 'software', label: 'Software' },
                { value: 'cloud', label: 'Cloud Resources' },
                { value: 'license', label: 'Licenses' },
              ]}
            />
          </Form.Item>
          <Form.Item name="category" label="Category">
            <Input placeholder="Category" style={{ width: 150 }} />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" icon={<Search />} onClick={handleSearch}>

                Search
              </Button>
              <Button onClick={handleReset}>Reset</Button>
              {showActions && (
                <Button
                  type="primary"
                  icon={<Plus />}
                  onClick={() => router.push('/assets/new')}
                >

                  Create assets
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
                  title: 'No asset data yet',
                  actionText: 'Add Asset',
                  onAction: () => router.push('/assets/new'),
                  showAction: canCreateAsset,
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
            showTotal: total => `Total ${total} items`,
          }}
        />
      </Card>
    </div>
  );
};

export default AssetList;
