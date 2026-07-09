'use client';

import React, { useState, useEffect } from 'react';
import { Layout, ConfigProvider, App } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { usePathname, useRouter } from 'next/navigation';
import { Header } from '@/components/layout/Header';
import { Sidebar } from '@/components/layout/Sidebar';
import { httpClient } from '@/lib/api/http-client';
import { LAYOUT_CONFIG } from '@/config/layout.config';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { NetworkStatus } from '@/components/common/NetworkStatus';
import { useLayoutStore } from '@/lib/store/layout-store';
import PageTransition from '@/components/common/PageTransition';
import { useAuthStore } from '@/lib/store/auth-store';
import type { Tenant } from '@/lib/api/api-config';

const { Content } = Layout;

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
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [checkingAuth, setCheckingAuth] = useState(true);
  const pathname = usePathname();
  const router = useRouter();

  // 处理客户端挂载和认证检查
  useEffect(() => {
    setMounted(true);
    const checkAuth = async () => {
      // 分别请求用户信息和租户信息，避免一个失败导致整体失败
      let userInfo = null;
      let tenantInfo = null;

      try {
        userInfo = await httpClient.get<any>('/api/v1/auth/me');
      } catch (e) {
        console.error('Failed to fetch user info:', e);
      }

      try {
        tenantInfo = await httpClient.get<any>('/api/v1/auth/tenants');
      } catch (e) {
        console.error('Failed to fetch tenant info:', e);
      }

      // 如果两个都失败，则认为未认证
      if (!userInfo && !tenantInfo) {
        setIsAuthenticated(false);
        router.push(`/login?redirect=${encodeURIComponent(pathname || '/')}`);
        setCheckingAuth(false);
        return;
      }

      const tenants = Array.isArray(tenantInfo?.tenants) ? tenantInfo.tenants : [];
      const currentTenant = tenants[0];

      const { login, setCurrentTenant } = useAuthStore.getState();
      login(
        {
          id: Number(userInfo?.id || 0),
          username: String(userInfo?.username || ''),
          email: String(userInfo?.email || ''),
          name: String(userInfo?.name || ''),
          role: String(userInfo?.role || 'end_user'),
          department: userInfo?.department,
          tenantId: userInfo?.tenantId
            ? Number(userInfo.tenantId)
            : userInfo?.tenantId
              ? Number(userInfo.tenantId)
              : undefined,
          permissions: userInfo?.permissions,
          createdAt: userInfo?.createdAt || userInfo?.createdAt,
          updatedAt: userInfo?.updatedAt || userInfo?.updatedAt,
        },
        'authenticated',
        currentTenant
          ? {
              id: Number(currentTenant.id),
              name: String(currentTenant.name),
              code: String(currentTenant.code),
              type: currentTenant.type,
              status: currentTenant.status,
              createdAt: new Date().toISOString(),
              updatedAt: new Date().toISOString(),
            }
          : undefined
      );
      if (currentTenant) {
        const tenantData: Tenant = {
          id: Number(currentTenant.id),
          name: String(currentTenant.name),
          code: String(currentTenant.code),
          type: currentTenant.type || 'standard',
          status: currentTenant.status || 'active',
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        };
        setCurrentTenant(tenantData);
      }
      setIsAuthenticated(true);
      setCheckingAuth(false);
    };
    checkAuth();
  }, [router, pathname]);

  // 响应式布局：在移动端自动折叠侧边栏
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

  // 正在检查认证状态时显示 loading
  if (checkingAuth) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  // 未认证时不渲染布局，直接重定向（由上面的 useEffect 处理）
  if (!isAuthenticated) {
    return null;
  }

  // 根据官方布局模式，使用单一容器控制侧边栏占位
  return (
    <ConfigProvider locale={zhCN}>
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
          className="min-h-screen bg-[#f5f7fb]"
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
          <Sidebar collapsed={collapsed} onCollapse={setCollapsed} />

          {/* 主区域 */}
          <Layout className="bg-[#f5f7fb] min-h-screen">
            {/* 顶部导航栏 */}
            <Header collapsed={collapsed} onCollapse={setCollapsed} showBreadcrumb={true} />

            {/* 内容区域 */}
            <Content
              id="main-content"
              tabIndex={-1}
              onClick={handleContentClick}
              className="bg-[#f5f7fb] w-auto min-w-0 max-w-full overflow-x-hidden shadow-none outline-none"
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
    </ConfigProvider>
  );
}
