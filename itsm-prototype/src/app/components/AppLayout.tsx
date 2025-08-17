"use client";

import React, { useState } from "react";
import { Layout, Button } from "antd";
import { ArrowLeft } from "lucide-react";
import { useRouter } from "next/navigation";
import { Sidebar } from "./layout/Sidebar";
import { Header } from "./layout/Header";

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
    <Layout style={{ minHeight: "100vh" }}>
      {/* 侧边栏 */}
      <Sidebar collapsed={collapsed} onCollapse={setCollapsed} />

      <Layout>
        {/* 头部 */}
        <Header
          collapsed={collapsed}
          onCollapse={setCollapsed}
          title={title}
          breadcrumb={breadcrumb}
          showBackButton={showBackButton}
          extra={extra}
          showBreadcrumb={showBreadcrumb}
        />

        {/* 主内容区域 */}
        <Content
          style={{
            margin: "24px",
            padding: "24px",
            background: "#fff",
            borderRadius: "8px",
            minHeight: "calc(100vh - 112px)",
          }}
        >
          {/* 返回按钮 */}
          {showBackButton && (
            <div style={{ marginBottom: 16 }}>
              <Button
                icon={<ArrowLeft size={16} />}
                onClick={() => router.back()}
                style={{ marginBottom: 16 }}
              >
                返回
              </Button>
            </div>
          )}

          {/* 页面头部区域 */}
          {showPageHeader && (title || description || extra) && (
            <div
              style={{
                marginBottom: 24,
                paddingBottom: 16,
                borderBottom: "1px solid #f0f0f0",
              }}
            >
              {/* 标题和描述 */}
              {(title || description) && (
                <div style={{ marginBottom: extra ? 16 : 0 }}>
                  {title && (
                    <h1
                      style={{
                        fontSize: "24px",
                        fontWeight: "600",
                        color: "#1f2937",
                        margin: 0,
                        marginBottom: description ? 8 : 0,
                      }}
                    >
                      {title}
                    </h1>
                  )}
                  {description && (
                    <p
                      style={{
                        fontSize: "14px",
                        color: "#6b7280",
                        margin: 0,
                        lineHeight: "1.5",
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
                    display: "flex",
                    justifyContent: "flex-end",
                    alignItems: "center",
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
