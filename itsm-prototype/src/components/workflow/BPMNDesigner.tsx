'use client';

import React from 'react';
import { Button } from 'antd';

interface BPMNDesignerProps {
  xml: string;
  onSave: (xml: string) => void;
  onChange?: (xml: string) => void;
  onDeploy?: (xml: string) => void;
  readOnly?: boolean;
  height?: number | string;
  showPropertiesPanel?: boolean;
}

const BPMNDesigner: React.FC<BPMNDesignerProps> = ({
  xml = '',
  onSave,
  onChange,
  onDeploy,
  readOnly = false,
  height = 600,
  showPropertiesPanel = true,
}) => {
  const handleSave = () => {
    const finalXML =
      xml ||
      `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="Process_1" name="工作流">
  </bpmn:process>
</bpmn:definitions>`;

    if (onSave) {
      onSave(finalXML);
    }
  };

  const handleDeploy = () => {
    if (onDeploy) {
      onDeploy(xml);
    }
  };

  const handleMockChange = () => {
    if (onChange) {
      // Mock a change
      const newXML = xml + ' '; // Dummy change
      onChange(newXML);
    }
  };

  return (
    <div
      style={{
        height,
        border: '1px solid #d9d9d9',
        borderRadius: '6px',
        padding: '16px',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <div className='text-center text-gray-500 flex-1 flex flex-col justify-center items-center'>
        <div className='mb-4'>
          <h3 className='text-lg font-medium'>BPMN 流程设计器</h3>
          <p className='text-sm text-gray-400 mt-2'>当前 XML 内容长度: {xml?.length || 0} 字符</p>
          {readOnly && <p className='text-amber-500'>(只读模式)</p>}
        </div>
        <div className='space-x-4'>
          {!readOnly && (
            <>
              <Button type='primary' onClick={handleSave}>
                保存设计
              </Button>
              {onChange && <Button onClick={handleMockChange}>模拟修改</Button>}
              {onDeploy && <Button onClick={handleDeploy}>部署流程</Button>}
            </>
          )}
        </div>
        <div className='mt-8 text-xs text-gray-300 max-w-md mx-auto'>
          <p>此处应集成 bpmn-js 或其他图形化设计库。</p>
          <p>当前为占位组件，已连接后端 API。</p>
        </div>
      </div>
      {showPropertiesPanel && (
        <div className='mt-4 p-3 border-t bg-gray-50 text-sm text-gray-500'>
          <p>属性面板 (Properties Panel)</p>
        </div>
      )}
    </div>
  );
};

export default BPMNDesigner;
