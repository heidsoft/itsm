'use client';

import React, { useState, useEffect } from 'react';
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
  message,
  Spin,
  Divider,
  Row,
  Col,
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
} from 'lucide-react';
import {
  TicketNotificationApi,
  TicketNotification,
  NotificationPreferencesResponse,
  NotificationPreferenceItem,
} from '@/lib/api/ticket-notification-api';
import { useAuthStore } from '@/lib/store/auth-store';

const { Title, Text } = Typography;
const { TabPane } = Tabs;

// 通知事件类型配置
const EVENT_TYPES = [
  { type: 'ticket_created', name: '工单创建', description: '当有新工单创建时' },
  { type: 'ticket_assigned', name: '工单分配', description: '当工单分配给您时' },
  { type: 'ticket_updated', name: '工单更新', description: '当工单状态或信息更新时' },
  { type: 'ticket_commented', name: '新增评论', description: '当工单有新评论时' },
  { type: 'sla_warning', name: 'SLA警告', description: '当工单即将超出SLA时限时' },
  { type: 'sla_violated', name: 'SLA违规', description: '当工单超出SLA时限时' },
  { type: 'ticket_resolved', name: '工单已解决', description: '当工单被标记为已解决时' },
  { type: 'ticket_closed', name: '工单已关闭', description: '当工单被关闭时' },
  { type: 'approval_required', name: '待审批', description: '当有待审批的工单时' },
  { type: 'approval_completed', name: '审批完成', description: '当审批结果出来时' },
];

