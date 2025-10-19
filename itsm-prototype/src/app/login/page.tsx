'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { User, Lock, Shield, ArrowRight, Eye, EyeOff } from 'lucide-react';
import {
  Typography,
  Space,
  Alert,
  Divider,
  ConfigProvider,
  Form,
  Input,
  Button,
  Checkbox,
  Card,
  Row,
  Col,
  Flex,
} from 'antd';
import { antdTheme } from '@/app/lib/antd-theme';
import { colors } from '@/lib/design-system/colors';

const { Text, Title } = Typography;

/**
 * 登录页面组件
 * 使用统一的设计系统和Ant Design组件，保持与系统内部一致的视觉风格
 */
export default function LoginPage() {
  const router = useRouter();
  const [form] = Form.useForm();

  // 状态管理
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [rememberMe, setRememberMe] = useState(false);

  // 真实的后端认证服务
  const authService = {
    async login(username: string, password: string) {
      try {
        const response = await fetch('http://localhost:8090/api/v1/login', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            username,
            password,
            tenant_code: '', // 可选，暂时为空
          }),
        });

        if (!response.ok) {
          const errorData = await response.json();
          throw new Error(errorData.message || '登录失败');
        }

        const data = await response.json();
        console.log('后端响应:', data);

        if (data.code === 0 && data.data) {
          return {
            success: true,
            user: data.data.user,
            token: data.data.access_token,
            refreshToken: data.data.refresh_token,
            tenant: data.data.tenant,
          };
        } else {
          throw new Error(data.message || '登录失败');
        }
      } catch (error) {
        console.error('登录请求失败:', error);
        throw error;
      }
    },
  };

  // 处理登录提交
  const handleLogin = async (values: { username: string; password: string }) => {
    console.log('开始登录:', values);
    setLoading(true);
    setError('');

    try {
      const result = await authService.login(values.username, values.password);
      console.log('登录结果:', result);

      if (result.success) {
        // 存储认证信息到localStorage
        localStorage.setItem('auth_token', result.token);
        localStorage.setItem('refresh_token', result.refreshToken);
        localStorage.setItem('user_info', JSON.stringify(result.user));
        localStorage.setItem('tenant_info', JSON.stringify(result.tenant));

        // 设置认证cookie（中间件需要）
        document.cookie = `auth-token=${result.token}; path=/; max-age=900; SameSite=Lax`; // 15分钟
        document.cookie = `refresh-token=${result.refreshToken}; path=/; max-age=604800; SameSite=Lax`; // 7天
        console.log('认证信息已存储，准备跳转');

        // 跳转到仪表板
        router.push('/dashboard');
        console.log('已执行跳转命令');
      }
    } catch (err) {
      console.error('登录错误:', err);
      setError(err instanceof Error ? err.message : '登录失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  return (
    <ConfigProvider theme={antdTheme}>
      <div
        style={{
          minHeight: '100vh',
          background: `linear-gradient(135deg, ${colors.functional.background.secondary} 0%, ${colors.primary[50]} 100%)`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '20px',
        }}
      >
        <div style={{ width: '100%', maxWidth: '1000px' }}>
          <Row gutter={[32, 0]} align='middle'>
            {/* 左侧品牌区域 - 紧凑版 */}
            <Col xs={0} lg={10}>
              <div
                style={{
                  background: `linear-gradient(135deg, ${colors.primary[600]} 0%, ${colors.primary[700]} 100%)`,
                  padding: '40px 32px',
                  borderRadius: '12px',
                  height: '480px',
                  display: 'flex',
                  flexDirection: 'column',
                  justifyContent: 'center',
                  position: 'relative',
                  overflow: 'hidden',
                }}
              >
                {/* 简化的装饰元素 */}
                <div
                  style={{
                    position: 'absolute',
                    top: '-30px',
                    right: '-30px',
                    width: '120px',
                    height: '120px',
                    background: 'rgba(255, 255, 255, 0.1)',
                    borderRadius: '50%',
                    filter: 'blur(20px)',
                  }}
                />

                <div style={{ position: 'relative', zIndex: 1 }}>
                  <Title
                    level={1}
                    style={{
                      color: 'white',
                      marginBottom: '12px',
                      fontSize: '28px',
                      fontWeight: '700',
                    }}
                  >
                    ITSM Pro
                  </Title>
                  <Text
                    style={{
                      color: 'rgba(255, 255, 255, 0.9)',
                      fontSize: '14px',
                      display: 'block',
                      marginBottom: '32px',
                    }}
                  >
                    智能IT服务管理平台
                  </Text>

                  {/* 简化的特性列表 */}
                  <Space direction='vertical' size='middle' style={{ width: '100%' }}>
                    <Flex align='center' gap={10}>
                      <div
                        style={{
                          width: '32px',
                          height: '32px',
                          background: 'rgba(255, 255, 255, 0.2)',
                          borderRadius: '6px',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                        }}
                      >
                        <Shield size={16} color='white' />
                      </div>
                      <div>
                        <Text
                          style={{
                            color: 'white',
                            fontWeight: '600',
                            fontSize: '14px',
                            display: 'block',
                          }}
                        >
                          企业级安全
                        </Text>
                        <Text style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: '12px' }}>
                          多层安全防护
                        </Text>
                      </div>
                    </Flex>

                    <Flex align='center' gap={10}>
                      <div
                        style={{
                          width: '32px',
                          height: '32px',
                          background: 'rgba(255, 255, 255, 0.2)',
                          borderRadius: '6px',
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'center',
                        }}
                      >
                        <ArrowRight size={16} color='white' />
                      </div>
                      <div>
                        <Text
                          style={{
                            color: 'white',
                            fontWeight: '600',
                            fontSize: '14px',
                            display: 'block',
                          }}
                        >
                          智能自动化
                        </Text>
                        <Text style={{ color: 'rgba(255, 255, 255, 0.8)', fontSize: '12px' }}>
                          AI驱动流程
                        </Text>
                      </div>
                    </Flex>
                  </Space>
                </div>
              </div>
            </Col>

            {/* 右侧登录表单 - 紧凑版 */}
            <Col xs={24} lg={14}>
              <Card
                style={{
                  borderRadius: '12px',
                  boxShadow: '0 8px 24px rgba(0, 0, 0, 0.12)',
                  border: 'none',
                }}
                styles={{ body: { padding: '40px' } }}
              >
                <div style={{ textAlign: 'center', marginBottom: '24px' }}>
                  <Title
                    level={2}
                    style={{
                      marginBottom: '8px',
                      color: colors.functional.text.primary,
                      fontSize: '24px',
                    }}
                  >
                    欢迎回来
                  </Title>
                  <Text style={{ color: colors.functional.text.secondary, fontSize: '14px' }}>
                    请登录您的账户以继续使用服务
                  </Text>
                </div>

                {/* 错误提示 */}
                {error && (
                  <Alert message={error} type='error' style={{ marginBottom: '20px' }} showIcon />
                )}

                {/* 登录表单 */}
                <Form form={form} onFinish={handleLogin} layout='vertical' size='middle'>
                  <Form.Item
                    name='username'
                    label='用户名'
                    rules={[
                      { required: true, message: '请输入用户名' },
                      { min: 3, message: '用户名至少3个字符' },
                    ]}
                  >
                    <Input
                      prefix={<User size={14} style={{ color: colors.functional.text.tertiary }} />}
                      placeholder='请输入用户名'
                      disabled={loading}
                    />
                  </Form.Item>

                  <Form.Item
                    name='password'
                    label='密码'
                    rules={[
                      { required: true, message: '请输入密码' },
                      { min: 6, message: '密码至少6个字符' },
                    ]}
                  >
                    <Input.Password
                      prefix={<Lock size={14} style={{ color: colors.functional.text.tertiary }} />}
                      placeholder='请输入密码'
                      disabled={loading}
                    />
                  </Form.Item>

                  <Form.Item style={{ marginBottom: '20px' }}>
                    <Flex justify='space-between' align='center'>
                      <Checkbox
                        checked={rememberMe}
                        onChange={e => setRememberMe(e.target.checked)}
                        disabled={loading}
                      >
                        记住我
                      </Checkbox>
                      <Button
                        type='link'
                        style={{ padding: 0, height: 'auto', fontSize: '12px' }}
                        disabled={loading}
                      >
                        忘记密码？
                      </Button>
                    </Flex>
                  </Form.Item>

                  <Form.Item>
                    <Button
                      type='primary'
                      htmlType='submit'
                      loading={loading}
                      size='large'
                      style={{
                        width: '100%',
                        height: '40px',
                        borderRadius: '6px',
                        fontSize: '14px',
                        fontWeight: '600',
                      }}
                      icon={<ArrowRight size={14} />}
                    >
                      {loading ? '登录中...' : '登录'}
                    </Button>
                  </Form.Item>
                </Form>

                {/* 其他登录方式 */}
                <Divider style={{ margin: '20px 0' }}>
                  <Text style={{ color: colors.functional.text.tertiary, fontSize: '12px' }}>
                    或
                  </Text>
                </Divider>

                <Button
                  size='middle'
                  style={{
                    width: '100%',
                    height: '40px',
                    borderRadius: '6px',
                    fontSize: '14px',
                  }}
                  disabled={loading}
                  icon={<Shield size={14} />}
                >
                  SSO 单点登录
                </Button>

                {/* 底部链接 */}
                <div style={{ textAlign: 'center', marginTop: '20px' }}>
                  <Text style={{ color: colors.functional.text.tertiary, fontSize: '12px' }}>
                    还没有账户？{' '}
                    <Button type='link' style={{ padding: 0, height: 'auto', fontSize: '12px' }}>
                      立即注册
                    </Button>
                  </Text>
                </div>
              </Card>
            </Col>
          </Row>
        </div>
      </div>
    </ConfigProvider>
  );
}
