'use client';

import React, { useState, useCallback } from 'react';
import {
  Card,
  Button,
  Input,
  Space,
  Tabs,
  Pagination,
  message,
  Empty,
  Typography,
  Dropdown,
  Grid,
  type MenuProps,
} from 'antd';
import type { MenuProps as AntdMenuProps } from 'antd';
import {
  Plus,
  Search,
  Filter,
  RotateCcw,
  Download,
  MoreVertical,
  Table,
  LayoutGrid,
  Bell,
  Settings,
} from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useDebounce } from '@/lib/component-utils';

const { Title, Text } = Typography;

// ============================================
// 类型定义
// ============================================

export interface PageStats {
  label: string;
  value: number;
  suffix?: React.ReactNode;
  color?: string;
  icon?: React.ReactNode;
}

export interface TabConfig {
  key: string;
  label: React.ReactNode;
  content: React.ReactNode;
}

export interface FilterConfig {
  visible: boolean;
  onToggle: () => void;
  content?: React.ReactNode;
}

export interface PaginationConfig {
  current: number;
  pageSize: number;
  total: number;
  onChange: (page: number, pageSize: number) => void;
  showSizeChanger?: boolean;
  showTotal?: (total: number) => string;
  pageSizeOptions?: string[];
}

export interface ActionButton {
  key: string;
  label: string;
  icon?: React.ReactNode;
  onClick: () => void;
  disabled?: boolean;
  danger?: boolean;
}

export interface BusinessPageTemplateProps {
  // 页面基本信息
  title: string;
  description?: string;

  // 统计数据
  stats?: PageStats[];
  statsLoading?: boolean;

  // 搜索配置
  searchPlaceholder?: string;
  searchValue?: string;
  onSearch?: (value: string) => void;
  searchLoading?: boolean;

  // 筛选配置
  filters?: FilterConfig;

  // 视图切换 (列表/看板)
  showViewSwitch?: boolean;
  activeView?: 'list' | 'kanban';
  onViewChange?: (view: 'list' | 'kanban') => void;

  // Tab 配置
  tabs?: TabConfig[];
  activeTab?: string;
  onTabChange?: (tab: string) => void;

  // 主要操作按钮
  primaryAction?: {
    label: string;
    onClick: () => void;
    icon?: React.ReactNode;
  };

  // 右侧操作按钮组
  extraActions?: ActionButton[];

  // 通知/预警
  alertBadge?: number;
  onAlertClick?: () => void;

  // 内容区域
  loading?: boolean;
  empty?: boolean;
  emptyDescription?: string;
  emptyAction?: {
    label: string;
    onClick: () => void;
  };
  /** 加载失败态（与 empty 区分，优先于 empty 展示） */
  error?: boolean;
  errorDescription?: string;
  /** 错误态重试回调 */
  onRetry?: () => void;
  children?: React.ReactNode;

  // 分页
  pagination?: PaginationConfig;

  // 自定义类名
  className?: string;
}

// ============================================
// 统计卡片组件
// ============================================

const StatsCard: React.FC<PageStats & { loading?: boolean }> = ({
  label,
  value,
  suffix,
  color = '#3b82f6',
  icon,
  loading,
}) => (
  <Card
    size="small"
    className="h-full rounded-lg shadow-sm transition-shadow hover:shadow-md"
    loading={loading}
    aria-label={`${label}统计`}
  >
    <div className="flex min-h-20 items-center justify-between gap-4">
      <div className="min-w-0">
        <Text type="secondary" className="block truncate text-sm">
          {label}
        </Text>
        <div className="mt-1 flex items-baseline gap-1 text-2xl font-semibold leading-none" style={{ color }}>
          {typeof value === 'number' ? value.toLocaleString() : value}
          {suffix && <span className="text-xs font-normal text-slate-500">{suffix}</span>}
        </div>
      </div>
      {icon && (
        <div
          className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl"
          style={{ color, backgroundColor: `${color}14` }}
          aria-hidden="true"
        >
          {icon}
        </div>
      )}
    </div>
  </Card>
);

// ============================================
// 主组件
// ============================================

