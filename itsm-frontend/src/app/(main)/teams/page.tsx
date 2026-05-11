'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

/**
 * 团队管理页面
 * 重定向到 /admin/teams
 * 保留 /teams 路由以兼容旧链接
 */
export default function TeamsPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace('/admin/teams');
  }, [router]);

  return null;
}
