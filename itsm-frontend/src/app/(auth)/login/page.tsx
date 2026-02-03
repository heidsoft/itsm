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
import { antdTheme } from '@/lib/antd-theme';
import { colors } from '@/lib/design-system/colors';
import { AuthService } from '@/lib/services/auth-service';
import { logger } from '@/lib/env';
import { useAuthStoreHydration } from '@/lib/store/auth-store';

const { Text, Title } = Typography;

/**
 * 登录页面组件
 * 使用统一的设计系统和Ant Design组件，保持与系统内部一致的视觉风格
 */
export default function LoginPage() {
  const router = useRouter();
  const [form] = Form.useForm();

  // Hydrate auth store
  useAuthStoreHydration();

  // 状态管理
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [rememberMe, setRememberMe] = useState(false);

  // 处理登录提交
  const handleLogin = async (values: { username: string; password: string }) => {
    logger.info('开始登录:', values);
    setLoading(true);
    setError('');

    try {
      const success = await AuthService.login(values.username, values.password);

      if (success) {
        logger.info('认证信息已存储，准备跳转');
        // 跳转到仪表板
        router.push('/dashboard');
        logger.info('已执行跳转命令');
      } else {
        // AuthService.login already handles error messages via console.error,
        // but for UI display, we can catch a generic message or specific ones if AuthService throws them.
        setError('登录失败，请检查您的用户名和密码');
      }
    } catch (err) {
      logger.error('登录错误:', err);
      setError(err instanceof Error ? err.message : '登录失败，请重试');
    } finally {
      setLoading(false);
    }
  };

  return (
    <ConfigProvider theme={antdTheme}>
      <div className="min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50">
        <div className="w-full max-w-[1000px]">
          <Row gutter={[32, 0]} align='middle'>
            {/* 左侧品牌区域 - 紧凑版 */}
            <Col xs={0} lg={10}>
              <div className="bg-gradient-to-br from-blue-600 to-blue-700 p-10 px-8 rounded-xl h-[480px] flex flex-col justify-center relative overflow-hidden">
                {/* 简化的装饰元素 */}
                <div className="absolute -top-8 -right-8 w-32 h-32 bg-white/10 rounded-full blur-xl" />

                <div className="relative z-10">
                  <Title
                    level={1}
                    className="!text-white !mb-3 !text-3xl !font-bold"
                  >
                    ITSM Pro
                  </Title>
                  <Text className="!text-white/90 !text-sm block !mb-8">
                    智能IT服务管理平台
                  </Text>

                  {/* 简化的特性列表 */}
                  <Space orientation='vertical' size='middle' className="w-full">
                    <Flex align='center' gap={10}>
                      <div className="w-8 h-8 bg-white/20 rounded-md flex items-center justify-center">
                        <Shield size={16} color='white' />
                      </div>
                      <div>
                        <Text className="!text-white !font-semibold !text-sm block">
                          企业级安全
                        </Text>
                        <Text className="!text-white/80 !text-xs">
                          多层安全防护
                        </Text>
                      </div>
                    </Flex>

                    <Flex align='center' gap={10}>
                      <div className="w-8 h-8 bg-white/20 rounded-md flex items-center justify-center">
                        <ArrowRight size={16} color='white' />
                      </div>
                      <div>
                        <Text className="!text-white !font-semibold !text-sm block">
                          智能自动化
                        </Text>
                        <Text className="!text-white/80 !text-xs">
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
                className="rounded-xl shadow-xl border-none"
                styles={{ body: { padding: '40px' } }}
              >
                <div className="text-center mb-6">
                  <Title
                    level={2}
                    className="!mb-2 !text-gray-900 !text-2xl"
                  >
                    欢迎回来
                  </Title>
                  <Text className="!text-gray-500 !text-sm">
                    请登录您的账户以继续使用服务
                  </Text>
                </div>

                {/* 错误提示 */}
                {error && (
                  <Alert message="登录失败" description={error} type='error' className="mb-5" showIcon />
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
                      prefix={<User size={14} className="text-gray-400" />}
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
                      prefix={<Lock size={14} className="text-gray-400" />}
                      placeholder='请输入密码'
                      disabled={loading}
                    />
                  </Form.Item>

                  <Form.Item className="mb-5">
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
                        className="p-0 h-auto text-xs"
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
                      className="w-full h-10 rounded-md text-sm font-semibold"
                      icon={<ArrowRight size={14} />}
                    >
                      {loading ? '登录中...' : '登录'}
                    </Button>
                  </Form.Item>
                </Form>

                {/* 其他登录方式 */}
                <Divider className="my-5">
                  <Text className="text-gray-400 text-xs">
                    或
                  </Text>
                </Divider>

                <Button
                  size='middle'
                  className="w-full h-10 rounded-md text-sm"
                  disabled={loading}
                  icon={<Shield size={14} />}
                >
                  SSO 单点登录
                </Button>

                {/* 底部链接 */}
                <div className="text-center mt-5">
                  <Text className="text-gray-400 text-xs">
                    还没有账户？{' '}
                    <Button type='link' className="p-0 h-auto text-xs">
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
