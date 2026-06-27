// 工作流画布组件
// Workflow Canvas Component - BPMN 设计器画布

'use client';

import React, { forwardRef } from 'react';
import dynamic from 'next/dynamic';
import { Spin } from 'antd';
import type { BpmnNodeSelection } from '../BPMNDesigner';

// 动态导入 BPMN 设计器 - bpmn-js 库较大，按需加载
const BPMNDesigner = dynamic(() => import('../BPMNDesigner'), {
  ssr: false,
  loading: () => (
    <div className="flex items-center justify-center h-full">
      <Spin size="large" description="加载流程设计器..." />
    </div>
  ),
});

export interface BpmnDesignerApi {
  updateElementProperties: (elementId: string, properties: Record<string, unknown>) => boolean;
  fitViewport: () => void;
}

interface WorkflowCanvasProps {
  currentXML: string;
  onSave: (xml: string) => void;
  onChange: (xml: string) => void;
  onSelectionChange?: (selection: BpmnNodeSelection | null) => void;
}

/**
 * 用模块级 ref 桥接 dynamic 组件与 BPMNDesigner 的命令式 API。
 * 因为 dynamic 组件无法直接转发 ref，使用 module-scope ref 通信。
 */
const _apiRef: { current: BpmnDesignerApi | null } = { current: null };

const WorkflowCanvas = forwardRef<BpmnDesignerApi, WorkflowCanvasProps>(function WorkflowCanvas(
  { currentXML, onSave, onChange, onSelectionChange },
  _ref
) {
  return (
    <div className="h-[calc(100vh-200px)] bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
      <BPMNDesigner
        xml={currentXML}
        onSave={onSave}
        onChange={onChange}
        onSelectionChange={onSelectionChange}
        apiRef={_apiRef}
      />
    </div>
  );
});

/**
 * 父组件调用此函数获取 BPMNDesigner 的命令式 API 句柄
 */
export function getBpmnDesignerApi(): BpmnDesignerApi | null {
  return _apiRef.current;
}

export default WorkflowCanvas;
