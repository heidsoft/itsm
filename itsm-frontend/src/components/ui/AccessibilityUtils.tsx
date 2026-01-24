'use client';

import React from 'react';
import { cn } from '@/lib/utils';

/**
 * 视觉隐藏组件
 * 
 * 仅对屏幕阅读器可见，视觉上隐藏
 * 
 * @example
 * ```tsx
 * <VisuallyHidden>
 *   当前工单状态: 已完成
 * </VisuallyHidden>
 * <Badge status="success" />
 * ```
 */
export interface VisuallyHiddenProps {
  children: React.ReactNode;
  as?: React.ElementType;
  className?: string;
}

export const VisuallyHidden: React.FC<VisuallyHiddenProps> = ({
  children,
  as: Component = 'span',
  className,
}) => {
  return (
    <Component
      className={cn('sr-only', className)}
      style={{
        position: 'absolute',
        width: '1px',
        height: '1px',
        padding: 0,
        margin: '-1px',
        overflow: 'hidden',
        clip: 'rect(0, 0, 0, 0)',
        whiteSpace: 'nowrap',
        border: 0,
      }}
    >
      {children}
    </Component>
  );
};

/**
 * 焦点可见指示器
 * 
 * 为交互元素添加清晰的焦点指示器
 * 
 * @example
 * ```tsx
 * <FocusIndicator>
 *   <button>点击我</button>
 * </FocusIndicator>
 * ```
 */
export interface FocusIndicatorProps {
  children: React.ReactNode;
  className?: string;
  offset?: number;
}

export const FocusIndicator: React.FC<FocusIndicatorProps> = ({
  children,
  className,
  offset = 2,
}) => {
  return (
    <div
      className={cn(
        'focus-within:outline focus-within:outline-3 focus-within:outline-blue-600',
        'focus-within:rounded transition-all duration-200',
        className
      )}
      style={{
        outlineOffset: `${offset}px`,
      }}
    >
      {children}
    </div>
  );
};

/**
 * ARIA实时区域组件
 * 
 * 用于向屏幕阅读器公告动态内容更新
 * 
 * @example
 * ```tsx
 * <AriaLiveRegion politeness="polite">
 *   {loadingMessage}
 * </AriaLiveRegion>
 * ```
 */
export interface AriaLiveRegionProps {
  children: React.ReactNode;
  politeness?: 'polite' | 'assertive' | 'off';
  atomic?: boolean;
  relevant?: 'additions' | 'removals' | 'text' | 'all';
  className?: string;
}

export const AriaLiveRegion: React.FC<AriaLiveRegionProps> = ({
  children,
  politeness = 'polite',
  atomic = true,
  relevant = 'all',
  className,
}) => {
  return (
    <div
      role="status"
      aria-live={politeness}
      aria-atomic={atomic}
      aria-relevant={relevant}
      className={cn('sr-only', className)}
    >
      {children}
    </div>
  );
};

/**
 * 跳过链接组件
 * 
 * 允许键盘用户快速跳过导航
 * 
 * @example
 * ```tsx
 * <SkipLink href="#main-content">
 *   跳到主内容
 * </SkipLink>
 * <main id="main-content">...</main>
 * ```
 */
export interface SkipLinkProps {
  href: string;
  children: React.ReactNode;
  className?: string;
}

export const SkipLink: React.FC<SkipLinkProps> = ({
  href,
  children,
  className,
}) => {
  return (
    <a
      href={href}
      className={cn(
        'sr-only',
        'focus:not-sr-only',
        'focus:absolute focus:top-4 focus:left-4 focus:z-[9999]',
        'focus:px-4 focus:py-2',
        'focus:bg-blue-600 focus:text-white',
        'focus:rounded-lg focus:shadow-lg',
        'focus:outline-none',
        'transition-all duration-200',
        className
      )}
    >
      {children}
    </a>
  );
};

/**
 * 可访问的图标按钮
 * 
 * 确保图标按钮对屏幕阅读器友好
 * 
 * @example
 * ```tsx
 * <AccessibleIconButton
 *   icon={<CloseOutlined />}
 *   label="关闭对话框"
 *   onClick={handleClose}
 * />
 * ```
 */
export interface AccessibleIconButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  icon: React.ReactNode;
  label: string;
  tooltip?: string;
}

export const AccessibleIconButton: React.FC<AccessibleIconButtonProps> = ({
  icon,
  label,
  tooltip,
  className,
  ...props
}) => {
  return (
    <button
      aria-label={label}
      title={tooltip || label}
      className={cn(
        'inline-flex items-center justify-center',
        'min-w-[44px] min-h-[44px]',
        'rounded-lg',
        'hover:bg-gray-100 active:bg-gray-200',
        'transition-colors duration-200',
        'focus:outline-none focus:ring-2 focus:ring-blue-600 focus:ring-offset-2',
        className
      )}
      {...props}
    >
      {icon}
      <VisuallyHidden>{label}</VisuallyHidden>
    </button>
  );
};

