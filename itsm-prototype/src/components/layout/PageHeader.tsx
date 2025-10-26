"use client";

import React from "react";
import { Typography, Space, Breadcrumb, Button, Divider } from "antd";
import { HomeOutlined, RightOutlined } from "@ant-design/icons";
import Link from "next/link";

const { Title, Text } = Typography;

interface BreadcrumbItem {
  label: string;
  href?: string;
}

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  breadcrumbs?: BreadcrumbItem[];
  actions?: React.ReactNode;
  showDivider?: boolean;
  className?: string;
}

export const PageHeader: React.FC<PageHeaderProps> = ({
  title,
  subtitle,
  breadcrumbs = [],
  actions,
  showDivider = true,
  className = "",
}) => {
  const defaultBreadcrumbs = [
    { label: "首页", href: "/dashboard" },
    ...breadcrumbs,
  ];

  return (
    <div className={`mb-6 ${className}`}>
      {/* 面包屑导航 */}
      <Breadcrumb
        separator={<RightOutlined />}
        className="mb-4"
        items={defaultBreadcrumbs.map((item, index) => ({
          title: item.href ? (
            <Link href={item.href} className="text-gray-500 hover:text-blue-600">
              {index === 0 ? <HomeOutlined /> : item.label}
            </Link>
          ) : (
            <span className="text-gray-700">{item.label}</span>
          ),
        }))}
      />

      {/* 页面标题区域 */}
      <div className="flex items-center justify-between">
        <div className="flex-1">
          <Title level={2} className="mb-2">
            {title}
          </Title>
          {subtitle && (
            <Text type="secondary" className="text-base">
              {subtitle}
            </Text>
          )}
        </div>
        
        {/* 操作按钮区域 */}
        {actions && (
          <div className="flex items-center space-x-2">
            {actions}
          </div>
        )}
      </div>

      {/* 分隔线 */}
      {showDivider && <Divider className="my-4" />}
    </div>
  );
};
