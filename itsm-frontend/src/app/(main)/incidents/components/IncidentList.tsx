'use client';

import React from 'react';
import { Table, Tag, Button, Space, Badge } from 'antd';
import { Eye, Edit, MoreHorizontal, AlertTriangle } from 'lucide-react';
import { Incident } from '@/lib/api/incident-api';
import { useI18n } from '@/lib/i18n';

interface IncidentListProps {
  incidents: Incident[];
  loading: boolean;
  selectedRowKeys: React.Key[];
  onSelectedRowKeysChange: (keys: React.Key[]) => void;
  onEdit: (incident: Incident) => void;
  onView: (incident: Incident) => void;
}

export const IncidentList: React.FC<IncidentListProps> = ({
  incidents,
  loading,
  selectedRowKeys,
  onSelectedRowKeysChange,
  onEdit,
  onView,
}) => {
  const { t } = useI18n();
  
  const statusConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
    open: {
      color: '#fa8c16',
      text: t('incidents.statusOpen'),
      backgroundColor: '#fff7e6',
    },
    'in-progress': {
      color: '#1890ff',
      text: t('incidents.statusInProgress'),
      backgroundColor: '#e6f7ff',
    },
    resolved: {
      color: '#52c41a',
      text: t('incidents.statusResolved'),
      backgroundColor: '#f6ffed',
    },
    closed: {
      color: '#00000073',
      text: t('incidents.statusClosed'),
      backgroundColor: '#fafafa',
    },
  };

  const priorityConfig: Record<string, { color: string; text: string; backgroundColor: string }> =
    {
      low: {
        color: '#52c41a',
        text: t('incidents.priorityLow'),
        backgroundColor: '#f6ffed',
      },
      medium: {
        color: '#1890ff',
        text: t('incidents.priorityMedium'),
        backgroundColor: '#e6f7ff',
      },
      high: {
        color: '#fa8c16',
        text: t('incidents.priorityHigh'),
        backgroundColor: '#fff7e6',
      },
      critical: {
        color: '#ff4d4f',
        text: t('incidents.priorityCritical'),
        backgroundColor: '#fff2f0',
      },
    };

  const columns = [
    {
      title: t('incidents.incidentInfo'),
      key: 'incident_info',
      width: 300,
      render: (_: unknown, record: Incident) => (
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
            }}
          >
            <AlertTriangle size={20} style={{ color: '#1890ff' }} />
          </div>
          <div>
            <div style={{ fontWeight: 'medium', color: '#000', marginBottom: 4 }}>
              {record.title}
            </div>
            <div style={{ fontSize: 'small', color: '#666' }}>
              #{record.incident_number} • {record.category}
            </div>
          </div>
        </div>
      ),
    },
    {
      title: t('incidents.status'),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: string) => {
        const config = statusConfig[status] || {
          color: '#666',
          text: status || t('incidents.unknown'),
          backgroundColor: '#f5f5f5',
        };
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
      title: t('incidents.priority'),
      dataIndex: 'priority',
      key: 'priority',
      width: 100,
      render: (priority: string) => {
        const config = priorityConfig[priority] || {
          color: '#666',
          text: priority || t('incidents.unknown'),
          backgroundColor: '#f5f5f5',
        };
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
      title: t('incidents.impact'),
      dataIndex: 'impact',
      key: 'impact',
      width: 120,
      render: (impact: string) => {
        const impactConfig: Record<string, { color: string; text: string }> = {
          low: { color: 'green', text: t('incidents.impactLow') },
          medium: { color: 'orange', text: t('incidents.impactMedium') },
          high: { color: 'red', text: t('incidents.impactHigh') },
        };
        const config = impactConfig[impact] || {
          color: 'default',
          text: impact || t('incidents.unknown'),
        };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: t('incidents.reporter'),
      dataIndex: 'reporter',
      key: 'reporter',
      width: 150,
      render: (reporter: { name: string }) => (
        <div style={{ fontSize: 'small' }}>{reporter?.name || t('incidents.unknown')}</div>
      ),
    },
    {
      title: t('incidents.createdAt'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (created_at: string) => (
        <div style={{ fontSize: 'small', color: '#666' }}>
          {new Date(created_at).toLocaleDateString()}
        </div>
      ),
    },
    {
      title: t('incidents.operations'),
      key: 'actions',
      width: 150,
      render: (_: unknown, record: Incident) => (
        <Space size='small'>
          <Button
            type='text'
            size='small'
            icon={<Eye size={16} />}
            onClick={() => onView(record)}
            className='text-blue-600 hover:text-blue-700 hover:bg-blue-50 border-0 rounded-lg transition-all duration-200 p-2'
            title={t('incidents.viewDetails')}
          />
          <Button
            type='text'
            size='small'
            icon={<Edit size={16} />}
            onClick={() => onEdit(record)}
            className='text-green-600 hover:text-green-700 hover:bg-green-50 border-0 rounded-lg transition-all duration-200 p-2'
            title={t('incidents.editIncident')}
          />
          <Button
            type='text'
            size='small'
            icon={<MoreHorizontal size={16} />}
            className='text-gray-600 hover:text-gray-700 hover:bg-gray-50 border-0 rounded-lg transition-all duration-200 p-2'
            title={t('incidents.moreActions')}
          />
        </Space>
      ),
    },
  ];

  return (
    <div className='bg-white rounded-2xl border border-gray-200 shadow-lg overflow-hidden'>
      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: onSelectedRowKeysChange,
        }}
        columns={columns}
        dataSource={incidents}
        rowKey='id'
        loading={loading}
        pagination={false}
        scroll={{ x: 1200 }}
        className='[&_.ant-table-thead>tr>th]:bg-gray-50 [&_.ant-table-thead>tr>th]:border-b-2 [&_.ant-table-thead>tr>th]:border-gray-200 [&_.ant-table-thead>tr>th]:font-semibold [&_.ant-table-thead>tr>th]:text-gray-700 [&_.ant-table-tbody>tr:hover>td]:bg-blue-50 [&_.ant-table-tbody>tr>td]:border-b [&_.ant-table-tbody>tr>td]:border-gray-100 [&_.ant-table-tbody>tr>td]:py-4'
      />
    </div>
  );
};
