'use client';

import React from 'react';
import { Table, Tag, Space, Button, Badge, Card } from 'antd';
import {
  Eye,
  Edit,
  MoreHorizontal,
  Network,
  Server,
  Database,
  HardDrive,
} from 'lucide-react';
import {
  ClusterOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  DesktopOutlined,
  HddOutlined,
} from '@ant-design/icons';
import { useI18n } from '@/lib/i18n';

const getCiTypeIcon = (type: string) => {
  switch (type) {
    case 'Cloud Server':
      return <CloudServerOutlined />;
    case 'Physical Server':
      return <DesktopOutlined />;
    case 'Relational Database':
      return <DatabaseOutlined />;
    case 'Storage Device':
      return <HddOutlined />;
    default:
      return <ClusterOutlined />;
  }
};

const getCiTypeColor = (type: string) => {
  switch (type) {
    case 'Cloud Server':
      return 'blue';
    case 'Physical Server':
      return 'purple';
    case 'Relational Database':
      return 'green';
    case 'Storage Device':
      return 'orange';
    default:
      return 'default';
  }
};

const getStatusConfig = (status: string) => {
  switch (status) {
    case 'Running':
      return {
        color: '#52c41a',
        text: '运行中',
        backgroundColor: '#f6ffed',
      };
    case 'Maintenance':
      return {
        color: '#fa8c16',
        text: '维护中',
        backgroundColor: '#fff7e6',
      };
    case 'Disabled':
      return {
        color: '#00000073',
        text: '已停用',
        backgroundColor: '#fafafa',
      };
    default:
      return {
        color: '#00000073',
        text: status,
        backgroundColor: '#fafafa',
      };
  }
};

interface CIListProps {
  cis: any[];
  loading: boolean;
  selectedRowKeys: React.Key[];
  onSelectedRowKeysChange: (keys: React.Key[]) => void;
  onViewRelations: (ci: any) => void;
}

export const CIList: React.FC<CIListProps> = ({
  cis,
  loading,
  selectedRowKeys,
  onSelectedRowKeysChange,
  onViewRelations,
}) => {
    const { t } = useI18n();
  const columns = [
    {
      title: t('cmdb.info'),
      key: 'ci_info',
      width: 300,
      render: (_: unknown, record: any) => (
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <div
            style={{
              width: 40,
              height: 40,
              backgroundColor: '#e6f7ff',
              borderRadius: 8,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginRight: 12,
              color: '#1890ff',
              fontSize: 18,
            }}
          >
            {getCiTypeIcon(record.type)}
          </div>
          <div>
            <div style={{ fontWeight: 'medium', color: '#000', marginBottom: 4 }}>
              {record.name}
            </div>
            <div style={{ fontSize: 'small', color: '#666' }}>
              {record.id} • {record.ip}
            </div>
          </div>
        </div>
      ),
    },
    {
      title: t('cmdb.type'),
      dataIndex: 'type',
      key: 'type',
      width: 120,
      render: (type: string) => (
        <Tag color={getCiTypeColor(type)} icon={getCiTypeIcon(type)}>
          {type}
        </Tag>
      ),
    },
    {
      title: t('cmdb.status'),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => {
        const config = getStatusConfig(status);
        return (
          <span
            style={{
              padding: '4px 12px',
              borderRadius: 16,
              fontSize: 'small',
              fontWeight: 500,
              color: config.color,
              backgroundColor: config.backgroundColor,
            }}
          >
            {config.text}
          </span>
        );
      },
    },
    {
      title: t('cmdb.business'),
      dataIndex: 'business',
      key: 'business',
      width: 150,
    },
    {
      title: t('cmdb.owner'),
      dataIndex: 'owner',
      key: 'owner',
      width: 120,
    },
    {
      title: t('cmdb.location'),
      dataIndex: 'location',
      key: 'location',
      width: 150,
    },
    {
      title: t('cmdb.config'),
      key: 'config',
      width: 150,
      render: (_: unknown, record: any) => (
        <div style={{ fontSize: 'small' }}>
          <div>
            {record.cpu} / {record.memory}
          </div>
          <div style={{ color: '#666' }}>{record.disk}</div>
        </div>
      ),
    },
    {
      title: t('cmdb.actions'),
      key: 'actions',
      width: 150,
      render: (_: unknown, record: any) => (
        <Space size='small'>
          <Button
            type='text'
            size='small'
            icon={<Eye size={16} />}
            onClick={() => window.open(`/cmdb/${record.id}`)}
          />
          <Button
            type='text'
            size='small'
            icon={<Edit size={16} />}
            onClick={() => window.open(`/cmdb/${record.id}/edit`)}
          />
          <Button
            type='text'
            size='small'
            icon={<ClusterOutlined />}
            onClick={() => onViewRelations(record)}
          />
          <Button type='text' size='small' icon={<MoreHorizontal size={16} />} />
        </Space>
      ),
    },
  ];

  return (
    <Card style={{ borderRadius: 16 }} styles={{ body: { padding: 0 } }}>
      <div
        style={{
          padding: '24px 24px 0 24px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <h3 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>{t('cmdb.ciList')}</h3>
          {selectedRowKeys.length > 0 && (
            <Badge count={selectedRowKeys.length} showZero style={{ backgroundColor: '#667eea' }} />
          )}
        </div>
        <div style={{ fontSize: 14, color: '#666' }}>共 {cis.length} 个配置项</div>
      </div>

      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: onSelectedRowKeysChange,
        }}
        columns={columns}
        dataSource={cis}
        rowKey='id'
        loading={loading}
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
        }}
        scroll={{ x: 1200 }}
      />
    </Card>
  );
};
