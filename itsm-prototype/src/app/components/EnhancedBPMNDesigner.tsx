"use client";

import React from "react";
import { WorkflowDesigner } from "./WorkflowDesigner";

interface EnhancedBPMNDesignerProps {
  initialXML?: string;
  onSave: (xml: string) => void;
  onDeploy?: (xml: string) => void;
  height?: number;
  showPropertiesPanel?: boolean;
}

const EnhancedBPMNDesigner: React.FC<EnhancedBPMNDesignerProps> = ({
  initialXML,
  onSave,
  onDeploy,
  height = 600,
  showPropertiesPanel = true,
}) => {
  const handleSave = (data: any) => {
    // 将 WorkflowDesigner 的数据转换为 BPMN XML 格式
    // 这里简化处理，实际应该根据 elements 和 connections 生成 XML
    const mockXML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="Process_1" name="工作流">
    <!-- 这里应该根据实际数据生成 -->
  </bpmn:process>
</bpmn:definitions>`;

    onSave(mockXML);
  };

  const handleCancel = () => {
    // 取消操作
  };

  return (
    <div style={{ height }}>
      <WorkflowDesigner
        workflow={initialXML ? { bpmn_xml: initialXML } : undefined}
        onSave={handleSave}
        onCancel={handleCancel}
      />
    </div>
  );
};

export default EnhancedBPMNDesigner;
