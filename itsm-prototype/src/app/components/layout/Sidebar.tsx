"use client";

import React from "react";
import { Layout, Menu, theme, Badge } from "antd";
import {
  LayoutDashboard,
  FileText,
  AlertTriangle,
  BookOpen,
  BarChart3,
  Database,
  HelpCircle,
  Calendar,
  Workflow,
  Shield,
  TrendingUp,
} from "lucide-react";
import { useRouter, usePathname } from "next/navigation";
import { useAuthStore } from "../../lib/store";

const { Sider } = Layout;

// 菜单配置
const MENU_CONFIG = {
  main: [
    {
      key: "/dashboard",
      icon: <LayoutDashboard size={18} />,
      label: "仪表盘",
      path: "/dashboard",
      permission: "dashboard:view",
      description: "系统概览和关键指标",
    },
    {
      key: "/tickets",
      icon: <FileText size={18} />,
      label: "工单管理",
      path: "/tickets",
      permission: "ticket:view",
      description: "管理和跟踪IT工单",
      badge: "New",
    },
    {
      key: "/incidents",
      icon: <AlertTriangle size={18} />,
      label: "事件管理",
      path: "/incidents",
      permission: "incident:view",
      description: "处理IT事件和故障",
    },
    {
      key: "/problems",
      icon: <HelpCircle size={18} />,
      label: "问题管理",
      path: "/problems",
      permission: "problem:view",
      description: "分析根本原因和解决方案",
    },
    {
      key: "/changes",
      icon: <BarChart3 size={18} />,
      label: "变更管理",
      path: "/changes",
      permission: "change:view",
      description: "管理IT变更和发布",
    },
    {
      key: "/cmdb",
      icon: <Database size={18} />,
      label: "配置管理",
      path: "/cmdb",
      permission: "cmdb:view",
      description: "IT资产和配置项管理",
    },
    {
      key: "/service-catalog",
      icon: <BookOpen size={18} />,
      label: "服务目录",
      path: "/service-catalog",
      permission: "service:view",
      description: "IT服务目录和请求",
    },
    {
      key: "/knowledge-base",
      icon: <HelpCircle size={18} />,
      label: "知识库",
      path: "/knowledge-base",
      permission: "knowledge:view",
      description: "技术文档和解决方案",
    },
    {
      key: "/sla",
      icon: <Calendar size={18} />,
      label: "SLA管理",
      path: "/sla",
      permission: "sla:view",
      description: "服务级别协议管理",
    },
    {
      key: "/reports",
      icon: <TrendingUp size={18} />,
      label: "报表分析",
      path: "/reports",
      permission: "report:view",
      description: "数据分析和报表",
    },
  ],
  admin: [
    {
      key: "/workflow",
      icon: <Workflow size={18} />,
      label: "工作流管理",
      path: "/workflow",
      permission: "workflow:config",
      description: "配置业务流程和自动化",
    },
    {
      key: "/admin",
      icon: <Shield size={18} />,
      label: "系统管理",
      path: "/admin",
      permission: "admin:view",
      description: "用户、权限和系统配置",
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

  // 渲染菜单项，支持徽章和描述
  const renderMenuItems = (items: typeof MENU_CONFIG.main) => {
    return items.map((item) => ({
      key: item.key,
      icon: item.icon,
      label: (
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <span>{item.label}</span>
          {item.badge && (
            <Badge
              count={item.badge}
              size="small"
              style={{
                backgroundColor: "#10b981",
                fontSize: "10px",
                lineHeight: "12px",
              }}
            />
          )}
        </div>
      ),
      onClick: () => handleMenuClick({ key: item.key }),
    }));
  };

  return (
    <Sider
      trigger={null}
      collapsible
      collapsed={collapsed}
      onCollapse={onCollapse}
      style={{
        background: "linear-gradient(180deg, #ffffff 0%, #f8fafc 100%)",
        borderRight: `1px solid ${token.colorBorder}`,
        boxShadow: "2px 0 8px rgba(0, 0, 0, 0.1)",
      }}
      width={260}
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
          background: "linear-gradient(135deg, #667eea 0%, #764ba2 100%)",
        }}
      >
        <div
          style={{
            width: 40,
            height: 40,
            background: "rgba(255, 255, 255, 0.2)",
            borderRadius: "12px",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            color: "white",
            fontWeight: "bold",
            fontSize: "20px",
            backdropFilter: "blur(10px)",
            border: "1px solid rgba(255, 255, 255, 0.3)",
          }}
        >
          IT
        </div>
        {!collapsed && (
          <div style={{ marginLeft: 16 }}>
            <div
              style={{
                fontSize: "20px",
                fontWeight: "700",
                color: "white",
                lineHeight: "1",
              }}
            >
              ITSM
            </div>
            <div
              style={{
                fontSize: "10px",
                color: "rgba(255, 255, 255, 0.8)",
                textTransform: "uppercase",
                letterSpacing: "1px",
                marginTop: "2px",
              }}
            >
              System
            </div>
          </div>
        )}
      </div>

      {/* 主菜单 */}
      <div style={{ padding: "16px 0" }}>
        <Menu
          mode="inline"
          selectedKeys={[pathname]}
          style={{
            border: "none",
            background: "transparent",
          }}
          items={renderMenuItems(MENU_CONFIG.main)}
        />
      </div>

      {/* 管理员菜单 */}
      {isAdmin && (
        <div style={{ marginTop: "auto" }}>
          <div
            style={{
              padding: "16px 24px 8px",
              fontSize: "11px",
              color: token.colorTextSecondary,
              fontWeight: "600",
              textTransform: "uppercase",
              letterSpacing: "1px",
              opacity: collapsed ? 0 : 1,
              transition: "opacity 0.2s",
            }}
          >
            {!collapsed && "管理功能"}
          </div>
          <div
            style={{
              padding: "0 12px 16px",
              borderTop: collapsed ? "none" : `1px solid ${token.colorBorder}`,
              marginTop: collapsed ? 0 : "8px",
            }}
          >
            <Menu
              mode="inline"
              selectedKeys={[pathname]}
              style={{
                border: "none",
                background: "transparent",
              }}
              items={renderMenuItems(MENU_CONFIG.admin)}
            />
          </div>
        </div>
      )}

      {/* 底部用户信息 */}
      {!collapsed && (
        <div
          style={{
            padding: "16px 24px",
            borderTop: `1px solid ${token.colorBorder}`,
            background: "rgba(0, 0, 0, 0.02)",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <div
              style={{
                width: 32,
                height: 32,
                borderRadius: "50%",
                background: "linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%)",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                color: "white",
                fontWeight: "600",
                fontSize: "14px",
              }}
            >
              {user?.name?.[0] || user?.username?.[0] || "U"}
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div
                style={{
                  fontSize: "14px",
                  fontWeight: "600",
                  color: token.colorText,
                  marginBottom: "2px",
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                  whiteSpace: "nowrap",
                }}
              >
                {user?.name || user?.username}
              </div>
              <div
                style={{
                  fontSize: "12px",
                  color: token.colorTextSecondary,
                  textTransform: "capitalize",
                }}
              >
                {user?.role || "user"}
              </div>
            </div>
          </div>
        </div>
      )}
    </Sider>
  );
};
