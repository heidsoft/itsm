'use client';

import React, { useState, useEffect, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Lock, CheckCircle, AlertCircle, ArrowRight } from 'lucide-react';
import {
  Typography, Form, Input, Button, Card, Row, Col, Flex,
  ConfigProvider, message
} from 'antd';
import { antdTheme } from '@/lib/antd-theme';
import { AuthService } from '@/lib/services/auth-service';
import { logger } from '@/lib/env';

const { Text, Title } = Typography;

/**
 * 重置密码内容组件
 * 包含useSearchParams使用，需要Suspense包装
 */
function ResetPasswordContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [tokenValid, setTokenValid] = useState(true);
  const [success, setSuccess] = useState(false);

  const token = searchParams.get('token');
  const email = searchParams.get('email');

  useEffect(() => {
    // 验证 token 是否存在
    if (!token) {
      setTokenValid(false);
      return;
    }

    // 验证 token 是否有效
    const validateToken = async () => {
      try {
        const isValid = await AuthService.validateResetToken(token, email || '');
        if (!isValid) {
          setTokenValid(false);
        }
      } catch (err) {
        logger.error('验证令牌失败:', err);
        setTokenValid(false);
      }
    };

    if (token && email) {
      validateToken();
    }
  }, [token, email]);

  const getPasswordStrength = (password: string) => {
    let strength = 0;
    if (password.length >= 8) strength++;
    if (/[A-Z]/.test(password)) strength++;
    if (/[a-z]/.test(password)) strength++;
    if (/[0-9]/.test(password)) strength++;
    if (/[^A-Za-z0-9]/.test(password)) strength++;
    return strength;
  };

  const handleReset = async (values: { password: string; confirmPassword: string }) => {
    setLoading(true);

    try {
      const success = await AuthService.resetPassword({
        token: token || '',
        email: email || '',
        password: values.password,
        passwordConfirm: values.confirmPassword,
      });

      if (success) {
        setSuccess(true);
        message.success('密码重置成功！');

        // 3秒后跳转到登录页
        setTimeout(() => {
          router.push('/login');
        }, 3000);
      }
    } catch (err) {
      logger.error('重置密码失败:', err);
      message.error(err instanceof Error ? err.message : '重置失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  if (!tokenValid) {
    return (
      <ConfigProvider theme={antdTheme}>
        <div className="min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50">
          <div className="w-full max-w-[420px]">
            <Card className="rounded-xl shadow-xl border-none" styles={{ body: { padding: '40px' } }}>
              <div className="text-center">
                <div className="w-16 h-16 mx-auto mb-4 bg-red-100 rounded-full flex items-center justify-center">
                  <AlertCircle size={32} className="text-red-600" />
                </div>

                <Title level={3} className="!mb-2 !text-gray-900 !text-xl">
                  链接已失效
                </Title>

                <Text className="!text-gray-500 !text-sm block !mb-6">
                  密码重置链接已过期或无效<br />
                  请重新发起密码重置请求
                </Text>

                <Button
                  type='primary'
                  size='large'
                  className="w-full h-10 rounded-md text-sm font-semibold"
                  onClick={() => router.push('/forgot-password')}
                >
                  重新发送请求
                </Button>

                <div className="text-center mt-4">
                  <Text className="text-gray-400 text-xs">
                    记起密码了？{' '}
                    <a href="/login" className="text-blue-600 hover:underline">
                      立即登录
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

  if (success) {
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
                  密码重置成功
                </Title>

                <Text className="!text-gray-500 !text-sm block !mb-6">
                  您的密码已成功重置<br />
                  正在跳转到登录页...
                </Text>

                <Button
                  type='primary'
                  size='large'
                  className="w-full h-10 rounded-md text-sm font-semibold"
                  onClick={() => router.push('/login')}
                >
                  立即登录
                </Button>
              </div>
            </Card>
          </div>
        </div>
      </ConfigProvider>
    );
  }

  const password = form.getFieldValue('password');
  const strength = getPasswordStrength(password || '');
  const strengthColors = ['#ff4d4f', '#ff4d4f', '#faad14', '#faad14', '#52c41a', '#52c41a'];
  const strengthLabels = ['非常弱', '弱', '一般', '中等', '强', '非常强'];

  return (
    <ConfigProvider theme={antdTheme}>
      <div className="min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50">
        <div className="w-full max-w-[420px]">
          <Card className="rounded-xl shadow-xl border-none" styles={{ body: { padding: '40px' } }}>
            <div className="text-center mb-6">
              <Title level={2} className="!mb-2 !text-gray-900 !text-2xl">
                设置新密码
              </Title>
              <Text className="!text-gray-500 !text-sm">
                请输入您的新密码
              </Text>
              {email && (
                <Text className="!text-gray-400 !text-xs block !mt-1">
                  账户：{email}
                </Text>
              )}
            </div>

            <Form form={form} layout='vertical' size='middle' onFinish={handleReset}>
              <Form.Item
                name='password'
                label='新密码'
                rules={[
                  { required: true, message: '请输入新密码' },
                  { min: 8, message: '密码至少8个字符' }
                ]}
              >
                <Input.Password
                  prefix={<Lock size={14} className="text-gray-400" />}
                  placeholder='请输入新密码'
                  size='large'
                  disabled={loading}
                />
              </Form.Item>

              {password && password.length > 0 && (
                <div className="mb-4 -mt-2">
                  <div className="flex items-center gap-2 mb-1">
                    <div className="flex-1 h-1.5 bg-gray-200 rounded-full overflow-hidden">
                      <div
                        className="h-full rounded-full transition-all duration-300"
                        style={{
                          width: `${(strength / 5) * 100}%`,
                          backgroundColor: strengthColors[strength - 1] || '#ff4d4f'
                        }}
                      />
                    </div>
                    <Text className="text-xs" style={{ color: strengthColors[strength - 1] || '#ff4d4f' }}>
                      {strengthLabels[strength - 1] || '非常弱'}
                    </Text>
                  </div>
                </div>
              )}

              <Form.Item
                name='confirmPassword'
                label='确认新密码'
                dependencies={['password']}
                rules={[
                  { required: true, message: '请确认新密码' },
                  ({ getFieldValue }) => ({
                    validator(_, value) {
                      if (!value || getFieldValue('password') === value) {
                        return Promise.resolve();
                      }
                      return Promise.reject(new Error('两次输入的密码不一致'));
                    },
                  }),
                ]}
              >
                <Input.Password
                  prefix={<Lock size={14} className="text-gray-400" />}
                  placeholder='请再次输入新密码'
                  size='large'
                  disabled={loading}
                />
              </Form.Item>

              <Form.Item className="mb-2">
                <div className="bg-gray-50 rounded-lg p-3">
                  <Text className="!text-gray-600 !text-xs block !mb-2">
                    密码要求：
                  </Text>
                  <ul className="!text-gray-500 !text-xs list-disc list-inside space-y-0.5">
                    <li>至少8个字符</li>
                    <li>包含大小写字母</li>
                    <li>包含数字或特殊字符</li>
                  </ul>
                </div>
              </Form.Item>

              <Form.Item>
                <Button
                  type='primary'
                  htmlType='submit'
                  size='large'
                  className="w-full h-10 rounded-md text-sm font-semibold"
                  loading={loading}
                  icon={<ArrowRight size={14} />}
                >
                  {loading ? '重置中...' : '确认重置'}
                </Button>
              </Form.Item>
            </Form>

            <div className="text-center">
              <Text className="text-gray-400 text-xs">
                记起密码了？{' '}
                <a href="/login" className="text-blue-600 hover:underline">
                  立即登录
                </a>
              </Text>
            </div>
          </Card>
        </div>
      </div>
    </ConfigProvider>
  );
}

/**
 * 重置密码页面组件
 */
export default function ResetPasswordPage() {
  return (
    <Suspense fallback={
      <ConfigProvider theme={antdTheme}>
        <div className="min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50">
          <Card className="rounded-xl shadow-xl border-none" styles={{ body: { padding: '40px' } }}>
            <div className="text-center">
              <Title level={3} className="!mb-2 !text-gray-900 !text-xl">
                正在加载...
              </Title>
            </div>
          </Card>
        </div>
      </ConfigProvider>
    }>
      <ResetPasswordContent />
    </Suspense>
  );
}
