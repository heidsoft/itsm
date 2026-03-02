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
import {
  SearchOutlined,
  EyeOutlined,
  EditOutlined,
  PlusOutlined,
  KeyOutlined,
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { AssetApi, LicenseStatus, LicenseType } from '@/lib/api/asset-api';

const { Option } = Select;

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
      setData(resp.licenses || []);
      setTotal(resp.total || 0);
    } catch (error) {
      message.error('加载许可证列表失败');
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
      title: '许可证名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
      ellipsis: true,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className='truncate block' style={{ maxWidth: '180px' }}>
            {text || '-'}
          </span>
        </Tooltip>
      ),
    },
    {
      title: '类型',
      dataIndex: 'license_type',
      key: 'license_type',
      width: 120,
      render: (type: string) => (
        <Tag color={typeColors[type] || 'default'}>
          {type === 'perpetual'
            ? '永久'
            : type === 'subscription'
              ? '订阅'
              : type === 'per-user'
                ? '按用户'
                : type === 'per-seat'
                  ? '按席位'
                  : type === 'site'
                    ? '站点'
                    : type}
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
          {status === 'active'
            ? '有效'
            : status === 'expired'
              ? '已过期'
              : status === 'expiring-soon'
                ? '即将过期'
                : status === 'depleted'
                  ? '已耗尽'
                  : status}
        </Tag>
      ),
    },
    {
      title: '使用情况',
      key: 'usage',
      width: 150,
      render: (_: any, record: any) => {
        const percent =
          record.total_quantity > 0 ? (record.used_quantity / record.total_quantity) * 100 : 0;
        return (
          <Progress
            percent={Math.round(percent)}
            size='small'
            status={percent >= 100 ? 'exception' : percent >= 80 ? 'normal' : 'success'}
            format={() => `${record.used_quantity}/${record.total_quantity}`}
          />
        );
      },
    },
    {
      title: '供应商',
      dataIndex: 'vendor',
      key: 'vendor',
      width: 150,
      render: (text: string) => (
        <Tooltip title={text}>
          <span className='truncate block' style={{ maxWidth: '130px' }}>
            {text || '-'}
          </span>
        </Tooltip>
      ),
    },
    {
      title: '到期日期',
      dataIndex: 'expiry_date',
      key: 'expiry_date',
      width: 120,
      render: (date: string) => {
        if (!date) return '-';
        const days = dayjs(date).diff(dayjs(), 'day');
        return (
          <span style={{ color: days < 30 ? '#ff4d4f' : undefined }}>
            {date} {days > 0 && days < 30 && `(${days}天后到期)`}
          </span>
        );
      },
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
        <Space aria-label='操作按钮'>
          <Tooltip title='查看许可证详情'>
            <Button
              type='text'
              icon={<EyeOutlined />}
              onClick={() => router.push(`/licenses/${record.id}`)}
              aria-label={`查看许可证 ${record.name || '详情'}`}
            />
          </Tooltip>
          <Tooltip title='编辑许可证信息'>
            <Button
              type='text'
              icon={<EditOutlined />}
              onClick={() => router.push(`/licenses/${record.id}`)}
              aria-label={`编辑许可证 ${record.name || '详情'}`}
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
            <Statistic title='总许可证' value={stats.total || 0} prefix={<KeyOutlined />} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic title='有效' value={stats.active || 0} valueStyle={{ color: '#52c41a' }} />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title='即将过期'
              value={stats.expiring_soon || 0}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title='已过期'
              value={stats.expired || 0}
              valueStyle={{ color: '#ff4d4f' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6} xl={4}>
          <Card>
            <Statistic
              title='合规率'
              value={stats.compliance_rate || 0}
              suffix='%'
              valueStyle={{ color: (stats.compliance_rate || 0) >= 80 ? '#52c41a' : '#ff4d4f' }}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <Form form={form} layout='inline' style={{ marginBottom: 16 }}>
          <Form.Item name='status' label='状态'>
            <Select
              placeholder='选择状态'
              allowClear
              style={{ width: 150 }}
              onChange={handleSearch}
            >
              <Option value='active'>有效</Option>
              <Option value='expired'>已过期</Option>
              <Option value='expiring-soon'>即将过期</Option>
              <Option value='depleted'>已耗尽</Option>
            </Select>
          </Form.Item>
          <Form.Item name='license_type' label='类型'>
            <Select
              placeholder='选择类型'
              allowClear
              style={{ width: 150 }}
              onChange={handleSearch}
            >
              <Option value='perpetual'>永久</Option>
              <Option value='subscription'>订阅</Option>
              <Option value='per-user'>按用户</Option>
              <Option value='per-seat'>按席位</Option>
              <Option value='site'>站点</Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type='primary' icon={<SearchOutlined />} onClick={handleSearch}>
                搜索
              </Button>
              <Button onClick={handleReset}>重置</Button>
              <Button
                type='primary'
                icon={<PlusOutlined />}
                onClick={() => router.push('/licenses/new')}
              >
                创建许可证
              </Button>
            </Space>
          </Form.Item>
        </Form>

        <Table
          columns={columns}
          dataSource={data}
          rowKey='id'
          loading={loading}
          scroll={{ x: 'max-content' }}
          locale={{
            emptyText: (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description='暂无许可证数据'>
                <Button type='primary' onClick={() => router.push('/licenses/new')}>
                  创建第一个许可证
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
          }}
        />
      </Card>
    </div>
  );
};

export default LicenseList;
