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

// 系统状态兼容接口（兼容 snake_case 和 camelCase）
interface SystemStatusResponse {
  cpu?: { cores?: number; usage?: number };
  memory?: { usagePercent?: number; usage?: number };
  disk?: { usagePercent?: number; usage?: number };
  startTime?: string;
  startedAt?: string;
  uptime?: string;
  upTime?: string;
  goroutines?: number;
  cpuCores?: number;
  activeConnections?: number;
}

// 工具函数：提取兼容字段值
function getSystemStatusValue<T>(data: any, camelKey: string, snakeKey: string): T {
  return (data[camelKey] ?? data[snakeKey] ?? 0) as T;
}

export default function EnhancedSystemConfiguration() {
  const [form] = Form.useForm();
  const [activeTab, setActiveTab] = useState('general');
  const [config, setConfig] = useState<Record<string, unknown>>({});
  const [hasChanges, setHasChanges] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [systemStats, setSystemStats] = useState({
    uptime: 'Loading...',
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

      response.items.forEach((item: { key: string; value: unknown }) => {
        configMap[item.key] = normalizeConfigValue(item.key, item.value);
      });

      setConfig(configMap);
      setInitialConfig(configMap);
      form.setFieldsValue(configMap);
    } catch (error) {
      message.error('Failed to load system configuration');
    }
  };

  // 初始化加载配置
  useEffect(() => {
    loadConfig();

    // 定期获取真实系统状态
    const fetchSystemStats = async () => {
      try {
        const status = (await SystemConfigAPI.getSystemStatus()) as unknown as SystemStatusResponse || {};
        const cpu = status.cpu || {};
        const memory = status.memory || {};
        const disk = status.disk || {};
        const startTime = status.startTime || status.startedAt;
        const uptime = status.uptime || status.upTime;

        setSystemStats({
          uptime: typeof uptime === 'string' ? uptime : calculateUptime(typeof startTime === 'string' ? parseInt(startTime, 10) : startTime),
          goroutines: getSystemStatusValue<number>(status as any, 'goroutines', 'goroutines'),
          cpuCores: cpu.cores || getSystemStatusValue<number>(status as any, 'cpu_cores', 'cpu_cores'),
          memoryUsagePercent: Math.round(memory.usagePercent || memory.usage || 0),
          diskUsagePercent: Math.round(disk.usagePercent || 0),
          activeConnections: getSystemStatusValue<number>(status as any, 'activeConnections', 'activeConnections'),
        });
      } catch (error) {
        console.error('Failed to fetch system status:', error);
      }
    };

    // 模拟日志数据
    setLogs([
      {
        id: 1,
        action: 'User login',
        user: 'admin',
        timestamp: '2026-07-04 10:30:00',
        ip: '192.168.1.100',
        details: 'Successfully logged into the system',
      },
      {
        id: 2,
        action: 'Configuration update',
        user: 'admin',
        timestamp: '2026-07-04 09:45:00',
        ip: '192.168.1.100',
        details: 'Update system configuration: sessionTimeout',
      },
      {
        id: 3,
        action: 'Ticket Created',
        user: 'user1',
        timestamp: '2026-07-04 09:20:00',
        ip: '192.168.1.101',
        details: 'Create work order TKT-2026-001',
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
    if (!startTime) return 'Unknown';
    const diff = Date.now() - startTime;
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    return `${days} days ${hours} hours`;
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

      message.success('Configuration saved successfully');
      setHasChanges(false);
      setInitialConfig(values);
    } catch (error) {
      message.error('Failed to save configuration');
    } finally {
      setIsSaving(false);
    }
  };

  // 通用设置表单项
  const GeneralSettings = () => (
    <div className="space-y-6">
      <div>
        <Title level={5} className="!mb-4">

          Basic Settings
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="System name"
              name="systemName"
              rules={[{ required: true, message: 'Please enter system name' }]}
            >
              <Input placeholder="Please enter system name" prefix={<Globe className="w-4 h-4" />} />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="System URL"
              name="systemUrl"
              rules={[
                { required: true, message: 'Please enter the system URL' },
                { type: 'url', message: 'Please enter a valid URL' },
              ]}
            >
              <Input placeholder="https://example.com" prefix={<WebIcon />} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="Time zone"
              name="timezone"
              rules={[{ required: true, message: 'Please select a time zone' }]}
            >
              <Select placeholder="Please select a time zone" options={[{ value: "Asia/Shanghai", label: "Asia/Shanghai (UTC+8)" }, { value: "Asia/Tokyo", label: "Asia/Tokyo (UTC+9)" }, { value: "Europe/London", label: "Europe/London (UTC+0)" }, { value: "America/New_York", label: "America/New York (UTC-5)" }]} />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="Language"
              name="language"
              rules={[{ required: true, message: 'Please select language' }]}
            >
              <Select placeholder="Please select language" options={[{ value: "en-US", label: "English" }]} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="date format"
              name="dateFormat"
              rules={[{ required: true, message: 'Please select date format' }]}
            >
              <Select placeholder="Please select date format" options={[{ value: "YYYY-MM-DD", label: "YYYY-MM-DD" }, { value: "DD/MM/YYYY", label: "DD/MM/YYYY" }, { value: "MM/DD/YYYY", label: "MM/DD/YYYY" }]} />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="Time format"
              name="timeFormat"
              rules={[{ required: true, message: 'Please select time format' }]}
            >
              <Select placeholder="Please select time format" options={[{ value: "24h", label: "24-hour clock" }, { value: "12h", label: "12-hour clock" }]} />
            </Form.Item>
          </Col>
        </Row>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">

          Session and File Settings
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="Session timeout (minutes)"
              name="sessionTimeout"
              rules={[{ required: true, message: 'Please enter session timeout' }]}
            >
              <InputNumber min={1} max={1440} style={{ width: '100%' }} addonAfter="minutes" />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="Maximum file size (MB)"
              name="maxFileSize"
              rules={[{ required: true, message: 'Please enter the maximum file size' }]}
            >
              <InputNumber min={1} max={1024} style={{ width: '100%' }} addonAfter="MB" />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item
          label="Allowed File Types"
          name="allowedFileTypes"
          rules={[{ required: true, message: 'Please enter allowed file types' }]}
        >
          <Input placeholder=".pdf,.doc,.docx,.xls,.xlsx,.jpg,.png,.gif" />
        </Form.Item>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">

          System maintenance
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item label="Maintenance mode" name="maintenanceMode" valuePropName="checked">
              <Switch checkedChildren="On" unCheckedChildren="Close" />
            </Form.Item>
            <Text type="secondary">Once enabled, ordinary users will not be able to access the system.</Text>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item label="Audit Logs" name="enableAuditLog" valuePropName="checked">
              <Switch checkedChildren="On" unCheckedChildren="Close" />
            </Form.Item>
            <Text type="secondary">Record all user operation logs</Text>
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

          Password Policy
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="Minimum password length"
              name="passwordMinLength"
              rules={[{ required: true, message: 'Please enter the minimum password length' }]}
            >
              <InputNumber min={6} max={32} style={{ width: '100%' }} addonAfter="character" />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={12} md={6}>
            <Form.Item label="Capital letters required" name="passwordRequireUppercase" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
          <Col xs={12} md={6}>
            <Form.Item label="lowercase letters required" name="passwordRequireLowercase" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
          <Col xs={12} md={6}>
            <Form.Item label="Number required" name="passwordRequireNumbers" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
          <Col xs={12} md={6}>
            <Form.Item
              label="Special characters required"
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

          Account security
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="Limit on failed login attempts"
              name="loginMaxAttempts"
              rules={[{ required: true, message: 'Please enter the limit for failed login attempts' }]}
            >
              <InputNumber min={1} max={10} style={{ width: '100%' }} addonAfter="times" />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="Account lock time (minutes)"
              name="accountLockoutDuration"
              rules={[{ required: true, message: 'Please enter the account lock time' }]}
            >
              <InputNumber min={1} max={1440} style={{ width: '100%' }} addonAfter="minutes" />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24}>
            <Form.Item label="Enable two-factor authentication" name="enable2FA" valuePropName="checked">
              <Switch checkedChildren="On" unCheckedChildren="Close" />
            </Form.Item>
          </Col>
        </Row>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">
          <Key className="inline w-4 h-4 mr-2" />

          API security
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item label="API rate limit" name="apiRateLimit">
              <InputNumber min={10} max={10000} style={{ width: '100%' }} addonAfter="Requests/minute" />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item label="API key rotation period">
              <Select defaultValue="90" style={{ width: '100%' }} options={[{ value: "30", label: "30 days" }, { value: "60", label: "60 days" }, { value: "90", label: "90 days" }, { value: "180", label: "180 days" }]} />
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

          SMTP settings
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="SMTP Server"
              name="smtpHost"
              rules={[{ required: true, message: 'Please enter SMTP server' }]}
            >
              <Input placeholder="smtp.example.com" prefix={<Server className="w-4 h-4" />} />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item
              label="SMTP Port"
              name="smtpPort"
              rules={[{ required: true, message: 'Please enter SMTP port' }]}
            >
              <InputNumber min={1} max={65535} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Form.Item
              label="SMTP username"
              name="smtpUsername"
              rules={[{ required: true, message: 'Please enter SMTP username' }]}
            >
              <Input placeholder="username@example.com" />
            </Form.Item>
          </Col>
          <Col xs={24} md={12}>
            <Form.Item label="SMTP password" name="smtpPassword">
              <Password placeholder="Please enter password" />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item label="Enable SSL/TLS" name="smtpEnableSSL" valuePropName="checked">
          <Switch checkedChildren="On" unCheckedChildren="Close" />
        </Form.Item>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">
          <FileText className="inline w-4 h-4 mr-2" />

          Email Templates
        </Title>
        <Form.Item
          label="Sender's email address"
          name="emailFrom"
          rules={[
            { required: true, message: 'Please enter the sender\'s email address' },
            { type: 'email', message: 'Please enter a valid email address' },
          ]}
        >
          <Input placeholder="noreply@example.com" prefix={<Mail className="w-4 h-4" />} />
        </Form.Item>

        <Form.Item label="System notification template" name="systemNotificationTemplate">
          <TextArea rows={4} placeholder="Please enter the system notification email template" />
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

          Notification Channels
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={8}>
            <Card>
              <div className="text-center">
                <Mail className="w-8 h-8 mx-auto mb-2 text-blue-500" />
                <Text strong>Email notification</Text>
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

          Notification Type
        </Title>
        <Row gutter={[16, 16]}>
          <Col xs={24} md={12}>
            <Card size="small">
              <Form.Item label="Work order creation notification" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="Ticket Status Changed" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="Work order allocation notification" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="SLA Warning" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
            </Card>
          </Col>
          <Col xs={24} md={12}>
            <Card size="small">
              <Form.Item label="Change Approval Notice" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="Issue creation notification" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="System Alerts" valuePropName="checked">
                <Switch defaultChecked />
              </Form.Item>
              <Form.Item label="Weekly report summary" valuePropName="checked">
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

          System log
        </Title>
        <Card
          extra={
            <Button type="link" onClick={() => setShowLogModal(true)}>

              View All
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

          Backup Management
        </Title>
        <Card
          extra={
            <Button type="primary" onClick={() => setShowBackupModal(true)}>

              Create backup
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
                  {item.status === 'completed' ? 'Completed' : item.status === 'failed' ? 'Failed' : 'In Progress'}
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
          <Settings className="w-4 h-4" />  General settings
        </span>
      ),
      children: <GeneralSettings />,
    },
    {
      key: 'security',
      label: (
        <span>
          <Shield className="w-4 h-4" />  Security
        </span>
      ),
      children: <SecuritySettings />,
    },
    {
      key: 'email',
      label: (
        <span>
          <Mail className="w-4 h-4" />  Email settings
        </span>
      ),
      children: <EmailSettings />,
    },
    {
      key: 'notification',
      label: (
        <span>
          <Bell className="w-4 h-4" />  Notifications
        </span>
      ),
      children: <NotificationSettings />,
    },
    {
      key: 'monitoring',
      label: (
        <span>
          <Monitor className="w-4 h-4" />  Logging and Monitoring
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

          System Configuration
        </Title>
        <Text type="secondary">Manage system global settings and integration configuration</Text>
      </div>

      {/* 系统状态统计 */}
      <Row gutter={[16, 16]} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200">
            <Statistic
              title="System Uptime"
              value={systemStats.uptime}
              prefix={<Clock className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a', fontSize: '1.25rem' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200">
            <Statistic
              title="Number of Goroutines"
              value={systemStats.goroutines}
              prefix={<Network className="w-5 h-5" />}
              styles={{ content: { color: '#1890ff', fontSize: '1.25rem' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-500">Memory Usage</div>
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
                <div className="text-sm text-gray-500">Disk Usage</div>
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
          title="Configuration modified"
          description="You have unsaved configuration changes, please save them in time."
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

              Configuration Management
            </Title>
            <Space>
              <Button icon={<RefreshCw className="w-4 h-4" />} onClick={handleReset}>

                Reset
              </Button>
              <Button
                type="primary"
                icon={<Save className="w-4 h-4" />}
                loading={isSaving}
                onClick={handleSave}
              >
                {isSaving ? 'Saving...' : 'Save Configuration'}
              </Button>
            </Space>
          </div>

          <Tabs items={tabItems} type="card" className="custom-tabs" />
        </Form>
      </Card>

      {/* 日志详情弹窗 */}
      <Modal
        title="System log"
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
            { title: 'Actions', dataIndex: 'action', key: 'action' },
            { title: 'Users', dataIndex: 'user', key: 'user' },
            { title: 'Timestamp', dataIndex: 'timestamp', key: 'timestamp' },
            { title: 'IP', dataIndex: 'ip', key: 'ip' },
            { title: 'Details', dataIndex: 'details', key: 'details' },
          ]}
        />
      </Modal>

      {/* 创建备份弹窗 */}
      <Modal
        title="Create backup"
        open={showBackupModal}
        onCancel={() => setShowBackupModal(false)}
        footer={[
          <Button key="cancel" onClick={() => setShowBackupModal(false)}>

            Cancel
          </Button>,
          <Button key="backup" type="primary">

            Start backup
          </Button>,
        ]}
      >
        <div className="py-4">
          <Text>Are you sure you want to create a new system backup? The backup may take several minutes.</Text>
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
