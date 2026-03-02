// 工作流画布组件
// Workflow Canvas Component - BPMN 设计器画布

'use client';

import React from 'react';
import dynamic from 'next/dynamic';
import { Spin } from 'antd';

// 动态导入 BPMN 设计器 - bpmn-js 库较大，按需加载
const BPMNDesigner = dynamic(() => import('@/components/workflow/BPMNDesigner'), {
  ssr: false,
  loading: () => (
    <div className="flex items-center justify-center h-full">
      <Spin size="large" tip="加载流程设计器..." />
    </div>
  ),
});

interface WorkflowCanvasProps {
  currentXML: string;
  onSave: (xml: string) => void;
  onChange: (xml: string) => void;
}

export default function WorkflowCanvas({ currentXML, onSave, onChange }: WorkflowCanvasProps) {
  // XML 变更处理
  const handleChange = (xml: string) => {
    onChange(xml);
  };

  // 保存处理
  const handleSave = (xml: string) => {
    onSave(xml);
  };

  return (
    <div className="h-[calc(100vh-200px)] bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
      <BPMNDesigner xml={currentXML} onSave={handleSave} onChange={handleChange} />
    </div>
  );
}
