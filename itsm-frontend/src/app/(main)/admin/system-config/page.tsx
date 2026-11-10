'use client';

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
  App,
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
} from 'antd';
const { Title, Text } = Typography;
const { TextArea } = Input;
const { Password } = Input;

// 引入系统配置API
import { SystemConfigAPI } from '@/lib/api/system-config-api';
import { formatUptime } from '@/lib/utils/metric-format';
import type { SystemStats } from '@/types/dashboard';
import { UsageGuideCard } from '@/components/common/UsageGuideCard';

const BOOLEAN_CONFIG_KEYS = new Set([
  'passwordRequireUppercase',
  'passwordRequireLowercase',
  'passwordRequireNumbers',
  'passwordRequireSpecialChars',
  'enable2FA',
  'smtpEnableSSL',
]);

const NUMBER_CONFIG_KEYS = new Set([
  'sessionTimeout',
  'maxFileSize',
  'passwordMinLength',
  'loginMaxAttempts',
  'accountLockoutDuration',
  'smtpPort',
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

export default function SystemConfiguration() {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const [activeTab, setActiveTab] = useState('general');
  const [config, setConfig] = useState<Record<string, unknown>>({});
  const [hasChanges, setHasChanges] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [systemStats, setSystemStats] = useState<SystemStats | null>(null);
  const [initialConfig, setInitialConfig] = useState<Record<string, unknown>>({});

  const memoryMeasured = systemStats?.availability.memory === 'measured';

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

    // 运行指标与系统健康卡共用同一份后端 DTO；轮询不叠加在途请求，
    // 否则慢响应会覆盖新采样值。
    let inFlight = false;
    const fetchSystemStats = async () => {
      if (inFlight) return;
      inFlight = true;
      try {
        setSystemStats(await SystemConfigAPI.getSystemStatus());
      } catch (error) {
        console.error('Failed to fetch system status:', error);
        setSystemStats(null);
      } finally {
        inFlight = false;
      }
    };

    // 初始加载
    fetchSystemStats();

    // 定期更新
    const interval = setInterval(fetchSystemStats, 5000);
    return () => clearInterval(interval);
  }, []);

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
          <Col span={12}>
            <Form.Item
              label="System name"
              name="systemName"
              rules={[{ required: true, message: 'Please enter system name' }]}
            >
              <Input placeholder="Please enter system name" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label="System URL"
              name="systemUrl"
              rules={[
                { required: true, message: 'Please enter the system URL' },
                { type: 'url', message: 'Please enter a valid URL' },
              ]}
            >
              <Input placeholder="https://example.com" />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label="Time zone"
              name="timezone"
              rules={[{ required: true, message: 'Please select a time zone' }]}
            >
              <Select placeholder="Please select a time zone" options={[{ value: "Asia/Shanghai", label: "Asia/Shanghai" }, { value: "Asia/Tokyo", label: "Asia/Tokyo" }, { value: "Europe/London", label: "Europe/London" }, { value: "America/New_York", label: "America/New York" }]} />
            </Form.Item>
          </Col>
          <Col span={12}>
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
          <Col span={12}>
            <Form.Item
              label="date format"
              name="dateFormat"
              rules={[{ required: true, message: 'Please select date format' }]}
            >
              <Select placeholder="Please select date format" options={[{ value: "YYYY-MM-DD", label: "YYYY-MM-DD" }, { value: "DD/MM/YYYY", label: "DD/MM/YYYY" }, { value: "MM/DD/YYYY", label: "MM/DD/YYYY" }]} />
            </Form.Item>
          </Col>
          <Col span={12}>
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

          Session settings
        </Title>
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label="Session timeout (minutes)"
              name="sessionTimeout"
              rules={[{ required: true, message: 'Please enter session timeout' }]}
            >
              <InputNumber min={1} max={1440} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label="Maximum file size (MB)"
              name="maxFileSize"
              rules={[{ required: true, message: 'Please enter the maximum file size' }]}
            >
              <InputNumber min={1} max={1024} style={{ width: '100%' }} />
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
    </div>
  );

  // 安全设置表单项
  const SecuritySettings = () => (
    <div className="space-y-6">
      <div>
        <Title level={5} className="!mb-4">

          Password Policy
        </Title>
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label="Minimum password length"
              name="passwordMinLength"
              rules={[{ required: true, message: 'Please enter the minimum password length' }]}
            >
              <InputNumber min={6} max={32} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item label="Capital letters required" name="passwordRequireUppercase" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item label="lowercase letters required" name="passwordRequireLowercase" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item label="Number required" name="passwordRequireNumbers" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Col>
          <Col span={12}>
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

          Account security
        </Title>
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label="Limit on failed login attempts"
              name="loginMaxAttempts"
              rules={[{ required: true, message: 'Please enter the limit for failed login attempts' }]}
            >
              <InputNumber min={1} max={10} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label="Account lock time (minutes)"
              name="accountLockoutDuration"
              rules={[{ required: true, message: 'Please enter the account lock time' }]}
            >
              <InputNumber min={1} max={1440} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item label="Enable two-factor authentication" name="enable2FA" valuePropName="checked">
          <Switch />
        </Form.Item>
      </div>
    </div>
  );

  // 邮件设置表单项
  const EmailSettings = () => (
    <div className="space-y-6">
      <div>
        <Title level={5} className="!mb-4">

          SMTP settings
        </Title>
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label="SMTP Server"
              name="smtpHost"
              rules={[{ required: true, message: 'Please enter SMTP server' }]}
            >
              <Input placeholder="smtp.example.com" />
            </Form.Item>
          </Col>
          <Col span={12}>
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
          <Col span={12}>
            <Form.Item
              label="SMTP username"
              name="smtpUsername"
              rules={[{ required: true, message: 'Please enter SMTP username' }]}
            >
              <Input placeholder="username@example.com" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item label="SMTP password" name="smtpPassword">
              <Password placeholder="Please enter password" />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item label="Enable SSL/TLS" name="smtpEnableSSL" valuePropName="checked">
          <Switch />
        </Form.Item>
      </div>

      <Divider />

      <div>
        <Title level={5} className="!mb-4">

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
          <Input placeholder="noreply@example.com" />
        </Form.Item>

        <Form.Item label="System notification template" name="systemNotificationTemplate">
          <TextArea rows={4} placeholder="Please enter the system notification email template" />
        </Form.Item>
      </div>
    </div>
  );

  // 标签页配置
  const tabItems = [
    {
      key: 'general',
      label: 'General settings',
      children: <GeneralSettings />,
      icon: <Settings className="w-4 h-4" />,
    },
    {
      key: 'security',
      label: 'Security',
      children: <SecuritySettings />,
      icon: <Shield className="w-4 h-4" />,
    },
    {
      key: 'email',
      label: 'Email settings',
      children: <EmailSettings />,
      icon: <Mail className="w-4 h-4" />,
    },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title level={2} className="!mb-2 !text-gray-900">
          <Settings className="inline-block w-6 h-6 mr-2" />

          System Configuration
        </Title>
        <Text type="secondary">Manage system global settings and integration configuration</Text>
      </div>

      <UsageGuideCard
        style={{ marginBottom: 16 }}
        title="How to use system configuration"
        intro="This page manages global default values: basic information, session timeout, password policy, account security and SMTP email settings. Modifications will affect all users, please be careful."
        steps={[
          'The form is displayed in groups according to tabs; after modifying any field, a reminder "You have unsaved configuration changes" will appear on the page. Click "Save Configuration" in the upper right corner to submit it uniformly. Click "Reset" to abandon modifications and restore the last saved value; modifications before saving will not take effect, so please do not leave the page directly.',
          'After deployment, it is recommended to check the basic settings: system name and access URL, time zone, language and date and time format (affects time display and notification content), and session timeout.',
          'Upload restrictions (size and allowed types) affect the upload of all files such as work order attachments, and are adjusted according to organizational specifications.',
          'Password policy and account security adjustments will affect subsequent settings and resets; before tightening the policy, please confirm that it will not prevent existing users from logging in normally.',
          'SMTP (server, port, account, authorization code) must be configured when email notification is required; after saving, a real notification (such as work order allocation) will be triggered to verify email delivery.',
          'The "system running time/Goroutine/CPU core/memory usage" above is from the back-end runtime sampling, which is the same caliber as the system health card on the home page. It is only for health observation and does not require configuration. Indicators that cannot be measured in the running environment show "Not Connected" instead of 0%.',
        ]}
      />

      {/* 系统状态统计 */}
      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200">
            <Statistic
              title="System Uptime"
              value={systemStats ? formatUptime(systemStats.uptime) : '—'}
              prefix={<Clock className="w-5 h-5" />}
              styles={{ content: { color: '#52c41a' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200">
            <Statistic
              title="Number of Goroutines"
              value={systemStats?.goroutines ?? 0}
              prefix={<Network className="w-5 h-5" />}
              styles={{ content: { color: '#1890ff' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200">
            <Statistic
              title="Number of CPU cores"
              value={systemStats?.cores ?? 0}
              prefix={<Cpu className="w-5 h-5" />}
              styles={{ content: { color: '#fa8c16' } }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm text-gray-500">Memory Usage</div>
                <div className="text-2xl font-bold text-purple-600">
                  {memoryMeasured
                    ? `${Math.round(systemStats?.memoryUsage ?? 0)}%`
                    : 'Not measured'}
                </div>
              </div>
              <MemoryStick className="w-8 h-8 text-purple-600" />
            </div>
            {memoryMeasured ? (
              <Progress
                percent={Math.round(systemStats?.memoryUsage ?? 0)}
                size="small"
                strokeColor="#722ed1"
              />
            ) : (
              <Text type="secondary" className="text-xs">

                The current operating environment cannot detect the upper memory limit (the container has no cgroup restrictions and the host is not Linux)
              </Text>
            )}
          </Card>
        </Col>
      </Row>

      {/* 操作提示 */}
      {hasChanges && (
        <Alert
          title="The configuration has been modified, please save it in time"
          type="warning"
          showIcon
          closable
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
          <div className="mb-4 flex justify-between items-center">
            <Title level={4}>Configuration Management</Title>
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
    </div>
  );
}
