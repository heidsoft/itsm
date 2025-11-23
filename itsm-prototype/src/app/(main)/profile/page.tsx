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
import { useI18n } from '@/lib/i18n';

const { Title, Text } = Typography;
const { Option } = Select;
const { TabPane } = Tabs;

// ... (interfaces remain the same)

export default function ProfilePage() {
  const { t } = useI18n();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  // ... (rest of state)

  // ... (loadProfile and loadStats functions)

  // ... (handlers)

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
      <PageHeader
        title={t('profile.title')}
        subtitle={t('profile.subtitle')}
        breadcrumbs={[{ label: t('profile.title') }]}
        actions={
          <Button
            icon={editing ? <Save /> : <Edit />}
            type={editing ? 'primary' : 'default'}
            onClick={() => setEditing(!editing)}
          >
            {editing ? t('profile.save') : t('profile.edit')}
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
                  {t('profile.basicInfo')}
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
                        label={t('profile.name')}
                        rules={[{ required: true, message: t('profile.enterName') }]}
                      >
                        <Input prefix={<User />} placeholder={t('profile.enterName')} />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item
                        name='username'
                        label={t('profile.username')}
                        rules={[{ required: true, message: t('profile.enterUsername') }]}
                      >
                        <Input prefix={<User />} placeholder={t('profile.enterUsername')} />
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
                        <Input prefix={<Mail />} placeholder={t('profile.enterEmail')} />
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item
                        name='phone'
                        label={t('profile.phone')}
                        rules={[{ required: true, message: t('profile.enterPhone') }]}
                      >
                        <Input prefix={<Phone />} placeholder={t('profile.enterPhone')} />
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
                        <Select prefix={<Building />} placeholder={t('profile.selectDepartment')}>
                          <Option value='IT支持部'>IT支持部</Option>
                          <Option value='系统运维部'>系统运维部</Option>
                          <Option value='网络管理部'>网络管理部</Option>
                          <Option value='安全运维部'>安全运维部</Option>
                        </Select>
                      </Form.Item>
                    </Col>
                    <Col span={12}>
                      <Form.Item label={t('profile.tenant')} name='tenant'>
                        <Input prefix={<Building />} disabled />
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
            </TabPane>

            <TabPane
              tab={
                <span>
                  <Key size={16} className='mr-2' />
                  {t('profile.securitySettings')}
                </span>
              }
              key='security'
            >
              <Card>
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
                      prefix={<Key />}
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
                      prefix={<Key />}
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
                      prefix={<Key />}
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
            </TabPane>

            <TabPane
              tab={
                <span>
                  <Bell size={16} className='mr-2' />
                  {t('profile.preferences')}
                </span>
              }
              key='preferences'
            >
              <Card>
                <Form form={preferencesForm} layout='vertical' onFinish={handlePreferencesUpdate}>
                  <Form.Item name='notifications' label={t('profile.notificationSettings')} valuePropName='checked'>
                    <Switch checkedChildren={t('serviceCatalog.enabled')} unCheckedChildren={t('serviceCatalog.disabled')} />
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
            </TabPane>
          </Tabs>
        </Col>

        {/* 右侧：统计信息和账户状态 */}
        <Col xs={24} lg={8}>
          <div className='space-y-6'>
            {/* 账户状态 */}
            <Card title={t('profile.accountStatus')}>
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
                  <Text>{t('profile.lastLogin')}</Text>
                  <Text>{profile.last_login}</Text>
                </div>
              </div>
            </Card>

            {/* 工作统计 */}
            {stats && (
              <Card title={t('profile.workStats')}>
                <div className='space-y-4'>
                  <div className='text-center'>
                    <Statistic title={t('profile.totalTickets')} value={stats.totalTickets} prefix={<User />} />
                  </div>
                  <div className='text-center'>
                    <Statistic
                      title={t('profile.resolvedTickets')}
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
                    <Statistic title={t('profile.avgResolutionTime')} value={stats.avgResolutionTime} suffix='h' />
                  </div>
                  <div className='text-center'>
                    <Statistic
                      title={t('profile.satisfactionScore')}
                      value={stats.satisfactionScore}
                      precision={1}
                      suffix='/5.0'
                      valueStyle={{
                        color: getSatisfactionColor(stats.satisfactionScore),
                      }}
                    />
                  </div>
                  <div className='text-center'>
                    <Statistic title={t('profile.responseRate')} value={stats.responseRate} suffix='%' />
                  </div>
                </div>
              </Card>
            )}

            {/* 快速操作 */}
            <Card title={t('profile.quickActions')}>
              <div className='space-y-2'>
                <Button block icon={<Shield />}>
                  {t('profile.viewPermissions')}
                </Button>
                <Button block icon={<Bell />}>
                  {t('profile.notificationSettings')}
                </Button>
                <Button block icon={<User />}>
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
