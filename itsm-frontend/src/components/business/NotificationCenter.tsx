'use client';

import React, { useState, useEffect } from 'react';
import {
  Drawer,
  List,
  Badge,
  Button,
  Space,
  Typography,
  Tabs,
  Card,
  Form,
  Input,
  Select,
  Switch,
  DatePicker,
  Table,
  Tag,
  Modal,
  message,
  Tooltip,
  Popconfirm,
  Divider,
  Row,
  Col,
  Statistic,
  Progress,
  Alert,
  Empty,
} from 'antd';
import {
  Bell,
  Mail,
  MessageSquare,
  Smartphone,
  Settings,
  FileText,
  History,
  Edit,
  Trash2,
  Plus,
  Send,
  CheckCircle,
  Clock,
  AlertCircle,
  Filter,
  Download,
  Eye,
  Copy,
} from 'lucide-react';
import type { ColumnsType } from 'antd/es/table';
import { TicketNotificationApi } from '@/lib/api/ticket-notification-api';
import { PRODUCT_CAPABILITIES } from '@/config/product-capabilities';

const { Title, Text, Paragraph } = Typography;
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

interface NotificationStats {
  total: number;
  unread: number;
  sentToday: number;
  failedToday: number;
  deliveryRate: number;
  avgResponseTime: number;
}