/**
 * 可访问的表单标签
 * 
 * 确保表单字段有正确的标签关联
 * 
 * @example
 * ```tsx
 * <AccessibleLabel
 *   htmlFor="email"
 *   required
 *   tooltip="请输入您的工作邮箱"
 * >
 *   邮箱地址
 * </AccessibleLabel>
 * <input id="email" type="email" />
 * ```
 */
export interface AccessibleLabelProps
  extends React.LabelHTMLAttributes<HTMLLabelElement> {
  children: React.ReactNode;
  required?: boolean;
  tooltip?: string;
}

export const AccessibleLabel: React.FC<AccessibleLabelProps> = ({
  children,
  required,
  tooltip,
  className,
  ...props
}) => {
  return (
    <label
      className={cn(
        'block text-sm font-medium text-gray-700 mb-1',
        className
      )}
      {...props}
    >
      <span className="flex items-center gap-1">
        {children}
        {required && (
          <>
            <span className="text-red-500" aria-hidden="true">
              *
            </span>
            <VisuallyHidden>(必填)</VisuallyHidden>
          </>
        )}
        {tooltip && (
          <span
            className="text-gray-400 text-xs cursor-help"
            title={tooltip}
            aria-label={tooltip}
          >
            ⓘ
          </span>
        )}
      </span>
    </label>
  );
};

/**
 * 可访问的对话框包装器
 * 
 * 确保对话框符合ARIA规范
 * 
 * @example
 * ```tsx
 * <AccessibleDialog
 *   isOpen={isOpen}
 *   onClose={handleClose}
 *   title="创建工单"
 * >
 *   <CreateTicketForm />
 * </AccessibleDialog>
 * ```
 */
export interface AccessibleDialogProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  className?: string;
}

export const AccessibleDialog: React.FC<AccessibleDialogProps> = ({
  isOpen,
  onClose,
  title,
  children,
  className,
}) => {
  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleEscape);
      // 防止背景滚动
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = '';
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="dialog-title"
      className={cn(
        'fixed inset-0 z-50',
        'flex items-center justify-center',
        'bg-black/50',
        className
      )}
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          onClose();
        }
      }}
    >
      <div
        className="bg-white rounded-lg shadow-xl max-w-lg w-full mx-4 max-h-[90vh] overflow-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="border-b border-gray-200 px-6 py-4">
          <h2 id="dialog-title" className="text-xl font-semibold text-gray-900">
            {title}
          </h2>
        </div>
        <div className="px-6 py-4">{children}</div>
      </div>
    </div>
  );
};

/**
 * 键盘导航提示
 * 
 * 显示当前可用的键盘快捷键
 * 
 * @example
 * ```tsx
 * <KeyboardNavigationHint
 *   shortcuts={[
 *     { key: 'Ctrl+K', description: '打开搜索' },
 *     { key: 'Ctrl+N', description: '创建新工单' },
 *   ]}
 * />
 * ```
 */
export interface KeyboardNavigationHintProps {
  shortcuts: Array<{
    key: string;
    description: string;
  }>;
  className?: string;
}

export const KeyboardNavigationHint: React.FC<KeyboardNavigationHintProps> = ({
  shortcuts,
  className,
}) => {
  const [isVisible, setIsVisible] = React.useState(false);

  React.useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === '?' && e.shiftKey) {
        e.preventDefault();
        setIsVisible((prev) => !prev);
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  if (!isVisible) {
    return (
      <div className="fixed bottom-4 right-4 text-xs text-gray-500">
        按 <kbd className="px-2 py-1 bg-gray-100 rounded border">?</kbd> 查看快捷键
      </div>
    );
  }

  return (
    <div
      className={cn(
        'fixed bottom-4 right-4 z-50',
        'bg-white rounded-lg shadow-2xl',
        'border border-gray-200',
        'p-4 max-w-sm',
        className
      )}
      role="dialog"
      aria-label="键盘快捷键帮助"
    >
      <div className="flex items-center justify-between mb-3">
        <h3 className="font-semibold text-gray-900">键盘快捷键</h3>
        <button
          onClick={() => setIsVisible(false)}
          className="text-gray-400 hover:text-gray-600"
          aria-label="关闭"
        >
          ✕
        </button>
      </div>
      <div className="space-y-2">
        {shortcuts.map((shortcut, index) => (
          <div key={index} className="flex items-center justify-between gap-4">
            <kbd className="px-2 py-1 bg-gray-100 rounded border text-xs font-mono">
              {shortcut.key}
            </kbd>
            <span className="text-sm text-gray-700 flex-1">{shortcut.description}</span>
          </div>
        ))}
      </div>
    </div>
  );
};

export default {
  VisuallyHidden,
  FocusIndicator,
  AriaLiveRegion,
  SkipLink,
  AccessibleIconButton,
  AccessibleLabel,
  AccessibleDialog,
  KeyboardNavigationHint,
};

