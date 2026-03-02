'use client';

import React from 'react';
import { Card } from 'antd';
import A2UIFormRenderer from '@/components/a2ui/A2UIFormRenderer';

export default function AICreateTicketPage() {
  return (
    <div className='max-w-4xl mx-auto p-6'>
      <div className='mb-6'>
        <h2 className='text-2xl font-bold text-gray-900 m-0'>AI 创建工单</h2>
        <p className='text-gray-500 mt-1'>用自然语言描述您的需求，AI 将自动生成并提交工单</p>
      </div>

      <Card>
        <A2UIFormRenderer />
      </Card>
    </div>
  );
}
