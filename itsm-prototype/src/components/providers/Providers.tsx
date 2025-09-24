'use client';

import { useEffect } from 'react';
import { AuthService } from '@/lib/auth/auth-service';
import { useAuthStore } from '@/lib/store/auth-store';
import { NotificationContainer } from '@/components/ui/NotificationContainer';

/**
 * 全局提供者组件
 * 负责初始化应用状态和提供全局上下文
 */
export function Providers({ children }: { children: React.ReactNode }) {
  const { login, logout, setLoading } = useAuthStore();

  // 初始化认证状态
  useEffect(() => {
    const initAuth = async () => {
      setLoading(true);
      
      try {
        // 检查是否有有效的认证状态
        const isAuthenticated = AuthService.isAuthenticated();
        
        if (isAuthenticated) {
          // 获取当前用户信息
          const user = await AuthService.getCurrentUser();
          const token = AuthService.getToken();
          
          if (user && token) {
            login(user, token);
          }
        }
      } catch (error) {
        console.error('Failed to initialize auth:', error);
        // 清除可能损坏的认证状态
        AuthService.logout();
        logout();
      } finally {
        setLoading(false);
      }
    };

    initAuth();
  }, [login, logout, setLoading]);

  return (
    <>
      {children}
      <NotificationContainer />
    </>
  );
}