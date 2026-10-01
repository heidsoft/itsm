'use client';

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Layout, App } from 'antd';
import { useRouter } from 'next/navigation';
import { Header } from '@/components/layout/Header';
import { Sidebar } from '@/components/layout/Sidebar';
import { LAYOUT_CONFIG } from '@/config/layout.config';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { NetworkStatus } from '@/components/common/NetworkStatus';
import { AdminRouteGuard } from '@/components/common/AdminRouteGuard';
import { useLayoutStore } from '@/lib/store/layout-store';
import PageTransition from '@/components/common/PageTransition';
import { useAuthStore, useAuthStoreHydration } from '@/lib/store/auth-store';
import { AuthService } from '@/lib/services/auth-service';
import { refreshSession } from '@/lib/api/session-api';
import { useTheme } from '@/lib/design-system/theme';

const { Content } = Layout;

type SessionView = 'checking' | 'ready' | 'anonymous' | 'unavailable';

/** 会话续签提前量：留出请求往返时间，避免卡在过期边界上。 */
const SESSION_REFRESH_MARGIN_MS = 60_000;
/** 下限：后端未给出剩余时间或剩余极短时，也不得变成忙轮询。 */
const SESSION_REFRESH_MIN_DELAY_MS = 30_000;
/** 启动时遇到瞬时故障的有界重试；耗尽后明确进入「无法确认会话」而不是假登录态。 */
const BOOT_RETRY_DELAYS_MS = [1_000, 3_000];

function nextRefreshDelay(expiresIn: number): number {
  const remainingMs = Math.max(expiresIn, 0) * 1000 - SESSION_REFRESH_MARGIN_MS;
  return Math.max(remainingMs, SESSION_REFRESH_MIN_DELAY_MS);
}

function currentTargetPath(): string {
  if (typeof window === 'undefined') return '/';
  return `${window.location.pathname}${window.location.search}` || '/';
}

function loginPathForCurrentPage(expired = false): string {
  const params = new URLSearchParams({ redirect: currentTargetPath() });
  if (expired) params.set('expired', 'true');
  return `/login?${params.toString()}`;
}

/**
 * 主应用布局
 * 包含 Header、Sidebar 和 Content 区域
 * 需要用户认证才能访问
 */
