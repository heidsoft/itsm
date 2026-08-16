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
    // 审批配置已迁移到 BPMN 工作流管理，跳转到工作流管理页面
    router.replace('/admin/workflows');
  }, [router]);

  return null;
}