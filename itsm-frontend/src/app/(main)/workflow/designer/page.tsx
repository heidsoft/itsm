'use client';

import React, { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import { WorkflowDesigner } from '@/components/workflow/designer';

interface WorkflowDesignerPageProps {
  params?: Promise<{ id?: string }> | { id?: string };
}

const WorkflowDesignerPage: React.FC<WorkflowDesignerPageProps> = ({ params }) => {
  const searchParams = useSearchParams();
  const [resolvedParams, setResolvedParams] = useState<{ id?: string }>({ id: undefined });

  useEffect(() => {
    if (!params) return;
    if (typeof (params as Promise<{ id?: string }>).then === 'function') {
      (params as Promise<{ id?: string }>).then(setResolvedParams).catch(console.error);
      return;
    }
    setResolvedParams(params as { id?: string });
  }, [params]);

  const workflowId = resolvedParams?.id || searchParams.get('id');

  return <WorkflowDesigner workflowId={workflowId || undefined} />;
};

export default WorkflowDesignerPage;
