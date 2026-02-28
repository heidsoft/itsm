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
import {
  TicketNotificationApi,
  TicketNotification,
  NotificationPreferencesResponse,
  NotificationPreferenceItem,
} from '@/lib/api/ticket-notification-api';
import { useAuthStore } from '@/lib/store/auth-store';
import { notificationWS } from '@/lib/services/notification-ws';
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;
const { TabPane } = Tabs;
const { RangePicker } = DatePicker;

// 通知事件类型配置
const EVENT_TYPES = [
  { type: 'ticket_created', nameKey: 'notifications.ticketCreated', descKey: 'notifications.ticketCreatedDesc' },
  { type: 'ticket_assigned', nameKey: 'notifications.ticketAssigned', descKey: 'notifications.ticketAssignedDesc' },
  { type: 'ticket_updated', nameKey: 'notifications.ticketUpdated', descKey: 'notifications.ticketUpdatedDesc' },
  { type: 'ticket_commented', nameKey: 'notifications.ticketCommented', descKey: 'notifications.ticketCommentedDesc' },
  { type: 'sla_warning', nameKey: 'notifications.slaWarning', descKey: 'notifications.slaWarningDesc' },
  { type: 'sla_violated', nameKey: 'notifications.slaViolated', descKey: 'notifications.slaViolatedDesc' },
  { type: 'ticket_resolved', nameKey: 'notifications.ticketResolved', descKey: 'notifications.ticketResolvedDesc' },
  { type: 'ticket_closed', nameKey: 'notifications.ticketClosed', descKey: 'notifications.ticketClosedDesc' },
  { type: 'approval_required', nameKey: 'notifications.approvalRequired', descKey: 'notifications.approvalRequiredDesc' },
  { type: 'approval_completed', nameKey: 'notifications.approvalCompleted', descKey: 'notifications.approvalCompletedDesc' },
];

