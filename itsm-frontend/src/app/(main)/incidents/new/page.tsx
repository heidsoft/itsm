'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Spin } from 'antd';

/**
 * 事件创建页面 - 重定向到统一入口
 * @deprecated 请使用 /incidents/create
 */
export default function NewIncidentPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace('/incidents/create');
  }, [router]);

  return (
    <div className="flex items-center justify-center min-h-[400px]">
      <Spin size="large" tip="正在跳转到创建页面..." />
    </div>
  );
}
