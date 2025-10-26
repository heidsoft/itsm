'use client';

import React, { Suspense, lazy, ComponentType } from 'react';
import { Spin, Skeleton } from 'antd';

// 加载中组件
const LoadingSpinner: React.FC<{ size?: 'small' | 'default' | 'large' }> = ({
  size = 'default',
}) => (
  <div className='flex items-center justify-center p-8'>
    <Spin size={size} />
  </div>
);

// 骨架屏组件
const SkeletonCard: React.FC = () => (
  <div className='p-4 border border-gray-200 rounded-lg'>
    <Skeleton active paragraph={{ rows: 3 }} />
  </div>
);

const SkeletonTable: React.FC = () => (
  <div className='space-y-2'>
    <Skeleton active paragraph={{ rows: 1 }} />
    {Array.from({ length: 5 }).map((_, index) => (
      <Skeleton key={index} active paragraph={{ rows: 1 }} />
    ))}
  </div>
);

// 懒加载包装器
export const withLazyLoading = <P extends object>(
  Component: ComponentType<P>,
  fallback?: React.ReactNode
) => {
  const LazyComponent = lazy(() => Promise.resolve({ default: Component }));

  return (props: P) => (
    <Suspense fallback={fallback || <LoadingSpinner />}>
      <LazyComponent {...props} />
    </Suspense>
  );
};

// 懒加载的工单组件
export const LazyTicketStats = lazy(() =>
  import('@/components/business/TicketStats').then(module => ({
    default: module.TicketStats,
  }))
);

export const LazyTicketFilters = lazy(() =>
  import('@/components/business/TicketFilters').then(module => ({
    default: module.TicketFilters,
  }))
);

export const LazyTicketList = lazy(() =>
  import('@/components/business/TicketList').then(module => ({
    default: module.TicketList,
  }))
);

export const LazyVirtualizedTicketList = lazy(() =>
  import('@/components/business/VirtualizedTicketList').then(module => ({
    default: module.VirtualizedTicketList,
  }))
);

export const LazyTicketModal = lazy(() =>
  import('@/components/business/TicketModal').then(module => ({
    default: module.TicketModal,
  }))
);

export const LazyTicketTemplateModal = lazy(() =>
  import('@/components/business/TicketModal').then(module => ({
    default: module.TicketTemplateModal,
  }))
);

export const LazyDashboard = lazy(() =>
  import('@/app/dashboard/page').then(module => ({
    default: module.default,
  }))
);

export const LazyTicketsPage = lazy(() =>
  import('@/app/tickets/page').then(module => ({
    default: module.default,
  }))
);

export const LazyIncidentsPage = lazy(() =>
  import('@/app/incidents/page').then(module => ({
    default: module.default,
  }))
);

export const LazyProblemsPage = lazy(() =>
  import('@/app/problems/page').then(module => ({
    default: module.default,
  }))
);

export const LazyChangesPage = lazy(() =>
  import('@/app/changes/page').then(module => ({
    default: module.default,
  }))
);

export const LazyKnowledgePage = lazy(() =>
  import('@/app/knowledge-base/page').then(module => ({
    default: module.default,
  }))
);

export const LazyReportsPage = lazy(() =>
  import('@/app/reports/page').then(module => ({
    default: module.default,
  }))
);

export const LazyAdminPage = lazy(() =>
  import('@/app/admin/page').then(module => ({
    default: module.default,
  }))
);

// 懒加载的布局组件
export const LazyAppLayout = lazy(() =>
  import('@/components/layout/AppLayout').then(module => ({
    default: module.default,
  }))
);

export const LazySidebar = lazy(() =>
  import('@/components/layout/Sidebar.tsx').then(module => ({
    default: module.Sidebar,
  }))
);

export const LazyHeader = lazy(() =>
  import('@/components/layout/Header.tsx').then(module => ({
    default: module.Header,
  }))
);

// 懒加载的图表组件
export const LazyCharts = lazy(() =>
  import('@/app/dashboard/charts').then(module => ({
    default: module.default,
  }))
);

export const LazyTicketAssociation = lazy(() =>
  import('@/app/components/TicketAssociation').then(module => ({
    default: module.TicketAssociation,
  }))
);

export const LazySatisfactionDashboard = lazy(() =>
  import('@/components/business/SatisfactionDashboard').then(module => ({
    default: module.SatisfactionDashboard,
  }))
);

// 懒加载包装器组件
export const LazyWrapper: React.FC<{
  children: React.ReactNode;
  fallback?: React.ReactNode;
}> = ({ children, fallback }) => (
  <Suspense fallback={fallback || <LoadingSpinner />}>{children}</Suspense>
);

// 导出加载组件
export { LoadingSpinner, SkeletonCard, SkeletonTable };
