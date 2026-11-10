'use client';

import React from 'react';
import { Card } from 'antd';
import A2UIFormRenderer from '@/components/a2ui/A2UIFormRenderer';

export default function AICreateTicketPage() {
  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="mb-6">
        <h2 className="text-2xl font-bold text-gray-900 m-0">AI creates work order</h2>
        <p className="text-gray-500 mt-1">Use natural language to describe your needs, and AI will automatically generate and submit a work order</p>
      </div>

      <Card>
        <A2UIFormRenderer />
      </Card>
    </div>
  );
}