export default function NotificationsPage() {
  const { user } = useAuthStore();
  const [notifications, setNotifications] = useState<TicketNotification[]>([]);
  const [unreadNotifications, setUnreadNotifications] = useState<TicketNotification[]>([]);
  const [readNotifications, setReadNotifications] = useState<TicketNotification[]>([]);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState('all');
  const [preferences, setPreferences] = useState<NotificationPreferenceItem[]>([]);
  const [preferencesLoading, setPreferencesLoading] = useState(false);
  const [form] = Form.useForm();

  useEffect(() => {
    loadNotifications();
    loadPreferences();
  }, [user]);

  const loadNotifications = async () => {
    setLoading(true);
    try {
      const [allRes, unreadRes, readRes] = await Promise.all([
        TicketNotificationApi.getUserNotifications({ page: 1, page_size: 100 }),
        TicketNotificationApi.getUserNotifications({ page: 1, page_size: 100, read: false }),
        TicketNotificationApi.getUserNotifications({ page: 1, page_size: 100, read: true }),
      ]);

      setNotifications(allRes.notifications || []);
      setUnreadNotifications(unreadRes.notifications || []);
      setReadNotifications(readRes.notifications || []);
    } catch (error) {
      message.error('加载通知失败');
      console.error('Failed to load notifications:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadPreferences = async () => {
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
      message.error('加载通知偏好失败');
      console.error('Failed to load preferences:', error);
    } finally {
      setPreferencesLoading(false);
    }
  };

  const handleMarkRead = async (notificationId: number) => {
    try {
      await TicketNotificationApi.markNotificationRead(notificationId);
      message.success('已标记为已读');
      loadNotifications();
    } catch (error) {
      message.error('标记失败');
      console.error('Failed to mark notification as read:', error);
    }
  };

  const handleMarkAllRead = async () => {
    try {
      await TicketNotificationApi.markAllNotificationsRead();
      message.success('已全部标记为已读');
      loadNotifications();
    } catch (error) {
      message.error('标记失败');
      console.error('Failed to mark all notifications as read:', error);
    }
  };

  const handleSavePreferences = async () => {
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
      message.success('通知偏好已保存');
      loadPreferences();
    } catch (error) {
      message.error('保存失败');
      console.error('Failed to update preferences:', error);
    }
  };

  const handleResetPreferences = async () => {
    try {
      await TicketNotificationApi.resetNotificationPreferences();
      message.success('已重置为默认偏好');
      loadPreferences();
    } catch (error) {
      message.error('重置失败');
      console.error('Failed to reset preferences:', error);
    }
  };

  const getPreferenceValue = (eventType: string, field: 'email' | 'in_app' | 'sms'): boolean => {
    const pref = preferences.find(p => p.event_type === eventType);
    if (!pref) return field === 'in_app'; // 默认开启站内消息
    return pref[`${field}_enabled` as keyof NotificationPreferenceItem] as boolean;
  };

  const getNotificationTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      created: '工单创建',
      assigned: '工单分配',
      status_changed: '状态变更',
      commented: '新增评论',
      sla_warning: 'SLA警告',
      resolved: '工单已解决',
      closed: '工单已关闭',
    };
    return labels[type] || type;
  };

  const getChannelIcon = (channel: string) => {
    switch (channel) {
      case 'email':
        return <Mail className='w-4 h-4' />;
      case 'sms':
        return <Smartphone className='w-4 h-4' />;
      default:
        return <MessageSquare className='w-4 h-4' />;
    }
  };

  const getChannelColor = (channel: string) => {
    switch (channel) {
      case 'email':
        return 'blue';
      case 'sms':
        return 'green';
      default:
        return 'default';
    }
  };

  const formatDateTime = (dateString?: string) => {
    if (!dateString) return '';
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const renderNotificationList = (notificationList: TicketNotification[]) => {
    if (notificationList.length === 0) {
      return (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description='暂无通知'
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
                  key='read'
                  type='link'
                  size='small'
                  icon={<Eye className='w-4 h-4' />}
                  onClick={() => handleMarkRead(notification.id)}
                >
                  标记已读
                </Button>
              ),
            ].filter(Boolean)}
          >
            <List.Item.Meta
              avatar={
                <div className='flex items-center justify-center w-10 h-10 rounded-full bg-blue-100'>
                  <Bell className='w-5 h-5 text-blue-600' />
                </div>
              }
              title={
                <div className='flex items-center justify-between'>
                  <div className='flex items-center space-x-2'>
                    <Text strong={notification.status !== 'read'}>
                      {getNotificationTypeLabel(notification.type)}
                    </Text>
                    {notification.status !== 'read' && (
                      <Badge status='processing' />
                    )}
                    <Tag color={getChannelColor(notification.channel)} icon={getChannelIcon(notification.channel)}>
                      {notification.channel === 'email' ? '邮件' : notification.channel === 'sms' ? '短信' : '站内消息'}
                    </Tag>
                  </div>
                  <Text type='secondary' className='text-xs'>
                    {formatDateTime(notification.created_at)}
                  </Text>
                </div>
              }
              description={
                <div className='space-y-1'>
                  <Text>{notification.content}</Text>
                  {notification.sent_at && (
                    <div className='text-xs text-gray-500'>
                      发送时间: {formatDateTime(notification.sent_at)}
                    </div>
                  )}
                  {notification.read_at && (
                    <div className='text-xs text-gray-500'>
                      阅读时间: {formatDateTime(notification.read_at)}
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

  return (
    <div className='p-6'>
      <div className='mb-6'>
        <Title level={2}>通知中心</Title>
      </div>

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane
          tab={
            <span>
              <Bell className='w-4 h-4 inline mr-1' />
              全部通知
              {unreadNotifications.length > 0 && (
                <Badge count={unreadNotifications.length} className='ml-2' />
              )}
            </span>
          }
          key='all'
        >
          <Card>
            <div className='mb-4 flex justify-between items-center'>
              <Text>
                共 {notifications.length} 条通知，未读 {unreadNotifications.length} 条
              </Text>
              {unreadNotifications.length > 0 && (
                <Button onClick={handleMarkAllRead}>
                  全部标记已读
                </Button>
              )}
            </div>
            <Spin spinning={loading}>
              {renderNotificationList(notifications)}
            </Spin>
          </Card>
        </TabPane>

        <TabPane
          tab={
            <span>
              <Clock className='w-4 h-4 inline mr-1' />
              未读
              {unreadNotifications.length > 0 && (
                <Badge count={unreadNotifications.length} className='ml-2' />
              )}
            </span>
          }
          key='unread'
        >
          <Card>
            <Spin spinning={loading}>
              {renderNotificationList(unreadNotifications)}
            </Spin>
          </Card>
        </TabPane>

        <TabPane
          tab={
            <span>
              <CheckCircle className='w-4 h-4 inline mr-1' />
              已读
            </span>
          }
          key='read'
        >
          <Card>
            <Spin spinning={loading}>
              {renderNotificationList(readNotifications)}
            </Spin>
          </Card>
        </TabPane>

        <TabPane
          tab={
            <span>
              <Settings className='w-4 h-4 inline mr-1' />
              通知偏好
            </span>
          }
          key='preferences'
        >
          <Card>
            <div className='flex justify-between items-center mb-4'>
              <Title level={4}>通知偏好设置</Title>
              <Button
                icon={<RotateCcw className='w-4 h-4' />}
                onClick={handleResetPreferences}
              >
                重置为默认
              </Button>
            </div>
            <Spin spinning={preferencesLoading}>
              <Form
                form={form}
                layout='vertical'
                onFinish={handleSavePreferences}
              >
                <Row gutter={[16, 16]}>
                  <Col xs={24} md={8}>
                    <Text strong>事件类型</Text>
                  </Col>
                  <Col xs={24} md={5}>
                    <Text strong className='flex items-center'>
                      <Mail className='w-4 h-4 mr-1' /> 邮件
                    </Text>
                  </Col>
                  <Col xs={24} md={5}>
                    <Text strong className='flex items-center'>
                      <MessageSquare className='w-4 h-4 mr-1' /> 站内消息
                    </Text>
                  </Col>
                  <Col xs={24} md={6}>
                    <Text strong className='flex items-center'>
                      <Smartphone className='w-4 h-4 mr-1' /> 短信
                    </Text>
                  </Col>
                </Row>
                <Divider />
                {EVENT_TYPES.map(event => (
                  <Row key={event.type} gutter={[16, 16]} align='middle' className='mb-2'>
                    <Col xs={24} md={8}>
                      <div>
                        <Text>{event.name}</Text>
                        <br />
                        <Text type='secondary' className='text-xs'>
                          {event.description}
                        </Text>
                      </div>
                    </Col>
                    <Col xs={24} md={5}>
                      <Form.Item name={`${event.type}_email`} valuePropName='checked' noStyle>
                        <Switch checkedChildren='开' unCheckedChildren='关' />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={5}>
                      <Form.Item name={`${event.type}_in_app`} valuePropName='checked' noStyle>
                        <Switch checkedChildren='开' unCheckedChildren='关' />
                      </Form.Item>
                    </Col>
                    <Col xs={24} md={6}>
                      <Form.Item name={`${event.type}_sms`} valuePropName='checked' noStyle>
                        <Switch checkedChildren='开' unCheckedChildren='关' />
                      </Form.Item>
                    </Col>
                  </Row>
                ))}

                <Divider />

                <Form.Item>
                  <Button type='primary' htmlType='submit'>
                    保存设置
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

