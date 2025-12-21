import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { useEffect } from 'react';

// 主题类型
export type Theme = 'light' | 'dark' | 'system';

// 通知类型
export interface Notification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message?: string;
  duration?: number;
  timestamp: number;
}

// 侧边栏状态
export interface SidebarState {
  isCollapsed: boolean;
  isMobile: boolean;
  isOpen: boolean;
}

// UI状态接口
interface UIState {
  // 主题
  theme: Theme;
  
  // 侧边栏
  sidebar: SidebarState;
  
  // 通知
  notifications: Notification[];
  
  // 加载状态
  globalLoading: boolean;
  
  // 页面标题
  pageTitle: string;
  
  // 面包屑
  breadcrumbs: Array<{ label: string; href?: string }>;
  
  // 操作方法
  setTheme: (theme: Theme) => void;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setSidebarMobile: (isMobile: boolean) => void;
  setSidebarOpen: (isOpen: boolean) => void;
  
  addNotification: (notification: Omit<Notification, 'id' | 'timestamp'>) => void;
  removeNotification: (id: string) => void;
  clearNotifications: () => void;
  
  setGlobalLoading: (loading: boolean) => void;
  setPageTitle: (title: string) => void;
  setBreadcrumbs: (breadcrumbs: Array<{ label: string; href?: string }>) => void;
}

/**
 * UI状态管理 Store
 * 管理全局UI状态，包括主题、侧边栏、通知等
 */
export const useUIStore = create<UIState>()(
  persist(
    (set, get) => ({
      // 初始状态
      theme: 'system',
      sidebar: {
        isCollapsed: false,
        isMobile: false,
        isOpen: true,
      },
      notifications: [],
      globalLoading: false,
      pageTitle: 'ITSM 系统',
      breadcrumbs: [],

      // 主题操作
      setTheme: (theme: Theme) => {
        set({ theme });
        
        // 应用主题到 document
        const root = document.documentElement;
        if (theme === 'dark') {
          root.classList.add('dark');
        } else if (theme === 'light') {
          root.classList.remove('dark');
        } else {
          // system theme
          const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
          if (prefersDark) {
            root.classList.add('dark');
          } else {
            root.classList.remove('dark');
          }
        }
      },

      // 侧边栏操作
      toggleSidebar: () => {
        const { sidebar } = get();
        set({
          sidebar: {
            ...sidebar,
            isCollapsed: !sidebar.isCollapsed,
          },
        });
      },

      setSidebarCollapsed: (collapsed: boolean) => {
        const { sidebar } = get();
        set({
          sidebar: {
            ...sidebar,
            isCollapsed: collapsed,
          },
        });
      },

      setSidebarMobile: (isMobile: boolean) => {
        const { sidebar } = get();
        set({
          sidebar: {
            ...sidebar,
            isMobile,
            // 移动端默认收起侧边栏
            isCollapsed: isMobile ? true : sidebar.isCollapsed,
          },
        });
      },

      setSidebarOpen: (isOpen: boolean) => {
        const { sidebar } = get();
        set({
          sidebar: {
            ...sidebar,
            isOpen,
          },
        });
      },

      // 通知操作
      addNotification: (notification) => {
        const id = Math.random().toString(36).substr(2, 9);
        const newNotification: Notification = {
          ...notification,
          id,
          timestamp: Date.now(),
          duration: notification.duration || 5000,
        };

        set((state) => ({
          notifications: [...state.notifications, newNotification],
        }));

        // 自动移除通知
        if (newNotification.duration && newNotification.duration > 0) {
          setTimeout(() => {
            get().removeNotification(id);
          }, newNotification.duration);
        }
      },

      removeNotification: (id: string) => {
        set((state) => ({
          notifications: state.notifications.filter((n) => n.id !== id),
        }));
      },

      clearNotifications: () => {
        set({ notifications: [] });
      },

      // 全局加载状态
      setGlobalLoading: (loading: boolean) => {
        set({ globalLoading: loading });
      },

      // 页面标题
      setPageTitle: (title: string) => {
        set({ pageTitle: title });
        document.title = `${title} - ITSM 系统`;
      },

      // 面包屑
      setBreadcrumbs: (breadcrumbs) => {
        set({ breadcrumbs });
      },
    }),
    {
      name: 'ui-storage',
      partialize: (state) => ({
        theme: state.theme,
        sidebar: {
          isCollapsed: state.sidebar.isCollapsed,
        },
      }),
    }
  )
);

// 通知工具函数
export const useNotifications = () => {
  const addNotification = useUIStore((state) => state.addNotification);
  
  return {
    success: (title: string, message?: string) => 
      addNotification({ type: 'success', title, message }),
    
    error: (title: string, message?: string) => 
      addNotification({ type: 'error', title, message, duration: 8000 }),
    
    warning: (title: string, message?: string) => 
      addNotification({ type: 'warning', title, message }),
    
    info: (title: string, message?: string) => 
      addNotification({ type: 'info', title, message }),
  };
};

// 响应式工具 Hook
export const useResponsive = () => {
  const setSidebarMobile = useUIStore((state) => state.setSidebarMobile);
  
  // 监听屏幕尺寸变化
  useEffect(() => {
    const checkMobile = () => {
      const isMobile = window.innerWidth < 768;
      setSidebarMobile(isMobile);
    };
    
    checkMobile();
    window.addEventListener('resize', checkMobile);
    
    return () => window.removeEventListener('resize', checkMobile);
  }, [setSidebarMobile]);
};