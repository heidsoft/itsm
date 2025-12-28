import React from 'react';
import { cn } from '@/lib/utils';

interface ResponsiveLayoutProps {
  children: React.ReactNode;
  sidebar?: React.ReactNode;
  header?: React.ReactNode;
  footer?: React.ReactNode;
  className?: string;
  sidebarWidth?: 'narrow' | 'normal' | 'wide';
  variant?: 'default' | 'full-width' | 'centered';
}

export function ResponsiveLayout({
  children,
  sidebar,
  header,
  footer,
  className,
  sidebarWidth = 'normal',
  variant = 'default',
}: ResponsiveLayoutProps) {
  const sidebarWidths = {
    narrow: 'w-64',
    normal: 'w-72',
    wide: 'w-80',
  };

  const containerClasses = {
    'default': 'max-w-7xl mx-auto',
    'full-width': 'w-full',
    'centered': 'max-w-4xl mx-auto',
  };

  return (
    <div className={cn('min-h-screen flex flex-col bg-gray-50', className)}>
      {/* Header */}
      {header && (
        <header className="sticky top-0 z-40 bg-white border-b border-gray-200 shadow-sm">
          <div className={containerClasses[variant]}>
            {header}
          </div>
        </header>
      )}

      {/* Main Content Area */}
      <div className="flex-1 flex">
        {/* Sidebar */}
        {sidebar && (
          <aside className={cn(
            'hidden lg:block bg-white border-r border-gray-200',
            sidebarWidths[sidebarWidth],
            'sticky top-16 h-[calc(100vh-4rem)] overflow-y-auto'
          )}>
            {sidebar}
          </aside>
        )}

        {/* Main Content */}
        <main className="flex-1 flex flex-col">
          <div className={cn(
            'flex-1',
            variant !== 'full-width' && 'px-4 sm:px-6 lg:px-8 py-6'
          )}>
            {children}
          </div>
        </main>
      </div>

      {/* Footer */}
      {footer && (
        <footer className="bg-white border-t border-gray-200">
          <div className={containerClasses[variant]}>
            {footer}
          </div>
        </footer>
      )}
    </div>
  );
}

interface PageHeaderProps {
  title: string;
  subtitle?: string;
  children?: React.ReactNode;
  breadcrumbs?: Array<{ label: string; href?: string }>;
  actions?: React.ReactNode;
  className?: string;
}