export const BusinessPageTemplate: React.FC<BusinessPageTemplateProps> = ({
  // 页面基本信息
  title,
  description,

  // 统计数据
  stats,
  statsLoading = false,

  // 搜索配置
  searchPlaceholder = '搜索...',
  searchValue,
  onSearch,
  searchLoading = false,

  // 筛选配置
  filters,

  // 视图切换
  showViewSwitch = true,
  activeView = 'list',
  onViewChange,

  // Tab 配置
  tabs,
  activeTab,
  onTabChange,

  // 操作按钮
  primaryAction,
  extraActions = [],

  // 通知
  alertBadge,
  onAlertClick,

  // 内容
  loading = false,
  empty = false,
  emptyDescription = '暂无数据',
  emptyAction,
  error = false,
  errorDescription = '加载失败，请稍后重试',
  onRetry,
  children,

  // 分页
  pagination,

  // 自定义
  className = '',
}) => {
  const router = useRouter();
  const screens = Grid.useBreakpoint();
  const [localSearchValue, setLocalSearchValue] = useState(searchValue || '');

  // 防抖搜索
  const debouncedSearch = useDebounce(localSearchValue, 300);

  // 搜索变化处理
  const handleSearchChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      setLocalSearchValue(value);
      if (onSearch) {
        onSearch(value);
      }
    },
    [onSearch]
  );

  const handleSearch = useCallback(() => {
    if (onSearch) {
      onSearch(localSearchValue);
    }
  }, [localSearchValue, onSearch]);

  // Tab items 转换
  const tabItems = tabs?.map((tab) => ({
    key: tab.key,
    label: tab.label,
  }));

  // 视图切换 items
  const viewTabItems = [
    {
      key: 'list',
      label: (
        <span className="flex items-center gap-2">
          <Table size={14} />
          列表视图
        </span>
      ),
    },
    {
      key: 'kanban',
      label: (
        <span className="flex items-center gap-2">
          <LayoutGrid size={14} />
          看板视图
        </span>
      ),
    },
  ];

  // 构建操作按钮菜单
  const actionMenuItems: AntdMenuProps['items'] = extraActions.map((action) => ({
    key: action.key,
    label: action.label,
    icon: action.icon,
    onClick: action.onClick,
    disabled: action.disabled,
    danger: action.danger,
  }));

  return (
    <div className={`min-h-screen bg-[var(--color-background-primary)] ${className}`}>
      {/* ====== 页面头部区域 ====== */}
      <div className="bg-[var(--color-surface-primary)] border-b border-[var(--color-border-primary)]">
        <div className="w-full px-3 py-4 sm:px-6">
          {/* 标题行 */}
          <div className="mb-4 flex items-start justify-between gap-3">
            <div className="min-w-0">
              <Title level={2} style={{ marginBottom: 0, marginTop: 0 }}>
                {title}
              </Title>
              {description && (
                <Text type="secondary" className="mt-1 block">
                  {description}
                </Text>
              )}
            </div>

            {/* 右侧操作按钮 */}
            <Space>
              {/* 预警按钮 */}
              {alertBadge !== undefined && onAlertClick && (
                <Button
                  icon={<Bell />}
                  onClick={onAlertClick}
                  className={alertBadge > 0 ? 'text-orange-500' : ''}
                >
                  SLA 预警
                  {alertBadge > 0 && (
                    <span className="ml-1 px-1.5 py-0.5 text-xs bg-red-500 text-white rounded-full">
                      {alertBadge}
                    </span>
                  )}
                </Button>
              )}

              {/* 主要操作按钮 */}
              {primaryAction && (
                <Button
                  type="primary"
                  icon={primaryAction.icon || <Plus />}
                  onClick={primaryAction.onClick}
                  size="small"
                >
                  {primaryAction.label}
                </Button>
              )}

              {/* 更多操作下拉菜单 */}
              {extraActions.length > 0 && (
                <Dropdown menu={{ items: actionMenuItems }} placement="bottomRight">
                  <Button icon={<MoreVertical />} />
                </Dropdown>
              )}
            </Space>
          </div>

          {/* 统计卡片行 */}
          {stats && stats.length > 0 && (
            <div
              className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4"
              role="group"
              aria-label="页面统计"
            >
              {stats.map(stat => (
                <StatsCard key={stat.label} {...stat} loading={statsLoading} />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* ====== 搜索和筛选区域 ====== */}
      <div className="bg-[var(--color-surface-primary)] border-b border-[var(--color-border-primary)]">
        <div className="w-full px-3 py-3 sm:px-6">
          <div className="flex flex-col items-stretch gap-3 sm:flex-row sm:items-center sm:justify-between">
            {/* 搜索框 */}
            <Input.Search
              placeholder={searchPlaceholder}
              allowClear
              value={localSearchValue}
              onChange={handleSearchChange}
              onSearch={handleSearch}
              loading={searchLoading}
              className="w-full sm:w-72"
              enterButton
            />

            <Space wrap className="w-full sm:w-auto">
              {/* 筛选按钮 */}
              {filters && (
                <Button
                  icon={<Filter />}
                  onClick={filters.onToggle}
                  type={filters.visible ? 'primary' : 'default'}
                >
                  筛选
                </Button>
              )}

              {/* 刷新按钮 */}
              <Button icon={<RotateCcw />} onClick={() => window.location.reload()}>
                刷新
              </Button>

              {/* 导出按钮 */}
              <Button icon={<Download />}>导出</Button>
            </Space>
          </div>

          {/* 筛选面板 */}
          {filters?.visible && filters.content && (
            <div className="mt-3 p-3 bg-[var(--color-background-tertiary)] rounded-lg">{filters.content}</div>
          )}
        </div>
      </div>

      {/* ====== 主内容区域 ====== */}
      <div className="w-full px-3 py-4 sm:px-6">
        {/* Tab 切换 */}
        {tabs && tabs.length > 0 && (
          <Tabs
            activeKey={activeTab}
            onChange={onTabChange}
            className="mb-4"
            items={tabItems}
          />
        )}

        {/* 视图切换 (当有看板内容时) */}
        {showViewSwitch && onViewChange && (
          <Tabs
            activeKey={activeView}
            onChange={(key) => onViewChange(key as 'list' | 'kanban')}
            className="mb-4"
            items={viewTabItems}
          />
        )}

        {/* 内容区域 */}
        <Card
          className="rounded-lg shadow-sm"
          styles={{
            body: {
              padding: screens.md ? (activeView === 'kanban' ? 16 : 24) : 12,
            },
          }}
        >
          {loading ? (
            <div className="flex items-center justify-center py-20">
              <div className="text-gray-400">加载中...</div>
            </div>
          ) : error ? (
            <div className="py-12 text-center" role="alert">
              <div className="mb-2 text-base font-medium text-red-500">{errorDescription}</div>
              <Text type="secondary" className="mb-4 block text-sm">
                数据加载失败，并非暂无数据，请检查网络后重试
              </Text>
              {onRetry && (
                <Button type="primary" icon={<RotateCcw className="h-4 w-4" />} onClick={onRetry}>
                  重试
                </Button>
              )}
            </div>
          ) : empty ? (
            <Empty
              description={emptyDescription}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            >
              {emptyAction && (
                <Button type="primary" onClick={emptyAction.onClick}>
                  {emptyAction.label}
                </Button>
              )}
            </Empty>
          ) : (
            children
          )}

          {/* 分页 */}
          {pagination && pagination.total > 0 && (
            <div className="mt-4 flex justify-end">
              <Pagination
                current={pagination.current}
                pageSize={pagination.pageSize}
                total={pagination.total}
                onChange={pagination.onChange}
                showSizeChanger={screens.md && (pagination.showSizeChanger ?? true)}
                responsive
                showTotal={pagination.showTotal ?? ((total) => `共 ${total} 条`)}
                pageSizeOptions={pagination.pageSizeOptions ?? ['10', '20', '50', '100']}
              />
            </div>
          )}
        </Card>
      </div>

      {/* ====== 快捷浮动按钮 ====== */}
      {primaryAction && (
        <div className="fixed bottom-6 right-6 z-50 md:hidden">
          <Button
            type="primary"
            shape="circle"
            size="large"
            icon={primaryAction.icon || <Plus />}
            onClick={primaryAction.onClick}
            className="shadow-lg hover:scale-110 transition-transform"
          />
        </div>
      )}
    </div>
  );
};

export default BusinessPageTemplate;
