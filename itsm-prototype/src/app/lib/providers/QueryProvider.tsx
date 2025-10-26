'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode, useState, lazy, Suspense, useEffect } from 'react';

// 动态导入ReactQueryDevtools，只在开发环境中使用
// 使用 disableHydrationWarning 防止 SSR 水合错误
const ReactQueryDevtools =
  typeof window !== 'undefined' && process.env.NODE_ENV === 'development'
    ? lazy(() =>
        import('@tanstack/react-query-devtools').then(module => ({
          default: module.ReactQueryDevtools,
        }))
      )
    : null;

// 创建QueryClient实例
const createQueryClient = () => {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // 数据缓存时间（5分钟）
        staleTime: 5 * 60 * 1000,
        // 缓存保留时间（10分钟）
        gcTime: 10 * 60 * 1000,
        // 重试次数
        retry: 3,
        // 重试延迟
        retryDelay: attemptIndex => Math.min(1000 * 2 ** attemptIndex, 30000),
        // 窗口失焦时不重新获取数据
        refetchOnWindowFocus: false,
        // 网络重连时重新获取数据
        refetchOnReconnect: true,
        // 错误时重试
        refetchOnError: true,
      },
      mutations: {
        // 失败时重试
        retry: 1,
        // 重试延迟
        retryDelay: 1000,
      },
    },
  });
};

interface QueryProviderProps {
  children: ReactNode;
}

export const QueryProvider: React.FC<QueryProviderProps> = ({ children }) => {
  const [queryClient] = useState(() => createQueryClient());
  const [mounted, setMounted] = useState(false);

  // 使用 useEffect 确保只在客户端挂载
  useEffect(() => {
    setMounted(true);
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      {mounted && typeof window !== 'undefined' && ReactQueryDevtools && (
        <Suspense fallback={null}>
          <ReactQueryDevtools initialIsOpen={false} />
        </Suspense>
      )}
    </QueryClientProvider>
  );
};
