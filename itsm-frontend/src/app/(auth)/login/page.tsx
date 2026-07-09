'use client';

import React, { useState, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { User, Lock, Shield, ArrowRight } from 'lucide-react';
import { useI18n } from '@/lib/i18n/useI18n';
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
  Tooltip,
} from 'antd';
import { antdTheme } from '@/lib/antd-theme';
import { AuthService } from '@/lib/services/auth-service';
import { logger } from '@/lib/env';
import { useAuthStoreHydration } from '@/lib/store/auth-store';

const { Text, Title } = Typography;

/**
 * 登录表单子组件
 * 使用 useSearchParams 读取 expired 参数，需要被 Suspense 包裹
 */
function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { t } = useI18n();
  const [form] = Form.useForm();

  // Hydrate auth store
  useAuthStoreHydration();

  // 状态管理
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [rememberMe, setRememberMe] = useState(false);

  // 检查会话过期标记
  const isExpired = searchParams.get('expired') === 'true';
  const redirectPath = searchParams.get('redirect') || '/dashboard';

  // 处理登录提交
  const handleLogin = async (values: { username: string; password: string }) => {
    logger.info('开始登录:', values);
    setLoading(true);
    setError('');

    try {
      const success = await AuthService.login(
        values.username,
        values.password,
        undefined,
        rememberMe
      );

      if (success) {
        logger.info('认证信息已存储，准备跳转');
        router.push(redirectPath);
        logger.info('已执行跳转命令');
      } else {
        setError(t('auth.login.loginFailed'));
      }
    } catch (err) {
      logger.error('登录错误:', err);
      setError(err instanceof Error ? err.message : t('auth.login.loginFailed'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card className='rounded-xl shadow-xl border-none' styles={{ body: { padding: '40px' } }}>
      <div className='text-center mb-6'>
        <Title level={2} className='!mb-2 !text-gray-900 !text-2xl'>
          {t('auth.login.title')}
        </Title>
        <Text className='!text-gray-500 !text-sm'>{t('auth.login.subtitle')}</Text>
      </div>

      {/* 默认账号提示：开发环境使用 admin/admin123，生产环境必须修改 */}
      {process.env.NODE_ENV !== 'production' && (
        <Alert
          title='开发环境默认账号'
          description={
            <span>
              用户名：<strong>admin</strong>　密码：<strong>admin123</strong>
              <br />
              生产环境部署前必须修改默认密码、JWT_SECRET、数据库密码。
            </span>
          }
          type='info'
          className='mb-5'
          showIcon
          closable
        />
      )}

      {/* 会话过期提示 */}
      {isExpired && (
        <Alert
          title='会话已过期'
          description='您的会话已过期，请重新登录。'
          type='warning'
          className='mb-5'
          showIcon
          closable
        />
      )}

      {error && (
        <Alert
          title={t('auth.login.loginFailed')}
          description={error}
          type='error'
          className='mb-5'
          showIcon
        />
      )}

      <Form
        form={form}
        onFinish={handleLogin}
        onFinishFailed={({ values, errorFields }) => {
          logger.warn('表单验证失败:', errorFields);
          if (errorFields.length > 0) {
            setError(errorFields[0].errors[0] || t('auth.login.loginFailed'));
          }
        }}
        layout='vertical'
        size='middle'
      >
        <Form.Item
          name='username'
          label={t('auth.login.usernameLabel')}
          rules={[
            { required: true, message: t('auth.login.usernameRequired') },
            { min: 3, message: t('auth.login.usernameMinLength') },
          ]}
        >
          <Input
            prefix={<User size={14} className='text-gray-400' />}
            placeholder={t('auth.login.usernamePlaceholder')}
            disabled={loading}
          />
        </Form.Item>

        <Form.Item
          name='password'
          label={t('auth.login.passwordLabel')}
          rules={[
            { required: true, message: t('auth.login.passwordRequired') },
            { min: 6, message: t('auth.login.passwordMinLength') },
          ]}
        >
          <Input.Password
            prefix={<Lock size={14} className='text-gray-400' />}
            placeholder={t('auth.login.passwordPlaceholder')}
            disabled={loading}
          />
        </Form.Item>

        <Form.Item className='mb-5'>
          <Flex justify='space-between' align='center'>
            <Checkbox
              checked={rememberMe}
              onChange={e => setRememberMe(e.target.checked)}
              disabled={loading}
            >
              {t('auth.login.rememberMe')}
            </Checkbox>
            <Tooltip title={loading ? '登录中...' : ''}>
              <Link href='/forgot-password'>
                <Button type='link' className='p-0 h-auto text-xs' disabled={loading}>
                  {t('auth.login.forgotPassword')}
                </Button>
              </Link>
            </Tooltip>
          </Flex>
        </Form.Item>

        <Form.Item>
          <Button
            type='primary'
            htmlType='submit'
            loading={loading}
            size='large'
            className='w-full h-10 rounded-md text-sm font-semibold'
            icon={<ArrowRight size={14} />}
          >
            {loading ? t('auth.login.loggingIn') : t('auth.login.loginButton')}
          </Button>
        </Form.Item>
      </Form>

      {/* 默认凭据提示 */}
      <Alert
        title='默认管理员账户'
        description='admin / admin123（请及时修改密码）'
        type='info'
        className='mb-5'
        showIcon
      />

      <Divider className='my-5'>
        <Text className='text-gray-400 text-xs'>{t('auth.login.or')}</Text>
      </Divider>

      {/* SSO 登录按钮已隐藏 — 开源版本暂不支持 SSO */}
      {false && (
        <Button
          size='middle'
          className='w-full h-10 rounded-md text-sm'
          disabled={loading}
          icon={<Shield size={14} />}
        >
          {t('auth.login.ssoLogin')}
        </Button>
      )}

      <div className='text-center mt-5'>
        <Text className='text-gray-400 text-xs'>
          {t('auth.login.noAccount')}{' '}
          <Link href='/register'>
            <Button type='link' className='p-0 h-auto text-xs'>
              {t('auth.login.registerNow')}
            </Button>
          </Link>
        </Text>
      </div>
    </Card>
  );
}

/**
 * 登录页面组件
 * 使用统一的设计系统和 Ant Design 组件，保持与系统内部一致的视觉风格
 */
export default function LoginPage() {
  return (
    <ConfigProvider theme={antdTheme}>
      <div className='min-h-screen flex items-center justify-center p-5 bg-gradient-to-br from-gray-50 to-blue-50'>
        <div className='w-full max-w-[1000px]'>
          <Row gutter={[32, 0]} align='middle'>
            {/* 左侧品牌区域 */}
            <Col xs={0} lg={10}>
              <div className='bg-gradient-to-br from-blue-600 to-blue-700 p-10 px-8 rounded-xl h-[480px] flex flex-col justify-center relative overflow-hidden'>
                <div className='absolute -top-8 -right-8 w-32 h-32 bg-white/10 rounded-full blur-xl' />

                <div className='relative z-10'>
                  <Title level={1} className='!text-white !mb-3 !text-3xl !font-bold'>
                    AI-Native ITSM
                  </Title>
                  <Text className='!text-white/90 !text-sm block !mb-8'>
                    AI 驱动的 IT 服务管理系统
                  </Text>

                  <Space orientation='vertical' size='middle' className='w-full'>
                    <Flex align='center' gap={10}>
                      <div className='w-8 h-8 bg-white/20 rounded-md flex items-center justify-center'>
                        <Shield size={16} color='white' />
                      </div>
                      <div>
                        <Text className='!text-white !font-semibold !text-sm block'>
                          企业级安全
                        </Text>
                        <Text className='!text-white/80 !text-xs'>多层安全防护</Text>
                      </div>
                    </Flex>

                    <Flex align='center' gap={10}>
                      <div className='w-8 h-8 bg-white/20 rounded-md flex items-center justify-center'>
                        <ArrowRight size={16} color='white' />
                      </div>
                      <div>
                        <Text className='!text-white !font-semibold !text-sm block'>
                          智能自动化
                        </Text>
                        <Text className='!text-white/80 !text-xs'>AI 驱动流程</Text>
                      </div>
                    </Flex>
                  </Space>
                </div>
              </div>
            </Col>

            {/* 右侧登录表单 — 使用 Suspense 包裹 useSearchParams */}
            <Col xs={24} lg={14}>
              <Suspense
                fallback={
                  <Card
                    className='rounded-xl shadow-xl border-none'
                    styles={{ body: { padding: '40px' } }}
                  >
                    <div className='text-center py-20 text-gray-400'>加载中...</div>
                  </Card>
                }
              >
                <LoginForm />
              </Suspense>
            </Col>
          </Row>
        </div>
      </div>
    </ConfigProvider>
  );
}
