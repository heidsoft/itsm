/**
 * ITSM前端架构 - 统一状态管理系统
 * 
 * 状态管理策略：
 * 1. Zustand - 客户端状态管理
 * 2. React Query - 服务端状态管理
 * 3. 状态持久化 - localStorage/sessionStorage
 * 4. 状态同步 - 跨标签页同步
 * 5. 状态优化 - 选择性更新和缓存
 */

import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import { subscribeWithSelector } from 'zustand/middleware';
import { QueryClient, QueryClientProvider, useQuery, useMutation } from '@tanstack/react-query';
import { ReactNode } from 'react';

// 状态类型定义
export type StateType = 'client' | 'server' | 'persistent' | 'temporary';

// 状态配置接口
export interface StateConfig {
  name: string;
  type: StateType;
  persist?: boolean;
  storage?: 'localStorage' | 'sessionStorage';
  sync?: boolean;
  ttl?: number; // 生存时间（毫秒）
  version?: string;
}

// 状态管理器接口
export interface StateManager<T = any> {
  getState: () => T;
  setState: (state: Partial<T>) => void;
  subscribe: (callback: (state: T) => void) => () => void;
  reset: () => void;
  destroy: () => void;
}

// 客户端状态管理器
export class ClientStateManager<T> implements StateManager<T> {
  private store: any;
  private config: StateConfig;

  constructor(config: StateConfig, initialState: T) {
    this.config = config;
    this.store = create<T>()(
      subscribeWithSelector(
        persist(
          (set, get) => ({
            ...initialState,
            setState: (newState: Partial<T>) => set(newState),
            reset: () => set(initialState),
          }),
          {
            name: config.name,
            storage: config.storage ? 
              createJSONStorage(() => 
                config.storage === 'localStorage' ? localStorage : sessionStorage
              ) : undefined,
            partialize: (state) => {
              // 选择性持久化
              const { setState, reset, ...persistedState } = state as any;
              return persistedState;
            },
          }
        )
      )
    );
  }

  getState(): T {
    return this.store.getState();
  }

  setState(state: Partial<T>): void {
    this.store.setState(state);
  }

  subscribe(callback: (state: T) => void): () => void {
    return this.store.subscribe(callback);
  }

  reset(): void {
    this.store.getState().reset?.();
  }

  destroy(): void {
    // 清理资源
    this.store.destroy?.();
  }
}

// 服务端状态管理器（基于React Query）
export class ServerStateManager {
  private queryClient: QueryClient;

  constructor() {
    this.queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          staleTime: 5 * 60 * 1000, // 5分钟
          cacheTime: 10 * 60 * 1000, // 10分钟
          retry: 3,
          refetchOnWindowFocus: false,
        },
        mutations: {
          retry: 1,
        },
      },
    });
  }

  getQueryClient(): QueryClient {
    return this.queryClient;
  }

  // 查询数据
  useQuery<T>(queryKey: string[], queryFn: () => Promise<T>, options?: any) {
    return useQuery({
      queryKey,
      queryFn,
      ...options,
    });
  }

  // 变更数据
  useMutation<T, Error, any>(mutationFn: (variables: any) => Promise<T>, options?: any) {
    return useMutation({
      mutationFn,
      ...options,
    });
  }
}

// 状态同步管理器
export class StateSyncManager {
  private listeners: Map<string, Set<(data: any) => void>> = new Map();

  constructor() {
    if (typeof window !== 'undefined') {
      // 监听storage变化
      window.addEventListener('storage', this.handleStorageChange.bind(this));
      
      // 监听自定义同步事件
      window.addEventListener('state-sync', this.handleCustomSync.bind(this));
    }
  }

  private handleStorageChange(event: StorageEvent): void {
    if (event.key?.startsWith('itsm-state-')) {
      const stateName = event.key.replace('itsm-state-', '');
      const listeners = this.listeners.get(stateName);
      
      if (listeners && event.newValue) {
        try {
          const data = JSON.parse(event.newValue);
          listeners.forEach(callback => callback(data));
        } catch (error) {
          console.error('Failed to parse synced state:', error);
        }
      }
    }
  }

  private handleCustomSync(event: CustomEvent): void {
    const { stateName, data } = event.detail;
    const listeners = this.listeners.get(stateName);
    
    if (listeners) {
      listeners.forEach(callback => callback(data));
    }
  }

