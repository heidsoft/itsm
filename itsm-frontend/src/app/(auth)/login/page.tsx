'use client';

import React, { useState, useEffect, useRef, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { User, Lock, Shield, ArrowRight } from 'lucide-react';
import { useI18n } from '@/lib/i18n/useI18n';
import {
  Typography,
  Space,
  Alert,
  ConfigProvider,
  Form,
  Input,
  Button,
  Card,
  Divider,
  Row,
  Col,
  Flex,
} from 'antd';
import { antdTheme } from '@/lib/antd-theme';
import { AuthService } from '@/lib/services/auth-service';
import { AuthAPI, type SSOProviderBrief } from '@/lib/api/auth-api';
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
  const [countdown, setCountdown] = useState(0); // 限流倒计时（秒）
  const countdownTimer = useRef<ReturnType<typeof setInterval> | null>(null);
  // SSO 提供方（登录页数据源）。加载失败保持空数组：SSO 不可用时登录页
  // 只隐藏入口，账号密码登录必须始终可用。
  const [ssoProviders, setSsoProviders] = useState<SSOProviderBrief[]>([]);
  const [ssoLoading, setSsoLoading] = useState<string | null>(null);

  // 清理倒计时定时器
  useEffect(() => {
    return () => {
      if (countdownTimer.current) {
        clearInterval(countdownTimer.current);
      }
    };
  }, []);

  // 加载启用中的 SSO 提供方（无提供方/后端未接入时静默为空）
  useEffect(() => {
    let cancelled = false;
    AuthAPI.listSSOProviders().then(providers => {
      if (!cancelled && providers.length > 0) {
        setSsoProviders(providers);
      }
    });
    return () => {
      cancelled = true;
    };
  }, []);

  // 发起 SSO 登录：拿授权 URL 后整页跳转到 IdP。
  // 跳转是浏览器级导航（非同源 fetch），授权码由 IdP 回调到 /sso/callback 页处理。
  const handleSSOLogin = async (providerCode: string) => {
    setSsoLoading(providerCode);
    setError('');
    try {
      const result = await AuthAPI.initiateSSOLogin(providerCode);
      if (!result.success || !result.redirectUrl) {
        setError(result.error || t('auth.login.ssoFailed'));
        setSsoLoading(null);
        return;
      }
      logger.info('跳转 SSO 提供方', { provider: providerCode });
      window.location.href = result.redirectUrl;
    } catch (err) {
      logger.error('SSO 登录发起失败:', err);
      setError(err instanceof Error ? err.message : t('auth.login.ssoFailed'));
      setSsoLoading(null);
    }
  };

  // 启动倒计时
  const startCountdown = (seconds: number) => {
    if (countdownTimer.current) {
      clearInterval(countdownTimer.current);
    }
    setCountdown(seconds);
    setLoading(true);
    countdownTimer.current = setInterval(() => {
      setCountdown(prev => {
        if (prev <= 1) {
          if (countdownTimer.current) {
            clearInterval(countdownTimer.current);
            countdownTimer.current = null;
          }
          setLoading(false);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };

  // 检查会话过期标记
  const isExpired = searchParams.get('expired') === 'true';
  // 开放重定向防护：仅允许站内相对路径。
  // `//evil.com` 是协议相对 URL，浏览器会当作外部主机跳转；`http(s)://`、
  // 反斜杠变体（`\evil.com` 部分浏览器归一化为 //）同样拒绝，一律回退 /dashboard。
  const rawRedirect = searchParams.get('redirect') || '/dashboard';
  const redirectPath =
    rawRedirect.startsWith('/') && !rawRedirect.startsWith('//') && !rawRedirect.startsWith('/\\')
      ? rawRedirect
      : '/dashboard';

  // 处理登录提交
  const handleLogin = async (values: { username: string; password: string }) => {
    // 只记录用户名：表单值含明文密码，不得进入日志。
    logger.info('开始登录:', { username: values.username });
    setLoading(true);
    setError('');

    try {
      const result = await AuthService.login(values.username, values.password);

      if (result.ok) {
        setLoading(false);
        logger.info('认证信息已存储，准备跳转');
        try {
          router.push(redirectPath);
        } catch (navErr) {
          logger.error('导航失败:', navErr);
          setError(t('auth.login.navFailed'));
        }
        logger.info('已执行跳转命令');
        return;
      }

      // 限流：按钮显示剩余秒数，倒计时结束后再放开 loading。
      if (result.retryAfterSeconds && result.retryAfterSeconds > 0) {
        setError(`${result.message}${t('auth.login.retryIn', { seconds: result.retryAfterSeconds })}`);
        startCountdown(result.retryAfterSeconds);
        return;
      }

      setError(result.message || t('auth.login.loginFailedUnknown'));
    } catch (err) {
      logger.error('登录错误:', err);
      setError(err instanceof Error ? err.message : t('auth.login.loginFailedUnknown'));
    }
    setLoading(false);
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
      {process.env.NODE_ENV === 'development' && (
        <Alert
          title={t('auth.login.devAccountTitle')}
          description={
            <span>
              {t('auth.login.devAccountUsername')}
              <strong>admin</strong> · {t('auth.login.devAccountPassword')}
              <strong>admin123</strong>
              <br />
              {t('auth.login.devAccountWarning')}
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
          title={t('auth.login.sessionExpiredTitle')}
          description={t('auth.login.sessionExpiredDescription')}
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
          <Flex justify='flex-end'>
            <Link href='/forgot-password'>
              <Button type='link' className='p-0 h-auto text-xs' disabled={loading}>
                {t('auth.login.forgotPassword')}
              </Button>
            </Link>
          </Flex>
        </Form.Item>

        <Form.Item>
          <Button
            type='primary'
            htmlType='submit'
            loading={loading}
            disabled={countdown > 0}
            size='large'
            className='w-full h-10 rounded-md text-sm font-semibold'
            icon={<ArrowRight size={14} />}
          >
            {countdown > 0
              ? t('auth.login.waitSeconds', { seconds: countdown })
              : loading
                ? t('auth.login.loggingIn')
                : t('auth.login.loginButton')}
          </Button>
        </Form.Item>
      </Form>

      {/* SSO 登录入口：仅在租户配置了启用中的提供方时渲染 */}
      {ssoProviders.length > 0 && (
        <>
          <Divider plain className='!my-5'>
            <Text className='text-gray-400 text-xs'>{t('auth.login.ssoDivider')}</Text>
          </Divider>
          <div className='flex flex-col gap-2'>
            {ssoProviders.map(provider => (
              <Button
                key={provider.code}
                size='large'
                className='w-full h-10 rounded-md text-sm'
                icon={<Shield size={14} />}
                loading={ssoLoading === provider.code}
                disabled={loading || (ssoLoading !== null && ssoLoading !== provider.code)}
                onClick={() => handleSSOLogin(provider.code)}
              >
                {provider.name}
              </Button>
            ))}
          </div>
        </>
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
  const { t } = useI18n();
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
                    {t('auth.login.brandTagline')}
                  </Text>

                  <Space orientation='vertical' size='middle' className='w-full'>
                    <Flex align='center' gap={10}>
                      <div className='w-8 h-8 bg-white/20 rounded-md flex items-center justify-center'>
                        <Shield size={16} color='white' />
                      </div>
                      <div>
                        <Text className='!text-white !font-semibold !text-sm block'>
                          {t('auth.login.brandSecurityTitle')}
                        </Text>
                        <Text className='!text-white/80 !text-xs'>
                          {t('auth.login.brandSecurityDescription')}
                        </Text>
                      </div>
                    </Flex>

                    <Flex align='center' gap={10}>
                      <div className='w-8 h-8 bg-white/20 rounded-md flex items-center justify-center'>
                        <ArrowRight size={16} color='white' />
                      </div>
                      <div>
                        <Text className='!text-white !font-semibold !text-sm block'>
                          {t('auth.login.brandAutomationTitle')}
                        </Text>
                        <Text className='!text-white/80 !text-xs'>
                          {t('auth.login.brandAutomationDescription')}
                        </Text>
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
                    <div className='text-center py-20 text-gray-400'>{t('auth.login.loading')}</div>
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
