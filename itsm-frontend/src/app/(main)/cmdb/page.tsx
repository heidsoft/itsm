'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Statistic,
  Typography,
  Tabs,
  Table,
  Tag,
  Button,
  Progress,
  Space,
  Input,
  Select,
  App,
  Spin,
  Tooltip,
} from 'antd';
import {
  Database,
  Server,
  Cloud,
  Shield,
  Plus,
  AlertTriangle,
  CheckCircle,
  Search,
  Filter,
  RefreshCw,
  Sparkles,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import CIList from '@/components/cmdb/CIList';
import { FilterToolbarCard } from '@/components/ui/FilterToolbarCard';
import { LoadingEmptyError } from '@/components/ui/LoadingEmptyError';
import { ManagementNotice, ManagementPageHeader } from '@/components/ui/ManagementPageHeader';
import { StatsOverview } from '@/components/ui/StatsOverview';
import { CMDBApi } from '@/lib/api/cmdb-api';
import { httpClient } from '@/lib/api/http-client';

const { Text } = Typography;

const providerColors: Record<string, string> = {
  阿里云: 'blue',
  腾讯云: 'red',
  华为云: 'orange',
  AWS: 'yellow',
};

const resourceStatusColors: Record<string, string> = {
  running: 'green',
  stopped: 'red',
  maintenance: 'orange',
};

const reconciliationStatusColors: Record<string, string> = {
  matched: 'green',
  mismatch: 'red',
  missing: 'orange',
  extra: 'blue',
};

export default function CMDBPage() {
  const router = useRouter();
  const { message } = App.useApp();
  const [activeTab, setActiveTab] = useState('cis');
  const [loading, setLoading] = useState(false);
  const [stats, setStats] = useState({
    totalCIs: 0,
    online: 0,
    offline: 0,
    maintenance: 0,
  });
  const [ciTypeStats, setCiTypeStats] = useState<any[]>([]);
  const [cloudResources, setCloudResources] = useState<any[]>([]);
  const [reconciliationData, setReconciliationData] = useState<any[]>([]);
  // 云资源筛选状态
  const [cloudSearchKeyword, setCloudSearchKeyword] = useState('');
  const [cloudTypeFilter, setCloudTypeFilter] = useState<string | undefined>(undefined);
  const [cloudStatusFilter, setCloudStatusFilter] = useState<string | undefined>(undefined);
  // 云资源同步状态
  const [cloudSyncStatus, setCloudSyncStatus] = useState<{
    total_accounts: number;
    active_accounts: number;
    discovered_count: number;
    last_discovery: string;
  } | null>(null);
  const [syncLoading, setSyncLoading] = useState(false);

  // Fetch all data
  const fetchStats = async () => {
    try {
      const statsData = await CMDBApi.getCMDBStats();
      // Backend returns: { total_count, active_count, inactive_count, maintenance_count, ... }
      const d = (statsData as any)?.data ?? statsData;
      setStats({
        totalCIs: d?.total_count ?? d?.totalCIs ?? d?.total_cis ?? 0,
        online: d?.active_count ?? d?.online ?? 0,
        offline: d?.inactive_count ?? d?.offline ?? 0,
        maintenance: d?.maintenance_count ?? d?.maintenance ?? 0,
      });
    } catch (error) {
      console.error('Failed to fetch CMDB stats:', error);
    }
  };

  const fetchCiTypeStats = async () => {
    try {
      const typesData = await CMDBApi.getCITypes();
      // typesData is CIType[] - ensure each item has a unique id for React keys
      const typesWithIds = (typesData || []).map((item: any, index: number) => ({
        ...item,
        id: item.id ?? item.type ?? `type-${index}`,
      }));
      setCiTypeStats(typesWithIds);
    } catch (error) {
      console.error('Failed to fetch CI types:', error);
    }
  };

  // 获取云资源同步状态
  const fetchCloudSyncStatus = async () => {
    try {
      // 模拟获取同步状态 - 后端需要实现实际API
      const response = await httpClient.get<any>('/api/v1/configuration-items/discovery/status');
      if (response) {
        setCloudSyncStatus(response);
      }
    } catch (error) {
      console.error('Failed to fetch cloud sync status:', error);
    }
  };

  // 触发云资源发现
  const triggerCloudDiscovery = async () => {
    setSyncLoading(true);
    try {
      await httpClient.post('/api/v1/configuration-items/discovery/run', {});
      message.success('云资源发现任务已启动');
      // 延迟刷新状态
      setTimeout(() => fetchCloudSyncStatus(), 3000);
    } catch (error) {
      message.error('启动发现任务失败');
    } finally {
      setSyncLoading(false);
    }
  };

  const fetchCloudResources = async () => {
    try {
      const resourcesData = await CMDBApi.getCloudResources();
      if (resourcesData.data && Array.isArray(resourcesData.data)) {
        setCloudResources(resourcesData.data);
      } else if (Array.isArray(resourcesData)) {
        setCloudResources(resourcesData);
      }
    } catch (error) {
      console.error('Failed to fetch cloud resources:', error);
    }
  };

  const fetchReconciliationData = async () => {
    try {
      const result = await CMDBApi.getReconciliationResults();
      // Backend returns { summary, unbound_resources, orphan_cis, unlinked_cis }
      // Flatten into a display-friendly array for the table
      const d = (result as any)?.data ?? result;
      if (d && typeof d === 'object' && !Array.isArray(d)) {
        const rows: any[] = [];
        (d.unbound_resources ?? []).forEach((r: any) => rows.push({ ...r, _kind: 'unbound' }));
        (d.orphan_cis ?? []).forEach((r: any) => rows.push({ ...r, _kind: 'orphan' }));
        (d.unlinked_cis ?? []).forEach((r: any) => rows.push({ ...r, _kind: 'unlinked' }));
        setReconciliationData(rows);
      } else if (Array.isArray(d)) {
        setReconciliationData(d);
      }
    } catch (error) {
      console.error('Failed to fetch reconciliation data:', error);
    }
  };

  useEffect(() => {
    fetchStats();
    fetchCiTypeStats();
    fetchCloudResources();
    fetchReconciliationData();
    fetchCloudSyncStatus();
  }, []);

  const ciTypeColumns = [
    {
      title: '类型',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: any) => (
        <Tag
          icon={
            record.name === '服务器' ? (
              <Server />
            ) : record.name === '云资源' ? (
              <Cloud />
            ) : (
              <Database />
            )
          }
        >
          {name}
        </Tag>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      render: (description: string) => description || '-',
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (isActive: boolean) => (
        <Tag color={isActive ? 'green' : 'red'}>{isActive ? '启用' : '禁用'}</Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: () => (
        <Button size="small" type="link">
          查看
        </Button>
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
      render: (ip: string) =>
        ip === '-' ? <span className="text-gray-400">-</span> : <code>{ip}</code>,
    },
    {
      title: '创建时间',
      dataIndex: 'create_time',
      key: 'create_time',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: any) => (
        <Space>
          <Button size="small" type="link" onClick={() => router.push(`/cmdb/cis/${record.id}`)}>
            详情
          </Button>
          <Button
            size="small"
            type="link"
            onClick={async () => {
              try {
                await CMDBApi.runReconciliation();
                message.success(`资源 ${record.resource_id} 同步已触发`);
              } catch (e) {
                message.error('同步失败');
              }
            }}
          >
            同步
          </Button>
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
          <a onClick={() => router.push(`/cmdb/cis/${record.id}`)}>{text}</a>
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
          {status === 'matched'
            ? '匹配'
            : status === 'mismatch'
              ? '不匹配'
              : status === 'missing'
                ? '缺失'
                : '多余'}
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
      render: (_: any, record: any) => (
        <Space>
          <Button
            size="small"
            type="link"
            onClick={async () => {
              try {
                await CMDBApi.runReconciliation();
                message.success('同步已触发，请稍后刷新查看');
              } catch (e) {
                message.error('同步失败');
              }
            }}
          >
            同步
          </Button>
          <Tooltip title="忽略功能需要后端支持，暂未实现">
            <Button
              size="small"
              type="link"
              disabled
            >
              忽略
            </Button>
          </Tooltip>
        </Space>
      ),
    },
  ];

  // Calculate reconciliation summary
  const reconciliationSummary = React.useMemo(() => {
    const total = reconciliationData.length;
    const matched = reconciliationData.filter(r => r.status === 'matched').length;
    const mismatch = reconciliationData.filter(r => r.status === 'mismatch').length;
    const missing = reconciliationData.filter(r => r.status === 'missing').length;
    return { total, matched, mismatch, missing };
  }, [reconciliationData]);

  // 云资源筛选后的数据
  const filteredCloudResources = React.useMemo(() => {
    return cloudResources.filter(resource => {
      const matchesSearch =
        !cloudSearchKeyword ||
        resource.resource_name?.toLowerCase().includes(cloudSearchKeyword.toLowerCase()) ||
        resource.resource_id?.toLowerCase().includes(cloudSearchKeyword.toLowerCase());
      const matchesType = !cloudTypeFilter || resource.resource_type === cloudTypeFilter;
      const matchesStatus = !cloudStatusFilter || resource.status === cloudStatusFilter;
      return matchesSearch && matchesType && matchesStatus;
    });
  }, [cloudResources, cloudSearchKeyword, cloudTypeFilter, cloudStatusFilter]);

  // 云资源筛选重置
  const handleResetCloudFilters = () => {
    setCloudSearchKeyword('');
    setCloudTypeFilter(undefined);
    setCloudStatusFilter(undefined);
  };

  const handleSyncCloudResources = async () => {
    message.loading({ content: '正在同步云资源...', key: 'sync' });
    setLoading(true);
    try {
      await fetchCloudResources();
      message.success({ content: '云资源同步完成', key: 'sync' });
    } catch (error) {
      message.error({ content: '同步失败', key: 'sync' });
    } finally {
      setLoading(false);
    }
  };

  const headerActions = (
    <>
      <Button icon={<RefreshCw className="w-4 h-4" />} loading={loading} onClick={handleSyncCloudResources}>
        同步云资源
      </Button>
      <Button type="primary" icon={<Plus className="w-4 h-4" />} onClick={() => router.push('/cmdb/cis/create')}>
        新增配置项
      </Button>
    </>
  );

  const statsItems = [
    {
      key: 'total',
      title: '配置项总数',
      value: stats.totalCIs,
      prefix: <Database className="text-blue-500 mr-2" />,
      accentColor: '#1890ff',
    },
    {
      key: 'online',
      title: '在线',
      value: stats.online,
      prefix: <CheckCircle className="text-green-500 mr-2" />,
      accentColor: '#52c41a',
    },
    {
      key: 'offline',
      title: '离线',
      value: stats.offline,
      prefix: <AlertTriangle className="text-red-500 mr-2" />,
      accentColor: '#ff4d4f',
    },
    {
      key: 'maintenance',
      title: '维护中',
      value: stats.maintenance,
      prefix: <Server className="text-orange-500 mr-2" />,
      accentColor: '#fa8c16',
    },
  ];

  return (
    <div className="p-6 min-h-screen bg-gray-50">
      <ManagementPageHeader
        title="配置管理数据库 (CMDB)"
        description="管理配置项、云资源同步、关系拓扑和核对结果。"
        actions={headerActions}
        notice={
          <ManagementNotice
            message="复杂域页面开始统一收口"
            description="总览、云资源、核对和配置项详情将逐步复用同一套页面头部、统计卡和筛选栏基线。"
          />
        }
      />

      <StatsOverview items={statsItems} className="mb-6" />

      {/* 云资源同步状态卡片 */}
      <Card
        className="mb-6"
        title={
          <span className="flex items-center gap-2">
            <Cloud className="w-4 h-4" />
            云资源同步状态
          </span>
        }
        extra={
          <Button
            type="primary"
            icon={<Sparkles className="w-4 h-4" />}
            onClick={triggerCloudDiscovery}
            loading={syncLoading}
            size="small"
          >
            立即同步
          </Button>
        }
      >
        <Spin spinning={syncLoading}>
          <Row gutter={[16, 16]}>
            <Col xs={24} sm={8}>
              <Statistic
                title="云账号总数"
                value={cloudSyncStatus?.total_accounts ?? 0}
                prefix={<Cloud />}
              />
            </Col>
            <Col xs={24} sm={8}>
              <Statistic
                title="活跃账号"
                value={cloudSyncStatus?.active_accounts ?? 0}
                prefix={<CheckCircle className="text-green-500" />}
                styles={{ content: { color: '#52c41a' } }}
              />
            </Col>
            <Col xs={24} sm={8}>
              <Statistic
                title="已发现资源"
                value={cloudSyncStatus?.discovered_count ?? 0}
                prefix={<Server />}
                styles={{ content: { color: '#1890ff' } }}
              />
            </Col>
          </Row>
          {cloudSyncStatus?.last_discovery && (
            <div className="mt-4 text-sm text-gray-500">
              上次同步时间: {new Date(cloudSyncStatus.last_discovery).toLocaleString()}
            </div>
          )}
        </Spin>
      </Card>

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
                  rowKey="id"
                  pagination={false}
                />
              ),
            },
            {
              key: 'cloud',
              label: (
                <span className="flex items-center gap-2">
                  <Cloud />
                  云资源 ({cloudResources.length})
                </span>
              ),
              children: (
                <div>
                  {/* 云资源统计 */}
                  <Row gutter={16} className="mb-4">
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="ECS"
                          value={cloudResources.filter(r => r.resource_type === 'ECS').length}
                          prefix={<Server />}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="RDS"
                          value={cloudResources.filter(r => r.resource_type === 'RDS').length}
                          prefix={<Database />}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="OSS"
                          value={cloudResources.filter(r => r.resource_type === 'OSS').length}
                          prefix={<Cloud />}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="SLB"
                          value={cloudResources.filter(r => r.resource_type === 'SLB').length}
                          prefix={<Shield />}
                        />
                      </Card>
                    </Col>
                  </Row>

                  {/* 筛选栏 */}
                  <FilterToolbarCard
                    className="mb-4"
                    filters={
                      <>
                        <Input
                          placeholder="搜索资源名称"
                          prefix={<Search className="w-4 h-4" />}
                          className="min-w-[220px]"
                          value={cloudSearchKeyword}
                          onChange={e => setCloudSearchKeyword(e.target.value)}
                          allowClear
                        />
                        <Select
                          placeholder="资源类型"
                          className="min-w-[120px]"
                          allowClear
                          value={cloudTypeFilter}
                          onChange={setCloudTypeFilter}
                          options={[
                            { value: 'ECS', label: 'ECS' },
                            { value: 'RDS', label: 'RDS' },
                            { value: 'OSS', label: 'OSS' },
                            { value: 'SLB', label: 'SLB' },
                          ]}
                        />
                        <Select
                          placeholder="状态"
                          className="min-w-[120px]"
                          allowClear
                          value={cloudStatusFilter}
                          onChange={setCloudStatusFilter}
                          options={[
                            { value: 'running', label: '运行中' },
                            { value: 'stopped', label: '已停止' },
                          ]}
                        />
                      </>
                    }
                    actions={
                      <>
                        <Text type="secondary">筛选后 {filteredCloudResources.length} 条</Text>
                        <Button icon={<Filter className="w-4 h-4" />} onClick={handleResetCloudFilters}>
                          重置
                        </Button>
                      </>
                    }
                  />

                  {filteredCloudResources.length === 0 ? (
                    <LoadingEmptyError
                      state="empty"
                      empty={{
                        title: '暂无云资源',
                        description: '可以先同步云资源，或调整当前筛选条件。',
                        actionText: '立即同步',
                        onAction: triggerCloudDiscovery,
                      }}
                    />
                  ) : (
                    <Table
                      columns={cloudResourceColumns}
                      dataSource={filteredCloudResources}
                      rowKey="id"
                      pagination={{ pageSize: 10 }}
                    />
                  )}
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
                          styles={{ content: { color: '#52c41a' } }}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small" className="bg-red-50">
                        <Statistic
                          title="不匹配"
                          value={reconciliationSummary.mismatch}
                          prefix={<AlertTriangle className="text-red-500" />}
                          styles={{ content: { color: '#ff4d4f' } }}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small" className="bg-orange-50">
                        <Statistic
                          title="缺失"
                          value={reconciliationSummary.missing}
                          prefix={<AlertTriangle className="text-orange-500" />}
                          styles={{ content: { color: '#fa8c16' } }}
                        />
                      </Card>
                    </Col>
                    <Col span={6}>
                      <Card size="small">
                        <Statistic
                          title="匹配率"
                          value={
                            reconciliationSummary.total > 0
                              ? (
                                  (reconciliationSummary.matched / reconciliationSummary.total) *
                                  100
                                ).toFixed(1)
                              : '0'
                          }
                          suffix="%"
                          prefix={<CheckCircle />}
                        />
                      </Card>
                    </Col>
                  </Row>

                  {/* 核对操作栏 */}
                  <div className="mb-4 flex justify-between">
                    <Space>
                      <Button
                        type="primary"
                        icon={<RefreshCw className="w-4 h-4" />}
                        onClick={async () => {
                          message.loading({ content: '正在执行全量核对...', key: 'reconcile' });
                          try {
                            await CMDBApi.runReconciliation();
                            await fetchReconciliationData();
                            message.success({ content: '核对完成', key: 'reconcile' });
                          } catch (error) {
                            message.error({ content: '核对失败', key: 'reconcile' });
                          }
                        }}
                      >
                        执行全量核对
                      </Button>
                      <Button icon={<RefreshCw className="w-4 h-4" />}>增量同步</Button>
                    </Space>
                    <Text type="secondary">最后同步: {new Date().toLocaleString('zh-CN')}</Text>
                  </div>

                  {reconciliationData.length === 0 ? (
                    <LoadingEmptyError
                      state="empty"
                      empty={{
                        title: '暂无核对结果',
                        description: '执行全量核对后，这里会展示匹配、缺失和不一致项。',
                        actionText: '执行全量核对',
                        onAction: async () => {
                          await CMDBApi.runReconciliation();
                          await fetchReconciliationData();
                        },
                      }}
                    />
                  ) : (
                    <Table
                      columns={reconciliationColumns}
                      dataSource={reconciliationData}
                      rowKey="id"
                      pagination={{ pageSize: 10 }}
                    />
                  )}
                </div>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}
