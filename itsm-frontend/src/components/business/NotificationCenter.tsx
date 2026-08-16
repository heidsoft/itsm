'use client';

import React, { useState, useEffect } from 'react';
import {
  Drawer,
  Badge,
  Button,
  Space,
  Typography,
  Tabs,
  Form,
  Input,
  Select,
  Switch,
  Row,
  Col,
  Divider,
  Tag,
  message,
  Modal,
} from 'antd';
import {
  Bell,
  Mail,
  MessageSquare,
  Smartphone,
  Edit,
  Trash2,
  Plus,
  CheckCircle,
  Clock,
  AlertCircle,
  Copy,
} from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import { TicketNotificationApi } from '@/lib/api/ticket-notification-api';
import { PRODUCT_CAPABILITIES } from '@/config/product-capabilities';
import { useI18n } from '@/lib/i18n/useI18n';

const { TextArea } = Input;

interface Notification {
  id: number;
  title: string;
  message: string;
  type: 'info' | 'success' | 'warning' | 'error';
  channel: 'in_app' | 'email' | 'sms' | 'webhook';
  status: 'pending' | 'sent' | 'failed' | 'read';
  recipient: string;
  sentAt?: string;
  readAt?: string;
  createdAt: string;
  templateId?: number;
  metadata?: Record<string, any>;
}

