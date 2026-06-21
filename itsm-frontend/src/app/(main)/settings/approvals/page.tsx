'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

/**
 * 审批配置设置页面
 * 重定向到管理后台的审批配置页面
 */
export default function SettingsApprovalsPage() {
  const router = useRouter();

  useEffect(() => {
    // 重定向到管理后台的审批配置页面
    router.replace('/admin/approvals');
  }, [router]);

  return null;
}