export default function MainLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const { collapsed, setCollapsed } = useLayoutStore();
  const [mounted, setMounted] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [sessionView, setSessionView] = useState<SessionView>('checking');
  const [sessionError, setSessionError] = useState('');
  const [retryNonce, setRetryNonce] = useState(0);
  const router = useRouter();

  // 续签时长只信后端下发的相对剩余秒数，不用浏览器时钟推算「15 分钟到了」
  const expiresInRef = useRef(0);

  // 恢复持久化的 auth store，并同步租户上下文到内存
  useAuthStoreHydration();

  useEffect(() => {
    setMounted(true);
  }, []);

  // 会话真相：唯一来源是后端会话端点，本地持久化状态不参与推断。
  useEffect(() => {
    const controller = new AbortController();
    let cancelled = false;

    const checkSession = async () => {
      for (let attempt = 0; ; attempt += 1) {
        const outcome = await AuthService.syncSession(controller.signal);
        if (cancelled) return;

        if (outcome.state === 'authenticated') {
          expiresInRef.current = outcome.expiresIn;
          setSessionView('ready');
          return;
        }

        if (outcome.state === 'unauthenticated') {
          setSessionView('anonymous');
          router.replace(loginPathForCurrentPage());
          return;
        }

        const retryDelayMs = BOOT_RETRY_DELAYS_MS[attempt];
        if (retryDelayMs === undefined) {
          setSessionError(outcome.reason);
          setSessionView('unavailable');
          return;
        }
        // 未确认会话前保持 loading，既不渲染假登录界面也不登出用户。
        await new Promise(resolve => setTimeout(resolve, retryDelayMs));
        if (cancelled) return;
      }
    };

    void checkSession();
    return () => {
      cancelled = true;
      controller.abort();
    };
  }, [retryNonce, router]);

  // 按后端剩余时间主动续签；续签结果会带回新的剩余时间并据此重排。
  useEffect(() => {
    if (sessionView !== 'ready') return;

    let cancelled = false;
    let timer: ReturnType<typeof setTimeout>;

    const schedule = (delayMs: number) => {
      timer = setTimeout(async () => {
        if (cancelled) return;
        const outcome = await refreshSession();
        if (cancelled) return;

        if (outcome.state === 'refreshed') {
          expiresInRef.current = outcome.expiresIn;
          schedule(nextRefreshDelay(outcome.expiresIn));
          return;
        }

        if (outcome.state === 'expired') {
          useAuthStore.getState().logout();
          setSessionView('anonymous');
          router.replace(loginPathForCurrentPage(true));
          return;
        }

        // 后端抖动：稍后按同一节奏再试，真实掉线由请求的 401 路径给结论。
        schedule(SESSION_REFRESH_MIN_DELAY_MS);
      }, delayMs);
    };

    schedule(nextRefreshDelay(expiresInRef.current));
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [sessionView, router]);

  const retrySessionCheck = useCallback(() => {
    setSessionView('checking');
    setRetryNonce(prev => prev + 1);
  }, []);

  // 响应式布局：在移动端自动折叠侧边栏；从移动端拉宽回桌面时恢复展开
  useEffect(() => {
    const handleResize = () => {
      const mobile = window.innerWidth < 768;
      setIsMobile(mobile);
      if (mobile) {
        setCollapsed(true);
      }
    };

    handleResize();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  // 移动端 → 桌面端切换时恢复侧边栏展开（此前会永久停留在收起态）
  useEffect(() => {
    if (mounted && !isMobile) {
      setCollapsed(false);
    }
  }, [mounted, isMobile]);

  // 在移动端，点击内容区域时折叠侧边栏
  const handleContentClick = () => {
    if (isMobile && !collapsed) {
      setCollapsed(true);
    }
  };

  // 未挂载时显示 loading（避免服务端渲染问题）
  if (!mounted) {
    return null;
  }

  // 正在向后端确认会话时显示 loading
  if (sessionView === 'checking') {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  // 后端确认无会话：不渲染布局，重定向由上面的 effect 处理
  if (sessionView === 'anonymous') {
    return null;
  }

  // 无法确认会话：既不假登录也不把人踢出，给出可重试的明确状态
  if (sessionView === 'unavailable') {
    return (
      <div className="flex items-center justify-center min-h-screen p-4">
        <Alert
          title="暂时无法确认登录状态"
          description={
            <span>
              服务暂时不可用，当前页面无法判断会话是否仍然有效。
              {sessionError ? <span className="block mt-1 text-gray-500">{sessionError}</span> : null}
            </span>
          }
          type="warning"
          showIcon
          action={
            <Button type="primary" onClick={retrySessionCheck}>
              重新检查
            </Button>
          }
        />
      </div>
    );
  }

  // 根据官方布局模式，使用单一容器控制侧边栏占位
  return (
    <ThemedMainLayout
      collapsed={collapsed}
      isMobile={isMobile}
      handleContentClick={handleContentClick}
    >
      <AdminRouteGuard>
        <PageTransition>{children}</PageTransition>
      </AdminRouteGuard>
    </ThemedMainLayout>
  );
}

function ThemedMainLayout({
  collapsed,
  isMobile,
  handleContentClick,
  children,
}: Readonly<{
  collapsed: boolean;
  isMobile: boolean;
  handleContentClick: () => void;
  children: React.ReactNode;
}>) {
  const { isDark } = useTheme();
  const setCollapsed = useLayoutStore(s => s.setCollapsed);

  return (
    <App>
      {/* Skip to main content link for accessibility */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:bg-primary-600 focus:text-white focus:px-4 focus:py-2 focus:rounded-lg focus:shadow-lg focus:outline-none focus:ring-2 focus:ring-primary-400"
      >
        跳转到主要内容
      </a>
      <NetworkStatus />
      <Layout
        className="min-h-screen bg-[var(--color-background-primary)]"
        style={{
          paddingLeft: isMobile
            ? 0
            : collapsed
              ? LAYOUT_CONFIG.sider.collapsedWidth
              : LAYOUT_CONFIG.sider.width,
          transition: 'padding-left 0.2s ease',
        }}
      >
        {/* 侧边栏 */}
        <Sidebar
          collapsed={collapsed}
          onCollapse={setCollapsed}
          mobile={isMobile}
        />

        {/* 主区域 */}
        <Layout className="bg-[var(--color-background-primary)] min-h-screen">
          {/* 顶部导航栏 */}
          <Header collapsed={collapsed} onCollapse={setCollapsed} showBreadcrumb={true} />

          {/* 内容区域 */}
          <Content
            id="main-content"
            tabIndex={-1}
            onClick={handleContentClick}
            className="bg-[var(--color-background-primary)] w-auto min-w-0 max-w-full overflow-x-hidden shadow-none outline-none"
            style={{
              minHeight: LAYOUT_CONFIG.content.minHeight,
            }}
          >
            <div
              className="main-content"
              style={{
                padding: isMobile ? `${LAYOUT_CONFIG.content.paddingMobile}px` : '16px',
              }}
            >
              <PageTransition>{children}</PageTransition>
            </div>
          </Content>

          {/* 页脚（可选） */}
          <footer className="text-center p-4 bg-transparent text-gray-400 text-xs">
            AI-Native ITSM ©{new Date().getFullYear()} - AI驱动的IT服务管理系统
          </footer>
        </Layout>

      {/* 移动端遮罩层 */}
      {!collapsed && isMobile && (
        <div
          onClick={() => setCollapsed(true)}
          className="fixed inset-0 bg-black/45"
          style={{
            zIndex: LAYOUT_CONFIG.zIndex.sider - 1,
          }}
        />
      )}
      </Layout>
    </App>
  );
}
