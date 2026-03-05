'use client';

import React, { useState, useEffect } from 'react';
import { Card, Row, Col, Statistic, Typography, Tabs, Table, Tag, Button, Progress, Space, Input, Select, App } from 'antd';
import { Database, Server, Cloud, Shield, Plus, AlertTriangle, CheckCircle } from 'lucide-react';
import { SyncOutlined } from '@ant-design/icons';
import { SearchOutlined, FilterOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import CIList from '@/components/cmdb/CIList';

const { Title, Text } = Typography;

// Mock CI statistics by type
const ciTypeStats = [
  { type: '服务器', count: 45, status: 'online' },
  { type: '网络设备', count: 28, status: 'online' },
  { type: '存储设备', count: 15, status: 'online' },
  { type: '云资源', count: 120, status: 'active' },
  { type: '安全设备', count: 12, status: 'online' },
];

// Mock cloud resources data
const mockCloudResources = [
  { id: 1, resource_id: 'i-bp123456', resource_name: 'Web服务器-01', resource_type: 'ECS', provider: '阿里云', region: '华东1', status: 'running', ip: '47.xxx.xxx.1', create_time: '2024-01-01' },
  { id: 2, resource_id: 'i-bp234567', resource_name: 'Web服务器-02', resource_type: 'ECS', provider: '阿里云', region: '华东1', status: 'running', ip: '47.xxx.xxx.2', create_time: '2024-01-02' },
  { id: 3, resource_id: 'rm-mysql123', resource_name: 'MySQL主库', resource_type: 'RDS', provider: '阿里云', region: '华东1', status: 'running', ip: '172.xxx.xxx.1', create_time: '2024-01-03' },
  { id: 4, resource_id: 'oss-bucket001', resource_name: '文件存储桶', resource_type: 'OSS', provider: '阿里云', region: '华东1', status: 'running', ip: '-', create_time: '2024-01-04' },
  { id: 5, resource_id: 'lb-xtest001', resource_name: '负载均衡器', resource_type: 'SLB', provider: '阿里云', region: '华东1', status: 'running', ip: '47.xxx.xxx.10', create_time: '2024-01-05' },
  { id: 6, resource_id: 'i-bp345678', resource_name: 'API服务器', resource_type: 'ECS', provider: '阿里云', region: '华东2', status: 'stopped', ip: '47.xxx.xxx.3', create_time: '2024-01-06' },
];

// Mock reconciliation data
const mockReconciliationData = [
  { id: 1, ci_name: 'Web服务器-01', ci_type: '服务器', source: 'CMDB', target: '阿里云', source_value: '运行中', target_value: '运行中', status: 'matched', last_check: '2024-01-15 10:00' },
  { id: 2, ci_name: 'MySQL主库', ci_type: '数据库', source: 'CMDB', target: '阿里云', source_value: '运行中', target_value: '运行中', status: 'matched', last_check: '2024-01-15 10:00' },
  { id: 3, ci_name: 'Redis缓存', ci_type: '缓存', source: 'CMDB', target: '阿里云', source_value: '运行中', target_value: '已停止', status: 'mismatch', last_check: '2024-01-15 10:00' },
  { id: 4, ci_name: '文件存储桶', ci_type: '存储', source: 'CMDB', target: '阿里云', source_value: '存在', target_value: '存在', status: 'matched', last_check: '2024-01-15 10:00' },
  { id: 5, ci_name: '测试服务器', ci_type: '服务器', source: 'CMDB', target: 'VMware', source_value: '运行中', target_value: '不存在', status: 'missing', last_check: '2024-01-15 10:00' },
];

const providerColors: Record<string, string> = {
  '阿里云': 'blue',
  '腾讯云': 'red',
  '华为云': 'orange',
  'AWS': 'yellow',
};

const resourceStatusColors: Record<string, string> = {
  'running': 'green',
  'stopped': 'red',
  'maintenance': 'orange',
};

const reconciliationStatusColors: Record<string, string> = {
  'matched': 'green',
  'mismatch': 'red',
  'missing': 'orange',
  'extra': 'blue',
};

export default function CMDBPage() {
  const router = useRouter();
  const { message } = App.useApp();
  const [activeTab, setActiveTab] = useState('cis');
  const [stats, setStats] = useState({
    totalCIs: 0,
    online: 0,
    offline: 0,
    maintenance: 0,
  });

  // Fetch stats
  const fetchStats = async () => {
    try {
      // Mock data - in real app would call API
      setStats({
        totalCIs: 220,
        online: 195,
        offline: 15,
        maintenance: 10,
      });
    } catch (error) {
      console.error('Failed to fetch CMDB stats:', error);
    }
  };

  useEffect(() => {
    fetchStats();
  }, []);

  const ciTypeColumns = [
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string, record: any) => (
        <Tag icon={record.type === '服务器' ? <Server /> : record.type === '云资源' ? <Cloud /> : <Database />}>
          {type}
        </Tag>
      ),
    },
    {
      title: '数量',
      dataIndex: 'count',
      key: 'count',
      render: (count: number) => <strong>{count}</strong>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'online' ? 'green' : status === 'offline' ? 'red' : 'blue'}>
          {status === 'online' ? '在线' : status === 'offline' ? '离线' : '活跃'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Button size="small" type="link">查看</Button>
      ),
    },
  ];

  const cloudResourceColumns = [
    {
      title: '资源ID',
      dataIndex: 'resource_id',
      key: 'resource_id',
      render: (text: string) => <code className="text-xs">{text}</code>,
    },
    {
      title: '资源名称',
      dataIndex: 'resource_name',
      key: 'resource_name',
      render: (text: string) => <a>{text}</a>,
    },
    {
      title: '类型',
      dataIndex: 'resource_type',
      key: 'resource_type',
      render: (type: string) => <Tag>{type}</Tag>,
    },
    {
      title: '供应商',
      dataIndex: 'provider',
      key: 'provider',
      render: (provider: string) => <Tag color={providerColors[provider]}>{provider}</Tag>,
    },
    {
      title: '地域',
      dataIndex: 'region',
      key: 'region',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={resourceStatusColors[status]}>{status === 'running' ? '运行中' : '已停止'}</Tag>
      ),
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
      render: (ip: string) => ip === '-' ? <span className="text-gray-400">-</span> : <code>{ip}</code>,
    },
    {
      title: '创建时间',
      dataIndex: 'create_time',
      key: 'create_time',
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Space>
          <Button size="small" type="link">详情</Button>
          <Button size="small" type="link">同步</Button>
        </Space>
      ),
    },
  ];

  const reconciliationColumns = [
    {
      title: '配置项',
      dataIndex: 'ci_name',
      key: 'ci_name',
      render: (text: string, record: any) => (
        <div>
          <a>{text}</a>
          <div className="text-xs text-gray-400">{record.ci_type}</div>
        </div>
      ),
    },
    {
      title: '数据源',
      key: 'sources',
      render: (_: any, record: any) => (
        <div className="flex items-center gap-2">
          <Tag>{record.source}</Tag>
          <span>→</span>
          <Tag>{record.target}</Tag>
        </div>
      ),
    },
    {
      title: 'CMDB值',
      dataIndex: 'source_value',
      key: 'source_value',
    },
    {
      title: '目标值',
      dataIndex: 'target_value',
      key: 'target_value',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={reconciliationStatusColors[status]}>
          {status === 'matched' ? '匹配' : status === 'mismatch' ? '不匹配' : status === 'missing' ? '缺失' : '多余'}
        </Tag>
      ),
    },
    {
      title: '最后检查',
      dataIndex: 'last_check',
      key: 'last_check',
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Space>
          <Button size="small" type="link">同步</Button>
          <Button size="small" type="link">忽略</Button>
        </Space>
      ),
    },
  ];

  // Calculate reconciliation summary
  const reconciliationSummary = React.useMemo(() => {
    const total = mockReconciliationData.length;
    const matched = mockReconciliationData.filter(r => r.status === 'matched').length;
    const mismatch = mockReconciliationData.filter(r => r.status === 'mismatch').length;
    const missing = mockReconciliationData.filter(r => r.status === 'missing').length;
    return { total, matched, mismatch, missing };
  }, []);

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      {/* 页面头部 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <Title level={2} style={{ marginBottom: 4 }}>
            配置管理数据库 (CMDB)
          </Title>
          <Text type="secondary">
            管理IT基础设施的配置项和关系
          </Text>
        </div>
        <Space>
          <Button
            icon={<SyncOutlined />}
            loading={false}
            onClick={() => {
              message.loading({ content: '正在同步云资源...', key: 'sync' });
              setTimeout(() => {
                message.success({ content: '云资源同步完成', key: 'sync' });
              }, 2000);
            }}
          >
            同步云资源
          </Button>
          <Button
            type="primary"
            icon={<Plus className="w-4 h-4" />}
            onClick={() => router.push('/cmdb/cis/create')}
          >
            新增配置项
          </Button>
        </Space>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="配置项总数"
              value={stats.totalCIs}
              prefix={<Database className="text-blue-500 mr-2" />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="在线"
              value={stats.online}
              prefix={<CheckCircle className="text-green-500 mr-2" />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="离线"
              value={stats.offline}
              prefix={<AlertTriangle className="text-red-500 mr-2" />}
              valueStyle={{ color: '#ff4d4f' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm" variant="borderless">
            <Statistic
              title="维护中"
              value={stats.maintenance}
              prefix={<Server className="text-orange-500 mr-2" />}
              valueStyle={{ color: '#fa8c16' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 主要内容 */}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'cis',
              label: (
                <span className="flex items-center gap-2">
                  <Database />
                  配置项
                </span>
              ),
              children: <CIList />,
            },
            {
              key: 'types',
              label: (
                <span className="flex items-center gap-2">
                  <Server />
                  类型分布
                </span>
              ),
              children: (
                <Table
                  columns={ciTypeColumns}
                  dataSource={ciTypeStats}
                  rowKey="type"
                  pagination={false}
                />
              ),
            },
            {
              key: 'cloud',
              label: (
                <span className="flex items-center gap-2">
                  <Cloud />
                  云资源 ({mockCloudResources.length})
                </span>
              ),
              children: (
                <div>
                  {/* 云资源统计 */}
                  <Row gutter={16} className="mb-4">
                    <Col span={6}>
                      <Card size="small">
                        <Statistic title="ECS" value={mockCloudResources.filter(r => r.resource_type === 'ECS').length} prefix={<Server />} />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic title="RDS" value={mockCloudResources.filter(r => r.resource_type === 'RDS').length} prefix={<Database />} />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic title="OSS" value={mockCloudResources.filter(r => r.resource_type === 'OSS').length} prefix={<Cloud />} />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic title="SLB" value={mockCloudResources.filter(r => r.resource_type === 'SLB').length} prefix={<Shield />} />
                      </Card>
                    </Col>
                  </Row>

                  {/* 筛选栏 */}
                  <div className="mb-4 flex gap-2">
                    <Input placeholder="搜索资源名称" prefix={<SearchOutlined />} style={{ width: 200 }} />
                    <Select placeholder="资源类型" style={{ width: 120 }} allowClear options={[
                      { value: 'ECS', label: 'ECS' },
                      { value: 'RDS', label: 'RDS' },
                      { value: 'OSS', label: 'OSS' },
                      { value: 'SLB', label: 'SLB' },
                    ]} />
                    <Select placeholder="状态" style={{ width: 100 }} allowClear options={[
                      { value: 'running', label: '运行中' },
                      { value: 'stopped', label: '已停止' },
                    ]} />
                    <Button icon={<FilterOutlined />}>筛选</Button>
                  </div>

                  <Table
                    columns={cloudResourceColumns}
                    dataSource={mockCloudResources}
                    rowKey="id"
                    pagination={{ pageSize: 10 }}
                  />
                </div>
              ),
            },
            {
              key: 'reconciliation',
              label: (
                <span className="flex items-center gap-2">
                  <Shield />
                  核对 ({reconciliationSummary.total})
                </span>
              ),
              children: (
                <div>
                  {/* 核对统计 */}
                  <Row gutter={16} className="mb-4">
                    <Col span={6}>
                      <Card size="small" className="bg-green-50">
                        <Statistic
                          title="匹配"
                          value={reconciliationSummary.matched}
                          prefix={<CheckCircle className="text-green-500" />}
                          valueStyle={{ color: '#52c41a' }}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small" className="bg-red-50">
                        <Statistic
                          title="不匹配"
                          value={reconciliationSummary.mismatch}
                          prefix={<AlertTriangle className="text-red-500" />}
                          valueStyle={{ color: '#ff4d4f' }}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small" className="bg-orange-50">
                        <Statistic
                          title="缺失"
                          value={reconciliationSummary.missing}
                          prefix={<AlertTriangle className="text-orange-500" />}
                          valueStyle={{ color: '#fa8c16' }}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="匹配率"
                          value={(reconciliationSummary.matched / reconciliationSummary.total * 100).toFixed(1)}
                          suffix="%"
                          prefix={<CheckCircle />}
                        />
                      </Card>
                    </Col>
                  </Row>

                  {/* 核对操作栏 */}
                  <div className="mb-4 flex justify-between">
                    <Space>
                      <Button type="primary" icon={<SyncOutlined />}>执行全量核对</Button>
                      <Button icon={<SyncOutlined />}>增量同步</Button>
                    </Space>
                    <Text type="secondary">最后同步: 2024-01-15 10:00</Text>
                  </div>

                  <Table
                    columns={reconciliationColumns}
                    dataSource={mockReconciliationData}
                    rowKey="id"
                    pagination={{ pageSize: 10 }}
                  />
                </div>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}
