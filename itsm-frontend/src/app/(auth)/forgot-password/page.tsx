'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Mail, ArrowLeft, CheckCircle, Shield } from 'lucide-react';
import { useI18n } from '@/lib/i18n/useI18n';
import {
  Typography, Form, Input, Button, Card, ConfigProvider, message, Divider, Alert
} from 'antd';
import { antdTheme } from '@/lib/antd-theme';
import { AuthService } from '@/lib/services/auth-service';
import { logger } from '@/lib/env';

const { Text, Title } = Typography;

/**
 * 忘记密码页面组件
 */
export default function ForgotPasswordPage() {
  const router = useRouter();
  const [form] = Form.useForm();
  const { t } = useI18n();
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [email, setEmail] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = async (values: { email: string }) => {
    setLoading(true);
    setEmail(values.email);
    setError('');

    try {
      const success = await AuthService.forgotPassword(values.email);
      if (success) {
        setSubmitted(true);
      } else {
        setError(t('auth.forgotPassword.sendFailed'));
      }
    } catch (err) {
      logger.error('发送失败:', err);
      setError(t('auth.forgotPassword.sendFailed'));
    } finally {
      setLoading(false);
    }
  };

  const handleResend = async () => {
    setLoading(true);
    try {
      const success = await AuthService.forgotPassword(email);
      if (success) {
        message.success(t('auth.forgotPassword.sendSuccess'));
      }
    } catch (err) {
      message.error(t('auth.forgotPassword.sendFailed'));
    } finally {
      setLoading(false);
    }
  };

  if (submitted) {
    return (
      <ConfigProvider theme={antdTheme}>
        <div className="min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50">
          <div className="w-full max-w-[420px]">
            <Card className="rounded-xl shadow-xl border-none" styles={{ body: { padding: '40px' } }}>
              <div className="text-center">
                <div className="w-16 h-16 mx-auto mb-4 bg-green-100 rounded-full flex items-center justify-center">
                  <CheckCircle size={32} className="text-green-600" />
                </div>

                <Title level={3} className="!mb-2 !text-gray-900 !text-xl">
                  检查您的邮箱
                </Title>

                <Text className="!text-gray-500 !text-sm block !mb-6">
                  我们已将密码重置链接发送至<br />
                  <span className="text-blue-600 font-medium">{email}</span>
                </Text>

                <div className="bg-gray-50 rounded-lg p-4 mb-6 text-left">
                  <Text className="!text-gray-600 !text-xs block !mb-2">
                    没有收到邮件？请检查：
                  </Text>
                  <ul className="!text-gray-500 !text-xs list-disc list-inside space-y-1">
                    <li>垃圾邮件文件夹</li>
                    <li>邮箱地址是否正确</li>
                    <li>邮件是否被拦截</li>
                  </ul>
                </div>

                <Button type='link' className="mb-4" onClick={() => setSubmitted(false)} icon={<ArrowLeft size={14} />}>
                  {t('auth.forgotPassword.backToLogin')}
                </Button>

                <Button type='primary' size='large' className="w-full h-10 rounded-md text-sm" loading={loading} onClick={handleResend}>
                  重新发送
                </Button>

                <div className="text-center mt-4">
                  <Text className="text-gray-400 text-xs">
                    记起密码了？{' '}
                    <a href="/login" className="text-blue-600 hover:underline">
                      {t('auth.register.loginNow')}
                    </a>
                  </Text>
                </div>
              </div>
            </Card>
          </div>
        </div>
      </ConfigProvider>
    );
  }

  return (
    <ConfigProvider theme={antdTheme}>
      <div className="min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50">
        <div className="w-full max-w-[420px]">
          <Card className="rounded-xl shadow-xl border-none" styles={{ body: { padding: '40px' } }}>
            <div className="text-center mb-6">
              <Title level={2} className="!mb-2 !text-gray-900 !text-2xl">
                {t('auth.forgotPassword.title')}
              </Title>
              <Text className="!text-gray-500 !text-sm">
                {t('auth.forgotPassword.subtitle')}
              </Text>
            </div>

            {error && (
              <Alert message={t('auth.forgotPassword.sendFailed')} description={error} type="error" className="mb-4" showIcon />
            )}

            <Form form={form} layout='vertical' size='middle' onFinish={handleSubmit}>
              <Form.Item
                name='email'
                label={t('auth.forgotPassword.emailLabel')}
                rules={[
                  { required: true, message: t('auth.forgotPassword.emailRequired') },
                  { type: 'email', message: t('auth.forgotPassword.emailInvalid') }
                ]}
              >
                <Input
                  prefix={<Mail size={14} className="text-gray-400" />}
                  placeholder={t('auth.forgotPassword.emailPlaceholder')}
                  size='large'
                  disabled={loading}
                />
              </Form.Item>

              <Form.Item>
                <Button type='primary' htmlType='submit' size='large' className="w-full h-10 rounded-md text-sm font-semibold" loading={loading}>
                  {loading ? t('auth.forgotPassword.sending') : t('auth.forgotPassword.sendButton')}
                </Button>
              </Form.Item>
            </Form>

            <div className="text-center">
              <Text className="text-gray-400 text-xs">
                记起密码了？{' '}
                <a href="/login" className="text-blue-600 hover:underline">
                  {t('auth.register.loginNow')}
                </a>
              </Text>
            </div>

            <Divider className="my-5">
              <Text className="text-gray-400 text-xs">{t('auth.login.or')}</Text>
            </Divider>

            <Button size='middle' className="w-full h-10 rounded-md text-sm" disabled={loading} icon={<Shield size={14} />}>
              SSO 企业登录
            </Button>
          </Card>
        </div>
      </div>
    </ConfigProvider>
  );
}
