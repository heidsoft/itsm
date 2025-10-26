"use client";

import React from "react";
import { Button } from "antd";

interface EnhancedBPMNDesignerProps {
  xml: string;
  onSave: (xml: string) => void;
  onChange?: (xml: string) => void;
  height?: number;
}

const EnhancedBPMNDesigner: React.FC<EnhancedBPMNDesignerProps> = ({
  xml,
  onSave,
  onChange,
  height = 600,
}) => {
  const handleSave = (data: any) => {
    // 将 WorkflowDesigner 的数据转换为 BPMN XML 格式
    // 这里简化处理，实际应该根据 elements  and connections 生成 XML
    const mockXML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="Process_1" name="工作流">
    <!-- 这里应该根据实际数据生成 -->
  </bpmn:process>
</bpmn:definitions>`;

    onSave(mockXML);
  };

  const handleChange = (data: any) => {
    if (onChange) {
      // 模拟XML变化
      const mockXML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="Process_1" name="工作流">
    <!-- 这里应该根据实际数据生成 -->
  </bpmn:process>
</bpmn:definitions>`;

      onChange(mockXML);
    }
  };

  return (
    <div
      style={{
        height,
        border: "1px solid #d9d9d9",
        borderRadius: "6px",
        padding: "16px",
      }}
    >
      <div className="text-center text-gray-500">
        <div className="mb-4">
          <h3>BPMN流程设计器</h3>
          <p>当前XML内容长度: {xml.length} 字符</p>
        </div>
        <div className="space-y-2">
          <Button type="primary" onClick={handleSave}>
            保存流程
          </Button>
          <Button onClick={handleChange}>模拟变更</Button>
        </div>
        <div className="mt-4 text-xs text-gray-400">
          <p>这是一个简化的BPMN设计器演示</p>
          <p>实际项目中应该集成专业的BPMN设计器组件</p>
        </div>
      </div>
    </div>
  );
};

export default EnhancedBPMNDesigner;
