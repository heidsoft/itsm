"use client";

import React from "react";
import { Layout, Menu, theme } from "antd";
import {
  LayoutDashboard,
  FileText,
  AlertTriangle,
  Settings,
  BookOpen,
  BarChart3,
  Database,
  HelpCircle,
  Calendar,
  Workflow,
} from "lucide-react";
import { useRouter, usePathname } from "next/navigation";
import { useAuthStore } from "../../lib/store";

const { Sider } = Layout;

// 菜单配置
const MENU_CONFIG = {
  main: [
    {
      key: "/dashboard",
      icon: <LayoutDashboard size={16} />,
      label: "仪表盘",
      path: "/dashboard",
      permission: "dashboard:view",
    },
    {
      key: "/tickets",
      icon: <FileText size={16} />,
      label: "工单管理",
      path: "/tickets",
      permission: "ticket:view",
    },
    {
      key: "/incidents",
      icon: <AlertTriangle size={16} />,
      label: "事件管理",
      path: "/incidents",
      permission: "incident:view",
    },
    {
      key: "/problems",
      icon: <HelpCircle size={16} />,
      label: "问题管理",
      path: "/problems",
      permission: "problem:view",
    },
    {
      key: "/changes",
      icon: <BarChart3 size={16} />,
      label: "变更管理",
      path: "/changes",
      permission: "change:view",
    },
    {
      key: "/cmdb",
      icon: <Database size={16} />,
      label: "配置管理",
      path: "/cmdb",
      permission: "cmdb:view",
    },
    {
      key: "/service-catalog",
      icon: <BookOpen size={16} />,
      label: "服务目录",
      path: "/service-catalog",
      permission: "service:view",
    },
    {
      key: "/knowledge-base",
      icon: <BookOpen size={16} />,
      label: "知识库",
      path: "/knowledge-base",
      permission: "knowledge:view",
    },
    {
      key: "/sla",
      icon: <Calendar size={16} />,
      label: "SLA管理",
      path: "/sla",
      permission: "sla:view",
    },
    {
      key: "/reports",
      icon: <BarChart3 size={16} />,
      label: "报表分析",
      path: "/reports",
      permission: "report:view",
    },
  ],
  admin: [
    {
      key: "/workflow",
      icon: <Workflow size={16} />,
      label: "工作流管理",
      path: "/workflow",
      permission: "workflow:config",
    },
    {
      key: "/admin",
      icon: <Settings size={16} />,
      label: "系统管理",
      path: "/admin",
      permission: "admin:view",
    },
  ],
};

interface SidebarProps {
  collapsed: boolean;
  onCollapse: (collapsed: boolean) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ collapsed, onCollapse }) => {
  const { token } = theme.useToken();
  const router = useRouter();
  const pathname = usePathname();
  const { user } = useAuthStore();

  const handleMenuClick = ({ key }: { key: string }) => {
    router.push(key);
  };

  const isAdmin = user?.role === "admin" || user?.role === "super_admin";

  return (
    <Sider
      trigger={null}
      collapsible
      collapsed={collapsed}
      onCollapse={onCollapse}
      style={{
        background: token.colorBgContainer,
        borderRight: `1px solid ${token.colorBorder}`,
      }}
      width={240}
    >
      {/* Logo 区域 */}
      <div
        style={{
          height: 64,
          display: "flex",
          alignItems: "center",
          justifyContent: collapsed ? "center" : "flex-start",
          padding: collapsed ? "0 16px" : "0 24px",
          borderBottom: `1px solid ${token.colorBorder}`,
        }}
      >
        <div
          style={{
            width: 36,
            height: 36,
            background: "linear-gradient(135deg, #1890ff 0%, #722ed1 100%)",
            borderRadius: "8px",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            color: "white",
            fontWeight: "bold",
            fontSize: "18px",
          }}
        >
          IT
        </div>
        {!collapsed && (
          <span
            style={{
              marginLeft: 12,
              fontSize: "18px",
              fontWeight: "600",
              color: token.colorText,
            }}
          >
            ITSM
          </span>
        )}
      </div>

      {/* 主菜单 */}
      <Menu
        mode="inline"
        selectedKeys={[pathname]}
        onClick={handleMenuClick}
        style={{
          border: "none",
          background: "transparent",
        }}
        items={MENU_CONFIG.main}
      />

      {/* 管理员菜单 */}
      {isAdmin && (
        <>
          <div
            style={{
              padding: "16px 24px 8px",
              fontSize: "12px",
              color: token.colorTextSecondary,
              fontWeight: "500",
              textTransform: "uppercase",
              letterSpacing: "0.5px",
            }}
          >
            {!collapsed && "管理功能"}
          </div>
          <Menu
            mode="inline"
            selectedKeys={[pathname]}
            onClick={handleMenuClick}
            style={{
              border: "none",
              background: "transparent",
            }}
            items={MENU_CONFIG.admin}
          />
        </>
      )}
    </Sider>
  );
};
