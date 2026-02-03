'use client';

import React, { useState, useEffect } from 'react';
import { Layout, ConfigProvider, App } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { usePathname, useRouter } from 'next/navigation';
import { Header } from '@/components/layout/Header';
import { Sidebar } from '@/components/layout/Sidebar';
import { AuthService } from '@/lib/services/auth-service';
import { LAYOUT_CONFIG } from '@/config/layout.config';

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
  const [collapsed, setCollapsed] = useState(false);
  const [mounted, setMounted] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const pathname = usePathname();
  const router = useRouter();

  // 处理客户端挂载
  useEffect(() => {
    setMounted(true);
  }, []);

  // 认证检查
  useEffect(() => {
    if (mounted && !AuthService.isAuthenticated()) {
      router.push('/login');
    }
  }, [mounted, router]);

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

  // 根据官方布局模式，使用单一容器控制侧边栏占位
  return (
    <ConfigProvider locale={zhCN}>
      <App>
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
              onClick={handleContentClick}
              className="bg-[#f5f7fb] w-auto min-w-0 max-w-full overflow-x-hidden shadow-none"
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
                {children}
              </div>
            </Content>

            {/* 页脚（可选） */}
            <footer className="text-center p-4 bg-transparent text-gray-400 text-xs">
              ITSM Platform ©{new Date().getFullYear()} - IT服务管理平台
            </footer>
          </Layout>
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
      </App>
    </ConfigProvider>
  );
}
