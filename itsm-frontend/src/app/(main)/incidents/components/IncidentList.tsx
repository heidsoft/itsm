'use client';

import React from 'react';
import { Table, Tag, Button, Space, Dropdown, message, Modal } from 'antd';
import { Eye, Edit, MoreHorizontal, AlertTriangle, Trash2 } from 'lucide-react';
import type { MenuProps } from 'antd';
import dayjs from 'dayjs';
import type { Incident } from '@/lib/api/types';
import { IncidentAPI } from '@/lib/api/incident-api';
import { useI18n } from '@/lib/i18n';

interface IncidentListProps {
  incidents: Incident[];
  loading: boolean;
  selectedRowKeys: React.Key[];
  onSelectedRowKeysChange: (keys: React.Key[]) => void;
  onEdit: (incident: Incident) => void;
  onDelete?: (incident: Incident) => void;
  onRefresh?: () => void;
}

export const IncidentList: React.FC<IncidentListProps> = ({
  incidents,
  loading,
  selectedRowKeys,
  onSelectedRowKeysChange,
  onEdit,
  onDelete,
  onRefresh,
}) => {
  const { t } = useI18n();

  const statusConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
    // 新建/待处理状态
    new: {
      color: '#1890ff',
      text: t('incidents.statusNew') || 'Create',
      backgroundColor: '#e6f7ff',
    },
    // 已确认状态
    acknowledged: {
      color: '#722ed1',
      text: t('incidents.statusAcknowledged') || 'Acknowledged',
      backgroundColor: '#f9f0ff',
    },
    // 已分配状态
    assigned: {
      color: '#13c2c2',
      text: t('incidents.statusAssigned') || 'Assigned',
      backgroundColor: '#e6fffb',
    },
    // 调查中状态
    investigating: {
      color: '#722ed1',
      text: t('incidents.statusInvestigating') || 'Investigating',
      backgroundColor: '#f9f0ff',
    },
    // 处理中状态 (both camelCase and snake_case from API)
    inProgress: {
      color: '#1890ff',
      text: t('incidents.statusInProgress'),
      backgroundColor: '#e6f7ff',
    },
    'in_progress': {
      color: '#1890ff',
      text: t('incidents.statusInProgress'),
      backgroundColor: '#e6f7ff',
    },
    // 已分类状态
    triaged: {
      color: '#faad14',
      text: t('incidents.statusTriaged') || 'Triaged',
      backgroundColor: '#fffbe6',
    },
    // 已升级状态
    escalated: {
      color: '#ff4d4f',
      text: t('incidents.statusEscalated') || 'Escalated',
      backgroundColor: '#fff2f0',
    },
    // 已暂停
    'on_hold': {
      color: '#fa8c16',
      text: t('incidents.statusOnHold') || 'Paused',
      backgroundColor: '#fff7e6',
    },
    // 已解决状态
    resolved: {
      color: '#52c41a',
      text: t('incidents.statusResolved'),
      backgroundColor: '#f6ffed',
    },
    // 已取消状态
    cancelled: {
      color: '#00000073',
      text: t('incidents.statusCancelled') || 'Cancelled',
      backgroundColor: '#fafafa',
    },
    // 已关闭状态
    closed: {
      color: '#00000073',
      text: t('incidents.statusClosed'),
      backgroundColor: '#fafafa',
    },
    // 兼容旧的前端状态值
    open: {
      color: '#fa8c16',
      text: t('incidents.statusOpen'),
      backgroundColor: '#fff7e6',
    },
  };

  const priorityConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
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
      key:'incidentInfo',
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
              #{record.incidentNumber || (record as unknown as { incidentNumber?: string }).incidentNumber || '-'} • {record.category}
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
      dataIndex: 'reporterName',
      key: 'reporter',
      width: 150,
      render: (_: unknown, record: Incident) => {
        // 后端列表接口回填 reporterName；缺失时回退用户 ID，不再恒显「未知」
        const text = record.reporterName || (record.reporter?.name || (record.reporterId ? `User#${record.reporterId}` : t('incidents.unknown')));
        return <div style={{ fontSize: 'small' }}>{text}</div>;
      },
    },
    {
      title: 'Assignee',
      dataIndex: 'assigneeName',
      key: 'assignee',
      width: 130,
      render: (_: unknown, record: Incident) => {
        const text = record.assigneeName || record.assignee?.name || (record.assigneeId ? `User#${record.assigneeId}` : '-');
        return <div style={{ fontSize: 'small' }}>{text}</div>;
      },
    },
    {
      title: t('incidents.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (created_at: string) => (
        <div style={{ fontSize: 'small', color: '#666' }}>
          {created_at ? dayjs(created_at).format('YYYY-MM-DD HH:mm') : '-'}
        </div>
      ),
    },
    {
      title: t('incidents.operations'),
      key: 'actions',
      width: 150,
      render: (_: unknown, record: Incident) => {
        const handleDelete = () => {
          Modal.confirm({
            title: 'Confirm Delete',
            content: (
              <div>
                <p>Confirm to delete event "{record.title}"?"</p>
                <p style={{ color: '#ff4d4f' }}>This operation cannot be undone.</p>
              </div>
            ),
            okText: 'Confirm Delete',
            okType: 'danger',
            cancelText: 'Cancel',
            onOk: async () => {
              try {
                await IncidentAPI.deleteIncident(record.id);
                message.success(t('incidents.deleteSuccess') || 'Deleted');
                onRefresh?.();
              } catch (error) {
                console.error('Failed to delete incident:', error);
                message.error(t('incidents.deleteFailed') || 'Delete failed');
              }
            },
          });
        };

        const items: MenuProps['items'] = [
          {
            key: 'delete',
            label: t('common.delete') || 'Delete',
            icon: <Trash2 size={14} />,
            danger: true,
            onClick: handleDelete,
          },
        ];

        return (
          <Space size="small">
            <Button
              type="text"
              size="small"
              icon={<Eye size={16} />}
              href={`/incidents/${record.id}`}
              aria-label={`View events ${record.title}`}
              className="text-blue-600 hover:text-blue-700 hover:bg-blue-50 border-0 rounded-lg transition-all duration-200 p-2"
              title={t('incidents.viewDetails')}
            />
            <Button
              type="text"
              size="small"
              icon={<Edit size={16} />}
              onClick={() => onEdit(record)}
              aria-label={`Edit Incident ${record.title}`}
              className="text-green-600 hover:text-green-700 hover:bg-green-50 border-0 rounded-lg transition-all duration-200 p-2"
              title={t('incidents.editIncident')}
            />
            <Dropdown menu={{ items }} trigger={['click']} placement="bottomRight">
              <Button
                type="text"
                size="small"
                icon={<MoreHorizontal size={16} />}
                className="text-gray-600 hover:text-gray-700 hover:bg-gray-50 border-0 rounded-lg transition-all duration-200 p-2"
                title={t('incidents.moreActions')}
                onClick={e => e.preventDefault()}
              />
            </Dropdown>
          </Space>
        );
      },
    },
  ];

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: onSelectedRowKeysChange,
        }}
        columns={columns}
        dataSource={incidents}
        rowKey="id"
        loading={loading}
        pagination={false}
        scroll={{ x: 1330 }}
      />
    </div>
  );
};
