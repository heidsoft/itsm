'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Form,
  Input,
  Button,
  Avatar,
  Typography,
  Space,
  Divider,
  message,
  Row,
  Col,
  Tabs,
  Upload,
  Select,
  Switch,
  Alert,
  Statistic,
  Progress,
  Tag,
} from 'antd';
import { User, Mail, Phone, Building, Shield, Bell, Key, Camera, Save, Edit } from 'lucide-react';
import { PageHeader } from '@/components/layout/PageHeader';
import { UserAPI } from '../lib/user-api';

const { Title, Text } = Typography;
const { Option } = Select;
const { TabPane } = Tabs;

interface UserProfile {
  id: number;
  username: string;
  email: string;
  name: string;
  department: string;
  phone: string;
  avatar?: string;
  role: string;
  tenant: string;
  created_at: string;
  last_login: string;
  preferences: {
    notifications: boolean;
    language: string;
    theme: string;
  };
}

interface ProfileStats {
  totalTickets: number;
  resolvedTickets: number;
  avgResolutionTime: number;
  satisfactionScore: number;
  responseRate: number;
}

export default function ProfilePage() {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [stats, setStats] = useState<ProfileStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [editing, setEditing] = useState(false);
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [profileForm] = Form.useForm();
  const [passwordForm] = Form.useForm();
  const [preferencesForm] = Form.useForm();

  useEffect(() => {
    loadProfile();
    loadStats();
  }, []);

  const loadProfile = async () => {
    try {
      // 模拟API调用
      const mockProfile: UserProfile = {
        id: 1,
        username: 'zhangsan',
        email: 'zhangsan@company.com',
        name: '张三',
        department: 'IT支持部',
        phone: '13800138000',
        avatar: '',
        role: '技术支持工程师',
        tenant: '公司总部',
        created_at: '2024-01-01',
        last_login: '2024-01-15 14:30:00',
        preferences: {
          notifications: true,
          language: 'zh-CN',
          theme: 'light',
        },
      };

      setProfile(mockProfile);
      profileForm.setFieldsValue(mockProfile);
      preferencesForm.setFieldsValue(mockProfile.preferences);
    } catch (error) {
      message.error('加载个人信息失败');
    }
  };

  const loadStats = async () => {
    try {
      // 模拟统计数据
      const mockStats: ProfileStats = {
        totalTickets: 156,
        resolvedTickets: 142,
        avgResolutionTime: 4.2,
        satisfactionScore: 4.5,
        responseRate: 95.8,
      };

      setStats(mockStats);
    } catch (error) {
      message.error('加载统计数据失败');
    }
  };

  const handleProfileUpdate = async (values: any) => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));

      setProfile(prev => (prev ? { ...prev, ...values } : null));
      setEditing(false);
      message.success('个人信息更新成功');
    } catch (error) {
      message.error('更新失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  const handlePasswordChange = async (values: any) => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));

      passwordForm.resetFields();
      message.success('密码修改成功');
    } catch (error) {
      message.error('密码修改失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  const handlePreferencesUpdate = async (values: any) => {
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));

      setProfile(prev =>
        prev
          ? {
              ...prev,
              preferences: { ...prev.preferences, ...values },
            }
          : null
      );

      message.success('偏好设置更新成功');
    } catch (error) {
      message.error('偏好设置更新失败');
    }
  };

  const getRoleColor = (role: string) => {
    switch (role) {
      case '技术支持工程师':
        return 'blue';
      case '系统管理员':
        return 'purple';
      case '服务台经理':
        return 'gold';
      default:
        return 'default';
    }
  };

  const getSatisfactionColor = (score: number) => {
    if (score >= 4.5) return '#52c41a';
    if (score >= 4.0) return '#1890ff';
    if (score >= 3.5) return '#faad14';
    return '#f5222d';
  };

  if (!profile) {
    return (
      <div className='flex items-center justify-center h-64'>
        <div className='text-center'>
          <div className='animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4'></div>
          <Text>加载中...</Text>
        </div>
      </div>
    );
  }

  return (
    <div className='max-w-6xl mx-auto p-6'>
      <PageHeader
        title='个人资料'
        subtitle='管理您的个人信息、偏好设置和安全选项'
        breadcrumbs={[{ label: '个人资料' }]}
        actions={
          <Button
            icon={editing ? <Save /> : <Edit />}
            type={editing ? 'primary' : 'default'}
            onClick={() => setEditing(!editing)}
          >
            {editing ? '保存' : '编辑'}
          </Button>
        }
      />

      <Row gutter={[24, 24]}>
        {/* 左侧：个人信息 */}
        <Col xs={24} lg={16}>
          <Tabs defaultActiveKey='profile' size='large'>
            <TabPane
              tab={
                <span>
                  <User size={16} className='mr-2' />
                  基本信息
                </span>
              }
              key='profile'
            >
              <Card>
                <Form
                  form={profileForm}
                  layout='vertical'
                  onFinish={handleProfileUpdate}
                  disabled={!editing}
                >
                  <Row gutter={16}>
                    <Col span={24} className='text-center mb-6'>
                      <div className='relative inline-block'>
                        <Avatar
                          size={120}
                          src={profile.avatar}
                          icon={<User size={60} />}
                          className='border-4 border-gray-100 shadow-lg'
                        />
                        {editing && (
                          <Button
                            type='primary'
                            shape='circle'
                            icon={<Camera size={16} />}
                            size='small'
                            className='absolute bottom-0 right-0'
                          />
                        )}
                      </div>
                      <div className='mt-4'>
                        <Title level={4} className='mb-2'>
                          {profile.name}
                        </Title>
                        <Tag color={getRoleColor(profile.role)}>{profile.role}</Tag>
                      </div>
                    </Col>
                  </Row>

                  <Row gutter={16}>
                    <Col span={12}>
                      <Form.Item
                        name='name'
                        label='姓名'
                        rules={[{ required: true, message: '请输入姓名' }]}
                      >
                        <Input prefix={<User />} placeholder='请输入姓名' />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item
                        name='username'
                        label='用户名'
                        rules={[{ required: true, message: '请输入用户名' }]}
                      >
                        <Input prefix={<User />} placeholder='请输入用户名' />
                      </Form.Item>
                    </Col>
                  </Row>

                  <Row gutter={16}>
                    <Col span={12}>
                      <Form.Item
                        name='email'
                        label='邮箱'
                        rules={[
                          { required: true, message: '请输入邮箱' },
                          { type: 'email', message: '请输入有效的邮箱地址' },
                        ]}
                      >
                        <Input prefix={<Mail />} placeholder='请输入邮箱' />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item
                        name='phone'
                        label='手机号'
                        rules={[{ required: true, message: '请输入手机号' }]}
                      >
                        <Input prefix={<Phone />} placeholder='请输入手机号' />
                      </Form.Item>
                    </Col>
                  </Row>

                  <Row gutter={16}>
                    <Col span={12}>
                      <Form.Item
                        name='department'
                        label='部门'
                        rules={[{ required: true, message: '请选择部门' }]}
                      >
                        <Select prefix={<Building />} placeholder='请选择部门'>
                          <Option value='IT支持部'>IT支持部</Option>
                          <Option value='系统运维部'>系统运维部</Option>
                          <Option value='网络管理部'>网络管理部</Option>
                          <Option value='安全运维部'>安全运维部</Option>
                        </Select>
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item label='租户' name='tenant'>
                        <Input prefix={<Building />} disabled />
                      </Form.Item>
                    </Col>
                  </Row>

                  {editing && (
                    <div className='text-center mt-6'>
                      <Space size='middle'>
                        <Button onClick={() => setEditing(false)}>取消</Button>
                        <Button type='primary' htmlType='submit' loading={loading}>
                          保存更改
                        </Button>
                      </Space>
                    </div>
                  )}
                </Form>
              </Card>
            </TabPane>

            <TabPane
              tab={
                <span>
                  <Key size={16} className='mr-2' />
                  安全设置
                </span>
              }
              key='security'
            >
              <Card>
                <Form form={passwordForm} layout='vertical' onFinish={handlePasswordChange}>
                  <Alert
                    message='密码安全提示'
                    description='建议使用包含大小写字母、数字和特殊字符的强密码，长度至少8位。'
                    type='info'
                    showIcon
                    className='mb-6'
                  />

                  <Form.Item
                    name='currentPassword'
                    label='当前密码'
                    rules={[{ required: true, message: '请输入当前密码' }]}
                  >
                    <Input.Password
                      prefix={<Key />}
                      placeholder='请输入当前密码'
                      visibilityToggle={{
                        visible: passwordVisible,
                        onVisibleChange: setPasswordVisible,
                      }}
                    />
                  </Form.Item>

                  <Form.Item
                    name='newPassword'
                    label='新密码'
                    rules={[
                      { required: true, message: '请输入新密码' },
                      { min: 8, message: '密码长度至少8位' },
                    ]}
                  >
                    <Input.Password
                      prefix={<Key />}
                      placeholder='请输入新密码'
                      visibilityToggle={{
                        visible: passwordVisible,
                        onVisibleChange: setPasswordVisible,
                      }}
                    />
                  </Form.Item>

                  <Form.Item
                    name='confirmPassword'
                    label='确认新密码'
                    dependencies={['newPassword']}
                    rules={[
                      { required: true, message: '请确认新密码' },
                      ({ getFieldValue }) => ({
                        validator(_, value) {
                          if (!value || getFieldValue('newPassword') === value) {
                            return Promise.resolve();
                          }
                          return Promise.reject(new Error('两次输入的密码不一致'));
                        },
                      }),
                    ]}
                  >
                    <Input.Password
                      prefix={<Key />}
                      placeholder='请确认新密码'
                      visibilityToggle={{
                        visible: passwordVisible,
                        onVisibleChange: setPasswordVisible,
                      }}
                    />
                  </Form.Item>

                  <div className='text-center'>
                    <Button type='primary' htmlType='submit' loading={loading}>
                      修改密码
                    </Button>
                  </div>
                </Form>
              </Card>
            </TabPane>

            <TabPane
              tab={
                <span>
                  <Bell size={16} className='mr-2' />
                  偏好设置
                </span>
              }
              key='preferences'
            >
              <Card>
                <Form form={preferencesForm} layout='vertical' onFinish={handlePreferencesUpdate}>
                  <Form.Item name='notifications' label='通知设置' valuePropName='checked'>
                    <Switch checkedChildren='开启' unCheckedChildren='关闭' />
                  </Form.Item>

                  <Form.Item name='language' label='语言设置'>
                    <Select placeholder='请选择语言'>
                      <Option value='zh-CN'>简体中文</Option>
                      <Option value='en-US'>English</Option>
                    </Select>
                  </Form.Item>

                  <Form.Item name='theme' label='主题设置'>
                    <Select placeholder='请选择主题'>
                      <Option value='light'>浅色主题</Option>
                      <Option value='dark'>深色主题</Option>
                      <Option value='auto'>跟随系统</Option>
                    </Select>
                  </Form.Item>

                  <div className='text-center'>
                    <Button type='primary' htmlType='submit'>
                      保存偏好设置
                    </Button>
                  </div>
                </Form>
              </Card>
            </TabPane>
          </Tabs>
        </Col>

        {/* 右侧：统计信息和账户状态 */}
        <Col xs={24} lg={8}>
          <div className='space-y-6'>
            {/* 账户状态 */}
            <Card title='账户状态'>
              <div className='space-y-4'>
                <div className='flex items-center justify-between'>
                  <Text>账户状态</Text>
                  <Tag color='success'>活跃</Tag>
                </div>
                <div className='flex items-center justify-between'>
                  <Text>注册时间</Text>
                  <Text>{profile.created_at}</Text>
                </div>
                <div className='flex items-center justify-between'>
                  <Text>最后登录</Text>
                  <Text>{profile.last_login}</Text>
                </div>
              </div>
            </Card>

            {/* 工作统计 */}
            {stats && (
              <Card title='工作统计'>
                <div className='space-y-4'>
                  <div className='text-center'>
                    <Statistic title='总工单数' value={stats.totalTickets} prefix={<User />} />
                  </div>
                  <div className='text-center'>
                    <Statistic
                      title='已解决工单'
                      value={stats.resolvedTickets}
                      prefix={<User />}
                      suffix={`/ ${stats.totalTickets}`}
                    />
                    <Progress
                      percent={Math.round((stats.resolvedTickets / stats.totalTickets) * 100)}
                      size='small'
                      className='mt-2'
                    />
                  </div>
                  <div className='text-center'>
                    <Statistic title='平均解决时间' value={stats.avgResolutionTime} suffix='小时' />
                  </div>
                  <div className='text-center'>
                    <Statistic
                      title='满意度评分'
                      value={stats.satisfactionScore}
                      precision={1}
                      suffix='/5.0'
                      valueStyle={{
                        color: getSatisfactionColor(stats.satisfactionScore),
                      }}
                    />
                  </div>
                  <div className='text-center'>
                    <Statistic title='响应率' value={stats.responseRate} suffix='%' />
                  </div>
                </div>
              </Card>
            )}

            {/* 快速操作 */}
            <Card title='快速操作'>
              <div className='space-y-2'>
                <Button block icon={<Shield />}>
                  查看权限
                </Button>
                <Button block icon={<Bell />}>
                  通知设置
                </Button>
                <Button block icon={<User />}>
                  切换账户
                </Button>
              </div>
            </Card>
          </div>
        </Col>
      </Row>
    </div>
  );
}
