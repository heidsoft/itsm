/**
 * Zustand Store 模板
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

// Auth Store 示例
interface AuthState {
  user: { id: number; name: string; email: string } | null;
  isAuthenticated: boolean;
}

interface AuthActions {
  setUser: (user: AuthState['user']) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState & AuthActions>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      setUser: (user) => set({ user, isAuthenticated: !!user }),
      logout: () => set({ user: null, isAuthenticated: false }),
    }),
    { name: 'auth-storage' }
  )
);

// 通用 Store 创建函数
export function createSimpleStore<T extends object>(
  name: string,
  initialState: T
) {
  return create<T>()(
    persist(
      () => initialState,
      { name }
    )
  );
}
