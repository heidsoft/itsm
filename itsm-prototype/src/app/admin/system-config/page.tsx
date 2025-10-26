'use client';

import { RefreshCw, Save, Mail, Network, Globe, Database, MemoryStick } from 'lucide-react';

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
} from 'antd';
const { Title, Text } = Typography;
const { Option } = Select;
const { TextArea } = Input;
const { Password } = Input;

// 引入系统配置API
import { SystemConfigAPI } from '@/lib/api/system-config-api';

export default function SystemConfiguration() {
  const [form] = Form.useForm();
  const [activeTab, setActiveTab] = useState('general');
  const [config, setConfig] = useState<Record<string, unknown>>({});
  const [hasChanges, setHasChanges] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [systemStats, setSystemStats] = useState({
    uptime: '7天 12小时',
    connections: 128,
    diskUsage: 65,
    memoryUsage: 42,
  });
  const [initialConfig, setInitialConfig] = useState<Record<string, unknown>>({});

  // 加载系统配置
  const loadConfig = async () => {
    try {
      const response = await SystemConfigAPI.getConfigs();
      const configMap: Record<string, unknown> = {};

      response.configs.forEach(item => {
        configMap[item.key] = item.value;
      });

      setConfig(configMap);
      setInitialConfig(configMap);
      form.setFieldsValue(configMap);
    } catch (error) {
      console.error('加载系统配置失败:', error);
      message.error('加载系统配置失败');
    }
  };

  // 初始化加载配置
  useEffect(() => {
    loadConfig();

    // 模拟获取系统状态
    const interval = setInterval(() => {
      setSystemStats({
        uptime: '7天 12小时',
        connections: Math.floor(Math.random() * 200) + 50,
        diskUsage: Math.floor(Math.random() * 30) + 60,
        memoryUsage: Math.floor(Math.random() * 30) + 30,
      });
    }, 5000);

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
        value: String(value),
      }));

      await SystemConfigAPI.updateConfigs(updateRequests);

      message.success('配置保存成功');
      setHasChanges(false);
      setInitialConfig(values);
    } catch (error) {
      console.error('保存配置失败:', error);
      message.error('保存配置失败');
    } finally {
      setIsSaving(false);
    }
  };

  // 通用设置表单项
  const GeneralSettings = () => (
    <div className='space-y-6'>
      <div>
        <Title level={5} className='!mb-4'>
          基础设置
        </Title>
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label='系统名称'
              name='systemName'
              rules={[{ required: true, message: '请输入系统名称' }]}
            >
              <Input placeholder='请输入系统名称' />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label='系统URL'
              name='systemUrl'
              rules={[
                { required: true, message: '请输入系统URL' },
                { type: 'url', message: '请输入有效的URL' },
              ]}
            >
              <Input placeholder='https://example.com' />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label='时区'
              name='timezone'
              rules={[{ required: true, message: '请选择时区' }]}
            >
              <Select placeholder='请选择时区'>
                <Option value='Asia/Shanghai'>亚洲/上海</Option>
                <Option value='Asia/Tokyo'>亚洲/东京</Option>
                <Option value='Europe/London'>欧洲/伦敦</Option>
                <Option value='America/New_York'>美洲/纽约</Option>
              </Select>
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label='语言'
              name='language'
              rules={[{ required: true, message: '请选择语言' }]}
            >
              <Select placeholder='请选择语言'>
                <Option value='zh-CN'>简体中文</Option>
                <Option value='en-US'>English</Option>
                <Option value='ja-JP'>日本語</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label='日期格式'
              name='dateFormat'
              rules={[{ required: true, message: '请选择日期格式' }]}
            >
              <Select placeholder='请选择日期格式'>
                <Option value='YYYY-MM-DD'>YYYY-MM-DD</Option>
                <Option value='DD/MM/YYYY'>DD/MM/YYYY</Option>
                <Option value='MM/DD/YYYY'>MM/DD/YYYY</Option>
              </Select>
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label='时间格式'
              name='timeFormat'
              rules={[{ required: true, message: '请选择时间格式' }]}
            >
              <Select placeholder='请选择时间格式'>
                <Option value='24h'>24小时制</Option>
                <Option value='12h'>12小时制</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>
      </div>

      <Divider />

      <div>
        <Title level={5} className='!mb-4'>
          会话设置
        </Title>
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label='会话超时时间(分钟)'
              name='sessionTimeout'
              rules={[{ required: true, message: '请输入会话超时时间' }]}
            >
              <InputNumber min={1} max={1440} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label='最大文件大小(MB)'
              name='maxFileSize'
              rules={[{ required: true, message: '请输入最大文件大小' }]}
            >
              <InputNumber min={1} max={1024} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item
          label='允许的文件类型'
          name='allowedFileTypes'
          rules={[{ required: true, message: '请输入允许的文件类型' }]}
        >
          <Input placeholder='.pdf,.doc,.docx,.xls,.xlsx,.jpg,.png,.gif' />
        </Form.Item>
      </div>
    </div>
  );

  // 安全设置表单项
  const SecuritySettings = () => (
    <div className='space-y-6'>
      <div>
        <Title level={5} className='!mb-4'>
          密码策略
        </Title>
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label='密码最小长度'
              name='passwordMinLength'
              rules={[{ required: true, message: '请输入密码最小长度' }]}
            >
              <InputNumber min={6} max={32} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item label='需要大写字母' name='passwordRequireUppercase' valuePropName='checked'>
              <Switch />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item label='需要小写字母' name='passwordRequireLowercase' valuePropName='checked'>
              <Switch />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item label='需要数字' name='passwordRequireNumbers' valuePropName='checked'>
              <Switch />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label='需要特殊字符'
              name='passwordRequireSpecialChars'
              valuePropName='checked'
            >
              <Switch />
            </Form.Item>
          </Col>
        </Row>
      </div>

      <Divider />

      <div>
        <Title level={5} className='!mb-4'>
          账户安全
        </Title>
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label='登录失败次数限制'
              name='loginMaxAttempts'
              rules={[{ required: true, message: '请输入登录失败次数限制' }]}
            >
              <InputNumber min={1} max={10} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label='账户锁定时间(分钟)'
              name='accountLockoutDuration'
              rules={[{ required: true, message: '请输入账户锁定时间' }]}
            >
              <InputNumber min={1} max={1440} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item label='启用双因素认证' name='enable2FA' valuePropName='checked'>
          <Switch />
        </Form.Item>
      </div>
    </div>
  );

  // 邮件设置表单项
  const EmailSettings = () => (
    <div className='space-y-6'>
      <div>
        <Title level={5} className='!mb-4'>
          SMTP设置
        </Title>
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label='SMTP服务器'
              name='smtpHost'
              rules={[{ required: true, message: '请输入SMTP服务器' }]}
            >
              <Input placeholder='smtp.example.com' />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              label='SMTP端口'
              name='smtpPort'
              rules={[{ required: true, message: '请输入SMTP端口' }]}
            >
              <InputNumber min={1} max={65535} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={[16, 16]}>
          <Col span={12}>
            <Form.Item
              label='SMTP用户名'
              name='smtpUsername'
              rules={[{ required: true, message: '请输入SMTP用户名' }]}
            >
              <Input placeholder='username@example.com' />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item label='SMTP密码' name='smtpPassword'>
              <Password placeholder='请输入密码' />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item label='启用SSL/TLS' name='smtpEnableSSL' valuePropName='checked'>
          <Switch />
        </Form.Item>
      </div>

      <Divider />

      <div>
        <Title level={5} className='!mb-4'>
          邮件模板
        </Title>
        <Form.Item
          label='发件人邮箱'
          name='emailFrom'
          rules={[
            { required: true, message: '请输入发件人邮箱' },
            { type: 'email', message: '请输入有效的邮箱地址' },
          ]}
        >
          <Input placeholder='noreply@example.com' />
        </Form.Item>

        <Form.Item label='系统通知模板' name='systemNotificationTemplate'>
          <TextArea rows={4} placeholder='请输入系统通知邮件模板' />
        </Form.Item>
      </div>
    </div>
  );

  // 标签页配置
  const tabItems = [
    {
      key: 'general',
      label: '通用设置',
      children: <GeneralSettings />,
      icon: <Settings className='w-4 h-4' />,
    },
    {
      key: 'security',
      label: '安全设置',
      children: <SecuritySettings />,
      icon: <Shield className='w-4 h-4' />,
    },
    {
      key: 'email',
      label: '邮件设置',
      children: <EmailSettings />,
      icon: <Mail className='w-4 h-4' />,
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      <div className='mb-6'>
        <Title level={2} className='!mb-2'>
          <Settings className='inline-block w-6 h-6 mr-2' />
          系统配置
        </Title>
        <Text type='secondary'>管理系统全局设置和集成配置</Text>
      </div>

      {/* 系统状态统计 */}
      <Row gutter={[16, 16]} className='mb-6'>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='系统运行时间'
              value={systemStats.uptime}
              prefix={<Clock className='w-5 h-5' />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <Statistic
              title='当前连接数'
              value={systemStats.connections}
              prefix={<Network className='w-5 h-5' />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <div className='flex items-center justify-between'>
              <div>
                <div className='text-sm text-gray-500'>磁盘使用率</div>
                <div className='text-2xl font-bold text-orange-600'>{systemStats.diskUsage}%</div>
              </div>
              <HardDrive className='w-8 h-8 text-orange-600' />
            </div>
            <Progress percent={systemStats.diskUsage} size='small' strokeColor='#fa8c16' />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className='enterprise-card'>
            <div className='flex items-center justify-between'>
              <div>
                <div className='text-sm text-gray-500'>内存使用率</div>
                <div className='text-2xl font-bold text-purple-600'>{systemStats.memoryUsage}%</div>
              </div>
              <MemoryStick className='w-8 h-8 text-purple-600' />
            </div>
            <Progress percent={systemStats.memoryUsage} size='small' strokeColor='#722ed1' />
          </Card>
        </Col>
      </Row>

      {/* 操作提示 */}
      {hasChanges && (
        <Alert
          message='配置已修改'
          description='您有未保存的配置更改，请及时保存。'
          type='warning'
          showIcon
          closable
          className='mb-6'
        />
      )}

      {/* 配置表单 */}
      <Card className='enterprise-card'>
        <Form
          form={form}
          layout='vertical'
          initialValues={config}
          onValuesChange={handleFormChange}
        >
          <div className='mb-4 flex justify-between items-center'>
            <Title level={4}>配置管理</Title>
            <Space>
              <Button icon={<RefreshCw className='w-4 h-4' />} onClick={handleReset}>
                重置
              </Button>
              <Button
                type='primary'
                icon={<Save className='w-4 h-4' />}
                loading={isSaving}
                onClick={handleSave}
              >
                {isSaving ? '保存中...' : '保存配置'}
              </Button>
            </Space>
          </div>

          <Tabs items={tabItems} type='card' className='custom-tabs' />
        </Form>
      </Card>
    </div>
  );
}
