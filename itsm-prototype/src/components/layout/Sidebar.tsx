'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  LayoutDashboard,
  Ticket,
  AlertTriangle,
  Wrench,
  GitBranch,
  HardDrive,
  Users,
  Settings,
  BarChart3,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';

/**
 * 导航菜单项配置
 */
const navigationItems = [
  {
    name: '仪表盘',
    href: '/dashboard',
    icon: LayoutDashboard,
  },
  {
    name: '工单管理',
    href: '/tickets',
    icon: Ticket,
  },
  {
    name: '事件管理',
    href: '/incidents',
    icon: AlertTriangle,
  },
  {
    name: '问题管理',
    href: '/problems',
    icon: Wrench,
  },
  {
    name: '变更管理',
    href: '/changes',
    icon: GitBranch,
  },
  {
    name: '资产管理',
    href: '/assets',
    icon: HardDrive,
  },
  {
    name: '用户管理',
    href: '/users',
    icon: Users,
  },
  {
    name: '报表分析',
    href: '/reports',
    icon: BarChart3,
  },
  {
    name: '系统设置',
    href: '/settings',
    icon: Settings,
  },
];

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
}

/**
 * 侧边栏组件
 */
export function Sidebar({ collapsed, onToggle }: SidebarProps) {
  const pathname = usePathname();

  return (
    <div
      className={`fixed inset-y-0 left-0 z-50 bg-white shadow-lg transition-all duration-300 ${
        collapsed ? 'w-16' : 'w-64'
      }`}
    >
      {/* 侧边栏头部 */}
      <div className="flex items-center justify-between h-16 px-4 border-b border-gray-200">
        {!collapsed && (
          <div className="flex items-center space-x-2">
            <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center">
              <span className="text-white font-bold text-sm">IT</span>
            </div>
            <span className="font-semibold text-gray-900">ITSM 系统</span>
          </div>
        )}
        
        <button
          onClick={onToggle}
          className="p-1.5 rounded-md hover:bg-gray-100 transition-colors"
        >
          {collapsed ? (
            <ChevronRight className="w-4 h-4 text-gray-600" />
          ) : (
            <ChevronLeft className="w-4 h-4 text-gray-600" />
          )}
        </button>
      </div>

      {/* 导航菜单 */}
      <nav className="mt-4 px-2">
        <ul className="space-y-1">
          {navigationItems.map((item) => {
            const isActive = pathname === item.href || pathname.startsWith(item.href + '/');
            const Icon = item.icon;

            return (
              <li key={item.name}>
                <Link
                  href={item.href}
                  className={`
                    flex items-center px-3 py-2 rounded-md text-sm font-medium transition-colors
                    ${isActive
                      ? 'bg-blue-50 text-blue-700 border-r-2 border-blue-700'
                      : 'text-gray-700 hover:bg-gray-50 hover:text-gray-900'
                    }
                  `}
                  title={collapsed ? item.name : undefined}
                >
                  <Icon className={`flex-shrink-0 w-5 h-5 ${collapsed ? 'mx-auto' : 'mr-3'}`} />
                  {!collapsed && (
                    <span className="truncate">{item.name}</span>
                  )}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      {/* 侧边栏底部 */}
      {!collapsed && (
        <div className="absolute bottom-4 left-4 right-4">
          <div className="bg-blue-50 rounded-lg p-3">
            <div className="flex items-center space-x-2">
              <div className="w-2 h-2 bg-green-400 rounded-full"></div>
              <span className="text-xs text-gray-600">系统运行正常</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}