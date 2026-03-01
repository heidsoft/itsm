'use client';

import React, { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { WorkflowDesigner } from '@/components/workflow/designer';

interface WorkflowDesignerPageProps {
  params: Promise<{ id?: string }>;
}

const WorkflowDesignerPage: React.FC<WorkflowDesignerPageProps> = ({ params }) => {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [resolvedParams, setResolvedParams] = useState<{ id?: string }>({ id: undefined });

  // 解析 Promise params
  useEffect(() => {
    params.then(setResolvedParams).catch(console.error);
  }, [params]);

  // 从URL参数获取工作流ID
  const workflowId = resolvedParams?.id || searchParams.get('id');

  return <WorkflowDesigner workflowId={workflowId || undefined} />;
};

export default WorkflowDesignerPage;