  // 订阅状态同步
  subscribe(stateName: string, callback: (data: any) => void): () => void {
    if (!this.listeners.has(stateName)) {
      this.listeners.set(stateName, new Set());
    }
    
    this.listeners.get(stateName)!.add(callback);
    
    return () => {
      const listeners = this.listeners.get(stateName);
      if (listeners) {
        listeners.delete(callback);
        if (listeners.size === 0) {
          this.listeners.delete(stateName);
        }
      }
    };
  }

  // 广播状态变化
  broadcast(stateName: string, data: any): void {
    if (typeof window !== 'undefined') {
      // 通过storage事件同步到其他标签页
      const storageKey = `itsm-state-${stateName}`;
      localStorage.setItem(storageKey, JSON.stringify({
        data,
        timestamp: Date.now(),
      }));

      // 通过自定义事件同步到当前标签页
      window.dispatchEvent(new CustomEvent('state-sync', {
        detail: { stateName, data }
      }));
    }
  }
}

// 状态管理器工厂
export class StateManagerFactory {
  private managers: Map<string, StateManager> = new Map();
  private serverManager: ServerStateManager;
  private syncManager: StateSyncManager;

  constructor() {
    this.serverManager = new ServerStateManager();
    this.syncManager = new StateSyncManager();
  }

  // 创建客户端状态管理器
  createClientState<T>(config: StateConfig, initialState: T): StateManager<T> {
    const manager = new ClientStateManager<T>(config, initialState);
    this.managers.set(config.name, manager);

    // 如果启用同步，订阅状态变化
    if (config.sync) {
      manager.subscribe((state) => {
        this.syncManager.broadcast(config.name, state);
      });
    }

    return manager;
  }

  // 获取状态管理器
  getStateManager(name: string): StateManager | undefined {
    return this.managers.get(name);
  }

  // 获取服务端状态管理器
  getServerManager(): ServerStateManager {
    return this.serverManager;
  }

  // 获取同步管理器
  getSyncManager(): StateSyncManager {
    return this.syncManager;
  }

  // 销毁所有管理器
  destroy(): void {
    this.managers.forEach(manager => manager.destroy());
    this.managers.clear();
  }
}

// 全局状态管理器实例
export const stateManagerFactory = new StateManagerFactory();

// React Query Provider组件
export function StateProvider({ children }: { children: ReactNode }) {
  const queryClient = stateManagerFactory.getServerManager().getQueryClient();

  return (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
}

// 状态管理Hook
export function useStateManager<T>(name: string): StateManager<T> | undefined {
  return stateManagerFactory.getStateManager(name) as StateManager<T>;
}

// 服务端状态Hook
export function useServerState<T>(
  queryKey: string[],
  queryFn: () => Promise<T>,
  options?: any
) {
  const serverManager = stateManagerFactory.getServerManager();
  return serverManager.useQuery(queryKey, queryFn, options);
}

// 服务端变更Hook
export function useServerMutation<T, V>(
  mutationFn: (variables: V) => Promise<T>,
  options?: any
) {
  const serverManager = stateManagerFactory.getServerManager();
  return serverManager.useMutation<T, Error, V>(mutationFn, options);
}

// 状态同步Hook
export function useStateSync<T>(stateName: string, callback: (data: T) => void) {
  const syncManager = stateManagerFactory.getSyncManager();
  
  return syncManager.subscribe(stateName, callback);
}

// 状态优化工具
export class StateOptimizer {
  // 防抖状态更新
  static debounce<T>(
    setState: (state: T) => void,
    delay: number = 300
  ): (state: T) => void {
    let timeoutId: NodeJS.Timeout;
    
    return (state: T) => {
      clearTimeout(timeoutId);
      timeoutId = setTimeout(() => setState(state), delay);
    };
  }

  // 节流状态更新
  static throttle<T>(
    setState: (state: T) => void,
    delay: number = 100
  ): (state: T) => void {
    let lastCall = 0;
    
    return (state: T) => {
      const now = Date.now();
      if (now - lastCall >= delay) {
        lastCall = now;
        setState(state);
      }
    };
  }

  // 选择性状态更新
  static selectiveUpdate<T>(
    currentState: T,
    newState: Partial<T>,
    keys: (keyof T)[]
  ): T {
    const updatedState = { ...currentState };
    
    keys.forEach(key => {
      if (key in newState) {
        updatedState[key] = newState[key] as T[keyof T];
      }
    });
    
    return updatedState;
  }
}

export default {
  StateManagerFactory,
  stateManagerFactory,
  StateProvider,
  useStateManager,
  useServerState,
  useServerMutation,
  useStateSync,
  StateOptimizer,
};
