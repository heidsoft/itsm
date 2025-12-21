'use client';

import React from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { Badge } from 'antd';
import { useResponsive } from '@/hooks/useResponsive';
import { cn } from '@/lib/utils';
import {
  DashboardOutlined,
  FileTextOutlined,
  ExclamationCircleOutlined,
  UserOutlined,
  AppstoreOutlined,
} from '@ant-design/icons';

export interface MobileTabBarItem {
  key: string;
  path: string;
  icon: React.ReactNode;
  label: string;
  badge?: number | string;
  dot?: boolean;
}

export interface MobileTabBarProps {
  /**
   * 标签项配置
   */
  items?: MobileTabBarItem[];
  
  /**
   * 是否显示
   * @default true (移动端自动显示)
   */
  visible?: boolean;
  
  /**
   * 是否固定在底部
   * @default true
   */
  fixed?: boolean;
  
  /**
   * 激活的标签键
   */
  activeKey?: string;
  
  /**
   * 标签点击回调
   */
  onChange?: (key: string) => void;
  
  /**
   * 自定义类名
   */
  className?: string;
}

/**
 * 默认标签项配置
 */
const defaultItems: MobileTabBarItem[] = [
  {
    key: 'dashboard',
    path: '/dashboard',
    icon: <DashboardOutlined />,
    label: '首页',
  },
  {
    key: 'tickets',
    path: '/tickets',
    icon: <FileTextOutlined />,
    label: '工单',
  },
  {
    key: 'incidents',
    path: '/incidents',
    icon: <ExclamationCircleOutlined />,
    label: '事件',
  },
  {
    key: 'more',
    path: '/more',
    icon: <AppstoreOutlined />,
    label: '更多',
  },
  {
    key: 'profile',
    path: '/profile',
    icon: <UserOutlined />,
    label: '我的',
  },
];

/**
 * 移动端底部导航栏组件
 * 
 * 特性：
 * - 仅在移动端显示
 * - 支持徽章和红点
 * - 平滑切换动画
 * - 安全区域适配
 * - 激活状态高亮
 * 
 * @example
 * ```tsx
 * <MobileTabBar
 *   items={[
 *     { key: 'home', path: '/', icon: <HomeOutlined />, label: '首页' },
 *     { key: 'tickets', path: '/tickets', icon: <FileTextOutlined />, label: '工单', badge: 5 },
 *   ]}
 * />
 * ```
 */
export const MobileTabBar: React.FC<MobileTabBarProps> = ({
  items = defaultItems,
  visible = true,
  fixed = true,
  activeKey,
  onChange,
  className,
}) => {
  const router = useRouter();
  const pathname = usePathname();
  const { isMobile } = useResponsive();

  // 确定当前激活的标签
  const currentActiveKey = activeKey || items.find(item => pathname.startsWith(item.path))?.key || items[0]?.key;

  // 标签点击处理
  const handleTabClick = (item: MobileTabBarItem) => {
    if (onChange) {
      onChange(item.key);
    } else {
      router.push(item.path);
    }
  };

  // 如果不是移动端或不可见，不渲染
  if (!isMobile || !visible) {
    return null;
  }

  return (
    <>
      {/* 占位元素，防止内容被TabBar遮挡 */}
      {fixed && <div className="h-16" />}
      
      {/* 底部导航栏 */}
      <div
        className={cn(
          'mobile-tabbar',
          'bg-white border-t border-gray-200',
          'z-50',
          fixed && 'fixed bottom-0 left-0 right-0',
          className
        )}
        style={{
          // 安全区域适配（iOS）
          paddingBottom: 'env(safe-area-inset-bottom)',
        }}
      >
        <div className="flex justify-around items-stretch h-16">
          {items.map((item) => {
            const isActive = currentActiveKey === item.key;
            
            return (
              <button
                key={item.key}
                onClick={() => handleTabClick(item)}
                className={cn(
                  'flex flex-col items-center justify-center flex-1',
                  'transition-all duration-200 ease-in-out',
                  'relative',
                  'min-h-[44px]', // 确保点击区域足够大
                  isActive ? 'text-blue-600' : 'text-gray-600',
                  !isActive && 'hover:text-gray-900 active:bg-gray-50'
                )}
                aria-label={item.label}
                aria-current={isActive ? 'page' : undefined}
              >
                {/* 激活指示器 */}
                {isActive && (
                  <div
                    className="absolute top-0 left-1/2 -translate-x-1/2 w-12 h-1 bg-blue-600 rounded-b-full transition-all duration-300"
                    aria-hidden="true"
                  />
                )}
                
                {/* 图标 */}
                <div className="relative">
                  <div
                    className={cn(
                      'text-2xl mb-1 transition-transform duration-200',
                      isActive && 'scale-110'
                    )}
                  >
                    {item.icon}
                  </div>
                  
                  {/* 徽章 */}
                  {(item.badge || item.dot) && (
                    <Badge
                      count={item.badge}
                      dot={item.dot}
                      className="absolute -top-1 -right-1"
                      style={{
                        boxShadow: '0 0 0 2px #fff',
                      }}
                    />
                  )}
                </div>
                
                {/* 标签文字 */}
                <span
                  className={cn(
                    'text-xs font-medium transition-all duration-200',
                    isActive && 'font-semibold'
                  )}
                >
                  {item.label}
                </span>
              </button>
            );
          })}
        </div>
      </div>
    </>
  );
};