export function PageHeader({
  title,
  subtitle,
  children,
  breadcrumbs,
  actions,
  className,
}: PageHeaderProps) {
  return (
    <div className={cn('space-y-4', className)}>
      {/* Breadcrumbs */}
      {breadcrumbs && breadcrumbs.length > 0 && (
        <nav className="flex text-sm text-gray-500" aria-label="Breadcrumb">
          <ol className="flex items-center space-x-2">
            {breadcrumbs.map((crumb, index) => (
              <li key={index} className="flex items-center">
                {index > 0 && (
                  <svg
                    className="h-4 w-4 text-gray-300 mx-2"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path
                      fillRule="evenodd"
                      d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
                      clipRule="evenodd"
                    />
                  </svg>
                )}
                {crumb.href ? (
                  <a
                    href={crumb.href}
                    className="hover:text-gray-700 transition-colors"
                  >
                    {crumb.label}
                  </a>
                ) : (
                  <span className="text-gray-900 font-medium">{crumb.label}</span>
                )}
              </li>
            ))}
          </ol>
        </nav>
      )}

      {/* Header Content */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="min-w-0 flex-1">
          <h1 className="text-2xl font-bold text-gray-900 truncate">
            {title}
          </h1>
          {subtitle && (
            <p className="mt-1 text-sm text-gray-500 truncate">
              {subtitle}
            </p>
          )}
        </div>

        {/* Actions */}
        {actions && (
          <div className="flex-shrink-0">
            {actions}
          </div>
        )}
      </div>

      {/* Additional content */}
      {children && <div>{children}</div>}
    </div>
  );
}

interface CardProps {
  title?: string;
  subtitle?: string;
  children: React.ReactNode;
  className?: string;
  headerActions?: React.ReactNode;
  footer?: React.ReactNode;
  variant?: 'default' | 'elevated' | 'outlined' | 'filled';
  size?: 'sm' | 'default' | 'lg';
}

export function Card({
  title,
  subtitle,
  children,
  className,
  headerActions,
  footer,
  variant = 'default',
  size = 'default',
}: CardProps) {
  const variants = {
    default: 'bg-white border border-gray-200',
    elevated: 'bg-white shadow-lg border border-gray-100',
    outlined: 'bg-white border-2 border-gray-300',
    filled: 'bg-gray-50 border border-gray-200',
  };

  const sizes = {
    sm: 'p-4',
    default: 'p-6',
    lg: 'p-8',
  };

  return (
    <div className={cn(
      'rounded-lg',
      variants[variant],
      className
    )}>
      {/* Header */}
      {(title || headerActions) && (
        <div className={cn(
          'flex items-center justify-between border-b border-gray-200 pb-4 mb-4',
          sizes[size]
        )}>
          <div className="min-w-0 flex-1">
            {title && (
              <h3 className="text-lg font-medium text-gray-900 truncate">
                {title}
              </h3>
            )}
            {subtitle && (
              <p className="mt-1 text-sm text-gray-500 truncate">
                {subtitle}
              </p>
            )}
          </div>
          {headerActions && (
            <div className="flex-shrink-0 ml-4">
              {headerActions}
            </div>
          )}
        </div>
      )}

      {/* Content */}
      <div className={cn(!title && !headerActions && sizes[size])}>
        {children}
      </div>

      {/* Footer */}
      {footer && (
        <div className={cn(
          'border-t border-gray-200 pt-4 mt-4',
          sizes[size]
        )}>
          {footer}
        </div>
      )}
    </div>
  );
}

interface GridProps {
  children: React.ReactNode;
  cols?: 1 | 2 | 3 | 4 | 5 | 6;
  gap?: 'sm' | 'default' | 'lg' | 'xl';
  className?: string;
  responsive?: boolean;
}

export function Grid({
  children,
  cols = 3,
  gap = 'default',
  className,
  responsive = true,
}: GridProps) {
  const colClasses = {
    1: 'grid-cols-1',
    2: 'grid-cols-1 md:grid-cols-2',
    3: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
    4: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4',
    5: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5',
    6: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6',
  };

  const gapClasses = {
    sm: 'gap-2',
    default: 'gap-4',
    lg: 'gap-6',
    xl: 'gap-8',
  };

  return (
    <div className={cn(
      'grid',
      responsive ? colClasses[cols] : `grid-cols-${cols}`,
      gapClasses[gap],
      className
    )}>
      {children}
    </div>
  );
}

interface StackProps {
  children: React.ReactNode;
  direction?: 'vertical' | 'horizontal';
  spacing?: 'sm' | 'default' | 'lg' | 'xl';
  align?: 'start' | 'center' | 'end' | 'stretch';
  justify?: 'start' | 'center' | 'end' | 'between' | 'around' | 'evenly';
  className?: string;
  wrap?: boolean;
}

export function Stack({
  children,
  direction = 'vertical',
  spacing = 'default',
  align = 'stretch',
  justify = 'start',
  className,
  wrap = false,
}: StackProps) {
  const directionClasses = {
    vertical: 'flex-col',
    horizontal: 'flex-row',
  };

  const spacingClasses = {
    vertical: {
      sm: 'space-y-2',
      default: 'space-y-4',
      lg: 'space-y-6',
      xl: 'space-y-8',
    },
    horizontal: {
      sm: 'space-x-2',
      default: 'space-x-4',
      lg: 'space-x-6',
      xl: 'space-x-8',
    },
  };

  const alignClasses = {
    start: 'items-start',
    center: 'items-center',
    end: 'items-end',
    stretch: 'items-stretch',
  };

  const justifyClasses = {
    start: 'justify-start',
    center: 'justify-center',
    end: 'justify-end',
    between: 'justify-between',
    around: 'justify-around',
    evenly: 'justify-evenly',
  };

  return (
    <div className={cn(
      'flex',
      directionClasses[direction],
      spacingClasses[direction][spacing],
      alignClasses[align],
      justifyClasses[justify],
      wrap && 'flex-wrap',
      className
    )}>
      {children}
    </div>
  );
}

interface LoadingStateProps {
  isLoading: boolean;
  children: React.ReactNode;
  skeleton?: React.ReactNode;
  className?: string;
}

export function LoadingState({
  isLoading,
  children,
  skeleton,
  className,
}: LoadingStateProps) {
  if (isLoading) {
    return (
      <div className={cn('animate-pulse', className)}>
        {skeleton || <DefaultSkeleton />}
      </div>
    );
  }

  return <>{children}</>;
}

function DefaultSkeleton() {
  return (
    <div className="space-y-4">
      <div className="h-4 bg-gray-200 rounded w-3/4"></div>
      <div className="h-4 bg-gray-200 rounded w-1/2"></div>
      <div className="h-4 bg-gray-200 rounded w-5/6"></div>
    </div>
  );
}

interface EmptyStateProps {
  title: string;
  description?: string;
  icon?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
}

export function EmptyState({
  title,
  description,
  icon,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div className={cn(
      'text-center py-12',
      className
    )}>
      {icon && (
        <div className="mx-auto h-12 w-12 text-gray-400 mb-4">
          {icon}
        </div>
      )}
      
      <h3 className="mt-2 text-sm font-medium text-gray-900">
        {title}
      </h3>
      
      {description && (
        <p className="mt-1 text-sm text-gray-500">
          {description}
        </p>
      )}
      
      {action && (
        <div className="mt-6">
          {action}
        </div>
      )}
    </div>
  );
}

interface ErrorBoundaryFallbackProps {
  error: Error;
  resetErrorBoundary: () => void;
}

export function ErrorBoundaryFallback({
  error,
  resetErrorBoundary,
}: ErrorBoundaryFallbackProps) {
  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full bg-white shadow-lg rounded-lg p-6">
        <div className="flex items-center mb-4">
          <div className="flex-shrink-0">
            <svg
              className="h-6 w-6 text-red-500"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.732-.833-2.5 0L4.268 19.5c-.77.833.192 2.5 1.732 2.5z"
              />
            </svg>
          </div>
          <div className="ml-3">
            <h3 className="text-sm font-medium text-gray-900">
              应用程序遇到错误
            </h3>
          </div>
        </div>
        
        <div className="mb-4">
          <p className="text-sm text-gray-500 mb-2">
            {error.message}
          </p>
          <details className="text-xs text-gray-400">
            <summary className="cursor-pointer hover:text-gray-600">
              查看详细信息
            </summary>
            <pre className="mt-2 whitespace-pre-wrap break-all">
              {error.stack}
            </pre>
          </details>
        </div>
        
        <button
          onClick={resetErrorBoundary}
          className="w-full bg-blue-600 text-white py-2 px-4 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-colors"
        >
          重试
        </button>
      </div>
    </div>
  );
}

// Export all components
export {
  ResponsiveLayout,
  PageHeader,
  Card,
  Grid,
  Stack,
  LoadingState,
  EmptyState,
  ErrorBoundaryFallback,
};