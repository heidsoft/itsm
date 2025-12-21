'use client';

import React from 'react';
import { Button } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import IncidentDetail from '@/modules/incident/components/IncidentDetail';

export default function IncidentDetailPage() {
  const router = useRouter();

  return (
    <div style={{ padding: 24 }}>
      <div style={{ marginBottom: 16 }}>
        <Button
          type="link"
          icon={<ArrowLeftOutlined />}
          onClick={() => router.back()}
          style={{ paddingLeft: 0, color: '#666' }}
        >
          返回列表
        </Button>
      </div>
      <IncidentDetail />
    </div>
  );
}
