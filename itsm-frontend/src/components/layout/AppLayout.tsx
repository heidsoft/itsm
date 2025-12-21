'use client';

import React, { useState } from 'react';
import { Layout, Button } from 'antd';
import { ArrowLeft } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { LAYOUT_CONFIG } from '@/config/layout.config';

const { Content } = Layout;

interface AppLayoutProps {
  children: React.ReactNode;
  title?: string;
  breadcrumb?: Array<{ title: string; href?: string }>;
  showBackButton?: boolean;
  extra?: React.ReactNode;
  showBreadcrumb?: boolean;
  description?: string; // 新增描述字段
  showPageHeader?: boolean; // 新增控制是否显示页面头部的字段
}

export function AppLayout({
  children,
  title,
  breadcrumb,
  showBackButton = false,
  extra,
  showBreadcrumb = true,
  description,
  showPageHeader = true, // 默认显示页面头部
}: AppLayoutProps) {
  const router = useRouter();
  const [collapsed, setCollapsed] = useState(false);

  return (
    <Layout hasSider style={{ minHeight: '100vh' }}>
      {/* 侧边栏 - 使用 Ant Design 标准尺寸 */}
      <Sidebar collapsed={collapsed} onCollapse={setCollapsed} />

      <Layout
        style={{
          marginLeft: collapsed ? LAYOUT_CONFIG.sider.collapsedWidth : LAYOUT_CONFIG.sider.width,
          transition: LAYOUT_CONFIG.transitions.base,
        }}
      >
        {/* 头部 - 高度 64px (Ant Design 标准) */}
        <Header
          collapsed={collapsed}
          onCollapse={setCollapsed}
          title={title}
          breadcrumb={breadcrumb}
          showBackButton={showBackButton}
          extra={extra}
          showBreadcrumb={showBreadcrumb}
        />

        {/* 主内容区域 - 遵循 8px 栅格系统 */}
        <Content
          style={{
            margin: collapsed
              ? `${LAYOUT_CONFIG.content.marginCollapsed}px`
              : `${LAYOUT_CONFIG.content.marginExpanded}px`,
            padding: collapsed
              ? `${LAYOUT_CONFIG.content.paddingCollapsed}px`
              : `${LAYOUT_CONFIG.content.padding}px`,
            background: '#fff',
            minHeight: LAYOUT_CONFIG.content.minHeight,
            borderRadius: LAYOUT_CONFIG.borderRadius.lg,
            transition: LAYOUT_CONFIG.transitions.base,
          }}
          className='responsive-content'
        >
          {/* 返回按钮 */}
          {showBackButton && (
            <div style={{ marginBottom: LAYOUT_CONFIG.spacing.md }}>
              <Button icon={<ArrowLeft size={16} />} onClick={() => router.back()} size='small'>
                返回
              </Button>
            </div>
          )}

          {/* 页面头部区域 */}
          {showPageHeader && (title || description || extra) && (
            <div
              style={{
                marginBottom: LAYOUT_CONFIG.content.pageHeaderMarginBottom,
                paddingBottom: LAYOUT_CONFIG.content.pageHeaderPaddingBottom,
                borderBottom: '1px solid #e5e7eb',
              }}
            >
              {/* 标题和描述 */}
              {(title || description) && (
                <div style={{ marginBottom: extra ? LAYOUT_CONFIG.spacing.md : 0 }}>
                  {title && (
                    <h1
                      style={{
                        fontSize: `${LAYOUT_CONFIG.content.pageTitleFontSize}px`,
                        fontWeight: '600',
                        color: '#1f2937',
                        margin: 0,
                        marginBottom: description ? LAYOUT_CONFIG.spacing.xs : 0,
                      }}
                    >
                      {title}
                    </h1>
                  )}
                  {description && (
                    <p
                      style={{
                        fontSize: `${LAYOUT_CONFIG.content.pageDescFontSize}px`,
                        color: '#6b7280',
                        margin: 0,
                        lineHeight: '1.5',
                      }}
                    >
                      {description}
                    </p>
                  )}
                </div>
              )}

              {/* 操作按钮区域 */}
              {extra && (
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'flex-end',
                    alignItems: 'center',
                  }}
                >
                  {extra}
                </div>
              )}
            </div>
          )}

          {/* 页面内容 */}
          {children}
        </Content>
      </Layout>
    </Layout>
  );
}

export default AppLayout;
