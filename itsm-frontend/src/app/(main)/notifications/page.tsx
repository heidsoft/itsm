'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Card,
  List,
  Badge,
  Button,
  Space,
  Typography,
  Tabs,
  Tag,
  Empty,
  Switch,
  Form,
  App,
  Spin,
  Divider,
  Row,
  Col,
  Input,
  Select,
  DatePicker,
  Tooltip,
  Popconfirm,
} from 'antd';
import {
  Bell,
  Mail,
  MessageSquare,
  Smartphone,
  Settings,
  CheckCircle,
  Clock,
  Eye,
  RotateCcw,
  Search,
  Filter,
  Delete,
  Wifi,
  WifiOff,
  RefreshCw,
} from 'lucide-react';
import dayjs from 'dayjs';
import type {
  TicketNotification,
  NotificationPreferenceItem} from '@/lib/api/ticket-notification-api';
import {
  TicketNotificationApi
} from '@/lib/api/ticket-notification-api';
import { useAuthStore } from '@/lib/store/auth-store';
import { notificationWS } from '@/lib/services/notification-ws';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;
const { RangePicker } = DatePicker;

const normalizeNotification = (notification: TicketNotification): TicketNotification => ({
  ...notification,
  createdAt: notification.createdAt || notification.createdAt,
  readAt: notification.readAt || notification.readAt,
  sentAt: notification.sentAt || notification.sentAt,
  ticketId: notification.ticketId || notification.ticketId,
  userId: notification.userId || notification.userId,
});

// 通知事件类型配置
interface EventTypeConfig {
  type: string;
  nameKey: string;
  descKey: string;
}

const EVENT_TYPES: EventTypeConfig[] = [
  {
    type: 'ticket_created',
    nameKey: 'notifications.ticketCreated',
    descKey: 'notifications.ticketCreatedDesc',
  },
  {
    type: 'ticket_assigned',
    nameKey: 'notifications.ticketAssigned',
    descKey: 'notifications.ticketAssignedDesc',
  },
  {
    type: 'ticket_updated',
    nameKey: 'notifications.ticketUpdated',
    descKey: 'notifications.ticketUpdatedDesc',
  },
  {
    type: 'ticket_commented',
    nameKey: 'notifications.ticketCommented',
    descKey: 'notifications.ticketCommentedDesc',
  },
  {
    type: 'sla_warning',
    nameKey: 'notifications.slaWarning',
    descKey: 'notifications.slaWarningDesc',
  },
  {
    type: 'sla_violated',
    nameKey: 'notifications.slaViolated',
    descKey: 'notifications.slaViolatedDesc',
  },
  {
    type: 'ticket_resolved',
    nameKey: 'notifications.ticketResolved',
    descKey: 'notifications.ticketResolvedDesc',
  },
  {
    type: 'ticket_closed',
    nameKey: 'notifications.ticketClosed',
    descKey: 'notifications.ticketClosedDesc',
  },
  {
    type: 'approval_required',
    nameKey: 'notifications.approvalRequired',
    descKey: 'notifications.approvalRequiredDesc',
  },
  {
    type: 'approval_completed',
    nameKey: 'notifications.approvalCompleted',
    descKey: 'notifications.approvalCompletedDesc',
  },
];

// 通知渠道配置
interface ChannelConfig {
  key: string;
  nameKey: string;
  icon: React.ComponentType<{ className?: string }>;
  color: string;
}

const CHANNELS: ChannelConfig[] = [
  { key: 'email', nameKey: 'notifications.email', icon: Mail, color: 'blue' },
  { key:'inApp', nameKey: 'notifications.inApp', icon: MessageSquare, color: 'default' },
  { key: 'sms', nameKey: 'notifications.sms', icon: Smartphone, color: 'green' },
];

