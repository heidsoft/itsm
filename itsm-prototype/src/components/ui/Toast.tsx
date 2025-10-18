import React from 'react';
import { createPortal } from 'react-dom';
import { cn } from '@/lib/utils';
import { X, CheckCircle, AlertCircle, Info, AlertTriangle } from 'lucide-react';

export interface ToastProps {
  id: string;
  title?: string;
  message: string;
  type?: 'success' | 'error' | 'warning' | 'info';
  duration?: number;
  position?: 'top-right' | 'top-left' | 'bottom-right' | 'bottom-left' | 'top-center' | 'bottom-center';
  closable?: boolean;
  onClose?: (id: string) => void;
}

/**
 * 单个Toast组件
 */
export const Toast: React.FC<ToastProps> = ({
  id,
  title,
  message,
  type = 'info',
  duration = 5000,
  closable = true,
  onClose,
}) => {
  const [isVisible, setIsVisible] = React.useState(false);
  const [isLeaving, setIsLeaving] = React.useState(false);

  // 自动关闭
  React.useEffect(() => {
    if (duration > 0) {
      const timer = setTimeout(() => {
        handleClose();
      }, duration);
      return () => clearTimeout(timer);
    }
  }, [duration]);

  // 入场动画
  React.useEffect(() => {
    const timer = setTimeout(() => setIsVisible(true), 50);
    return () => clearTimeout(timer);
  }, []);

  const handleClose = () => {
    setIsLeaving(true);
    setTimeout(() => {
      onClose?.(id);
    }, 300);
  };

  // 类型样式
  const typeStyles = {
    success: {
      icon: CheckCircle,
      classes: 'bg-green-50 border-green-200 text-green-800',
      iconClasses: 'text-green-500',
    },
    error: {
      icon: AlertCircle,
      classes: 'bg-red-50 border-red-200 text-red-800',
      iconClasses: 'text-red-500',
    },
    warning: {
      icon: AlertTriangle,
      classes: 'bg-yellow-50 border-yellow-200 text-yellow-800',
      iconClasses: 'text-yellow-500',
    },
    info: {
      icon: Info,
      classes: 'bg-blue-50 border-blue-200 text-blue-800',
      iconClasses: 'text-blue-500',
    },
  };

  const { icon: Icon, classes, iconClasses } = typeStyles[type];

  return (
    <div
      className={cn(
        'flex items-start p-4 mb-3 bg-white border rounded-lg shadow-lg',
        'transform transition-all duration-300 ease-out',
        isVisible && !isLeaving && 'translate-x-0 opacity-100',
        !isVisible && 'translate-x-full opacity-0',
        isLeaving && 'translate-x-full opacity-0 scale-95',
        classes
      )}
      role="alert"
    >
      {/* 图标 */}
      <div className="flex-shrink-0">
        <Icon className={cn('w-5 h-5', iconClasses)} />
      </div>
      
      {/* 内容 */}
      <div className="ml-3 flex-1 min-w-0">
        {title && (
          <h4 className="text-sm font-semibold mb-1">
            {title}
          </h4>
        )}
        <p className="text-sm">
          {message}
        </p>
      </div>
      
      {/* 关闭按钮 */}
      {closable && (
        <button
          type="button"
          onClick={handleClose}
          className="ml-3 flex-shrink-0 p-1 rounded-md hover:bg-black/5 transition-colors"
          aria-label="关闭通知"
        >
          <X className="w-4 h-4" />
        </button>
      )}
    </div>
  );
};

// Toast容器组件
export interface ToastContainerProps {
  toasts: ToastProps[];
  position?: ToastProps['position'];
  onClose: (id: string) => void;
}

export const ToastContainer: React.FC<ToastContainerProps> = ({
  toasts,
  position = 'top-right',
  onClose,
}) => {
  const [mounted, setMounted] = React.useState(false);

  React.useEffect(() => {
    setMounted(true);
  }, []);

  if (!mounted || toasts.length === 0) {
    return null;
  }

  // 位置样式
  const positionClasses = {
    'top-right': 'top-4 right-4',
    'top-left': 'top-4 left-4',
    'bottom-right': 'bottom-4 right-4',
    'bottom-left': 'bottom-4 left-4',
    'top-center': 'top-4 left-1/2 transform -translate-x-1/2',
    'bottom-center': 'bottom-4 left-1/2 transform -translate-x-1/2',
  };

  const containerContent = (
    <div
      className={cn(
        'fixed z-50 flex flex-col w-full max-w-sm pointer-events-none',
        positionClasses[position]
      )}
    >
      {toasts.map((toast) => (
        <div key={toast.id} className="pointer-events-auto">
          <Toast
            {...toast}
            onClose={onClose}
          />
        </div>
      ))}
    </div>
  );

  return createPortal(containerContent, document.body);
};

// Toast Hook
interface ToastContextType {
  toasts: ToastProps[];
  addToast: (toast: Omit<ToastProps, 'id'>) => void;
  removeToast: (id: string) => void;
  clearToasts: () => void;
}

const ToastContext = React.createContext<ToastContextType | undefined>(undefined);

export const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = React.useState<ToastProps[]>([]);

  const addToast = React.useCallback((toast: Omit<ToastProps, 'id'>) => {
    const id = Math.random().toString(36).substr(2, 9);
    setToasts((prev) => [...prev, { ...toast, id }]);
  }, []);

  const removeToast = React.useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id));
  }, []);

  const clearToasts = React.useCallback(() => {
    setToasts([]);
  }, []);

  const value = React.useMemo(
    () => ({
      toasts,
      addToast,
      removeToast,
      clearToasts,
    }),
    [toasts, addToast, removeToast, clearToasts]
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <ToastContainer
        toasts={toasts}
        onClose={removeToast}
      />
    </ToastContext.Provider>
  );
};

export const useToast = () => {
  const context = React.useContext(ToastContext);
  if (context === undefined) {
    throw new Error('useToast must be used within a ToastProvider');
  }

  const { addToast } = context;

  return React.useMemo(
    () => ({
      ...context,
      success: (message: string, title?: string) =>
        addToast({ type: 'success', message, title }),
      error: (message: string, title?: string) =>
        addToast({ type: 'error', message, title }),
      warning: (message: string, title?: string) =>
        addToast({ type: 'warning', message, title }),
      info: (message: string, title?: string) =>
        addToast({ type: 'info', message, title }),
    }),
    [context, addToast]
  );
};