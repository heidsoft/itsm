'use client';

import React from 'react';
import { Button } from 'antd';
import { ArrowLeft } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';
import IncidentDetail from '@/components/incident/IncidentDetail';

// 动态路由参数类型
export default function IncidentDetailPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 16 }}>
        <Button
          type="link"
          icon={<ArrowLeft />}
          onClick={() => router.back()}
          style={{ paddingLeft: 0, color: '#666' }}
        >
          返回列表
        </Button>
      </div>
      {/* 将ID传递给IncidentDetail组件 */}
      <IncidentDetail id={id} />
    </div>
  );
}
