'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

/**
 * 工作流列表页面
 * 重定向到 /workflow
 * 保留 /workflows 路由以兼容旧链接
 */
export default function WorkflowsPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace('/workflow');
  }, [router]);

  return null;
}