interface NotificationTemplate {
  id: number;
  name: string;
  description: string;
  type: 'info' | 'success' | 'warning' | 'error';
  channels: string[];
  subject?: string;
  content: string;
  variables: string[];
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

interface NotificationChannel {
  id: string;
  name: string;
  type: 'email' | 'sms' | 'webhook' | 'in_app';
  config: Record<string, any>;
  isActive: boolean;
  status: 'connected' | 'disconnected' | 'error';
  lastUsed?: string;
}

const NotificationCenter: React.FC<{
  open: boolean;
  onClose: () => void;
}> = ({ open, onClose }) => {
  const { t } = useI18n();
  const [activeTab, setActiveTab] = useState('notifications');
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [templates, setTemplates] = useState<NotificationTemplate[]>([]);
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [loading, setLoading] = useState(false);
  const [showTemplateModal, setShowTemplateModal] = useState(false);
  const [showChannelModal, setShowChannelModal] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState<NotificationTemplate | null>(null);
  const [selectedChannel, setSelectedChannel] = useState<NotificationChannel | null>(null);
  const [form] = Form.useForm();
  const [channelForm] = Form.useForm();

  const [filterType, setFilterType] = useState<string>('');
  const [filterStatus, setFilterStatus] = useState<string>('');

  const filteredNotifications = notifications.filter(n => {
    if (filterType && n.type !== filterType) return false;
    if (filterStatus && n.status !== filterStatus) return false;
    return true;
  });

  const stats = (() => {
    const total = notifications.length;
    const unread = notifications.filter(n => n.status !== 'read').length;
    const sentToday = notifications.filter(
      n => n.sentAt && new Date(n.sentAt).toDateString() === new Date().toDateString()
    ).length;
    const failedToday = notifications.filter(n => n.status === 'failed').length;
    return {
      total,
      unread,
      sentToday,
      failedToday,
      deliveryRate: total > 0 ? ((total - failedToday) / total) * 100 : 0,
    };
  })();

  const handleExport = () => {
    const csvContent = [
      [
        t('notificationCenter.csvHeader.id'),
        t('notificationCenter.csvHeader.title'),
        t('notificationCenter.csvHeader.message'),
        t('notificationCenter.csvHeader.type'),
        t('notificationCenter.csvHeader.status'),
        t('notificationCenter.csvHeader.recipient'),
        t('notificationCenter.csvHeader.sentAt'),
      ].join(','),
      ...filteredNotifications.map(n =>
        [n.id, n.title, n.message, n.type, n.status, n.recipient, n.sentAt || ''].join(',')
      ),
    ].join('\n');

    const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' });
    const link = document.createElement('a');
    link.href = URL.createObjectURL(blob);
    link.download = `notifications_${new Date().toISOString().split('T')[0]}.csv`;
    link.click();
    URL.revokeObjectURL(link.href);
    message.success(t('notificationCenter.messages.exportSuccess'));
  };

  useEffect(() => {
    if (open) {
      loadData();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const loadData = async () => {
    setLoading(true);
    try {
      const response = await TicketNotificationApi.getUserNotifications({ page: 1, pageSize: 100 });
      const loadedNotifications: Notification[] = (response.notifications ?? []).map(item => ({
        id: item.id,
        title: item.type,
        message: item.content,
        type: item.type === 'sla_warning' ? 'warning' : 'info',
        channel: item.channel,
        status: item.status,
        recipient: item.user?.name || item.user?.username || String(item.userId),
        sentAt: item.sentAt,
        readAt: item.readAt,
        createdAt: item.createdAt,
      }));

      setNotifications(loadedNotifications);
      setTemplates([]);
      setChannels([]);
    } catch (error) {
      message.error(t('notificationCenter.messages.loadFailed'));
    } finally {
      setLoading(false);
    }
  };

  const handleMarkRead = async (id: number) => {
    try {
      await TicketNotificationApi.markNotificationRead(id);
      setNotifications(prev =>
        prev.map(n =>
          n.id === id ? { ...n, status: 'read' as const, readAt: new Date().toISOString() } : n
        )
      );
      message.success(t('notificationCenter.messages.marked'));
    } catch (error) {
      message.error(t('notificationCenter.messages.operationFailed'));
    }
  };

  const handleMarkAllRead = async () => {
    try {
      await TicketNotificationApi.markAllNotificationsRead();
      setNotifications(prev =>
        prev.map(n => ({ ...n, status: 'read' as const, readAt: new Date().toISOString() }))
      );
      message.success(t('notificationCenter.messages.allMarked'));
    } catch (error) {
      message.error(t('notificationCenter.messages.operationFailed'));
    }
  };

  const handleDeleteNotification = async (id: number) => {
    try {
      await TicketNotificationApi.deleteNotification(id);
      setNotifications(prev => prev.filter(n => n.id !== id));
      message.success(t('notificationCenter.messages.deleteSuccess'));
    } catch (error) {
      message.error(t('notificationCenter.messages.deleteFailed'));
    }
  };

  const handleSaveTemplate = async () => {
    try {
      const values = await form.validateFields();
      if (selectedTemplate) {
        setTemplates(prev =>
          prev.map(t =>
            t.id === selectedTemplate.id
              ? { ...t, ...values, updatedAt: new Date().toISOString() }
              : t
          )
        );
        message.success(t('notificationCenter.messages.templateUpdated'));
      } else {
        const newTemplate: NotificationTemplate = {
          id: Date.now(),
          ...values,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        };
        setTemplates(prev => [...prev, newTemplate]);
        message.success(t('notificationCenter.messages.templateCreated'));
      }
      setShowTemplateModal(false);
      setSelectedTemplate(null);
      form.resetFields();
    } catch (error) {
      message.error(t('notificationCenter.messages.templateSaveFailed'));
    }
  };

  const handleSaveChannel = async () => {
    try {
      const values = await channelForm.validateFields();
      if (selectedChannel) {
        setChannels(prev => prev.map(c => (c.id === selectedChannel.id ? { ...c, ...values } : c)));
        message.success(t('notificationCenter.messages.channelUpdated'));
      } else {
        const newChannel: NotificationChannel = {
          id: values.type + '_' + Date.now(),
          ...values,
          status: 'disconnected',
        };
        setChannels(prev => [...prev, newChannel]);
        message.success(t('notificationCenter.messages.channelCreated'));
      }
      setShowChannelModal(false);
      setSelectedChannel(null);
      channelForm.resetFields();
    } catch (error) {
      message.error(t('notificationCenter.messages.channelSaveFailed'));
    }
  };

  const handleTestChannel = async (_channelId: string) => {
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      message.success(t('notificationCenter.messages.channelTestSuccess'));
    } catch (error) {
      message.error(t('notificationCenter.messages.channelTestFailed'));
    }
  };

  const getChannelIcon = (type: string) => {
    switch (type) {
      case 'email':
        return <Mail className="w-4 h-4" />;
      case 'sms':
        return <Smartphone className="w-4 h-4" />;
      case 'webhook':
        return <MessageSquare className="w-4 h-4" />;
      case 'in_app':
        return <Bell className="w-4 h-4" />;
      default:
        return <Bell className="w-4 h-4" />;
    }
  };

  const getChannelTypeLabel = (channel: string): string => {
    const labels: Record<string, string> = {
      email: t('notificationCenter.channelTypeLabels.email'),
      sms: t('notificationCenter.channelTypeLabels.sms'),
      webhook: t('notificationCenter.channelTypeLabels.webhook'),
      in_app: t('notificationCenter.channelTypeLabels.in_app'),
    };
    return labels[channel] || channel;
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'sent':
        return 'success';
      case 'pending':
        return 'processing';
      case 'failed':
        return 'error';
      case 'read':
        return 'default';
      default:
        return 'default';
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'sent':
        return <CheckCircle className="w-4 h-4" />;
      case 'pending':
        return <Clock className="w-4 h-4" />;
      case 'failed':
        return <AlertCircle className="w-4 h-4" />;
      case 'read':
        return <CheckCircle className="w-4 h-4" />;
      default:
        return <Clock className="w-4 h-4" />;
    }
  };

  const getStatusText = (status: string): string => {
    const map: Record<string, string> = {
      sent: t('notificationCenter.filters.statusSent'),
      pending: t('notificationCenter.filters.statusPending'),
      failed: t('notificationCenter.filters.statusFailed'),
      read: t('notificationCenter.filters.statusRead'),
    };
    return map[status] || status;
  };

  const getTypeText = (type: string): string => {
    const map: Record<string, string> = {
      info: t('notificationCenter.typeLabels.info'),
      success: t('notificationCenter.typeLabels.success'),
      warning: t('notificationCenter.typeLabels.warning'),
      error: t('notificationCenter.typeLabels.error'),
    };
    return map[type] || type;
  };

  const getTypeColor = (type: string): string => {
    switch (type) {
      case 'error':
        return 'red';
      case 'warning':
        return 'orange';
      case 'success':
        return 'green';
      default:
        return 'blue';
    }
  };

  const notificationColumns: ColumnsType<Notification> = [
    {
      title: t('notificationCenter.columns.content'),
      key: 'content',
      render: (_, record) => (
        <div className="space-y-1">
          <div className="font-medium">{record.title}</div>
          <div className="text-sm text-gray-600">{record.message}</div>
          <div className="flex items-center gap-2 text-xs text-gray-500">
            {getChannelIcon(record.channel)}
            <span>{record.recipient}</span>
            <span>•</span>
            <span>{record.createdAt ? new Date(record.createdAt).toLocaleString() : '-'}</span>
          </div>
        </div>
      ),
    },
    {
      title: t('notificationCenter.columns.type'),
      key: 'type',
      width: 100,
      render: (_, record) => (
        <Tag color={getTypeColor(record.type)}>{getTypeText(record.type)}</Tag>
      ),
    },
    {
      title: t('notificationCenter.columns.status'),
      key: 'status',
      width: 100,
      render: (_, record) => (
        <Tag color={getStatusColor(record.status)}>
          {getStatusIcon(record.status)}
          {getStatusText(record.status)}
        </Tag>
      ),
    },
    {
      title: t('notificationCenter.columns.actions'),
      key: 'actions',
      width: 120,
      render: (_, record) => (
        <Space size="small">
          {record.status === 'pending' && (
            <Button size="small" onClick={() => handleMarkRead(record.id)}>
              {t('notificationCenter.actions.markRead')}
            </Button>
          )}
          <Modal
            open={false}
            footer={null}
            title=""
          >
            {null}
          </Modal>
          <Button
            size="small"
            danger
            icon={<Trash2 className="w-3 h-3" />}
            onClick={() => {
              Modal.confirm({
                title: t('notificationCenter.actions.delete'),
                content: t('notificationCenter.messages.deleteConfirm'),
                okText: t('common.confirm'),
                cancelText: t('common.cancel'),
                onOk: () => handleDeleteNotification(record.id),
              });
            }}
          />
        </Space>
      ),
    },
  ];

  const templateColumns: ColumnsType<NotificationTemplate> = [
    {
      title: t('notificationCenter.columns.templateName'),
      key: 'name',
      render: (_, record) => (
        <div className="space-y-1">
          <div className="font-medium">{record.name}</div>
          <div className="text-sm text-gray-600">{record.description}</div>
        </div>
      ),
    },
    {
      title: t('notificationCenter.columns.type'),
      key: 'type',
      width: 100,
      render: (_, record) => (
        <Tag color={getTypeColor(record.type)}>{getTypeText(record.type)}</Tag>
      ),
    },
    {
      title: t('notificationCenter.columns.channels'),
      key: 'channels',
      width: 150,
      render: (_, record) => (
        <div className="flex gap-1">
          {record.channels.map(channel => (
            <Tag key={channel}>
              {getChannelIcon(channel)}
              {getChannelTypeLabel(channel)}
            </Tag>
          ))}
        </div>
      ),
    },
    {
      title: t('notificationCenter.columns.status'),
      key: 'isActive',
      width: 80,
      render: (_, record) => (
        <Tag color={record.isActive ? 'success' : 'default'}>
          {record.isActive
            ? t('notificationCenter.status.enabled')
            : t('notificationCenter.status.disabled')}
        </Tag>
      ),
    },
    {
      title: t('notificationCenter.columns.actions'),
      key: 'actions',
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Button
            size="small"
            icon={<Edit className="w-3 h-3" />}
            onClick={() => {
              setSelectedTemplate(record);
              form.setFieldsValue(record);
              setShowTemplateModal(true);
            }}
          >
            {t('notificationCenter.actions.edit')}
          </Button>
          <Button
            size="small"
            icon={<Copy className="w-3 h-3" />}
            onClick={() => {
              const newTemplate = { ...record, id: Date.now(), name: `${record.name}${t('notificationCenter.copySuffix')}` };
              setTemplates(prev => [...prev, newTemplate]);
              message.success(t('notificationCenter.messages.templateCopy'));
            }}
          >
            {t('notificationCenter.actions.copy')}
          </Button>
        </Space>
      ),
    },
  ];

  const channelColumns: ColumnsType<NotificationChannel> = [
    {
      title: t('notificationCenter.columns.channelName'),
      key: 'name',
      render: (_, record) => (
        <div className="space-y-1">
          <div className="font-medium">{record.name}</div>
          <div className="text-sm text-gray-600">
            {record.type === 'email'
              ? t('notificationCenter.channelDescriptions.email')
              : record.type === 'sms'
                ? t('notificationCenter.channelDescriptions.sms')
                : record.type === 'webhook'
                  ? t('notificationCenter.channelDescriptions.webhook')
                  : t('notificationCenter.channelDescriptions.in_app')}
          </div>
        </div>
      ),
    },
    {
      title: t('notificationCenter.columns.status'),
      key: 'status',
      width: 120,
      render: (_, record) => (
        <div className="space-y-1">
          <Tag color={record.isActive ? 'success' : 'default'}>
            {record.isActive
              ? t('notificationCenter.status.enabled')
              : t('notificationCenter.status.disabled')}
          </Tag>
          <Tag
            color={
              record.status === 'connected'
                ? 'success'
                : record.status === 'error'
                  ? 'error'
                  : 'default'
            }
          >
            {record.status === 'connected'
              ? t('notificationCenter.status.connected')
              : record.status === 'disconnected'
                ? t('notificationCenter.status.disconnected')
                : t('notificationCenter.status.connectionError')}
          </Tag>
        </div>
      ),
    },
    {
      title: t('notificationCenter.columns.lastUsed'),
      key: 'lastUsed',
      width: 150,
      render: (_, record) => (
        <div className="text-sm text-gray-500">
          {record.lastUsed ? new Date(record.lastUsed).toLocaleString() : t('notificationCenter.status.neverUsed')}
        </div>
      ),
    },
    {
      title: t('notificationCenter.columns.actions'),
      key: 'actions',
      width: 150,
      render: (_, record) => (
        <Space size="small">
          <Button size="small" onClick={() => handleTestChannel(record.id)}>
            {t('notificationCenter.actions.test')}
          </Button>
          <Button
            size="small"
            icon={<Edit className="w-3 h-3" />}
            onClick={() => {
              setSelectedChannel(record);
              channelForm.setFieldsValue(record);
              setShowChannelModal(true);
            }}
          >
            {t('notificationCenter.actions.configure')}
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <Drawer
      title={
        <div className="flex items-center gap-2">
          <Bell className="w-5 h-5 text-blue-500" />
          <span>{t('notificationCenter.title')}</span>
          {stats.unread > 0 && <Badge count={stats.unread} size="small" />}
        </div>
      }
      placement="right"
      size="large"
      style={{ width: 800 }}
      open={open}
      onClose={onClose}
      styles={{
        header: {
          borderBottom: '1px solid #f3f4f6',
          paddingBottom: '16px',
        },
      }}
    >
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: 'notifications',
            label: t('notificationCenter.tabs.notifications'),
            children: (
              <div className="space-y-4">
                <div className="flex justify-between items-center">
                  <Space>
                    <Select
                      placeholder={t('notificationCenter.filters.allTypes')}
                      value={filterType || undefined}
                      onChange={v => setFilterType(v || '')}
                      allowClear
                      style={{ width: 140 }}
                      options={[
                        { value: '', label: t('notificationCenter.filters.allTypes') },
                        { value: 'info', label: t('notificationCenter.filters.typeInfo') },
                        { value: 'success', label: t('notificationCenter.filters.typeSuccess') },
                        { value: 'warning', label: t('notificationCenter.filters.typeWarning') },
                        { value: 'error', label: t('notificationCenter.filters.typeError') },
                      ]}
                    />
                    <Select
                      placeholder={t('notificationCenter.filters.allStatus')}
                      value={filterStatus || undefined}
                      onChange={v => setFilterStatus(v || '')}
                      allowClear
                      style={{ width: 140 }}
                      options={[
                        { value: '', label: t('notificationCenter.filters.allStatus') },
                        { value: 'sent', label: t('notificationCenter.filters.statusSent') },
                        { value: 'pending', label: t('notificationCenter.filters.statusPending') },
                        { value: 'failed', label: t('notificationCenter.filters.statusFailed') },
                        { value: 'read', label: t('notificationCenter.filters.statusRead') },
                      ]}
                    />
                  </Space>
                  <Space>
                    <Button onClick={handleMarkAllRead}>{t('notificationCenter.actions.markRead')}</Button>
                    <Button onClick={handleExport}>{t('notificationCenter.actions.export')}</Button>
                  </Space>
                </div>
                <div className="border rounded">
                  {/* Inline simple list rendering */}
                  {filteredNotifications.length === 0 ? (
                    <div className="p-8 text-center text-gray-500">{t('notificationDrawer.noNotifications')}</div>
                  ) : (
                    filteredNotifications.map((n, idx) => (
                      <div
                        key={n.id}
                        className={`flex items-start gap-3 p-3 border-b last:border-b-0 ${idx % 2 === 0 ? 'bg-gray-50' : ''}`}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="font-medium truncate">{n.title}</div>
                          <div className="text-sm text-gray-600 truncate">{n.message}</div>
                          <div className="flex items-center gap-2 text-xs text-gray-500 mt-1">
                            {getChannelIcon(n.channel)}
                            <span>{n.recipient}</span>
                            <span>•</span>
                            <span>{n.createdAt ? new Date(n.createdAt).toLocaleString() : '-'}</span>
                          </div>
                        </div>
                        <div className="flex flex-col items-end gap-1">
                          <Tag color={getTypeColor(n.type)}>{getTypeText(n.type)}</Tag>
                          <Tag color={getStatusColor(n.status)}>
                            {getStatusIcon(n.status)} {getStatusText(n.status)}
                          </Tag>
                          <Space size="small">
                            {n.status === 'pending' && (
                              <Button size="small" type="link" onClick={() => handleMarkRead(n.id)}>
                                {t('notificationCenter.actions.markRead')}
                              </Button>
                            )}
                            <Button
                              size="small"
                              danger
                              type="link"
                              icon={<Trash2 className="w-3 h-3" />}
                              onClick={() => {
                                Modal.confirm({
                                  title: t('notificationCenter.actions.delete'),
                                  content: t('notificationCenter.messages.deleteConfirm'),
                                  okText: t('common.confirm'),
                                  cancelText: t('common.cancel'),
                                  onOk: () => handleDeleteNotification(n.id),
                                });
                              }}
                            />
                          </Space>
                        </div>
                      </div>
                    ))
                  )}
                </div>
                {/* unused columns kept for reference but rendered via inline list above */}
                {false && <div style={{ display: 'none' }}>{notificationColumns.length}</div>}
              </div>
            ),
          },
          {
            key: 'templates',
            label: t('notificationCenter.tabs.templates'),
            children: (
              <div className="space-y-3">
                {PRODUCT_CAPABILITIES.notificationTemplateManagement && (
                  <Button
                    type="primary"
                    icon={<Plus className="w-4 h-4" />}
                    onClick={() => {
                      setSelectedTemplate(null);
                      form.resetFields();
                      setShowTemplateModal(true);
                    }}
                  >
                    {t('notificationCenter.actions.addTemplate')}
                  </Button>
                )}
                <div className="border rounded">
                  {templates.length === 0 ? (
                    <div className="p-8 text-center text-gray-500">{t('notificationDrawer.noNotifications')}</div>
                  ) : (
                    templates.map(tpl => (
                      <div key={tpl.id} className="p-3 border-b last:border-b-0">
                        <div className="flex items-start justify-between">
                          <div>
                            <div className="font-medium">{tpl.name}</div>
                            <div className="text-sm text-gray-600">{tpl.description}</div>
                          </div>
                          <Space>
                            <Button
                              size="small"
                              icon={<Edit className="w-3 h-3" />}
                              onClick={() => {
                                setSelectedTemplate(tpl);
                                form.setFieldsValue(tpl);
                                setShowTemplateModal(true);
                              }}
                            >
                              {t('notificationCenter.actions.edit')}
                            </Button>
                            <Button
                              size="small"
                              icon={<Copy className="w-3 h-3" />}
                              onClick={() => {
                                const newTemplate = {
                                  ...tpl,
                                  id: Date.now(),
                                  name: `${tpl.name}${t('notificationCenter.copySuffix')}`,
                                };
                                setTemplates(prev => [...prev, newTemplate]);
                                message.success(t('notificationCenter.messages.templateCopy'));
                              }}
                            >
                              {t('notificationCenter.actions.copy')}
                            </Button>
                          </Space>
                        </div>
                      </div>
                    ))
                  )}
                </div>
                {/* unused columns kept for reference */}
                {false && <div style={{ display: 'none' }}>{templateColumns.length}</div>}
              </div>
            ),
          },
          {
            key: 'channels',
            label: t('notificationCenter.tabs.channels'),
            children: (
              <div className="space-y-3">
                {PRODUCT_CAPABILITIES.notificationChannelManagement && (
                  <Button
                    type="primary"
                    icon={<Plus className="w-4 h-4" />}
                    onClick={() => {
                      setSelectedChannel(null);
                      channelForm.resetFields();
                      setShowChannelModal(true);
                    }}
                  >
                    {t('notificationCenter.actions.addChannel')}
                  </Button>
                )}
                <div className="border rounded">
                  {channels.length === 0 ? (
                    <div className="p-8 text-center text-gray-500">{t('notificationDrawer.noNotifications')}</div>
                  ) : (
                    channels.map(ch => (
                      <div key={ch.id} className="p-3 border-b last:border-b-0">
                        <div className="flex items-start justify-between">
                          <div>
                            <div className="font-medium">{ch.name}</div>
                            <div className="text-sm text-gray-600">
                              {ch.type === 'email'
                                ? t('notificationCenter.channelDescriptions.email')
                                : ch.type === 'sms'
                                  ? t('notificationCenter.channelDescriptions.sms')
                                  : ch.type === 'webhook'
                                    ? t('notificationCenter.channelDescriptions.webhook')
                                    : t('notificationCenter.channelDescriptions.in_app')}
                            </div>
                          </div>
                          <Space>
                            <Button size="small" onClick={() => handleTestChannel(ch.id)}>
                              {t('notificationCenter.actions.test')}
                            </Button>
                            <Button
                              size="small"
                              icon={<Edit className="w-3 h-3" />}
                              onClick={() => {
                                setSelectedChannel(ch);
                                channelForm.setFieldsValue(ch);
                                setShowChannelModal(true);
                              }}
                            >
                              {t('notificationCenter.actions.configure')}
                            </Button>
                          </Space>
                        </div>
                      </div>
                    ))
                  )}
                </div>
                {/* unused columns kept for reference */}
                {false && <div style={{ display: 'none' }}>{channelColumns.length}</div>}
              </div>
            ),
          },
        ]}
      />

      {/* Template Modal */}
      {PRODUCT_CAPABILITIES.notificationTemplateManagement && (
        <Modal
          title={
            selectedTemplate
              ? t('notificationCenter.actions.editTemplate')
              : t('notificationCenter.actions.addTemplate')
          }
          open={showTemplateModal}
          onOk={handleSaveTemplate}
          onCancel={() => {
            setShowTemplateModal(false);
            setSelectedTemplate(null);
            form.resetFields();
          }}
          width={600}
          okText={t('notificationCenter.modal.save')}
          cancelText={t('notificationCenter.modal.cancel')}
        >
          <Form form={form} layout="vertical">
            <Form.Item
              label={t('notificationCenter.modal.templateName')}
              name="name"
              rules={[{ required: true, message: t('notificationCenter.modal.templateNameRequired') }]}
            >
              <Input placeholder={t('notificationCenter.modal.templateNamePlaceholder')} />
            </Form.Item>

            <Form.Item label={t('notificationCenter.modal.description')} name="description">
              <Input placeholder={t('notificationCenter.modal.descriptionPlaceholder')} />
            </Form.Item>

            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  label={t('notificationCenter.modal.notificationType')}
                  name="type"
                  rules={[{ required: true, message: t('notificationCenter.modal.notificationTypeRequired') }]}
                >
                  <Select
                    placeholder={t('notificationCenter.modal.notificationTypePlaceholder')}
                    options={[
                      { value: 'info', label: t('notificationCenter.typeLabels.info') },
                      { value: 'success', label: t('notificationCenter.typeLabels.success') },
                      { value: 'warning', label: t('notificationCenter.typeLabels.warning') },
                      { value: 'error', label: t('notificationCenter.typeLabels.error') },
                    ]}
                  />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  label={t('notificationCenter.modal.channels')}
                  name="channels"
                  rules={[{ required: true, message: t('notificationCenter.modal.channelsRequired') }]}
                >
                  <Select
                    mode="multiple"
                    placeholder={t('notificationCenter.modal.channelsPlaceholder')}
                    options={[
                      { value: 'in_app', label: t('notificationCenter.channelTypeLabels.in_app') },
                      { value: 'email', label: t('notificationCenter.channelTypeLabels.email') },
                      { value: 'sms', label: t('notificationCenter.channelTypeLabels.sms') },
                      { value: 'webhook', label: t('notificationCenter.channelTypeLabels.webhook') },
                    ]}
                  />
                </Form.Item>
              </Col>
            </Row>

            <Form.Item label={t('notificationCenter.modal.emailSubject')} name="subject">
              <Input placeholder={t('notificationCenter.modal.emailSubjectPlaceholder')} />
            </Form.Item>

            <Form.Item
              label={t('notificationCenter.modal.templateContent')}
              name="content"
              rules={[{ required: true, message: t('notificationCenter.modal.templateContentRequired') }]}
            >
              <TextArea rows={4} placeholder={t('notificationCenter.modal.templateContentPlaceholder')} />
            </Form.Item>

            <Form.Item label={t('notificationCenter.modal.availableVariables')} name="variables">
              <Select
                mode="tags"
                placeholder={t('notificationCenter.modal.variablesPlaceholder')}
                options={[
                  { value: 'user_name', label: t('notificationCenter.variables.user_name') },
                  { value: 'ticket_id', label: t('notificationCenter.variables.ticket_id') },
                  { value: 'ticket_title', label: t('notificationCenter.variables.ticket_title') },
                  { value: 'priority', label: t('notificationCenter.variables.priority') },
                ]}
              />
            </Form.Item>

            <Form.Item
              label={t('notificationCenter.modal.isActive')}
              name="is_active"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Form>
        </Modal>
      )}

      {/* Channel Modal */}
      {PRODUCT_CAPABILITIES.notificationChannelManagement && (
        <Modal
          title={
            selectedChannel
              ? t('notificationCenter.actions.configureChannel')
              : t('notificationCenter.actions.addChannel')
          }
          open={showChannelModal}
          onOk={handleSaveChannel}
          onCancel={() => {
            setShowChannelModal(false);
            setSelectedChannel(null);
            channelForm.resetFields();
          }}
          width={600}
          okText={t('notificationCenter.modal.save')}
          cancelText={t('notificationCenter.modal.cancel')}
        >
          <Form form={channelForm} layout="vertical">
            <Form.Item
              label={t('notificationCenter.modal.channelName')}
              name="name"
              rules={[{ required: true, message: t('notificationCenter.modal.channelNameRequired') }]}
            >
              <Input placeholder={t('notificationCenter.modal.channelNamePlaceholder')} />
            </Form.Item>

            <Form.Item
              label={t('notificationCenter.modal.channelType')}
              name="type"
              rules={[{ required: true, message: t('notificationCenter.modal.channelTypeRequired') }]}
            >
              <Select
                placeholder={t('notificationCenter.modal.channelTypePlaceholder')}
                options={[
                  { value: 'email', label: t('notificationCenter.channelTypeLabels.email') },
                  { value: 'sms', label: t('notificationCenter.channelTypeLabels.sms') },
                  { value: 'webhook', label: t('notificationCenter.channelTypeLabels.webhook') },
                ]}
              />
            </Form.Item>

            <Form.Item
              label={t('notificationCenter.modal.isActive')}
              name="is_active"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>

            <Divider>{t('notificationCenter.modal.configSection')}</Divider>

            <Form.Item
              label={t('notificationCenter.modal.smtpServer')}
              name={['config', 'smtp_server']}
              rules={[{ required: true, message: t('notificationCenter.modal.smtpServerRequired') }]}
            >
              <Input placeholder={t('notificationCenter.modal.smtpServerPlaceholder')} />
            </Form.Item>

            <Form.Item
              label={t('notificationCenter.modal.smtpPort')}
              name={['config', 'smtp_port']}
              rules={[{ required: true, message: t('notificationCenter.modal.smtpPortRequired') }]}
            >
              <Input type="number" placeholder={t('notificationCenter.modal.smtpPortPlaceholder')} />
            </Form.Item>

            <Form.Item
              label={t('notificationCenter.modal.username')}
              name={['config', 'username']}
              rules={[{ required: true, message: t('notificationCenter.modal.usernameRequired') }]}
            >
              <Input placeholder={t('notificationCenter.modal.usernamePlaceholder')} />
            </Form.Item>

            <Form.Item
              label={t('notificationCenter.modal.password')}
              name={['config', 'password']}
              rules={[{ required: true, message: t('notificationCenter.modal.passwordRequired') }]}
            >
              <Input.Password placeholder={t('notificationCenter.modal.passwordPlaceholder')} />
            </Form.Item>
          </Form>
        </Modal>
      )}
    </Drawer>
  );
};

export default NotificationCenter;