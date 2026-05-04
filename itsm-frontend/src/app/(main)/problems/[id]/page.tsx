'use client';

import React from 'react';
import { Button } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useRouter, useParams } from 'next/navigation';
import ProblemDetail from '@/components/problem/ProblemDetail';

export default function ProblemDetailPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;

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
      <ProblemDetail id={id} />
    </div>
  );
}
