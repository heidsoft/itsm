"use client";

import React from "react";
import { Card, Row, Col, Button, Space, Breadcrumb } from "antd";
import {
  ArrowLeftOutlined,
  PlusOutlined,
  ReloadOutlined,
  DownloadOutlined,
} from "@ant-design/icons";

// 企业级页面模板接口
interface EnterprisePageTemplateProps {
  title: string;
  breadcrumb?: Array<{ title: string; href?: string }>;
  showBackButton?: boolean;
  extra?: React.ReactNode;
  children: React.ReactNode;
  showQuickActions?: boolean;
  quickActions?: React.ReactNode;
  showStats?: boolean;
  stats?: React.ReactNode;
  showToolbar?: boolean;
  toolbar?: React.ReactNode;
  showContent?: boolean;
  content?: React.ReactNode;
  className?: string;
}

// 页面容器组件
export const PageContainer: React.FC<EnterprisePageTemplateProps> = ({
  title,
  breadcrumb = [],
  showBackButton = false,
  extra,
  children,
  showQuickActions = false,
  quickActions,
  showStats = false,
  stats,
  showToolbar = false,
  toolbar,
  showContent = true,
  content,
  className = "",
}) => {
  // 默认的快速操作
  const defaultQuickActions = (
    <Space>
      <Button
        type="primary"
        icon={<PlusOutlined />}
        className="enterprise-btn enterprise-btn-primary"
      >
        新建
      </Button>
      <Button
        icon={<ReloadOutlined />}
        className="enterprise-btn enterprise-btn-ghost"
      >
        刷新
      </Button>
      <Button
        icon={<DownloadOutlined />}
        className="enterprise-btn enterprise-btn-ghost"
      >
        导出
      </Button>
    </Space>
  );

  // 默认的工具栏
  const defaultToolbar = (
    <Card className="enterprise-toolbar">
      <Row gutter={[16, 16]} align="middle">
        <Col xs={24} sm={12} md={8}>
          <div className="flex items-center space-x-2">
            {showBackButton && (
              <Button
                type="text"
                icon={<ArrowLeftOutlined />}
                className="enterprise-btn enterprise-btn-ghost"
              >
                返回
              </Button>
            )}
            <span className="text-lg font-semibold text-gray-900">{title}</span>
          </div>
        </Col>
        <Col xs={24} sm={12} md={8}>
          {showQuickActions && (quickActions || defaultQuickActions)}
        </Col>
        <Col xs={24} sm={24} md={8}>
          <div className="flex justify-end">{extra}</div>
        </Col>
      </Row>
    </Card>
  );

  return (
    <div className={`page-container ${className}`}>
      {/* 页面头部区域 */}
      <div
        style={{
          marginBottom: 24,
          paddingBottom: 16,
          borderBottom: '1px solid #e5e7eb',
        }}
      >
        {/* 面包屑导航 */}
        {(showBackButton || breadcrumb.length > 0) && (
          <div className="flex items-center mb-4">
             {showBackButton && (
              <Button
                type="text"
                icon={<ArrowLeftOutlined />}
                className="mr-2"
                onClick={() => window.history.back()}
              >
                返回
              </Button>
            )}
            {breadcrumb.length > 0 && (
              <Breadcrumb>
                {breadcrumb.map((item, index) => (
                  <Breadcrumb.Item key={index}>
                    {item.href ? <a href={item.href}>{item.title}</a> : item.title}
                  </Breadcrumb.Item>
                ))}
              </Breadcrumb>
            )}
          </div>
        )}

        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-2xl font-semibold text-gray-900 mb-1">{title}</h1>
          </div>
          {extra && <div>{extra}</div>}
        </div>
      </div>

      <div className="space-y-6">
        {/* 统计卡片 */}
        {showStats && stats && <div className="mb-6">{stats}</div>}

        {/* 工具栏 */}
        {showToolbar && <div className="mb-6">{toolbar || defaultToolbar}</div>}

        {/* 主要内容 */}
        {showContent && (
          <div className="enterprise-fade-in">{content || children}</div>
        )}
      </div>
    </div>
  );
};

// 企业级卡片组件
export const EnterpriseCard: React.FC<{
  title?: string;
  extra?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}> = ({ title, extra, children, className = "" }) => (
  <Card title={title} extra={extra} className={`enterprise-card ${className}`}>
    {children}
  </Card>
);

// 企业级统计卡片组件
export const EnterpriseStatCard: React.FC<{
  title: string;
  value: number | string;
  prefix?: React.ReactNode;
  suffix?: string;
  valueStyle?: React.CSSProperties;
  className?: string;
}> = ({ title, value, prefix, suffix, valueStyle, className = "" }) => (
  <Card className={`enterprise-stat-card ${className}`}>
    <div className="text-center">
      {prefix && <div className="mb-2">{prefix}</div>}
      <div className="text-3xl font-bold text-gray-900" style={valueStyle}>
        {value}
        {suffix && <span className="text-lg text-gray-500 ml-1">{suffix}</span>}
      </div>
      <div className="text-sm text-gray-500 mt-2">{title}</div>
    </div>
  </Card>
);

// 企业级表格容器组件
export const EnterpriseTableContainer: React.FC<{
  title?: string;
  extra?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}> = ({ title, extra, children, className = "" }) => (
  <Card
    title={title}
    extra={extra}
    className={`enterprise-card enterprise-table ${className}`}
  >
    {children}
  </Card>
);

// 企业级表单容器组件
export const EnterpriseFormContainer: React.FC<{
  title?: string;
  children: React.ReactNode;
  className?: string;
}> = ({ title, children, className = "" }) => (
  <Card
    title={title}
    className={`enterprise-card enterprise-form ${className}`}
  >
    {children}
  </Card>
);

// 企业级模态框容器组件
export const EnterpriseModalContainer: React.FC<{
  title: string;
  visible: boolean;
  onCancel: () => void;
  children: React.ReactNode;
  width?: number;
  footer?: React.ReactNode;
}> = ({ title, visible, onCancel, children, width = 600, footer }) => (
  <div className="enterprise-modal">
    {/* 这里可以包装Modal组件 */}
    {children}
  </div>
);

export default PageContainer;
