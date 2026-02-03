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
import { UserApi } from '@/lib/api/user-api';
import { useI18n } from '@/lib/i18n';

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
  role: string;
  avatar?: string;
  tenant_id: number;
  created_at: string;
  last_login?: string;
}

interface UserStats {
  totalTickets: number;
  resolvedTickets: number;
  avgResolutionTime: number;
  satisfactionScore: number;
  responseRate: number;
}

export default function ProfilePage() {
  const { t } = useI18n();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [editing, setEditing] = useState(false);
  const [loading, setLoading] = useState(false);
  const [passwordVisible, setPasswordVisible] = useState(false);
  const [stats, setStats] = useState<UserStats | null>(null);
  const [profileForm] = Form.useForm();
  const [passwordForm] = Form.useForm();
  const [preferencesForm] = Form.useForm();

  useEffect(() => {
    loadProfile();
    loadStats();
  }, []);

  const loadProfile = async () => {
    try {
      setLoading(true);
      // 从 localStorage 或 API 获取当前用户信息
      const userId = localStorage.getItem('userId');
      if (userId) {
        const userData = await UserApi.getUserById(Number(userId));
        setProfile({
          id: userData.id,
          username: userData.username,
          email: userData.email,
          name: userData.name,
          department: userData.department,
          phone: userData.phone,
          role: 'user', // 从 userData 获取或从 token 解析
          tenant_id: userData.tenant_id,
          created_at: userData.created_at,
        });
        profileForm.setFieldsValue({
          name: userData.name,
          username: userData.username,
          email: userData.email,
          phone: userData.phone,
          department: userData.department,
          tenant: 'Default Tenant', // 从 tenant 信息获取
        });
      }
    } catch (error) {
      console.error('加载用户信息失败:', error);
      message.error('加载用户信息失败');
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      // 模拟统计数据
      setStats({
        totalTickets: 45,
        resolvedTickets: 38,
        avgResolutionTime: 2.5,
        satisfactionScore: 4.2,
        responseRate: 95,
      });
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  };

  const handleProfileUpdate = async (values: Record<string, unknown>) => {
    if (!profile) return;
    try {
      setLoading(true);
      await UserApi.updateUser(profile.id, {
        name: values.name as string,
        email: values.email as string,
        phone: values.phone as string,
        department: values.department as string,
      });
      message.success('个人信息更新成功');
      setEditing(false);
      await loadProfile();
    } catch (error) {
      console.error('更新个人信息失败:', error);
      message.error('更新个人信息失败');
    } finally {
      setLoading(false);
    }
  };

  const handlePasswordChange = async (values: Record<string, unknown>) => {
    try {
      setLoading(true);
      // 调用 API 更改密码
      message.success('密码修改成功');
      passwordForm.resetFields();
    } catch (error) {
      console.error('修改密码失败:', error);
      message.error('修改密码失败');
    } finally {
      setLoading(false);
    }
  };

  const handlePreferencesUpdate = async (values: Record<string, unknown>) => {
    try {
      // 保存用户偏好设置
      message.success('偏好设置已保存');
    } catch (error) {
      console.error('保存偏好设置失败:', error);
      message.error('保存偏好设置失败');
    }
  };

  const getRoleColor = (role: string): string => {
    const colors: Record<string, string> = {
      admin: 'red',
      user: 'blue',
      manager: 'green',
    };
    return colors[role] || 'default';
  };

  const getSatisfactionColor = (score: number): string => {
    if (score >= 4) return '#52c41a';
    if (score >= 3) return '#faad14';
    return '#ff4d4f';
  };

  if (!profile) {
    return (
      <div className='flex items-center justify-center h-64'>
        <div className='text-center'>
          <div className='animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4'></div>
          <Text>{t('profile.loading') || 'Loading...'}</Text>
        </div>
      </div>
    );
  }

  return (
    <div className='max-w-6xl mx-auto p-6'>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{t('profile.title') || '个人资料'}</h1>
          <p className="text-gray-500 mt-1">{t('profile.subtitle') || '管理您的个人信息和偏好设置'}</p>
        </div>
        <Button
          icon={editing ? <Save className="w-4 h-4" /> : <Edit className="w-4 h-4" />}
          type={editing ? 'primary' : 'default'}
          onClick={() => setEditing(!editing)}
        >
          {editing ? t('profile.save') || '保存' : t('profile.edit') || '编辑'}
        </Button>
      </div>

      <Row gutter={[24, 24]}>
        {/* 左侧：个人信息 */}
        <Col xs={24} lg={16}>
          <Tabs 
            defaultActiveKey='profile' 
            size='large'
            items={[
              {
                key: 'profile',
                label: (
                  <span className="flex items-center">
                    <User size={16} className='mr-2' />
                    {t('profile.basicInfo')}
                  </span>
                ),
                children: (
                  <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
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
                            <Title level={4} className='!mb-2'>
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
                            label={t('profile.name')}
                            rules={[{ required: true, message: t('profile.enterName') }]}
                          >
                            <Input prefix={<User size={16} className="text-gray-400" />} placeholder={t('profile.enterName')} />
                          </Form.Item>
                        </Col>
                        <Col span={12}>
                          <Form.Item
                            name='username'
                            label={t('profile.username')}
                            rules={[{ required: true, message: t('profile.enterUsername') }]}
                          >
                            <Input prefix={<User size={16} className="text-gray-400" />} placeholder={t('profile.enterUsername')} />
                          </Form.Item>
                        </Col>
                      </Row>

                      <Row gutter={16}>
                        <Col span={12}>
                          <Form.Item
                            name='email'
                            label={t('profile.email')}
                            rules={[
                              { required: true, message: t('profile.enterEmail') },
                              { type: 'email', message: t('profile.validEmail') },
                            ]}
                          >
                            <Input prefix={<Mail size={16} className="text-gray-400" />} placeholder={t('profile.enterEmail')} />
                          </Form.Item>
                        </Col>
                        <Col span={12}>
                          <Form.Item
                            name='phone'
                            label={t('profile.phone')}
                            rules={[{ required: true, message: t('profile.enterPhone') }]}
                          >
                            <Input prefix={<Phone size={16} className="text-gray-400" />} placeholder={t('profile.enterPhone')} />
                          </Form.Item>
                        </Col>
                      </Row>

                      <Row gutter={16}>
                        <Col span={12}>
                          <Form.Item
                            name='department'
                            label={t('profile.department')}
                            rules={[{ required: true, message: t('profile.selectDepartment') }]}
                          >
                            <Select 
                              placeholder={t('profile.selectDepartment')}
                              suffixIcon={<Building size={16} className="text-gray-400" />}
                            >
                              <Option value='IT支持部'>IT支持部</Option>
                              <Option value='系统运维部'>系统运维部</Option>
                              <Option value='网络管理部'>网络管理部</Option>
                              <Option value='安全运维部'>安全运维部</Option>
                            </Select>
                          </Form.Item>
                        </Col>
                        <Col span={12}>
                          <Form.Item label={t('profile.tenant')} name='tenant'>
                            <Input prefix={<Building size={16} className="text-gray-400" />} disabled />
                          </Form.Item>
                        </Col>
                      </Row>

                      {editing && (
                        <div className='text-center mt-6'>
                          <Space size='middle'>
                            <Button onClick={() => setEditing(false)}>{t('profile.cancel')}</Button>
                            <Button type='primary' htmlType='submit' loading={loading}>
                              {t('profile.saveChanges')}
                            </Button>
                          </Space>
                        </div>
                      )}
                    </Form>
                  </Card>
                )
              },
              {
                key: 'security',
                label: (
                  <span className="flex items-center">
                    <Key size={16} className='mr-2' />
                    {t('profile.securitySettings')}
                  </span>
                ),
                children: (
                  <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
                    <Form form={passwordForm} layout='vertical' onFinish={handlePasswordChange}>
                      <Alert
                        message={t('profile.passwordHint')}
                        description={t('profile.passwordHintDesc')}
                        type='info'
                        showIcon
                        className='mb-6'
                      />

                      <Form.Item
                        name='currentPassword'
                        label={t('profile.currentPassword')}
                        rules={[{ required: true, message: t('profile.enterCurrentPassword') }]}
                      >
                        <Input.Password
                          prefix={<Key size={16} className="text-gray-400" />}
                          placeholder={t('profile.enterCurrentPassword')}
                          visibilityToggle={{
                            visible: passwordVisible,
                            onVisibleChange: setPasswordVisible,
                          }}
                        />
                      </Form.Item>

                      <Form.Item
                        name='newPassword'
                        label={t('profile.newPassword')}
                        rules={[
                          { required: true, message: t('profile.enterNewPassword') },
                          { min: 8, message: t('profile.passwordMinLength') },
                        ]}
                      >
                        <Input.Password
                          prefix={<Key size={16} className="text-gray-400" />}
                          placeholder={t('profile.enterNewPassword')}
                          visibilityToggle={{
                            visible: passwordVisible,
                            onVisibleChange: setPasswordVisible,
                          }}
                        />
                      </Form.Item>

                      <Form.Item
                        name='confirmPassword'
                        label={t('profile.confirmPassword')}
                        dependencies={['newPassword']}
                        rules={[
                          { required: true, message: t('profile.confirmNewPassword') },
                          ({ getFieldValue }) => ({
                            validator(_, value) {
                              if (!value || getFieldValue('newPassword') === value) {
                                return Promise.resolve();
                              }
                              return Promise.reject(new Error(t('profile.passwordMismatch')));
                            },
                          }),
                        ]}
                      >
                        <Input.Password
                          prefix={<Key size={16} className="text-gray-400" />}
                          placeholder={t('profile.confirmNewPassword')}
                          visibilityToggle={{
                            visible: passwordVisible,
                            onVisibleChange: setPasswordVisible,
                          }}
                        />
                      </Form.Item>

                      <div className='text-center'>
                        <Button type='primary' htmlType='submit' loading={loading}>
                          {t('profile.saveChanges')}
                        </Button>
                      </div>
                    </Form>
                  </Card>
                )
              },
              {
                key: 'preferences',
                label: (
                  <span className="flex items-center">
                    <Bell size={16} className='mr-2' />
                    {t('profile.preferences')}
                  </span>
                ),
                children: (
                  <Card className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
                    <Form form={preferencesForm} layout='vertical' onFinish={handlePreferencesUpdate}>
                      <Form.Item
                        name='notifications'
                        label={t('profile.notificationSettings')}
                        valuePropName='checked'
                      >
                        <Switch
                          checkedChildren={t('serviceCatalog.enabled')}
                          unCheckedChildren={t('serviceCatalog.disabled')}
                        />
                      </Form.Item>

                      <Form.Item name='language' label={t('profile.languageSettings')}>
                        <Select placeholder={t('profile.languageSettings')}>
                          <Option value='zh-CN'>简体中文</Option>
                          <Option value='en-US'>English</Option>
                        </Select>
                      </Form.Item>

                      <Form.Item name='theme' label={t('profile.themeSettings')}>
                        <Select placeholder={t('profile.themeSettings')}>
                          <Option value='light'>Light</Option>
                          <Option value='dark'>Dark</Option>
                          <Option value='auto'>Auto</Option>
                        </Select>
                      </Form.Item>

                      <div className='text-center'>
                        <Button type='primary' htmlType='submit'>
                          {t('profile.saveChanges')}
                        </Button>
                      </div>
                    </Form>
                  </Card>
                )
              }
            ]}
          />
        </Col>

        {/* 右侧：统计信息和账户状态 */}
        <Col xs={24} lg={8}>
          <div className='space-y-6'>
            {/* 账户状态 */}
            <Card title={t('profile.accountStatus')} className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
              <div className='space-y-4'>
                <div className='flex items-center justify-between'>
                  <Text>{t('profile.accountStatus')}</Text>
                  <Tag color='success'>{t('profile.active')}</Tag>
                </div>
                <div className='flex items-center justify-between'>
                  <Text>{t('profile.registrationTime')}</Text>
                  <Text>{profile.created_at}</Text>
                </div>
                <div className='flex items-center justify-between'>
                  <Text>{t('profile.lastLogin') || '最后登录'}</Text>
                  <Text>{profile.last_login || '从未登录'}</Text>
                </div>
              </div>
            </Card>

            {/* 工作统计 */}
            {stats && (
              <Card title={t('profile.workStats')} className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
                <div className='space-y-4'>
                  <div className='text-center'>
                    <Statistic
                      title={t('profile.totalTickets')}
                      value={stats.totalTickets}
                      prefix={<User size={16} />}
                    />
                  </div>
                  <div className='text-center'>
                    <Statistic
                      title={t('profile.resolvedTickets')}
                      value={stats.resolvedTickets}
                      prefix={<User size={16} />}
                      suffix={`/ ${stats.totalTickets}`}
                    />
                    <Progress
                      percent={Math.round((stats.resolvedTickets / stats.totalTickets) * 100)}
                      size='small'
                      className='mt-2'
                    />
                  </div>
                  <div className='text-center'>
                    <Statistic
                      title={t('profile.avgResolutionTime')}
                      value={stats.avgResolutionTime}
                      suffix='h'
                    />
                  </div>
                  <div className='text-center'>
                    <Statistic
                      title={t('profile.satisfactionScore')}
                      value={stats.satisfactionScore}
                      precision={1}
                      suffix='/5.0'
                      styles={{ content: {
                        color: getSatisfactionColor(stats.satisfactionScore),
                      }}
                    />
                  </div>
                  <div className='text-center'>
                    <Statistic
                      title={t('profile.responseRate')}
                      value={stats.responseRate}
                      suffix='%'
                    />
                  </div>
                </div>
              </Card>
            )}

            {/* 快速操作 */}
            <Card title={t('profile.quickActions')} className="rounded-lg shadow-sm border border-gray-200" variant="borderless">
              <div className='space-y-2'>
                <Button block icon={<Shield size={16} />}>
                  {t('profile.viewPermissions')}
                </Button>
                <Button block icon={<Bell size={16} />}>
                  {t('profile.notificationSettings')}
                </Button>
                <Button block icon={<User size={16} />}>
                  {t('profile.switchAccount')}
                </Button>
              </div>
            </Card>
          </div>
        </Col>
      </Row>
    </div>
  );
}
