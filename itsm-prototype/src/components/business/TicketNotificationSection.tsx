'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  List,
  Button,
  Space,
  Typography,
  Tag,
  Empty,
  Badge,
  Modal,
  Form,
  Select,
  Input,
  message,
  Tooltip,
  Divider,
} from 'antd';
import {
  Bell,
  Mail,
  MessageSquare,
  Smartphone,
  Send,
  Eye,
  CheckCircle,
  Clock,
  AlertCircle,
} from 'lucide-react';
import { TicketNotificationApi, TicketNotification, SendTicketNotificationRequest } from '@/lib/api/ticket-notification-api';
import { UserSelect } from '@/components/common/UserSelect';
import { useAuthStore } from '@/lib/store/auth-store';
import { App } from 'antd';

const { Text, Title } = Typography;
const { TextArea } = Input;
const { Option } = Select;

interface TicketNotificationSectionProps {
  ticketId: number;
  canSend?: boolean;
  onNotificationSent?: (notification: TicketNotification) => void;
}

/**
 * 工单通知管理组件
 */
export const TicketNotificationSection: React.FC<TicketNotificationSectionProps> = ({
  ticketId,
  canSend = true,
  onNotificationSent,
}) => {
  const { message: antMessage } = App.useApp();
  const { user } = useAuthStore();
  const [notifications, setNotifications] = useState<TicketNotification[]>([]);
  const [loading, setLoading] = useState(false);
  const [sendModalVisible, setSendModalVisible] = useState(false);
  const [form] = Form.useForm();

  // 加载通知列表
  const loadNotifications = async () => {
    setLoading(true);
    try {
      const response = await TicketNotificationApi.getTicketNotifications(ticketId);
      setNotifications(response.notifications || []);
    } catch (error) {
      console.error('Failed to load notifications:', error);
      antMessage.error('加载通知列表失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (ticketId) {
      loadNotifications();
    }
  }, [ticketId]);

  // 发送通知
  const handleSendNotification = async (values: {
    user_ids: number[];
    type: string;
    channel: 'email' | 'in_app' | 'sms';
    content: string;
  }) => {
    try {
      const request: SendTicketNotificationRequest = {
        user_ids: values.user_ids,
        type: values.type,
        channel: values.channel,
        content: values.content,
      };
      await TicketNotificationApi.sendTicketNotification(ticketId, request);
      antMessage.success('通知发送成功');
      setSendModalVisible(false);
      form.resetFields();
      await loadNotifications();
      // 触发回调
      if (onNotificationSent) {
        // 创建一个临时通知对象用于回调
        const tempNotification: TicketNotification = {
          id: Date.now(),
          ticket_id: ticketId,
          user_id: values.user_ids[0],
          type: values.type as any,
          channel: values.channel,
          content: values.content,
          status: 'sent',
          created_at: new Date().toISOString(),
        };
        onNotificationSent(tempNotification);
      }
    } catch (error: any) {
      console.error('Failed to send notification:', error);
      antMessage.error(error.message || '通知发送失败');
    }
  };

  // 标记为已读
  const handleMarkRead = async (notificationId: number) => {
    try {
      await TicketNotificationApi.markNotificationRead(notificationId);
      antMessage.success('已标记为已读');
      await loadNotifications();
    } catch (error: any) {
      console.error('Failed to mark notification as read:', error);
      antMessage.error(error.message || '标记失败');
    }
  };

  // 获取通知类型标签
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

  // 获取通知类型颜色
  const getNotificationTypeColor = (type: string) => {
    const colors: Record<string, string> = {
      created: 'blue',
      assigned: 'cyan',
      status_changed: 'orange',
      commented: 'purple',
      sla_warning: 'red',
      resolved: 'green',
      closed: 'default',
    };
    return colors[type] || 'default';
  };

  // 获取渠道图标
  const getChannelIcon = (channel: string) => {
    switch (channel) {
      case 'email':
        return <Mail style={{ fontSize: 16 }} />;
      case 'sms':
        return <Smartphone style={{ fontSize: 16 }} />;
      default:
        return <MessageSquare style={{ fontSize: 16 }} />;
    }
  };

  // 获取渠道颜色
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

  // 格式化时间
  const formatDateTime = (dateString?: string) => {
    if (!dateString) return '';
    return new Date(dateString).toLocaleString('zh-CN');
  };

  // 未读通知数量
  const unreadCount = notifications.filter(n => n.status !== 'read').length;

  return (
    <div className="space-y-4">
      {/* 操作栏 */}
      <div className="flex justify-between items-center">
        <Space>
          <Title level={5} style={{ margin: 0 }}>
            通知历史
          </Title>
          {unreadCount > 0 && (
            <Badge count={unreadCount} showZero>
              <Bell style={{ fontSize: 20 }} />
            </Badge>
          )}
        </Space>
        {canSend && (
          <Button
            type="primary"
            icon={<Send />}
            onClick={() => setSendModalVisible(true)}
          >
            发送通知
          </Button>
        )}
      </div>

      {/* 通知列表 */}
      <Card loading={loading}>
        {notifications.length === 0 ? (
          <Empty description="暂无通知" />
        ) : (
          <List
            dataSource={notifications}
            renderItem={(notification) => (
              <List.Item
                className={notification.status === 'read' ? 'opacity-70' : ''}
                actions={[
                  notification.status !== 'read' && (
                    <Tooltip title="标记为已读" key="read">
                      <Button
                        type="text"
                        size="small"
                        icon={<Eye />}
                        onClick={() => handleMarkRead(notification.id)}
                      >
                        标记已读
                      </Button>
                    </Tooltip>
                  ),
                ].filter(Boolean)}
              >
                <List.Item.Meta
                  avatar={
                    <div className="flex items-center justify-center w-10 h-10 rounded-full bg-blue-50">
                      {notification.status === 'read' ? (
                        <CheckCircle style={{ fontSize: 20, color: '#52c41a' }} />
                      ) : (
                        <Bell style={{ fontSize: 20, color: '#1890ff' }} />
                      )}
                    </div>
                  }
                  title={
                    <Space>
                      <Tag color={getNotificationTypeColor(notification.type)}>
                        {getNotificationTypeLabel(notification.type)}
                      </Tag>
                      <Tag color={getChannelColor(notification.channel)} icon={getChannelIcon(notification.channel)}>
                        {notification.channel === 'email' ? '邮件' : notification.channel === 'sms' ? '短信' : '站内消息'}
                      </Tag>
                      {notification.status !== 'read' && (
                        <Badge status="processing" text="未读" />
                      )}
                    </Space>
                  }
                  description={
                    <div className="space-y-1">
                      <Text>{notification.content}</Text>
                      <div className="flex items-center space-x-4 text-xs text-gray-500">
                        <span>
                          <Clock style={{ fontSize: 12, marginRight: 4 }} />
                          创建时间: {formatDateTime(notification.created_at)}
                        </span>
                        {notification.sent_at && (
                          <span>
                            <Send style={{ fontSize: 12, marginRight: 4 }} />
                            发送时间: {formatDateTime(notification.sent_at)}
                          </span>
                        )}
                        {notification.read_at && (
                          <span>
                            <Eye style={{ fontSize: 12, marginRight: 4 }} />
                            阅读时间: {formatDateTime(notification.read_at)}
                          </span>
                        )}
                        {notification.user && (
                          <span>
                            接收人: {notification.user.name || notification.user.username}
                          </span>
                        )}
                      </div>
                    </div>
                  }
                />
              </List.Item>
            )}
          />
        )}
      </Card>

      {/* 发送通知模态框 */}
      <Modal
        title="发送通知"
        open={sendModalVisible}
        onOk={() => form.submit()}
        onCancel={() => {
          setSendModalVisible(false);
          form.resetFields();
        }}
        okText="发送"
        cancelText="取消"
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSendNotification}
          initialValues={{
            channel: 'in_app',
            type: 'commented',
          }}
        >
          <Form.Item
            label="接收人"
            name="user_ids"
            rules={[{ required: true, message: '请选择接收人' }]}
          >
            <UserSelect
              mode="multiple"
              placeholder="请选择接收人"
            />
          </Form.Item>

          <Form.Item
            label="通知类型"
            name="type"
            rules={[{ required: true, message: '请选择通知类型' }]}
          >
            <Select placeholder="请选择通知类型">
              <Option value="created">工单创建</Option>
              <Option value="assigned">工单分配</Option>
              <Option value="status_changed">状态变更</Option>
              <Option value="commented">新增评论</Option>
              <Option value="sla_warning">SLA警告</Option>
              <Option value="resolved">工单已解决</Option>
              <Option value="closed">工单已关闭</Option>
            </Select>
          </Form.Item>

          <Form.Item
            label="通知渠道"
            name="channel"
            rules={[{ required: true, message: '请选择通知渠道' }]}
          >
            <Select placeholder="请选择通知渠道">
              <Option value="in_app">
                <Space>
                  <MessageSquare />
                  站内消息
                </Space>
              </Option>
              <Option value="email">
                <Space>
                  <Mail />
                  邮件
                </Space>
              </Option>
              <Option value="sms">
                <Space>
                  <Smartphone />
                  短信
                </Space>
              </Option>
            </Select>
          </Form.Item>

          <Form.Item
            label="通知内容"
            name="content"
            rules={[{ required: true, message: '请输入通知内容' }]}
          >
            <TextArea
              rows={4}
              placeholder="请输入通知内容..."
              showCount
              maxLength={500}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