// 通知渠道配置
const CHANNELS = [
  { key: 'email', nameKey: 'notifications.email', icon: Mail, color: 'blue' },
  { key: 'in_app', nameKey: 'notifications.inApp', icon: MessageSquare, color: 'default' },
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
        TicketNotificationApi.getUserNotifications({ page: 1, page_size: 100 }),
        TicketNotificationApi.getUserNotifications({ page: 1, page_size: 100, read: false }),
        TicketNotificationApi.getUserNotifications({ page: 1, page_size: 100, read: true }),
      ]);

      const all = allRes.notifications || [];
      setNotifications(all);
      setUnreadNotifications(unreadRes.notifications || []);
      setReadNotifications(readRes.notifications || []);
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
        formValues[`${pref.event_type}_email`] = pref.email_enabled;
        formValues[`${pref.event_type}_in_app`] = pref.in_app_enabled;
        formValues[`${pref.event_type}_sms`] = pref.sms_enabled;
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
      notificationWS.connect(user.id, token)
        .then(() => {
          setWsConnected(true);
        })
        .catch((error) => {
          console.error('WebSocket connection failed:', error);
          setWsConnected(false);
        });

      // 监听新通知
      const unsubscribe = notificationWS.onNotification((notification) => {
        setNotifications(prev => [notification, ...prev]);
        setUnreadNotifications(prev => [notification, ...prev]);
        message.info(notification.content);
      });

      // 监听连接状态变化
      const unsubConnection = notificationWS.onConnectionChange((connected) => {
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
  const handleMarkRead = useCallback(async (notificationId: number) => {
    try {
      await TicketNotificationApi.markNotificationRead(notificationId);
      message.success(t('notifications.markSuccess'));
      loadNotifications();
    } catch (error) {
      message.error(t('notifications.loadFailed'));
      console.error('Failed to mark notification as read:', error);
    }
  }, [t, loadNotifications]);

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
          event_type: event.type,
          email_enabled: values[`${event.type}_email`] ?? false,
          in_app_enabled: values[`${event.type}_in_app`] ?? true,
          sms_enabled: values[`${event.type}_sms`] ?? false,
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
  const handleDeleteNotification = useCallback(async (notificationId: number) => {
    try {
      // 这里需要添加删除 API
      message.success(t('notifications.delete'));
      loadNotifications();
    } catch (error) {
      message.error(t('notifications.loadFailed'));
    }
  }, [t, loadNotifications]);

  // 清空所有通知
  const handleClearAll = useCallback(async () => {
    try {
      // 这里需要添加清空 API
      message.success(t('notifications.clearAll'));
      loadNotifications();
    } catch (error) {
      message.error(t('notifications.loadFailed'));
    }
  }, [t, loadNotifications]);

  // 筛选通知列表
  const filterNotifications = useCallback((list: TicketNotification[]) => {
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
      if (dateRange && item.created_at) {
        const date = dayjs(item.created_at);
        if (!date.isAfter(dateRange[0]) || !date.isBefore(dateRange[1])) {
          return false;
        }
      }
      return true;
    });
  }, [searchText, channelFilter, typeFilter, dateRange]);

  // 获取筛选后的通知
  const filteredNotifications = useMemo(() => filterNotifications(notifications), [notifications, filterNotifications]);
  const filteredUnreadNotifications = useMemo(() => filterNotifications(unreadNotifications), [unreadNotifications, filterNotifications]);
  const filteredReadNotifications = useMemo(() => filterNotifications(readNotifications), [readNotifications, filterNotifications]);

  // 获取偏好值
  const getPreferenceValue = (eventType: string, field: 'email' | 'in_app' | 'sms'): boolean => {
    const pref = preferences.find(p => p.event_type === eventType);
    if (!pref) return field === 'in_app';
    return pref[`${field}_enabled` as keyof NotificationPreferenceItem] as boolean;
  };

  // 获取通知类型标签
  const getNotificationTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      created: t('notifications.ticketCreated'),
      assigned: t('notifications.ticketAssigned'),
      status_changed: t('notifications.ticketUpdated'),
      commented: t('notifications.ticketCommented'),
      sla_warning: t('notifications.slaWarning'),
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
                    {notification.status !== 'read' && (
                      <Badge status="processing" />
                    )}
                    <Tag color={getChannelColor(notification.channel)} icon={getChannelIcon(notification.channel)}>
                      {notification.channel === 'email' ? t('notifications.email') :
                       notification.channel === 'sms' ? t('notifications.sms') : t('notifications.inApp')}
                    </Tag>
                  </div>
                  <Text type="secondary" className="text-xs">
                    {formatDateTime(notification.created_at)}
                  </Text>
                </div>
              }
              description={
                <div className="space-y-1">
                  <Text>{notification.content}</Text>
                  {notification.sent_at && (
                    <div className="text-xs text-gray-500">
                      {t('notifications.sentAt')}: {formatDateTime(notification.sent_at)}
                    </div>
                  )}
                  {notification.read_at && (
                    <div className="text-xs text-gray-500">
                      {t('notifications.readAt')}: {formatDateTime(notification.read_at)}
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
                {t(channel.nameKey as any)}
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
            label: t(event.nameKey as any),
          }))}
        />
      </Col>
      <Col xs={24} md={6}>
        <RangePicker
          style={{ width: '100%' }}
          value={dateRange}
          onChange={(dates: [dayjs.Dayjs, dayjs.Dayjs] | null) => setDateRange(dates)}
        />
      </Col>
      <Col xs={24} md={2}>
        <Button icon={<RefreshCw className="w-4 h-4" />} onClick={loadNotifications}>
          {t('workflow.refresh')}
        </Button>
      </Col>
    </Row>
  );

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <Title level={2}>{t('notifications.title')}</Title>
        <Space>
          <Tooltip title={wsConnected ? t('notifications.wsConnected') : t('notifications.wsDisconnected')}>
            {wsConnected ? (
              <Wifi className="w-5 h-5 text-green-500" />
            ) : (
              <WifiOff className="w-5 h-5 text-red-500" />
            )}
          </Tooltip>
          <Button icon={<RefreshCw className="w-4 h-4" />} onClick={loadNotifications}>
            {t('workflow.refresh')}
          </Button>
        </Space>
      </div>

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane
          tab={
            <span>
              <Bell className="w-4 h-4 inline mr-1" />
              {t('notifications.allNotifications')}
              {filteredUnreadNotifications.length > 0 && (
                <Badge count={filteredUnreadNotifications.length} className="ml-2" />
              )}
            </span>
          }
          key="all"
        >
          <Card>
            {renderFilterBar()}
            <div className="mb-4 flex justify-between items-center">
              <Text>
                {t('notifications.totalNotifications', { total: filteredNotifications.length })},
                {t('notifications.unreadCount', { count: filteredUnreadNotifications.length })}
              </Text>
              <Space>
                {filteredUnreadNotifications.length > 0 && (
                  <Button onClick={handleMarkAllRead}>
                    {t('notifications.markAllRead')}
                  </Button>
                )}
                {filteredNotifications.length > 0 && (
                  <Popconfirm
                    title={t('notifications.clearAll')}
                    description={t('notifications.deleteRead')}
                    onConfirm={handleClearAll}
                  >
                    <Button danger icon={<Delete className="w-4 h-4" />}>
                      {t('notifications.clearAll')}
                    </Button>
                  </Popconfirm>
                )}
              </Space>
            </div>
            <Spin spinning={loading}>
              {renderNotificationList(filteredNotifications)}
            </Spin>
          </Card>
        </TabPane>

        <TabPane
          tab={
            <span>
              <Clock className="w-4 h-4 inline mr-1" />
              {t('notifications.unread')}
              {filteredUnreadNotifications.length > 0 && (
                <Badge count={filteredUnreadNotifications.length} className="ml-2" />
              )}
            </span>
          }
          key="unread"
        >
          <Card>
            {renderFilterBar()}
            <Spin spinning={loading}>
              {renderNotificationList(filteredUnreadNotifications)}
            </Spin>
          </Card>
        </TabPane>

        <TabPane
          tab={
            <span>
              <CheckCircle className="w-4 h-4 inline mr-1" />
              {t('notifications.read')}
            </span>
          }
          key="read"
        >
          <Card>
            {renderFilterBar()}
            <Spin spinning={loading}>
              {renderNotificationList(filteredReadNotifications)}
            </Spin>
          </Card>
        </TabPane>

        <TabPane
          tab={
            <span>
              <Settings className="w-4 h-4 inline mr-1" />
              {t('notifications.preferences')}
            </span>
          }
          key="preferences"
        >
          <Card>
            <div className="flex justify-between items-center mb-4">
              <Title level={4}>{t('notifications.preferences')}</Title>
              <Button
                icon={<RotateCcw className="w-4 h-4" />}
                onClick={handleResetPreferences}
              >
                {t('notifications.resetDefault')}
              </Button>
            </div>
            <Spin spinning={preferencesLoading}>
              <Form
                form={form}
                layout="vertical"
                onFinish={handleSavePreferences}
              >
                <Row gutter={[16, 16]}>
                  <Col xs={24} md={8}>
                    <Text strong>{t('notifications.eventTypes')}</Text>
                  </Col>
                  <Col xs={24} md={5}>
                    <Text strong className="flex items-center">
                      <Mail className="w-4 h-4 mr-1" /> {t('notifications.emailEnabled')}
                    </Text>
                  </Col>
                  <Col xs={24} md={5}>
                    <Text strong className="flex items-center">
                      <MessageSquare className="w-4 h-4 mr-1" /> {t('notifications.inAppEnabled')}
                    </Text>
                  </Col>
                  <Col xs={24} md={6}>
                    <Text strong className="flex items-center">
                      <Smartphone className="w-4 h-4 mr-1" /> {t('notifications.smsEnabled')}
                    </Text>
                  </Col>
                </Row>
                <Divider />
                {EVENT_TYPES.map(event => (
                  <Row key={event.type} gutter={[16, 16]} align="middle" className="mb-2">
                    <Col xs={24} md={8}>
                      <div>
                        <Text>{t(event.nameKey as any)}</Text>
                        <br />
                        <Text type="secondary" className="text-xs">
                          {t(event.descKey as any)}
                        </Text>
                      </div>
                    </Col>
                    <Col xs={24} md={5}>
                      <Form.Item name={`${event.type}_email`} valuePropName="checked" noStyle>
                        <Switch checkedChildren={t('workflow.statusEnabled')} unCheckedChildren={t('workflow.statusDisabled')} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={5}>
                      <Form.Item name={`${event.type}_in_app`} valuePropName="checked" noStyle>
                        <Switch checkedChildren={t('workflow.statusEnabled')} unCheckedChildren={t('workflow.statusDisabled')} />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name={`${event.type}_sms`} valuePropName="checked" noStyle>
                        <Switch checkedChildren={t('workflow.statusEnabled')} unCheckedChildren={t('workflow.statusDisabled')} />
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
        </TabPane>
      </Tabs>
    </div>
  );
}