export default function NotificationsPage() {
  const { t } = useI18n();
  const { user, token } = useAuthStore();
  const { message } = App.useApp();
  const [notifications, setNotifications] = useState<TicketNotification[]>([]);
  const [unreadNotifications, setUnreadNotifications] = useState<TicketNotification[]>([]);
  const [readNotifications, setReadNotifications] = useState<TicketNotification[]>([]);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('all');
  const [preferences, setPreferences] = useState<NotificationPreferenceItem[]>([]);
  const [preferencesLoading, setPreferencesLoading] = useState(false);
  const [form] = Form.useForm();

  // 筛选状态
  const [searchText, setSearchText] = useState('');
  const [channelFilter, setChannelFilter] = useState<string | null>(null);
  const [typeFilter, setTypeFilter] = useState<string | null>(null);
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null);

  // WebSocket 状态
  const [wsConnected, setWsConnected] = useState(false);

  // 加载通知
  const loadNotifications = useCallback(async () => {
    setLoading(true);
    try {
      const [allRes, unreadRes, readRes] = await Promise.all([
        TicketNotificationApi.getUserNotifications({ page: 1, pageSize: 100 }),
        TicketNotificationApi.getUserNotifications({ page: 1, pageSize: 100, read: false }),
        TicketNotificationApi.getUserNotifications({ page: 1, pageSize: 100, read: true }),
      ]);

      const all = (allRes.notifications || []).map(normalizeNotification);
      setNotifications(all);
      setUnreadNotifications((unreadRes.notifications || []).map(normalizeNotification));
      setReadNotifications((readRes.notifications || []).map(normalizeNotification));
    } catch (error) {
      message.error(t('notifications.loadFailed'));
      console.error('Failed to load notifications:', error);
    } finally {
      setLoading(false);
    }
  }, [t]);

  // 加载偏好设置
  const loadPreferences = useCallback(async () => {
    setPreferencesLoading(true);
    try {
      const prefs = await TicketNotificationApi.getNotificationPreferences();
      setPreferences(prefs.preferences || []);

      // 设置表单初始值
      const formValues: Record<string, boolean> = {};
      prefs.preferences?.forEach(pref => {
        formValues[`${pref.eventType}_email`] = pref.emailEnabled;
        formValues[`${pref.eventType}_in_app`] = pref.inAppEnabled;
        formValues[`${pref.eventType}_sms`] = pref.smsEnabled;
      });
      form.setFieldsValue(formValues);
    } catch (error) {
      message.error(t('notifications.loadFailed'));
      console.error('Failed to load preferences:', error);
    } finally {
      setPreferencesLoading(false);
    }
  }, [t, form]);

  // 初始化 WebSocket 连接
  useEffect(() => {
    if (user?.id && token) {
      // 连接 WebSocket
      notificationWS
        .connect(user.id, token)
        .then(() => {
          setWsConnected(true);
        })
        .catch(error => {
          console.error('WebSocket connection failed:', error);
          setWsConnected(false);
        });

      // 监听新通知
      const unsubscribe = notificationWS.onNotification(notification => {
        const normalized = normalizeNotification(notification);
        setNotifications(prev => [normalized, ...prev]);
        setUnreadNotifications(prev => [normalized, ...prev]);
        message.info(notification.content);
      });

      // 监听连接状态变化
      const unsubConnection = notificationWS.onConnectionChange(connected => {
        setWsConnected(connected);
      });

      return () => {
        unsubscribe();
        unsubConnection();
        notificationWS.disconnect();
      };
    }
  }, [user?.id, token]);

  // 初始加载
  useEffect(() => {
    loadNotifications();
    loadPreferences();
  }, [loadNotifications, loadPreferences]);

  // 标记已读
  const handleMarkRead = useCallback(
    async (notificationId: number) => {
      try {
        await TicketNotificationApi.markNotificationRead(notificationId);
        message.success(t('notifications.markSuccess'));
        loadNotifications();
      } catch (error) {
        message.error(t('notifications.loadFailed'));
        console.error('Failed to mark notification as read:', error);
      }
    },
    [t, loadNotifications]
  );

  // 标记全部已读
  const handleMarkAllRead = useCallback(async () => {
    try {
      await TicketNotificationApi.markAllNotificationsRead();
      message.success(t('notifications.markSuccess'));
      loadNotifications();
    } catch (error) {
      message.error(t('notifications.loadFailed'));
      console.error('Failed to mark all notifications as read:', error);
    }
  }, [t, loadNotifications]);

  // 保存偏好设置
  const handleSavePreferences = useCallback(async () => {
    try {
      const values = form.getFieldsValue();
      const prefs: NotificationPreferenceItem[] = [];

      EVENT_TYPES.forEach(event => {
        prefs.push({
          eventType: event.type,
          emailEnabled: values[`${event.type}_email`] ?? false,
          inAppEnabled: values[`${event.type}_in_app`] ?? true,
          smsEnabled: values[`${event.type}_sms`] ?? false,
        });
      });

      await TicketNotificationApi.updateNotificationPreferences({ preferences: prefs });
      message.success(t('notifications.saveSuccess'));
      loadPreferences();
    } catch (error) {
      message.error(t('notifications.saveFailed'));
      console.error('Failed to update preferences:', error);
    }
  }, [t, form, loadPreferences]);

  // 重置偏好设置
  const handleResetPreferences = useCallback(async () => {
    try {
      await TicketNotificationApi.resetNotificationPreferences();
      message.success(t('notifications.resetSuccess'));
      loadPreferences();
    } catch (error) {
      message.error(t('notifications.saveFailed'));
      console.error('Failed to reset preferences:', error);
    }
  }, [t, loadPreferences]);

  // 删除单个通知
  const handleDeleteNotification = useCallback(
    async (notificationId: number) => {
      try {
        await TicketNotificationApi.deleteNotification(notificationId);
        message.success(t('notifications.delete'));
        loadNotifications();
      } catch (error) {
        message.error(t('notifications.loadFailed'));
        console.error('Failed to delete notification:', error);
      }
    },
    [t, loadNotifications]
  );

  // 清空所有通知（获取全部通知后逐条删除）
  const handleClearAll = useCallback(async () => {
    try {
      // 获取所有通知（含已读与未读），逐条删除
      const response = await TicketNotificationApi.getUserNotifications({
        page: 1,
        pageSize: 200,
      });
      const allItems = response.notifications || [];
      await Promise.all(
        allItems.map(item => TicketNotificationApi.deleteNotification(item.id))
      );
      message.success(t('notifications.clearAll'));
      loadNotifications();
    } catch (error) {
      message.error(t('notifications.loadFailed'));
      console.error('Failed to clear all notifications:', error);
    }
  }, [t, loadNotifications]);

  // 筛选通知列表
  const filterNotifications = useCallback(
    (list: TicketNotification[]) => {
      return list.filter(item => {
        // 搜索筛选
        if (searchText && !item.content.toLowerCase().includes(searchText.toLowerCase())) {
          return false;
        }
        // 渠道筛选
        if (channelFilter && item.channel !== channelFilter) {
          return false;
        }
        // 类型筛选
        if (typeFilter && item.type !== typeFilter) {
          return false;
        }
        // 日期范围筛选
        if (dateRange && item.createdAt) {
          const date = dayjs(item.createdAt);
          if (!date.isAfter(dateRange[0]) || !date.isBefore(dateRange[1])) {
            return false;
          }
        }
        return true;
      });
    },
    [searchText, channelFilter, typeFilter, dateRange]
  );

  // 获取筛选后的通知
  const filteredNotifications = useMemo(
    () => filterNotifications(notifications),
    [notifications, filterNotifications]
  );
  const filteredUnreadNotifications = useMemo(
    () => filterNotifications(unreadNotifications),
    [unreadNotifications, filterNotifications]
  );
  const filteredReadNotifications = useMemo(
    () => filterNotifications(readNotifications),
    [readNotifications, filterNotifications]
  );

  // 获取偏好值
  const getPreferenceValue = (eventType: string, field: 'email' | 'in_app' | 'sms'): boolean => {
    const pref = preferences.find(p => p.eventType === eventType);
    if (!pref) return field === 'in_app';
    return pref[`${field}_enabled` as keyof NotificationPreferenceItem] as boolean;
  };

  // 获取通知类型标签
  const getNotificationTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      created: t('notifications.ticketCreated'),
      assigned: t('notifications.ticketAssigned'),
      statusChanged: t('notifications.ticketUpdated'),
      commented: t('notifications.ticketCommented'),
      slaWarning: t('notifications.slaWarning'),
      resolved: t('notifications.ticketResolved'),
      closed: t('notifications.ticketClosed'),
    };
    return labels[type] || type;
  };

  // 获取渠道图标
  const getChannelIcon = (channel: string) => {
    const config = CHANNELS.find(c => c.key === channel);
    const IconComponent = config?.icon || MessageSquare;
    return <IconComponent className="w-4 h-4" />;
  };

  // 获取渠道颜色
  const getChannelColor = (channel: string) => {
    const config = CHANNELS.find(c => c.key === channel);
    return config?.color || 'default';
  };

  // 格式化时间
  const formatDateTime = (dateString?: string) => {
    if (!dateString) return '';
    const date = dayjs(dateString);
    return date.format('YYYY-MM-DD HH:mm');
  };

  // 渲染通知列表
  const renderNotificationList = (notificationList: TicketNotification[]) => {
    if (notificationList.length === 0) {
      return (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={t('notifications.noNotifications')}
        />
      );
    }

    return (
      <List
        dataSource={notificationList}
        renderItem={notification => (
          <List.Item
            className={notification.status === 'read' ? 'opacity-70' : ''}
            actions={[
              notification.status !== 'read' && (
                <Button
                  key="read"
                  type="link"
                  size="small"
                  icon={<Eye className="w-4 h-4" />}
                  onClick={() => handleMarkRead(notification.id)}
                >
                  {t('notifications.markRead')}
                </Button>
              ),
              <Popconfirm
                key="delete"
                title={t('notifications.delete')}
                description={t('notifications.deleteRead')}
                onConfirm={() => handleDeleteNotification(notification.id)}
              >
                <Button type="link" size="small" danger icon={<Delete className="w-4 h-4" />}>
                  {t('notifications.delete')}
                </Button>
              </Popconfirm>,
            ].filter(Boolean)}
          >
            <List.Item.Meta
              avatar={
                <div className="flex items-center justify-center w-10 h-10 rounded-full bg-blue-100">
                  <Bell className="w-5 h-5 text-blue-600" />
                </div>
              }
              title={
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <Text strong={notification.status !== 'read'}>
                      {getNotificationTypeLabel(notification.type)}
                    </Text>
                    {notification.status !== 'read' && <Badge status="processing" />}
                    <Tag
                      color={getChannelColor(notification.channel)}
                      icon={getChannelIcon(notification.channel)}
                    >
                      {notification.channel === 'email'
                        ? t('notifications.email')
                        : notification.channel === 'sms'
                          ? t('notifications.sms')
                          : t('notifications.inApp')}
                    </Tag>
                  </div>
                  <Text type="secondary" className="text-xs">
                    {formatDateTime(notification.createdAt)}
                  </Text>
                </div>
              }
              description={
                <div className="space-y-1">
                  <Text>{notification.content}</Text>
                  {notification.sentAt && (
                    <div className="text-xs text-gray-500">
                      {t('notifications.sentAt')}: {formatDateTime(notification.sentAt)}
                    </div>
                  )}
                  {notification.readAt && (
                    <div className="text-xs text-gray-500">
                      {t('notifications.readAt')}: {formatDateTime(notification.readAt)}
                    </div>
                  )}
                </div>
              }
            />
          </List.Item>
        )}
      />
    );
  };

  // 渲染筛选栏
  const renderFilterBar = () => (
    <Row gutter={[16, 16]} className="mb-4">
      <Col xs={24} md={6}>
        <Input
          placeholder={t('notifications.searchPlaceholder')}
          prefix={<Search className="w-4 h-4" />}
          value={searchText}
          onChange={e => setSearchText(e.target.value)}
          allowClear
        />
      </Col>
      <Col xs={24} md={5}>
        <Select
          placeholder={t('notifications.channel')}
          style={{ width: '100%' }}
          value={channelFilter}
          onChange={setChannelFilter}
          allowClear
          options={CHANNELS.map(channel => ({
            value: channel.key,
            label: (
              <Space>
                {getChannelIcon(channel.key)}
                {t(channel.nameKey)}
              </Space>
            ),
          }))}
        />
      </Col>
      <Col xs={24} md={5}>
        <Select
          placeholder={t('notifications.allTypes')}
          style={{ width: '100%' }}
          value={typeFilter}
          onChange={setTypeFilter}
          allowClear
          options={EVENT_TYPES.map(event => ({
            value: event.type,
            label: t(event.nameKey),
          }))}
        />
      </Col>
      <Col xs={24} md={6}>
        <RangePicker
          style={{ width: '100%' }}
          value={dateRange}
          onChange={(dates: any) => {
            if (dates && dates.length === 2) {
              setDateRange([dates[0], dates[1]]);
            } else {
              setDateRange(null);
            }
          }}
        />
      </Col>
      <Col xs={24} md={2}>
        <Button icon={<RefreshCw className="w-4 h-4" />} onClick={loadNotifications}>
          {t('workflow.refresh')}
        </Button>
      </Col>
    </Row>
  );

  const tabItems = [
    {
      key: 'all',
      label: (
        <span>
          <Bell className="mr-1 inline h-4 w-4" />
          {t('notifications.allNotifications')}
          {filteredUnreadNotifications.length > 0 && (
            <Badge count={filteredUnreadNotifications.length} className="ml-2" />
          )}
        </span>
      ),
      children: (
        <Card>
          {renderFilterBar()}
          <div className="mb-4 flex items-center justify-between">
            <Text>
              {t('notifications.totalNotifications', { total: filteredNotifications.length })},
              {t('notifications.unreadCount', { count: filteredUnreadNotifications.length })}
            </Text>
            <Space>
              {filteredUnreadNotifications.length > 0 && (
                <Button onClick={handleMarkAllRead}>{t('notifications.markAllRead')}</Button>
              )}
              {filteredNotifications.length > 0 && (
                <Popconfirm
                  title={t('notifications.clearAll')}
                  description={t('notifications.deleteRead')}
                  onConfirm={handleClearAll}
                >
                  <Button danger icon={<Delete className="h-4 w-4" />}>
                    {t('notifications.clearAll')}
                  </Button>
                </Popconfirm>
              )}
            </Space>
          </div>
          <Spin spinning={loading}>{renderNotificationList(filteredNotifications)}</Spin>
        </Card>
      ),
    },
    {
      key: 'unread',
      label: (
        <span>
          <Clock className="mr-1 inline h-4 w-4" />
          {t('notifications.unread')}
          {filteredUnreadNotifications.length > 0 && (
            <Badge count={filteredUnreadNotifications.length} className="ml-2" />
          )}
        </span>
      ),
      children: (
        <Card>
          {renderFilterBar()}
          <Spin spinning={loading}>{renderNotificationList(filteredUnreadNotifications)}</Spin>
        </Card>
      ),
    },
    {
      key: 'read',
      label: (
        <span>
          <CheckCircle className="mr-1 inline h-4 w-4" />
          {t('notifications.read')}
        </span>
      ),
      children: (
        <Card>
          {renderFilterBar()}
          <Spin spinning={loading}>{renderNotificationList(filteredReadNotifications)}</Spin>
        </Card>
      ),
    },
    {
      key: 'preferences',
      label: (
        <span>
          <Settings className="mr-1 inline h-4 w-4" />
          {t('notifications.preferences')}
        </span>
      ),
      children: (
        <Card>
          <div className="mb-4 flex items-center justify-between">
            <Title level={4}>{t('notifications.preferences')}</Title>
            <Button icon={<RotateCcw className="h-4 w-4" />} onClick={handleResetPreferences}>
              {t('notifications.resetDefault')}
            </Button>
          </div>
          <Spin spinning={preferencesLoading}>
            <Form form={form} layout="vertical" onFinish={handleSavePreferences}>
              <Row gutter={[16, 16]}>
                <Col xs={24} md={8}>
                  <Text strong>{t('notifications.eventTypes')}</Text>
                </Col>
                <Col xs={24} md={5}>
                  <Text strong className="flex items-center">
                    <Mail className="mr-1 h-4 w-4" /> {t('notifications.emailEnabled')}
                  </Text>
                </Col>
                <Col xs={24} md={5}>
                  <Text strong className="flex items-center">
                    <MessageSquare className="mr-1 h-4 w-4" /> {t('notifications.inAppEnabled')}
                  </Text>
                </Col>
                <Col xs={24} md={6}>
                  <Text strong className="flex items-center">
                    <Smartphone className="mr-1 h-4 w-4" /> {t('notifications.smsEnabled')}
                  </Text>
                </Col>
              </Row>
              <Divider />
              {EVENT_TYPES.map(event => (
                <Row key={event.type} gutter={[16, 16]} align="middle" className="mb-2">
                  <Col xs={24} md={8}>
                    <div>
                      <Text>{t(event.nameKey)}</Text>
                      <br />
                      <Text type="secondary" className="text-xs">
                        {t(event.descKey)}
                      </Text>
                    </div>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={`${event.type}_email`} valuePropName="checked" noStyle>
                      <Switch
                        checkedChildren={t('workflow.statusEnabled')}
                        unCheckedChildren={t('workflow.statusDisabled')}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={`${event.type}_in_app`} valuePropName="checked" noStyle>
                      <Switch
                        checkedChildren={t('workflow.statusEnabled')}
                        unCheckedChildren={t('workflow.statusDisabled')}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={6}>
                    <Form.Item name={`${event.type}_sms`} valuePropName="checked" noStyle>
                      <Switch
                        checkedChildren={t('workflow.statusEnabled')}
                        unCheckedChildren={t('workflow.statusDisabled')}
                      />
                    </Form.Item>
                  </Col>
                </Row>
              ))}

              <Divider />

              <Form.Item>
                <Button type="primary" htmlType="submit">
                  {t('notifications.saveSettings')}
                </Button>
              </Form.Item>
            </Form>
          </Spin>
        </Card>
      ),
    },
  ];

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <Title level={2}>{t('notifications.title')}</Title>
        <Space>
          <Tooltip
            title={wsConnected ? t('notifications.wsConnected') : t('notifications.wsDisconnected')}
          >
            {wsConnected ? (
              <Wifi className="h-5 w-5 text-green-500" />
            ) : (
              <WifiOff className="h-5 w-5 text-red-500" />
            )}
          </Tooltip>
          <Button icon={<RefreshCw className="h-4 w-4" />} onClick={loadNotifications}>
            {t('workflow.refresh')}
          </Button>
        </Space>
      </div>

      <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} />
    </div>
  );
}
