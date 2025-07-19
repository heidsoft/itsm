"use client";

import React, { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Users,
  Shield,
  Settings,
  Workflow,
  Bell,
  Mail,
  Database,
  Globe,
  FileText,
  Calendar,
  Zap,
  BookOpen,
  ChevronDown,
  ChevronRight,
  Home,
  Building2,
  HelpCircle,
  LogOut,
} from "lucide-react";

interface AdminLayoutProps {
  children: React.ReactNode;
}

// 系统设置菜单配置 - 增强版
const ADMIN_MENU_GROUPS = {
  overview: {
    title: "概览",
    items: [{ href: "/admin", icon: Home, label: "管理中心首页" }],
  },
  userManagement: {
    title: "用户与权限管理",
    items: [
      { href: "/admin/users", icon: Users, label: "用户管理", badge: "1,234" },
      { href: "/admin/roles", icon: Shield, label: "角色管理", badge: "15" },
      { href: "/admin/groups", icon: Users, label: "用户组管理", badge: "28" },
      {
        href: "/admin/permissions",
        icon: Shield,
        label: "权限配置",
        badge: "156",
      },
      // 添加租户管理菜单项
      {
        href: "/admin/tenants",
        icon: Building2,
        label: "租户管理",
        badge: "5",
      },
    ],
  },
  processConfig: {
    title: "流程配置",
    items: [
      {
        href: "/admin/workflows",
        icon: Workflow,
        label: "工作流配置",
        badge: "45",
      },
      {
        href: "/admin/approval-chains",
        icon: FileText,
        label: "审批链配置",
        badge: "12",
      },
      {
        href: "/admin/sla-definitions",
        icon: Calendar,
        label: "SLA定义",
        badge: "8",
      },
      {
        href: "/admin/escalation-rules",
        icon: Zap,
        label: "升级规则",
        badge: "6",
      },
    ],
  },
  serviceConfig: {
    title: "服务配置",
    items: [
      {
        href: "/admin/service-catalogs",
        icon: BookOpen,
        label: "服务目录管理",
        badge: "89",
      },
      {
        href: "/admin/categories",
        icon: FileText,
        label: "分类管理",
        badge: "24",
      },
      {
        href: "/admin/priority-matrix",
        icon: Zap,
        label: "优先级矩阵",
        badge: "5",
      },
      {
        href: "/admin/business-rules",
        icon: Settings,
        label: "业务规则",
        badge: "32",
      },
    ],
  },
  systemConfig: {
    title: "系统配置",
    items: [
      {
        href: "/admin/notifications",
        icon: Bell,
        label: "通知配置",
        badge: "24",
      },
      {
        href: "/admin/email-templates",
        icon: Mail,
        label: "邮件模板",
        badge: "18",
      },
      {
        href: "/admin/data-sources",
        icon: Database,
        label: "数据源配置",
        badge: "5",
      },
      {
        href: "/admin/integrations",
        icon: Globe,
        label: "集成配置",
        badge: "12",
      },
      {
        href: "/admin/system-properties",
        icon: Settings,
        label: "系统属性",
        badge: "67",
      },
    ],
  },
};

const AdminLayout = ({ children }: AdminLayoutProps) => {
  const pathname = usePathname();
  const [collapsedGroups, setCollapsedGroups] = useState<
    Record<string, boolean>
  >({});

  const toggleGroup = (groupKey: string) => {
    setCollapsedGroups((prev) => ({
      ...prev,
      [groupKey]: !prev[groupKey],
    }));
  };

  return (
    <div className="flex h-screen bg-gray-50 font-sans">
      {/* 系统设置侧边栏 - 增强版 */}
      <aside className="w-80 bg-white shadow-xl border-r border-gray-200 flex flex-col">
        {/* 头部 */}
        <div className="p-6 border-b border-gray-200 bg-gradient-to-r from-blue-600 to-purple-600">
          <h1 className="text-2xl font-bold text-white">系统设置</h1>
          <p className="text-sm text-blue-100 mt-1">企业级ITSM系统管理中心</p>
        </div>

        {/* 导航菜单 */}
        <nav className="flex-1 overflow-y-auto p-4">
          <div className="space-y-2">
            {Object.entries(ADMIN_MENU_GROUPS).map(([groupKey, group]) => {
              const isCollapsed = collapsedGroups[groupKey];
              const hasActiveItem = group.items.some(
                (item) => pathname === item.href
              );

              return (
                <div key={groupKey} className="mb-4">
                  {/* 分组标题 */}
                  <button
                    onClick={() => toggleGroup(groupKey)}
                    className={`w-full flex items-center justify-between px-3 py-2 text-sm font-semibold rounded-lg transition-colors ${
                      hasActiveItem
                        ? "bg-blue-50 text-blue-700"
                        : "text-gray-700 hover:bg-gray-100"
                    }`}
                  >
                    <span className="text-xs uppercase tracking-wider">
                      {group.title}
                    </span>
                    {isCollapsed ? (
                      <ChevronRight className="w-4 h-4" />
                    ) : (
                      <ChevronDown className="w-4 h-4" />
                    )}
                  </button>

                  {/* 菜单项 */}
                  {!isCollapsed && (
                    <div className="mt-2 space-y-1 ml-2">
                      {group.items.map((item) => {
                        const isActive = pathname === item.href;
                        return (
                          <Link
                            key={item.href}
                            href={item.href}
                            className={`flex items-center justify-between px-3 py-2.5 text-sm font-medium rounded-lg transition-all duration-200 group ${
                              isActive
                                ? "bg-blue-100 text-blue-700 border-l-4 border-blue-700 shadow-sm"
                                : "text-gray-700 hover:bg-gray-50 hover:text-gray-900 hover:translate-x-1"
                            }`}
                          >
                            <div className="flex items-center">
                              <item.icon
                                className={`w-4 h-4 mr-3 flex-shrink-0 ${
                                  isActive
                                    ? "text-blue-600"
                                    : "text-gray-500 group-hover:text-gray-700"
                                }`}
                              />
                              <span>{item.label}</span>
                            </div>
                            {item.badge && (
                              <span
                                className={`px-2 py-1 text-xs rounded-full font-medium ${
                                  isActive
                                    ? "bg-blue-200 text-blue-800"
                                    : "bg-gray-200 text-gray-600 group-hover:bg-gray-300"
                                }`}
                              >
                                {item.badge}
                              </span>
                            )}
                          </Link>
                        );
                      })}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </nav>

        {/* 底部信息 */}
        <div className="p-4 border-t border-gray-200 bg-gray-50">
          <div className="space-y-3">
            {/* 帮助链接 */}
            <Link
              href="#"
              className="flex items-center px-3 py-2 text-sm text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors"
            >
              <HelpCircle className="w-4 h-4 mr-2" />
              帮助文档
            </Link>

            {/* 退出链接 */}
            <Link
              href="/dashboard"
              className="flex items-center px-3 py-2 text-sm text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-lg transition-colors"
            >
              <LogOut className="w-4 h-4 mr-2" />
              返回主界面
            </Link>

            {/* 版本信息 */}
            <div className="text-xs text-gray-500 px-3">
              <div className="font-medium">ITSM Pro v2.0.1</div>
              <div>企业级IT服务管理平台</div>
              <div className="mt-1 text-green-600">● 系统运行正常</div>
            </div>
          </div>
        </div>
      </aside>

      {/* 主内容区域 */}
      <main className="flex-1 overflow-y-auto">
        <div className="p-8">{children}</div>
      </main>
    </div>
  );
};

export default AdminLayout;
