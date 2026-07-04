'use client';

/**
 * 增强版系统配置页面
 * 添加更多功能标签和改进的响应式设计
 */

import {
  RefreshCw,
  Save,
  Mail,
  Network,
  Globe,
  Database,
  MemoryStick,
  Settings,
  Shield,
  Clock,
  Cpu,
  Bell,
  FileText,
  Lock,
  Monitor,
  Smartphone,
  Cloud,
  Server,
  Database as DatabaseIcon,
  Key,
  UserCheck,
  Webhook,
  Slack,
} from 'lucide-react';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Tabs,
  Form,
  Input,
  Select,
  Switch,
  Button,
  message,
  Space,
  Row,
  Col,
  Typography,
  Divider,
  InputNumber,
  Alert,
  Tooltip,
  Badge,
  Statistic,
  Tag,
  Modal,
  Progress,
  List,
  Avatar,
  Empty,
  Collapse,
  Table,
} from 'antd';
const { Title, Text, Paragraph } = Typography;
const { Option } = Select;
const { TextArea } = Input;
const { Password } = Input;
const { Panel } = Collapse;

// 引入系统配置API
import { SystemConfigAPI } from '@/lib/api/system-config-api';

const BOOLEAN_CONFIG_KEYS = new Set([
  'passwordRequireUppercase',
  'passwordRequireLowercase',
  'passwordRequireNumbers',
  'passwordRequireSpecialChars',
  'enable2FA',
  'smtpEnableSSL',
  'enableNotification',
  'enableAuditLog',
  'maintenanceMode',
]);

const NUMBER_CONFIG_KEYS = new Set([
  'sessionTimeout',
  'maxFileSize',
  'passwordMinLength',
  'loginMaxAttempts',
  'accountLockoutDuration',
  'smtpPort',
  'maxUploadSize',
  'apiRateLimit',
]);

const normalizeConfigValue = (key: string, value: unknown): unknown => {
  if (BOOLEAN_CONFIG_KEYS.has(key)) {
    if (typeof value === 'boolean') return value;
    if (typeof value === 'string') return value.toLowerCase() === 'true';
    return Boolean(value);
  }

  if (NUMBER_CONFIG_KEYS.has(key)) {
    if (typeof value === 'number') return value;
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : undefined;
  }

  return value;
};

const serializeConfigValue = (value: unknown): string => {
  if (value === null || value === undefined) return '';
  if (typeof value === 'string') return value;
  if (typeof value === 'boolean') return value ? 'true' : 'false';
  if (typeof value === 'number') return String(value);
  return JSON.stringify(value);
};

// 日志条目类型
interface LogEntry {
  id: number;
  action: string;
  user: string;
  timestamp: string;
  ip: string;
  details: string;
}

// 备份记录类型
interface BackupRecord {
  id: number;
  name: string;
  size: string;
  createdAt: string;
  status: 'completed' | 'failed' | 'in_progress';
}

