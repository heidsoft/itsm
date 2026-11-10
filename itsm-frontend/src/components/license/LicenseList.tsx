'use client';

/**
 * 许可证列表组件
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
  Progress,
  Empty,
} from 'antd';
import { Search, Plus, Pencil, Eye, Key } from 'lucide-react';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { AssetApi, LicenseStatus, LicenseType } from '@/lib/api/asset-api';

// 状态颜色映射
const statusColors: Record<string, string> = {
  active: 'success',
  expired: 'error',
  'expiring-soon': 'warning',
  depleted: 'default',
};

// 类型颜色映射
const typeColors: Record<string, string> = {
  perpetual: 'blue',
  subscription: 'purple',
  'per-user': 'cyan',
  'per-seat': 'orange',
  site: 'green',
};

const LicenseList: React.FC = () => {
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
      const resp = await AssetApi.getLicenses({
        ...query,
        ...values,
      });
      setData(resp.items);
      setTotal(resp.total);
    } catch (error) {
      message.error('Failed to load license list');
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      const resp = await AssetApi.getLicenseStats();
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
      title: 'LicensesName',
      dataIndex: 'name',
      key: 'name',
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
      dataIndex:'licenseType',
      key:'licenseType',
      width: 120,
      render: (type: string) => (
        <Tag color={typeColors[type] || 'default'}>
          {type === 'perpetual'
            ? 'Permanent'
            : type === 'subscription'
              ? 'Subscribe'
              : type === 'per-user'
                ? 'By user'
                : type === 'per-seat'
                  ? 'By seat'
                  : type === 'site'
                    ? 'site'
                    : type}
        </Tag>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>
          {status === 'active'
            ? 'valid'
            : status === 'expired'
              ? 'Expired'
              : status === 'expiring-soon'
                ? 'Expiring soon'
                : status === 'depleted'
                  ? 'Exhausted'
                  : status}
        </Tag>
      ),
    },
    {
      title: 'Usage',
      key: 'usage',
      width: 150,
      render: (_: any, record: any) => {
        const percent =
          record.totalQuantity > 0 ? (record.usedQuantity / record.totalQuantity) * 100 : 0;
        return (
          <Progress
            percent={Math.round(percent)}
            size="small"
            status={percent >= 100 ? 'exception' : percent >= 80 ? 'normal' : 'success'}
            format={() => `${record.usedQuantity}/${record.totalQuantity}`}
          />
        );
      },
    },
    {
      title: 'Supplier',
      dataIndex: 'vendor',
      key: 'vendor',
      width: 150,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className="truncate block" style={{ maxWidth: '130px' }}>
            {text || '-'}
          </span>
        </Tooltip>
      ),
    },
    {
      title: 'Expiration date',
      dataIndex:'expiryDate',
      key:'expiryDate',
      width: 120,
      render: (date: string) => {
        // 处理空值和空白字符串
        if (!date || date.trim() === '') return 'Permanent';
        const days = dayjs(date).diff(dayjs(), 'day');
        return (
          <span style={{ color: days < 30 ? '#ff4d4f' : undefined }}>
            {date} {days > 0 && days < 30 && `(${days} Expires days later)`}
          </span>
        );
      },
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
          <Tooltip title="View license details">
            <Button
              type="text"
              icon={<Eye />}
              onClick={() => router.push(`/licenses/${record.id}`)}
              aria-label={`View license ${record.name || 'Details'}`}
            />
          </Tooltip>
          <Tooltip title="Edit license information">
            <Button
              type="text"
              icon={<Pencil />}
              onClick={() => router.push(`/licenses/${record.id}/edit`)}
              aria-label={`Edit license ${record.name || 'Details'}`}
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
            <Statistic title="General license" value={stats.total || 0} prefix={<Key />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="valid"
              value={stats.active || 0}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="Expiring soon"
              value={stats.expiringSoon || 0}
              styles={{ content: { color: '#faad14' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="Expired"
              value={stats.expired || 0}
              styles={{ content: { color: '#ff4d4f' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title="Compliance Rate"
              value={stats.complianceRate || 0}
              suffix="%"
              styles={{
                content: { color: (stats.complianceRate || 0) >= 80 ? '#52c41a' : '#ff4d4f' },
              }}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <Form form={form} layout="inline" style={{ marginBottom: 16 }}>
          <Form.Item name="status" label="Status">
            <Select
              placeholder="Select status"
              allowClear
              style={{ width: 150 }}
              onChange={handleSearch}
              options={[
                { value: 'active', label: 'valid' },
                { value: 'expired', label: 'Expired' },
                { value: 'expiring-soon', label: 'Expiring soon' },
                { value: 'depleted', label: 'Exhausted' },
              ]}
            />
          </Form.Item>
          <Form.Item name="licenseType" label="Type">
            <Select
              placeholder="Select"
              allowClear
              style={{ width: 150 }}
              onChange={handleSearch}
              options={[
                { value: 'perpetual', label: 'Permanent' },
                { value: 'subscription', label: 'Subscribe' },
                { value: 'per-user', label: 'By user' },
                { value: 'per-seat', label: 'By seat' },
                { value: 'site', label: 'site' },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" icon={<Search />} onClick={handleSearch}>

                Search
              </Button>
              <Button onClick={handleReset}>Reset</Button>
              <Button
                type="primary"
                icon={<Plus />}
                onClick={() => router.push('/licenses/new')}
              >

                Create license
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
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="No license data yet">
                <Button type="primary" onClick={() => router.push('/licenses/new')}>

                  Create license
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

export default LicenseList;
