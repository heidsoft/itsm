"use client";

import React, { useState } from "react";
import { usePathname } from "next/navigation";
import Link from "next/link";
import {
  LayoutDashboard,
  BookOpen,
  AlertTriangle,
  Search,
  GitMerge,
  Target,
  TrendingUp,
  Database,
  User,
  Ticket,
  BarChart,
  Settings,
  ChevronDown,
  ChevronRight,
  LogOut,
} from "lucide-react";
import { AuthService } from "../lib/auth-service";
import { useRouter } from "next/navigation";

// 菜单分组配置
const MENU_GROUPS = {
  core: {
    title: "核心功能",
    items: [
      { href: "/dashboard", icon: LayoutDashboard, label: "仪表盘" },
      { href: "/tickets", icon: Ticket, label: "所有工单" },
      { href: "/service-catalog", icon: BookOpen, label: "服务目录" },
      { href: "/my-requests", icon: User, label: "我的请求" },
    ],
  },
  management: {
    title: "流程管理",
    items: [
      { href: "/incidents", icon: AlertTriangle, label: "事件管理" },
      { href: "/problems", icon: Search, label: "问题管理" },
      { href: "/changes", icon: GitMerge, label: "变更管理" },
      { href: "/sla", icon: Target, label: "服务级别管理" },
      { href: "/improvements", icon: TrendingUp, label: "持续改进" },
    ],
  },
  system: {
    title: "系统管理",
    items: [
      { href: "/cmdb", icon: Database, label: "配置管理" },
      { href: "/knowledge-base", icon: BookOpen, label: "知识库" },
      { href: "/reports", icon: BarChart, label: "报告分析" },
      // 修改系统设置为完整的管理入口
      { href: "/admin", icon: Settings, label: "系统设置" },
    ],
  },
};

export const Sidebar = () => {
  const router = useRouter();
  const [collapsedGroups, setCollapsedGroups] = useState<
    Record<string, boolean>
  >({});
  const [isCollapsed, setIsCollapsed] = useState(false);

  const handleLogout = () => {
    AuthService.clearToken();
    router.push("/login");
  };

  const toggleGroup = (groupKey: string) => {
    setCollapsedGroups((prev) => ({
      ...prev,
      [groupKey]: !prev[groupKey],
    }));
  };

  return (
    <aside
      className={`${
        isCollapsed ? "w-16" : "w-72"
      } bg-gray-900 text-white flex flex-col transition-all duration-300 ease-in-out relative`}
    >
      {/* 头部 - 固定高度 */}
      <div className="flex items-center justify-center p-4 border-b border-gray-700 h-16 flex-shrink-0">
        {!isCollapsed && (
          <h1 className="text-xl font-bold tracking-wider">
            ITSM<span className="text-blue-400">Pro</span>
          </h1>
        )}
        <button
          onClick={() => setIsCollapsed(!isCollapsed)}
          className={`${
            isCollapsed ? "mx-auto" : "ml-auto"
          } p-1 hover:bg-gray-800 rounded transition-colors`}
        >
          <ChevronRight
            className={`w-4 h-4 transition-transform ${
              isCollapsed ? "" : "rotate-180"
            }`}
          />
        </button>
      </div>

      {/* 导航区域 - 可滚动 */}
      <nav className="flex-1 overflow-y-auto scrollbar-thin scrollbar-thumb-gray-600 scrollbar-track-gray-800">
        <div className="p-2 space-y-1">
          {Object.entries(MENU_GROUPS).map(([groupKey, group]) => (
            <div key={groupKey} className="mb-2">
              {!isCollapsed && (
                <button
                  onClick={() => toggleGroup(groupKey)}
                  className="w-full flex items-center justify-between p-2 text-xs font-semibold text-gray-400 hover:text-gray-200 transition-colors"
                >
                  <span>{group.title}</span>
                  <ChevronDown
                    className={`w-3 h-3 transition-transform ${
                      collapsedGroups[groupKey] ? "-rotate-90" : ""
                    }`}
                  />
                </button>
              )}

              {(!collapsedGroups[groupKey] || isCollapsed) && (
                <div className="space-y-1">
                  {group.items.map((item) => (
                    <NavItem
                      key={item.href}
                      {...item}
                      isCollapsed={isCollapsed}
                    />
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      </nav>

      {/* 底部用户区域 - 固定 */}
      <div className="border-t border-gray-700 p-3 flex-shrink-0">
        {!isCollapsed && (
          <div className="mb-3 p-2 bg-gray-800 rounded-lg">
            <div className="text-xs text-gray-400">当前用户</div>
            <div className="text-sm font-medium">管理员</div>
          </div>
        )}

        <button
          className={`w-full flex items-center ${
            isCollapsed ? "justify-center" : "justify-start"
          } space-x-2 p-2 text-red-400 hover:text-red-300 hover:bg-gray-800 rounded-lg transition-all duration-200 group`}
          onClick={handleLogout}
          title="退出登录"
        >
          <LogOut className="w-4 h-4" />
          {!isCollapsed && <span className="text-sm">退出登录</span>}
        </button>
      </div>
    </aside>
  );
};

const NavItem = ({
  href,
  icon: Icon,
  label,
  isCollapsed,
}: {
  href: string;
  icon: React.ElementType;
  label: string;
  isCollapsed: boolean;
}) => {
  const pathname = usePathname();
  const isActive = pathname === href;

  return (
    <Link
      href={href}
      className={`flex items-center ${
        isCollapsed ? "justify-center" : "justify-start"
      } space-x-3 p-2 rounded-lg transition-all duration-200 group relative ${
        isActive
          ? "bg-blue-600 text-white shadow-lg"
          : "text-gray-300 hover:bg-gray-800 hover:text-white"
      }`}
      title={isCollapsed ? label : undefined}
    >
      <Icon className="w-4 h-4 flex-shrink-0" />
      {!isCollapsed && <span className="text-sm truncate">{label}</span>}

      {/* 折叠状态下的提示 */}
      {isCollapsed && (
        <div className="absolute left-full ml-2 px-2 py-1 bg-gray-800 text-white text-xs rounded opacity-0 group-hover:opacity-100 transition-opacity duration-200 pointer-events-none whitespace-nowrap z-50">
          {label}
        </div>
      )}
    </Link>
  );
};