export default function EnhancedSystemConfiguration() {
  const [form] = Form.useForm();
  const [activeTab, setActiveTab] = useState('general');
  const [config, setConfig] = useState<Record<string, unknown>>({});
  const [hasChanges, setHasChanges] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [systemStats, setSystemStats] = useState({
    uptime: '加载中...',
    goroutines: 0,
    cpuCores: 0,
    memoryUsagePercent: 0,
    diskUsagePercent: 0,
    activeConnections: 0,
  });
  const [initialConfig, setInitialConfig] = useState<Record<string, unknown>>({});

  // 新增状态
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [backups, setBackups] = useState<BackupRecord[]>([]);
  const [activeConnections, setActiveConnections] = useState<number>(0);
  const [showLogModal, setShowLogModal] = useState(false);
  const [showBackupModal, setShowBackupModal] = useState(false);

  // 加载系统配置
  const loadConfig = async () => {
    try {
      const response = await SystemConfigAPI.getConfigs();
      const configMap: Record<string, unknown> = {};

      response.configs.forEach((item: { key: string; value: unknown }) => {
        configMap[item.key] = normalizeConfigValue(item.key, item.value);
      });

      setConfig(configMap);
      setInitialConfig(configMap);
      form.setFieldsValue(configMap);
    } catch (error) {
      message.error('加载系统配置失败');
    }
  };

  // 初始化加载配置
  useEffect(() => {
    loadConfig();

    // 定期获取真实系统状态
    const fetchSystemStats = async () => {
      try {
        const status = (await SystemConfigAPI.getSystemStatus()) || {};
        const cpu = (status as any).cpu || {};
        const memory = (status as any).memory || {};
        const disk = (status as any).disk || {};
        const startTime =
          (status as any).startTime ||
          (status as any).start_time ||
          (status as any).startedAt ||
          (status as any).started_at;
        const uptime = (status as any).uptime || (status as any).upTime;

        setSystemStats({
          uptime: typeof uptime === 'string' ? uptime : calculateUptime(startTime),
          goroutines: (status as any).goroutines || 0,
          cpuCores: cpu.cores || (status as any).cpu_cores || 0,
          memoryUsagePercent: Math.round(memory.usage_percent || memory.usage || 0),
          diskUsagePercent: Math.round(disk.usage_percent || 0),
          activeConnections: (status as any).activeConnections || 0,
        });
      } catch (error) {
        console.error('Failed to fetch system status:', error);
      }
    };

    // 模拟日志数据
    setLogs([
      {
        id: 1,
        action: '用户登录',
        user: 'admin',
        timestamp: '2026-07-04 10:30:00',
        ip: '192.168.1.100',
        details: '成功登录系统',
      },
      {
        id: 2,
        action: '配置更新',
        user: 'admin',
        timestamp: '2026-07-04 09:45:00',
        ip: '192.168.1.100',
        details: '更新系统配置: sessionTimeout',
      },
      {
        id: 3,
        action: '工单创建',
        user: 'user1',
        timestamp: '2026-07-04 09:20:00',
        ip: '192.168.1.101',
        details: '创建工单 TKT-2026-001',
      },
    ]);

    // 模拟备份数据
    setBackups([
      {
        id: 1,
        name: 'full_backup_20260704',
        size: '2.5 GB',
        createdAt: '2026-07-04 02:00:00',
        status: 'completed',
      },
      {
        id: 2,
        name: 'incremental_backup_20260703',
        size: '350 MB',
        createdAt: '2026-07-03 02:00:00',
        status: 'completed',
      },
      {
        id: 3,
        name: 'full_backup_20260702',
        size: '2.4 GB',
        createdAt: '2026-07-02 02:00:00',
        status: 'completed',
      },
    ]);

    // 初始加载
    fetchSystemStats();

    // 定期更新
    const interval = setInterval(fetchSystemStats, 5000);
    return () => clearInterval(interval);
  }, []);

  // 计算运行时长
  const calculateUptime = (startTime?: number): string => {
    if (!startTime) return '未知';
    const diff = Date.now() - startTime;
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    return `${days}天 ${hours}小时`;
  };

  // 表单值变化时标记为有更改
  const handleFormChange = () => {
    setHasChanges(true);
  };

  // 重置表单
  const handleReset = () => {
    form.setFieldsValue(initialConfig);
    setHasChanges(false);
  };

  // 保存配置
  const handleSave = async () => {
    try {
      setIsSaving(true);
      const values = form.getFieldsValue();

      // 构造更新请求
      const updateRequests = Object.entries(values).map(([key, value]) => ({
        key,
        value: serializeConfigValue(value),
      }));

      await SystemConfigAPI.updateConfigs(updateRequests);

      message.success('配置保存成功');
      setHasChanges(false);
      setInitialConfig(values);
    } catch (error) {
      message.error('保存配置失败');
    } finally {
      setIsSaving(false);
    }
  };

  // 通用设置表单项
  const GeneralSettings = () => (
    <div className="space-y-6">
      <div>
        <Title level={5} className="!mb-4">
          基础设置
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="系统名称"
              name="systemName"
              rules={[{ required: true, message: '请输入系统名称' }]}
            >
              <Input placeholder="请输入系统名称" prefix={<Globe className="w-4 h-4" />} />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="系统URL"
              name="systemUrl"
              rules={[
                { required: true, message: '请输入系统URL' },
                { type: 'url', message: '请输入有效的URL' },
              ]}
            >
              <Input placeholder="https://example.com" prefix={<WebIcon />} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="时区"
              name="timezone"
              rules={[{ required: true, message: '请选择时区' }]}
            >
              <Select placeholder="请选择时区">
                <Option value="Asia/Shanghai">亚洲/上海 (UTC+8)</Option>
                <Option value="Asia/Tokyo">亚洲/东京 (UTC+9)</Option>
                <Option value="Europe/London">欧洲/伦敦 (UTC+0)</Option>
                <Option value="America/New_York">美洲/纽约 (UTC-5)</Option>
              </Select>
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="语言"
              name="language"
              rules={[{ required: true, message: '请选择语言' }]}
            >
              <Select placeholder="请选择语言">
                <Option value="zh-CN">简体中文</Option>
                <Option value="en-US">English</Option>
                <Option value="ja-JP">日本語</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="日期格式"
              name="dateFormat"
              rules={[{ required: true, message: '请选择日期格式' }]}
            >
              <Select placeholder="请选择日期格式">
                <Option value="YYYY-MM-DD">YYYY-MM-DD</Option>
                <Option value="DD/MM/YYYY">DD/MM/YYYY</Option>
                <Option value="MM/DD/YYYY">MM/DD/YYYY</Option>
              </Select>
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="时间格式"
              name="timeFormat"
              rules={[{ required: true, message: '请选择时间格式' }]}
            >
              <Select placeholder="请选择时间格式">
                <Option value="24h">24小时制</Option>
                <Option value="12h">12小时制</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">
          会话与文件设置
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="会话超时时间(分钟)"
              name="sessionTimeout"
              rules={[{ required: true, message: '请输入会话超时时间' }]}
            >
              <InputNumber min={1} max={1440} style={{ width: '100%' }} addonAfter="分钟" />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="最大文件大小(MB)"
              name="maxFileSize"
              rules={[{ required: true, message: '请输入最大文件大小' }]}
            >
              <InputNumber min={1} max={1024} style={{ width: '100%' }} addonAfter="MB" />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item
          label="允许的文件类型"
          name="allowedFileTypes"
          rules={[{ required: true, message: '请输入允许的文件类型' }]}
        >
          <Input placeholder=".pdf,.doc,.docx,.xls,.xlsx,.jpg,.png,.gif" />
        </Form.Item>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">
          系统维护
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item label="维护模式" name="maintenanceMode" valuePropName="checked">
              <Switch checkedChildren="开启" unCheckedChildren="关闭" />
            </Form.Item>
            <Text type="secondary">开启后，普通用户将无法访问系统</Text>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item label="审计日志" name="enableAuditLog" valuePropName="checked">
              <Switch checkedChildren="开启" unCheckedChildren="关闭" />
            </Form.Item>
            <Text type="secondary">记录所有用户操作日志</Text>
          </Col>
        </Row>
      </div>
    </div>
  );

  // 安全设置表单项
  const SecuritySettings = () => (
    <div className="space-y-6">
      <div>
        <Title level={5} className="!mb-4">
          <Lock className="inline w-4 h-4 mr-2" />
          密码策略
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="密码最小长度"
              name="passwordMinLength"
              rules={[{ required: true, message: '请输入密码最小长度' }]}
            >
              <InputNumber min={6} max={32} style={{ width: '100%' }} addonAfter="字符" />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={12} md={6}>
            <Form.Item label="需要大写字母" name="passwordRequireUppercase" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
          <Col xs={12} md={6}>
            <Form.Item label="需要小写字母" name="passwordRequireLowercase" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
          <Col xs={12} md={6}>
            <Form.Item label="需要数字" name="passwordRequireNumbers" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
          <Col xs={12} md={6}>
            <Form.Item
              label="需要特殊字符"
              name="passwordRequireSpecialChars"
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Col>
        </Row>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">
          <UserCheck className="inline w-4 h-4 mr-2" />
          账户安全
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="登录失败次数限制"
              name="loginMaxAttempts"
              rules={[{ required: true, message: '请输入登录失败次数限制' }]}
            >
              <InputNumber min={1} max={10} style={{ width: '100%' }} addonAfter="次" />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="账户锁定时间(分钟)"
              name="accountLockoutDuration"
              rules={[{ required: true, message: '请输入账户锁定时间' }]}
            >
              <InputNumber min={1} max={1440} style={{ width: '100%' }} addonAfter="分钟" />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24}>
            <Form.Item label="启用双因素认证" name="enable2FA" valuePropName="checked">
              <Switch checkedChildren="开启" unCheckedChildren="关闭" />
            </Form.Item>
          </Col>
        </Row>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">
          <Key className="inline w-4 h-4 mr-2" />
          API 安全
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item label="API速率限制" name="apiRateLimit">
              <InputNumber min={10} max={10000} style={{ width: '100%' }} addonAfter="请求/分钟" />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item label="API密钥轮换周期">
              <Select defaultValue="90" style={{ width: '100%' }}>
                <Option value="30">30天</Option>
                <Option value="60">60天</Option>
                <Option value="90">90天</Option>
                <Option value="180">180天</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>
      </div>
    </div>
  );

  // 邮件设置表单项
  const EmailSettings = () => (
    <div className="space-y-6">
      <div>
        <Title level={5} className="!mb-4">
          <Mail className="inline w-4 h-4 mr-2" />
          SMTP设置
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="SMTP服务器"
              name="smtpHost"
              rules={[{ required: true, message: '请输入SMTP服务器' }]}
            >
              <Input placeholder="smtp.example.com" prefix={<Server className="w-4 h-4" />} />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="SMTP端口"
              name="smtpPort"
              rules={[{ required: true, message: '请输入SMTP端口' }]}
            >
              <InputNumber min={1} max={65535} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="SMTP用户名"
              name="smtpUsername"
              rules={[{ required: true, message: '请输入SMTP用户名' }]}
            >
              <Input placeholder="username@example.com" />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item label="SMTP密码" name="smtpPassword">
              <Password placeholder="请输入密码" />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item label="启用SSL/TLS" name="smtpEnableSSL" valuePropName="checked">
          <Switch checkedChildren="开启" unCheckedChildren="关闭" />
        </Form.Item>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">
          <FileText className="inline w-4 h-4 mr-2" />
          邮件模板
        </Title>
        <Form.Item
          label="发件人邮箱"
          name="emailFrom"
          rules={[
            { required: true, message: '请输入发件人邮箱' },
            { type: 'email', message: '请输入有效的邮箱地址' },
          ]}
        >
          <Input placeholder="noreply@example.com" prefix={<Mail className="w-4 h-4" />} />
        </Form.Item>

        <Form.Item label="系统通知模板" name="systemNotificationTemplate">
          <TextArea rows={4} placeholder="请输入系统通知邮件模板" />
        </Form.Item>
      </div>
    </div>
  );

  // 通知设置
  const NotificationSettings = () => (
    <div className="space-y-6">
      <div>
        <Title level={5} className="!mb-4">
          <Bell className="inline w-4 h-4 mr-2" />
          通知渠道
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={8}>
            <Card>
              <div className="text-center">
                <Mail className="w-8 h-8 mx-auto mb-2 text-blue-500" />
                <Text strong>邮件通知</Text>
                <div className="mt-2">
                  <Switch defaultChecked />
                </div>
              </div>
            </Card>
          </Col>
          <Col xs={24} md={8}>
            <Card>
              <div className="text-center">
                <Slack className="w-8 h-8 mx-auto mb-2 text-purple-500" />
                <Text strong>Slack</Text>
                <div className="mt-2">
                  <Switch />
                </div>
              </div>
            </Card>
          </Col>
          <Col xs={24} md={8}>
            <Card>
              <div className="text-center">
                <Webhook className="w-8 h-8 mx-auto mb-2 text-green-500" />
                <Text strong>Webhook</Text>
                <div className="mt-2">
                  <Switch />
                </div>
              </div>
            </Card>
          </Col>
        </Row>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">
          通知类型
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Card size="small">
              <Form.Item label="工单创建通知" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="工单状态变更" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="工单分配通知" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="SLA警告" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
            </Card>
          </Col>
          <Col xs={24} md={12}>
            <Card size="small">
              <Form.Item label="变更审批通知" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="问题创建通知" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="系统告警" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="周报汇总" valuePropName="checked">
                <Switch />
              </Form.Item>
            </Card>
          </Col>
        </Row>
      </div>
    </div>
  );

  // 日志与监控
  const MonitoringSettings = () => (
    <div className="space-y-6">
      <div>
        <Title level={5} className="!mb-4">
          <Monitor className="inline w-4 h-4 mr-2" />
          系统日志
        </Title>
        <Card
          extra={
            <Button type="link" onClick={() => setShowLogModal(true)}>
              查看全部
            </Button>
          }
        >
          <List
            size="small"
            dataSource={logs.slice(0, 5)}
            renderItem={item => (
              <List.Item>
                <List.Item.Meta
                  avatar={<Avatar size="small" style={{ backgroundColor: '#1890ff' }}>{item.user.charAt(0)}</Avatar>}
                  title={<Text>{item.action}</Text>}
                  description={
                    <Space>
                      <Text type="secondary">{item.timestamp}</Text>
                      <Text type="secondary">IP: {item.ip}</Text>
                    </Space>
                  }
                />
              </List.Item>
            )}
          />
        </Card>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">
          <DatabaseIcon className="inline w-4 h-4 mr-2" />
          备份管理
        </Title>
        <Card
          extra={
            <Button type="primary" onClick={() => setShowBackupModal(true)}>
              创建备份
            </Button>
          }
        >
          <List
            size="small"
            dataSource={backups}
            renderItem={item => (
              <List.Item>
                <List.Item.Meta
                  avatar={<DatabaseIcon className="w-5 h-5 text-blue-500" />}
                  title={<Text>{item.name}</Text>}
                  description={
                    <Space>
                      <Text type="secondary">{item.size}</Text>
                      <Text type="secondary">{item.createdAt}</Text>
                    </Space>
                  }
                />
                <Tag color={item.status === 'completed' ? 'success' : item.status === 'failed' ? 'error' : 'processing'}>
                  {item.status === 'completed' ? '已完成' : item.status === 'failed' ? '失败' : '进行中'}
                </Tag>
              </List.Item>
            )}
          />
        </Card>
      </div>
    </div>
  );

  // 标签页配置
  const tabItems = [
    {
      key: 'general',
      label: (
        <span>
          <Settings className="w-4 h-4" /> 通用设置
        </span>
      ),
      children: <GeneralSettings />,
    },
    {
      key: 'security',
      label: (
        <span>
          <Shield className="w-4 h-4" /> 安全设置
        </span>
      ),
      children: <SecuritySettings />,
    },
    {
      key: 'email',
      label: (
        <span>
          <Mail className="w-4 h-4" /> 邮件设置
        </span>
      ),
      children: <EmailSettings />,
    },
    {
      key: 'notification',
      label: (
        <span>
          <Bell className="w-4 h-4" /> 通知设置
        </span>
      ),
      children: <NotificationSettings />,
    },
    {
      key: 'monitoring',
      label: (
        <span>
          <Monitor className="w-4 h-4" /> 日志与监控
        </span>
      ),
      children: <MonitoringSettings />,
    },
  ];

  return (
    <div className="p-6">
      <div className="mb-6">
        <Title level={2} className="!mb-2 !text-gray-900">
          <Settings className="inline-block w-6 h-6 mr-2" />
          系统配置
        </Title>
        <Text type="secondary">管理系统全局设置和集成配置</Text>
      </div>

      {/* 系统状态统计 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200">
            <Statistic
              title="系统运行时间"
              value={systemStats.uptime}
              prefix={<Clock className="w-5 h-5" />}
              valueStyle={{ color: '#52c41a', fontSize: '1.25rem' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200">
            <Statistic
              title="Goroutine 数"
              value={systemStats.goroutines}
              prefix={<Network className="w-5 h-5" />}
              valueStyle={{ color: '#1890ff', fontSize: '1.25rem' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-500">内存使用率</div>
                <div className="text-2xl font-bold text-purple-600">{systemStats.memoryUsagePercent}%</div>
              </div>
              <MemoryStick className="w-8 h-8 text-purple-600" />
            </div>
            <Progress percent={systemStats.memoryUsagePercent} size="small" strokeColor="#722ed1" />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-500">磁盘使用率</div>
                <div className="text-2xl font-bold text-orange-500">{systemStats.diskUsagePercent}%</div>
              </div>
              <Database className="w-8 h-8 text-orange-500" />
            </div>
            <Progress percent={systemStats.diskUsagePercent} size="small" strokeColor="#fa8c16" />
          </Card>
        </Col>
      </Row>

      {/* 操作提示 */}
      {hasChanges && (
        <Alert
          message="配置已修改"
          description="您有未保存的配置更改，请及时保存。"
          type="warning"
          showIcon
          closable
          className="mb-6"
        />
      )}

      {/* 配置表单 */}
      <Card className="rounded-lg shadow-sm border border-gray-200">
        <Form
          form={form}
          layout="vertical"
          initialValues={config}
          onValuesChange={handleFormChange}
        >
          <div className="mb-4 flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
            <Title level={4} className="!mb-0">
              配置管理
            </Title>
            <Space>
              <Button icon={<RefreshCw className="w-4 h-4" />} onClick={handleReset}>
                重置
              </Button>
              <Button
                type="primary"
                icon={<Save className="w-4 h-4" />}
                loading={isSaving}
                onClick={handleSave}
              >
                {isSaving ? '保存中...' : '保存配置'}
              </Button>
            </Space>
          </div>

          <Tabs items={tabItems} type="card" className="custom-tabs" />
        </Form>
      </Card>

      {/* 日志详情弹窗 */}
      <Modal
        title="系统日志"
        open={showLogModal}
        onCancel={() => setShowLogModal(false)}
        footer={null}
        width={800}
      >
        <Table
          dataSource={logs}
          rowKey="id"
          size="small"
          pagination={{ pageSize: 10 }}
          columns={[
            { title: '操作', dataIndex: 'action', key: 'action' },
            { title: '用户', dataIndex: 'user', key: 'user' },
            { title: '时间', dataIndex: 'timestamp', key: 'timestamp' },
            { title: 'IP', dataIndex: 'ip', key: 'ip' },
            { title: '详情', dataIndex: 'details', key: 'details' },
          ]}
        />
      </Modal>

      {/* 创建备份弹窗 */}
      <Modal
        title="创建备份"
        open={showBackupModal}
        onCancel={() => setShowBackupModal(false)}
        footer={[
          <Button key="cancel" onClick={() => setShowBackupModal(false)}>
            取消
          </Button>,
          <Button key="backup" type="primary">
            开始备份
          </Button>,
        ]}
      >
        <div className="py-4">
          <Text>确定要创建新的系统备份吗？备份可能需要几分钟时间。</Text>
        </div>
      </Modal>
    </div>
  );
}

// 图标组件
const WebIcon = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
    <circle cx="12" cy="12" r="10" />
    <line x1="2" y1="12" x2="22" y2="12" />
    <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
  </svg>
);
