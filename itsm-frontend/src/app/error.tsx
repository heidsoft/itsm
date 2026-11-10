'use client';

import React, { useEffect } from 'react';
import { Result, Button } from 'antd';
import { LayoutDashboard, RotateCcw } from 'lucide-react';
import { useRouter } from 'next/navigation';

/**
 * 路由级错误边界
 * 捕获子路由中的运行时错误并显示降级 UI
 */
export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const router = useRouter();

  useEffect(() => {
    // 记录错误到控制台（后续可替换为 logger 服务）
    console.error('[RouteError]', error);
  }, [error]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <Result
        status="500"
        title="Page error"
        subTitle="Sorry, the page encountered an unexpected error. Please try again or return to the dashboard."
        extra={[
          <Button key="retry" type="primary" icon={<RotateCcw />} onClick={() => reset()}>

            Retry
          </Button>,
          <Button
            key="dashboard"
            icon={<LayoutDashboard />}
            onClick={() => router.push('/dashboard')}
          >

            Return to dashboard
          </Button>,
        ]}
      />
    </div>
  );
}
