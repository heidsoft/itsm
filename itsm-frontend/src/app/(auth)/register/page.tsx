'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Mail, Lock, User, Building2, Phone, ArrowRight,
  CheckCircle, AlertCircle, Shield
} from 'lucide-react';
import {
  Typography, Form, Input, Button, Card, Row, Col, Flex,
  Divider, ConfigProvider, message, Select
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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [step, setStep] = useState(1); // 1: 基本信息, 2: 详细信息

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
        message.success('注册成功！请前往登录');
        router.push('/login');
      } else {
        setError('注册失败，请稍后重试');
      }
    } catch (err) {
      logger.error('注册错误:', err);
      setError(err instanceof Error ? err.message : '注册失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  // 密码强度检查
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
                创建账户
              </Title>
              <Text className="!text-gray-500 !text-sm">
                立即注册，体验智能IT服务管理
              </Text>
            </div>

            {error && (
              <AlertCircle className="mb-4 text-red-500 w-full" />
            )}

            <Form form={form} layout='vertical' size='middle' onFinish={handleRegister}>
              {step === 1 ? (
                <>
                  <Form.Item
                    name='username'
                    label='用户名'
                    rules={[
                      { required: true, message: '请输入用户名' },
                      { min: 3, message: '用户名至少3个字符' },
                      { max: 20, message: '用户名最多20个字符' },
                      { pattern: /^[a-zA-Z0-9_]+$/, message: '用户名只能包含字母、数字和下划线' }
                    ]}
                  >
                    <Input
                      prefix={<User size={14} className="text-gray-400" />}
                      placeholder='请输入用户名'
                      disabled={loading}
                    />
                  </Form.Item>

                  <Form.Item
                    name='email'
                    label='邮箱'
                    rules={[
                      { required: true, message: '请输入邮箱' },
                      { type: 'email', message: '请输入有效的邮箱地址' }
                    ]}
                  >
                    <Input
                      prefix={<Mail size={14} className="text-gray-400" />}
                      placeholder='请输入邮箱'
                      disabled={loading}
                    />
                  </Form.Item>

                  <Form.Item
                    name='password'
                    label='密码'
                    rules={[
                      { required: true, message: '请输入密码' },
                      { min: 8, message: '密码至少8个字符' }
                    ]}
                  >
                    <Input.Password
                      prefix={<Lock size={14} className="text-gray-400" />}
                      placeholder='请输入密码'
                      disabled={loading}
                    />
                  </Form.Item>

                  {password && password.length > 0 && (
                    <div className="mb-4">
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
                    label='确认密码'
                    dependencies={['password']}
                    rules={[
                      { required: true, message: '请确认密码' },
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
                      placeholder='请再次输入密码'
                      disabled={loading}
                    />
                  </Form.Item>

                  <Form.Item>
                    <Button
                      type='primary'
                      htmlType='button'
                      size='large'
                      className="w-full h-10 rounded-md text-sm font-semibold"
                      icon={<ArrowRight size={14} />}
                      onClick={handleNextStep}
                      disabled={loading}
                    >
                      下一步
                    </Button>
                  </Form.Item>
                </>
              ) : (
                <>
                  <Form.Item
                    name='fullName'
                    label='姓名'
                    rules={[{ required: true, message: '请输入姓名' }]}
                  >
                    <Input
                      prefix={<User size={14} className="text-gray-400" />}
                      placeholder='请输入您的姓名'
                      disabled={loading}
                    />
                  </Form.Item>

                  <Form.Item
                    name='phone'
                    label='手机号'
                    rules={[
                      { required: true, message: '请输入手机号' },
                      { pattern: /^1[3-9]\d{9}$/, message: '请输入有效的手机号' }
                    ]}
                  >
                    <Input
                      prefix={<Phone size={14} className="text-gray-400" />}
                      placeholder='请输入手机号'
                      disabled={loading}
                    />
                  </Form.Item>

                  <Form.Item
                    name='company'
                    label='公司名称'
                  >
                    <Input
                      prefix={<Building2 size={14} className="text-gray-400" />}
                      placeholder='请输入公司名称（选填）'
                      disabled={loading}
                    />
                  </Form.Item>

                  <Form.Item
                    name='role'
                    label='角色'
                    initialValue='end_user'
                  >
                    <Select
                      placeholder="请选择您的角色"
                      options={[
                        { value: 'end_user', label: '普通用户' },
                        { value: 'it_admin', label: 'IT管理员' },
                        { value: 'manager', label: '经理' },
                      ]}
                    />
                  </Form.Item>

                  <Form.Item
                    name='terms'
                    valuePropName='checked'
                    rules={[
                      {
                        validator: (_, value) =>
                          value ? Promise.resolve() : Promise.reject(new Error('请同意服务条款')),
                      },
                    ]}
                  >
                    <div className="flex items-center gap-2">
                      <input type="checkbox" id="terms" />
                      <label htmlFor="terms" className="text-sm text-gray-600">
                        我已阅读并同意
                        <a href="#" className="text-blue-600 hover:underline"> 服务条款</a>
                        和
                        <a href="#" className="text-blue-600 hover:underline"> 隐私政策</a>
                      </label>
                    </div>
                  </Form.Item>

                  <Form.Item>
                    <Flex gap={12}>
                      <Button
                        size='large'
                        className="flex-1 h-10 rounded-md text-sm"
                        onClick={() => setStep(1)}
                        disabled={loading}
                      >
                        返回
                      </Button>
                      <Button
                        type='primary'
                        htmlType='submit'
                        size='large'
                        className="flex-1 h-10 rounded-md text-sm font-semibold"
                        loading={loading}
                        icon={<CheckCircle size={14} />}
                      >
                        {loading ? '注册中...' : '立即注册'}
                      </Button>
                    </Flex>
                  </Form.Item>
                </>
              )}
            </Form>

            <Divider className="my-5">
              <Text className="text-gray-400 text-xs">或</Text>
            </Divider>

            <Button
              size='middle'
              className="w-full h-10 rounded-md text-sm"
              disabled={loading}
              icon={<Shield size={14} />}
            >
              SSO 企业登录
            </Button>

            <div className="text-center mt-5">
              <Text className="text-gray-400 text-xs">
                已有账户？{' '}
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
