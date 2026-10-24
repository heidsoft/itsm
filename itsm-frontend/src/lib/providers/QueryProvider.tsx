'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode} from 'react';
import { useState, lazy, Suspense, useEffect } from 'react';
import { ApiRequestError } from '@/lib/api/http-client';

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
//
// 重试策略按「请求是否会改变服务端状态」分开，这是本文件最容易出事的一处配置：
// 写请求一旦重试，就可能重复生效。典型场景是「服务端已受理但响应超时/断连」——
// 客户端看到的是失败，服务端却已经把审批决策、建单写进去了，1 秒后的自动重放
// 会再写一次。因此 mutation 一律不重试；query 只读，重试是安全的，但 4xx 是
// 服务端明确拒绝（无权限/参数错/不存在），重试只是让用户多等几秒并刷一堆错误日志。
// 导出以便回归测试直接断言重试策略（写操作不得重试是最容易悄悄退化的约定）。
export const createQueryClient = () => {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // 数据缓存时间（5分钟）
        staleTime: 5 * 60 * 1000,
        // 缓存保留时间（10分钟）
        gcTime: 10 * 60 * 1000,
        // 重试次数：仅对「可能自愈」的失败重试（网络错误 / 5xx / 429 限流）
        retry: (failureCount, error) => {
          if (failureCount >= 3) return false;
          return isRetryableError(error);
        },
        // 重试延迟
        retryDelay: attemptIndex => Math.min(1000 * 2 ** attemptIndex, 30000),
        // 窗口失焦时不重新获取数据
        refetchOnWindowFocus: false,
        // 网络重连时重新获取数据
        refetchOnReconnect: true,
      },
      mutations: {
        // 写操作不重试：无法区分「没发出去」和「已受理但响应丢失」，
        // 盲目重试会让审批决策、建单等动作重复生效。需要重试的场景由调用方
        // 自行带上幂等键显式发起。
        retry: 0,
      },
    },
  });
};

/**
 * 是否值得重试。
 *
 * - 无 status 的错误 = 网络层失败（断网、DNS、CORS、超时），可能自愈，重试。
 * - 5xx = 服务端暂时不可用，可能自愈，重试。
 * - 429 = 限流，退避后重试。
 * - 其余 4xx（401/403/404/409/422…）= 服务端明确拒绝，重试只会重复同样的失败。
 */
function isRetryableError(error: unknown): boolean {
  if (!(error instanceof ApiRequestError)) {
    // 非 ApiRequestError（例如网络异常）无法判定状态码，按可重试处理。
    return true;
  }
  const status = error.status;
  if (status === undefined) return true;
  if (status >= 500) return true;
  if (status === 429) return true;
  return false;
}

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
