'use client';

/**
 * 工作流实例详情页（深链接路由）
 *
 * 从审批中心等处跳转到 /workflow/instances/[id] 时展示。
 * 复用 WorkflowInstanceDetail 组件，含基本信息 / 任务列表 / 执行历史三个 Tab。
 */

import React, { Suspense } from 'react';
import { useParams } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { Button, Space, Spin } from 'antd';
import { useRouter } from 'next/navigation';

import { WorkflowInstanceDetail } from '@/components/workflow/WorkflowInstanceDetail';

function InstanceDetailContent() {
  const router = useRouter();
  const params = useParams();
  const instanceId = params?.id as string | undefined;

  if (!instanceId) {
    return <div className="p-6 text-center text-gray-500">缺少实例 ID</div>;
  }

  return (
    <div className="p-6">
      <WorkflowInstanceDetail
        instanceId={instanceId}
        onBack={() => router.push('/workflow/instances')}
      />
    </div>
  );
}

export default function WorkflowInstanceDetailPage() {
  return (
    <Suspense
      fallback={
        <div className="flex items-center justify-center h-64">
          <Spin tip="加载中…" />
        </div>
      }
    >
      <InstanceDetailContent />
    </Suspense>
  );
}