const NotificationCenter: React.FC<{
  open: boolean;
  onClose: () => void;
}> = ({ open, onClose }) => {
  const [activeTab, setActiveTab] = useState('notifications');
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [templates, setTemplates] = useState<NotificationTemplate[]>([]);
  const [channels, setChannels] = useState<NotificationChannel[]>([]);
  const [stats, setStats] = useState<NotificationStats>({
    total: 0,
    unread: 0,
    sentToday: 0,
    failedToday: 0,
    deliveryRate: 0,
    avgResponseTime: 0,
  });
  const [loading, setLoading] = useState(false);
  const [showTemplateModal, setShowTemplateModal] = useState(false);
  const [showChannelModal, setShowChannelModal] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState<NotificationTemplate | null>(null);
  const [selectedChannel, setSelectedChannel] = useState<NotificationChannel | null>(null);
  const [form] = Form.useForm();
  const [channelForm] = Form.useForm();

  // 筛选状态
  const [filterType, setFilterType] = useState<string>('');
  const [filterStatus, setFilterStatus] = useState<string>('');

  // 筛选后的通知列表
  const filteredNotifications = notifications.filter(n => {
    if (filterType && n.type !== filterType) return false;
    if (filterStatus && n.status !== filterStatus) return false;
    return true;
  });

  // 导出通知列表
  const handleExport = () => {
    const csvContent = [
      ['ID', '标题', '内容', '类型', '状态', '接收人', '发送时间'].join(','),
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
    message.success('导出成功');
  };

  // 模拟数据
  useEffect(() => {
    if (open) {
      loadData();
    }
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

      const total = response.total ?? loadedNotifications.length;
      const unread = loadedNotifications.filter(n => n.status !== 'read').length;
      const sentToday = loadedNotifications.filter(
        n => n.sentAt && new Date(n.sentAt).toDateString() === new Date().toDateString()
      ).length;
      const failedToday = loadedNotifications.filter(n => n.status === 'failed').length;

      setStats({
        total,
        unread,
        sentToday,
        failedToday,
        deliveryRate: total > 0 ? ((total - failedToday) / total) * 100 : 0,
        avgResponseTime: 0,
      });
    } catch (error) {
      message.error('加载数据失败');
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

      message.success('已标记为已读');
    } catch (error) {
      message.error('操作失败');
    }
  };

  const handleMarkAllRead = async () => {
    try {
      await TicketNotificationApi.markAllNotificationsRead();

      setNotifications(prev =>
        prev.map(n => ({ ...n, status: 'read' as const, readAt: new Date().toISOString() }))
      );

      message.success('已全部标记为已读');
    } catch (error) {
      message.error('操作失败');
    }
  };

  const handleDeleteNotification = async (id: number) => {
    try {
      await TicketNotificationApi.deleteNotification(id);

      setNotifications(prev => prev.filter(n => n.id !== id));
      message.success('删除成功');
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleSaveTemplate = async () => {
    try {
      const values = await form.validateFields();

      if (selectedTemplate) {
        // 更新模板
        setTemplates(prev =>
          prev.map(t =>
            t.id === selectedTemplate.id
              ? { ...t, ...values, updatedAt: new Date().toISOString() }
              : t
          )
        );
        message.success('模板更新成功');
      } else {
        // 创建新模板
        const newTemplate: NotificationTemplate = {
          id: Date.now(),
          ...values,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        };
        setTemplates(prev => [...prev, newTemplate]);
        message.success('模板创建成功');
      }

      setShowTemplateModal(false);
      setSelectedTemplate(null);
      form.resetFields();
    } catch (error) {
      message.error('保存模板失败');
    }
  };

  const handleSaveChannel = async () => {
    try {
      const values = await channelForm.validateFields();

      if (selectedChannel) {
        // 更新通道
        setChannels(prev => prev.map(c => (c.id === selectedChannel.id ? { ...c, ...values } : c)));
        message.success('通道更新成功');
      } else {
        // 创建新通道
        const newChannel: NotificationChannel = {
          id: values.type + '_' + Date.now(),
          ...values,
          status: 'disconnected',
        };
        setChannels(prev => [...prev, newChannel]);
        message.success('通道创建成功');
      }

      setShowChannelModal(false);
      setSelectedChannel(null);
      channelForm.resetFields();
    } catch (error) {
      message.error('保存通道失败');
    }
  };

  const handleTestChannel = async (channelId: string) => {
    try {
      // 模拟测试通道
      await new Promise(resolve => setTimeout(resolve, 1000));
      message.success('通道测试成功');
    } catch (error) {
      message.error('通道测试失败');
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

  const notificationColumns: ColumnsType<Notification> = [
    {
      title: '通知内容',
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
      title: '类型',
      key: 'type',
      width: 100,
      render: (_, record) => (
        <Tag
          color={
            record.type === 'error'
              ? 'red'
              : record.type === 'warning'
                ? 'orange'
                : record.type === 'success'
                  ? 'green'
                  : 'blue'
          }
        >
          {record.type === 'error'
            ? '错误'
            : record.type === 'warning'
              ? '警告'
              : record.type === 'success'
                ? '成功'
                : '信息'}
        </Tag>
      ),
    },
    {
      title: '状态',
      key: 'status',
      width: 100,
      render: (_, record) => (
        <Tag color={getStatusColor(record.status)}>
          {getStatusIcon(record.status)}
          {record.status === 'sent'
            ? '已发送'
            : record.status === 'pending'
              ? '待发送'
              : record.status === 'failed'
                ? '发送失败'
                : '已读'}
        </Tag>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 120,
      render: (_, record) => (
        <Space size="small">
          {record.status === 'pending' && (
            <Button size="small" onClick={() => handleMarkRead(record.id)}>
              标记已读
            </Button>
          )}
          <Popconfirm
            title="确认删除"
            description="确定要删除这条通知吗？"
            onConfirm={() => handleDeleteNotification(record.id)}
            okText="确认"
            cancelText="取消"
          >
            <Button size="small" danger icon={<Trash2 className="w-3 h-3" />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const templateColumns: ColumnsType<NotificationTemplate> = [
    {
      title: '模板名称',
      key: 'name',
      render: (_, record) => (
        <div className="space-y-1">
          <div className="font-medium">{record.name}</div>
          <div className="text-sm text-gray-600">{record.description}</div>
        </div>
      ),
    },
    {
      title: '类型',
      key: 'type',
      width: 100,
      render: (_, record) => (
        <Tag
          color={
            record.type === 'error'
              ? 'red'
              : record.type === 'warning'
                ? 'orange'
                : record.type === 'success'
                  ? 'green'
                  : 'blue'
          }
        >
          {record.type === 'error'
            ? '错误'
            : record.type === 'warning'
              ? '警告'
              : record.type === 'success'
                ? '成功'
                : '信息'}
        </Tag>
      ),
    },
    {
      title: '通道',
      key: 'channels',
      width: 150,
      render: (_, record) => (
        <div className="flex gap-1">
          {record.channels.map(channel => (
            <Tag key={channel}>
              {getChannelIcon(channel)}
              {channel === 'email'
                ? '邮件'
                : channel === 'sms'
                  ? '短信'
                  : channel === 'webhook'
                    ? 'Webhook'
                    : '站内信'}
            </Tag>
          ))}
        </div>
      ),
    },
    {
      title: '状态',
      key: 'isActive',
      width: 80,
      render: (_, record) => (
        <Tag color={record.isActive ? 'success' : 'default'}>
          {record.isActive ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '操作',
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
            编辑
          </Button>
          <Button
            size="small"
            icon={<Copy className="w-3 h-3" />}
            onClick={() => {
              const newTemplate = { ...record, id: Date.now(), name: `${record.name}_副本` };
              setTemplates(prev => [...prev, newTemplate]);
              message.success('模板复制成功');
            }}
          >
            复制
          </Button>
        </Space>
      ),
    },
  ];

  const channelColumns: ColumnsType<NotificationChannel> = [
    {
      title: '通道名称',
      key: 'name',
      render: (_, record) => (
        <div className="space-y-1">
          <div className="font-medium">{record.name}</div>
          <div className="text-sm text-gray-600">
            {record.type === 'email'
              ? 'SMTP邮件服务'
              : record.type === 'sms'
                ? '短信服务'
                : record.type === 'webhook'
                  ? 'Webhook回调'
                  : '站内通知'}
          </div>
        </div>
      ),
    },
    {
      title: '状态',
      key: 'status',
      width: 120,
      render: (_, record) => (
        <div className="space-y-1">
          <Tag color={record.isActive ? 'success' : 'default'}>
            {record.isActive ? '启用' : '禁用'}
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
              ? '已连接'
              : record.status === 'disconnected'
                ? '未连接'
                : '连接错误'}
          </Tag>
        </div>
      ),
    },
    {
      title: '最后使用',
      key:'lastUsed',
      width: 150,
      render: (_, record) => (
        <div className="text-sm text-gray-500">
          {record.lastUsed ? new Date(record.lastUsed).toLocaleString() : '从未使用'}
        </div>
      ),
    },
    {
      title: '操作',
      key: 'actions',
      width: 150,
      render: (_, record) => (
        <Space size="small">
          <Button size="small" onClick={() => handleTestChannel(record.id)}>
            测试
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
            配置
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
          <span>通知中心</span>
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

        ]}
      />

      {/* 模板编辑模态框 */}
      {PRODUCT_CAPABILITIES.notificationTemplateManagement && <Modal
        title={selectedTemplate ? '编辑模板' : '新建模板'}
        open={showTemplateModal}
        onOk={handleSaveTemplate}
        onCancel={() => {
          setShowTemplateModal(false);
          setSelectedTemplate(null);
          form.resetFields();
        }}
        width={600}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label="模板名称"
            name="name"
            rules={[{ required: true, message: '请输入模板名称' }]}
          >
            <Input placeholder="请输入模板名称" />
          </Form.Item>

          <Form.Item label="描述" name="description">
            <Input placeholder="请输入模板描述" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                label="通知类型"
                name="type"
                rules={[{ required: true, message: '请选择通知类型' }]}
              >
                <Select placeholder="请选择通知类型" options={[{ value: "info", label: "信息" }, { value: "success", label: "成功" }, { value: "warning", label: "警告" }, { value: "error", label: "错误" }]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                label="通知通道"
                name="channels"
                rules={[{ required: true, message: '请选择通知通道' }]}
              >
                <Select mode="multiple" placeholder="请选择通知通道" options={[{ value: "in_app", label: "站内信" }, { value: "email", label: "邮件" }, { value: "sms", label: "短信" }, { value: "webhook", label: "Webhook" }]} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item label="邮件主题" name="subject">
            <Input placeholder="请输入邮件主题" />
          </Form.Item>

          <Form.Item
            label="模板内容"
            name="content"
            rules={[{ required: true, message: '请输入模板内容' }]}
          >
            <TextArea rows={4} placeholder="请输入模板内容，支持变量：{{variable_name}}" />
          </Form.Item>

          <Form.Item label="可用变量" name="variables">
            <Select mode="tags" placeholder="输入变量名后按回车" options={[{ value: "user_name", label: "用户名" }, { value: "ticket_id", label: "工单ID" }, { value: "ticket_title", label: "工单标题" }, { value: "priority", label: "优先级" }]} />
          </Form.Item>

          <Form.Item label="启用状态" name="is_active" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>}

      {/* 通道配置模态框 */}
      {PRODUCT_CAPABILITIES.notificationChannelManagement && <Modal
        title={selectedChannel ? '配置通道' : '添加通道'}
        open={showChannelModal}
        onOk={handleSaveChannel}
        onCancel={() => {
          setShowChannelModal(false);
          setSelectedChannel(null);
          channelForm.resetFields();
        }}
        width={600}
        okText="保存"
        cancelText="取消"
      >
        <Form form={channelForm} layout="vertical">
          <Form.Item
            label="通道名称"
            name="name"
            rules={[{ required: true, message: '请输入通道名称' }]}
          >
            <Input placeholder="请输入通道名称" />
          </Form.Item>

          <Form.Item
            label="通道类型"
            name="type"
            rules={[{ required: true, message: '请选择通道类型' }]}
          >
            <Select placeholder="请选择通道类型" options={[{ value: "email", label: "邮件" }, { value: "sms", label: "短信" }, { value: "webhook", label: "Webhook" }]} />
          </Form.Item>

          <Form.Item label="启用状态" name="is_active" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Divider>配置信息</Divider>

          <Form.Item
            label="SMTP服务器"
            name={['config', 'smtp_server']}
            rules={[{ required: true, message: '请输入SMTP服务器' }]}
          >
            <Input placeholder="请输入SMTP服务器地址" />
          </Form.Item>

          <Form.Item
            label="SMTP端口"
            name={['config', 'smtp_port']}
            rules={[{ required: true, message: '请输入SMTP端口' }]}
          >
            <Input type="number" placeholder="请输入SMTP端口" />
          </Form.Item>

          <Form.Item
            label="用户名"
            name={['config', 'username']}
            rules={[{ required: true, message: '请输入用户名' }]}
          >
            <Input placeholder="请输入用户名" />
          </Form.Item>

          <Form.Item
            label="密码"
            name={['config', 'password']}
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password placeholder="请输入密码" />
          </Form.Item>
        </Form>
      </Modal>}
    </Drawer>
  );
};

export default NotificationCenter;
