'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Mail, Lock, User, Building2, Phone, ArrowRight,
  CheckCircle, AlertCircle, Shield
} from 'lucide-react';
import { useI18n } from '@/lib/i18n/useI18n';
import {
  Typography, Form, Input, Button, Card, Row, Col, Flex,
  Divider, ConfigProvider, message, Select, Alert
} from 'antd';
import { antdTheme } from '@/lib/antd-theme';
import { AuthService } from '@/lib/services/auth-service';
import { logger } from '@/lib/env';

const { Text, Title } = Typography;

/**
 * 注册页面组件
 */
export default function RegisterPage() {
  const router = useRouter();
  const [form] = Form.useForm();
  const { t } = useI18n();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [step, setStep] = useState(1);

  const handleNextStep = async () => {
    try {
      await form.validateFields(['username', 'email', 'password', 'confirmPassword']);
      setStep(2);
    } catch (err) {
      // Validation failed, stay on current step
    }
  };

  const handleRegister = async (values: Record<string, string>) => {
    setLoading(true);
    setError('');

    try {
      const success = await AuthService.register({
        username: values.username,
        email: values.email,
        password: values.password,
        fullName: values.fullName,
        phone: values.phone,
        company: values.company,
        role: values.role,
      });

      if (success) {
        message.success(t('auth.register.registerSuccess'));
        router.push('/login');
      } else {
        setError(t('auth.register.registerFailed'));
      }
    } catch (err) {
      logger.error('注册错误:', err);
      setError(err instanceof Error ? err.message : t('auth.register.registerFailed'));
    } finally {
      setLoading(false);
    }
  };

  const getPasswordStrength = (password: string) => {
    let strength = 0;
    if (password.length >= 8) strength++;
    if (/[A-Z]/.test(password)) strength++;
    if (/[a-z]/.test(password)) strength++;
    if (/[0-9]/.test(password)) strength++;
    if (/[^A-Za-z0-9]/.test(password)) strength++;
    return strength;
  };

  const password = form.getFieldValue('password');
  const strength = getPasswordStrength(password || '');
  const strengthColors = ['#ff4d4f', '#ff4d4f', '#faad14', '#faad14', '#52c41a', '#52c41a'];
  const strengthLabels = ['非常弱', '弱', '一般', '中等', '强', '非常强'];

  return (
    <ConfigProvider theme={antdTheme}>
      <div className="min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50">
        <div className="w-full max-w-[480px]">
          <Card className="rounded-xl shadow-xl border-none" styles={{ body: { padding: '40px' } }}>
            <div className="text-center mb-6">
              <Title level={2} className="!mb-2 !text-gray-900 !text-2xl">
                {t('auth.register.title')}
              </Title>
              <Text className="!text-gray-500 !text-sm">
                {t('auth.register.subtitle')}
              </Text>
            </div>

            {error && (
              <Alert message={t('auth.register.registerFailed')} description={error} type="error" className="mb-4" showIcon />
            )}

            {step === 1 && (
              <>
                <Form form={form} onFinish={handleRegister} layout='vertical' size='middle'>
                  <Form.Item
                    name='username'
                    label={t('auth.register.usernameLabel')}
                    rules={[
                      { required: true, message: t('auth.register.usernameRequired') },
                      { min: 3, message: t('auth.register.usernameMinLength') },
                    ]}
                  >
                    <Input prefix={<User size={14} className="text-gray-400" />} placeholder={t('auth.register.usernamePlaceholder')} disabled={loading} />
                  </Form.Item>

                  <Form.Item
                    name='email'
                    label={t('auth.register.emailLabel')}
                    rules={[
                      { required: true, message: t('auth.register.emailRequired') },
                      { type: 'email', message: t('auth.register.emailInvalid') },
                    ]}
                  >
                    <Input prefix={<Mail size={14} className="text-gray-400" />} placeholder={t('auth.register.emailPlaceholder')} disabled={loading} />
                  </Form.Item>

                  <Form.Item
                    name='password'
                    label={t('auth.register.passwordLabel')}
                    rules={[
                      { required: true, message: t('auth.register.passwordRequired') },
                      { min: 8, message: t('auth.register.passwordMinLength') },
                    ]}
                  >
                    <Input.Password prefix={<Lock size={14} className="text-gray-400" />} placeholder={t('auth.register.passwordPlaceholder')} disabled={loading} />
                  </Form.Item>

                  <Form.Item
                    name='confirmPassword'
                    label={t('auth.register.confirmPasswordLabel')}
                    dependencies={['password']}
                    rules={[
                      { required: true, message: t('auth.register.confirmPasswordRequired') },
                      ({ getFieldValue }) => ({
                        validator(_, value) {
                          if (!value || getFieldValue('password') === value) {
                            return Promise.resolve();
                          }
                          return Promise.reject(new Error(t('auth.register.passwordMismatch')));
                        },
                      }),
                    ]}
                  >
                    <Input.Password prefix={<Lock size={14} className="text-gray-400" />} placeholder={t('auth.register.confirmPasswordPlaceholder')} disabled={loading} />
                  </Form.Item>

                  <Form.Item>
                    <Button type='primary' onClick={handleNextStep} className="w-full" icon={<ArrowRight size={14} />}>
                      下一步
                    </Button>
                  </Form.Item>
                </Form>

                <Divider className="my-4">
                  <Text className="text-gray-400 text-xs">{t('auth.login.or')}</Text>
                </Divider>

                <div className="text-center">
                  <Text className="text-gray-400 text-xs">
                    {t('auth.register.hasAccount')}{' '}
                    <Button type='link' className="p-0 h-auto text-xs" onClick={() => router.push('/login')}>
                      {t('auth.register.loginNow')}
                    </Button>
                  </Text>
                </div>
              </>
            )}

            {step === 2 && (
              <>
                <Form form={form} onFinish={handleRegister} layout='vertical' size='middle'>
                  <Form.Item
                    name='fullName'
                    label={t('auth.register.nameLabel')}
                    rules={[{ required: true, message: t('auth.register.nameRequired') }]}
                  >
                    <Input prefix={<User size={14} className="text-gray-400" />} placeholder={t('auth.register.namePlaceholder')} disabled={loading} />
                  </Form.Item>

                  <Form.Item
                    name='phone'
                    label={t('auth.register.phoneLabel')}
                    rules={[{ required: true, message: t('auth.register.phoneRequired') }]}
                  >
                    <Input prefix={<Phone size={14} className="text-gray-400" />} placeholder={t('auth.register.phonePlaceholder')} disabled={loading} />
                  </Form.Item>

                  <Form.Item name='company' label={t('auth.register.companyLabel')}>
                    <Input prefix={<Building2 size={14} className="text-gray-400" />} placeholder={t('auth.register.companyPlaceholder')} disabled={loading} />
                  </Form.Item>

                  <Form.Item
                    name='role'
                    label={t('auth.register.roleLabel')}
                    rules={[{ required: true, message: t('auth.register.roleRequired') }]}
                  >
                    <Select placeholder={t('auth.register.rolePlaceholder')} disabled={loading}>
                      <Select.Option value="developer">开发人员</Select.Option>
                      <Select.Option value="manager">项目经理</Select.Option>
                      <Select.Option value="admin">系统管理员</Select.Option>
                      <Select.Option value="user">普通用户</Select.Option>
                    </Select>
                  </Form.Item>

                  <Form.Item>
                    <Flex gap={8}>
                      <Button onClick={() => setStep(1)} className="flex-1" disabled={loading}>
                        上一步
                      </Button>
                      <Button type='primary' htmlType='submit' loading={loading} className="flex-1" icon={<CheckCircle size={14} />}>
                        {loading ? t('auth.register.registering') : t('auth.register.registerButton')}
                      </Button>
                    </Flex>
                  </Form.Item>
                </Form>
              </>
            )}
          </Card>
        </div>
      </div>
    </ConfigProvider>
  );
}