/**
 * 移动端顶部应用栏
 * 
 * 用于移动端的顶部导航栏
 */
export interface MobileAppBarProps {
  title?: string;
  left?: React.ReactNode;
  right?: React.ReactNode;
  onBack?: () => void;
  fixed?: boolean;
  className?: string;
}

export const MobileAppBar: React.FC<MobileAppBarProps> = ({
  title,
  left,
  right,
  onBack,
  fixed = true,
  className,
}) => {
  const { isMobile } = useResponsive();
  const router = useRouter();

  if (!isMobile) {
    return null;
  }

  return (
    <>
      {/* 占位元素 */}
      {fixed && <div className="h-14" />}
      
      {/* 顶部应用栏 */}
      <div
        className={cn(
          'mobile-appbar',
          'bg-white border-b border-gray-200',
          'z-50',
          fixed && 'fixed top-0 left-0 right-0',
          className
        )}
        style={{
          paddingTop: 'env(safe-area-inset-top)',
        }}
      >
        <div className="flex items-center justify-between h-14 px-4">
          {/* 左侧 */}
          <div className="flex items-center gap-3">
            {onBack && (
              <button
                onClick={onBack}
                className="text-blue-600 text-lg p-2 -ml-2 hover:bg-blue-50 rounded-lg transition-colors"
                aria-label="返回"
              >
                ←
              </button>
            )}
            {left}
          </div>
          
          {/* 标题 */}
          {title && (
            <h1 className="flex-1 text-center text-base font-semibold text-gray-900 truncate px-4">
              {title}
            </h1>
          )}
          
          {/* 右侧 */}
          <div className="flex items-center gap-2">
            {right}
          </div>
        </div>
      </div>
    </>
  );
};

/**
 * 移动端浮动操作按钮
 * 
 * 用于快速访问主要操作
 */
export interface MobileFabProps {
  icon: React.ReactNode;
  onClick: () => void;
  label?: string;
  position?: 'bottom-right' | 'bottom-center' | 'bottom-left';
  badge?: number | string;
  className?: string;
}

export const MobileFab: React.FC<MobileFabProps> = ({
  icon,
  onClick,
  label,
  position = 'bottom-right',
  badge,
  className,
}) => {
  const { isMobile } = useResponsive();

  if (!isMobile) {
    return null;
  }

  const positionMap = {
    'bottom-right': 'bottom-20 right-4',
    'bottom-center': 'bottom-20 left-1/2 -translate-x-1/2',
    'bottom-left': 'bottom-20 left-4',
  };

  return (
    <button
      onClick={onClick}
      aria-label={label}
      className={cn(
        'mobile-fab',
        'fixed z-40',
        'w-14 h-14 rounded-full',
        'bg-blue-600 text-white',
        'shadow-lg hover:shadow-xl',
        'active:scale-95',
        'transition-all duration-200',
        'flex items-center justify-center',
        'text-2xl',
        positionMap[position],
        className
      )}
      style={{
        minHeight: 44,
        minWidth: 44,
      }}
    >
      {badge && (
        <Badge
          count={badge}
          className="absolute -top-1 -right-1"
          style={{
            boxShadow: '0 0 0 2px #fff',
          }}
        />
      )}
      {icon}
    </button>
  );
};

export default MobileTabBar;

