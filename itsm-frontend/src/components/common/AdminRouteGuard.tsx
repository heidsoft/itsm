'use client';

import { Button, Result } from 'antd';
import { usePathname, useRouter } from 'next/navigation';
import { usePermissions } from '@/lib/hooks/use-permissions';

export function AdminRouteGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const { isAdmin } = usePermissions();
  const router = useRouter();

  if (pathname.startsWith('/admin') && !isAdmin()) {
    return (
      <Result
        status="403"
        title="403"
        subTitle="抱歉，您没有权限访问此页面。"
        extra={
          <Button type="primary" onClick={() => router.push('/')}>
            返回首页
          </Button>
        }
      />
    );
  }

  return <>{children}</>;
}
