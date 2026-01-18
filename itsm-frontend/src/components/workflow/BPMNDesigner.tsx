'use client';

import React, { useEffect, useRef, useState, useCallback } from 'react';
import { Button, Tooltip, message } from 'antd';
import {
  Save,
  PlayCircle,
  ZoomIn,
  ZoomOut,
  Maximize,
  Undo,
  Redo,
  Trash2,
  FileJson,
  Download,
  Upload,
} from 'lucide-react';
import BpmnModeler from 'bpmn-js/lib/Modeler';

import 'bpmn-js/dist/assets/diagram-js.css';
import 'bpmn-js/dist/assets/bpmn-font/css/bpmn.css';

interface BPMNDesignerProps {
  xml: string;
  onSave: (xml: string) => void;
  onChange?: (xml: string) => void;
  onDeploy?: (xml: string) => void;
  readOnly?: boolean;
  height?: number | string;
}

const BPMNDesigner: React.FC<BPMNDesignerProps> = ({
  xml = '',
  onSave,
  onChange,
  onDeploy,
  readOnly = false,
  height = 600,
}) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const modelerRef = useRef<BpmnModeler | null>(null);
  const [currentXML, setCurrentXML] = useState(xml);
  const [zoom, setZoom] = useState(1);
  const [history, setHistory] = useState<string[]>([xml]);
  const [historyIndex, setHistoryIndex] = useState(0);

  // 初始化 BPMN Modeler
  useEffect(() => {
    if (!containerRef.current || modelerRef.current) return;

    const modeler = new BpmnModeler({
      container: containerRef.current,
      keyboard: {
        bindTo: window,
      },
    });

    modelerRef.current = modeler;

    // 加载初始 XML
    if (xml) {
      modeler.importXML(xml).catch((err: Error) => {
        console.error('Failed to import XML:', err);
        message.error('加载流程图失败');
      });
    } else {
      // 创建空白流程
      const blankXML = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="Process_1" name="新流程" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1" name="开始"/>
  </bpmn:process>
</bpmn:definitions>`;
      modeler.importXML(blankXML).catch((err: Error) => {
        console.error('Failed to create blank diagram:', err);
      });
    }

    // 监听变化
    modeler.on('commandStack.changed', () => {
      modeler.saveXML({ format: true }).then((result) => {
        if (result.xml) {
          setCurrentXML(result.xml);
          onChange?.(result.xml);

          // 更新历史记录
          setHistory((prev) => {
            const newHistory = prev.slice(0, historyIndex + 1);
            newHistory.push(result.xml!);
            return newHistory.slice(-20); // 最多保留20步
          });
        }
      });
    });

    return () => {
      modeler.destroy();
      modelerRef.current = null;
    };
  }, []);

  // 更新 XML
  useEffect(() => {
    if (xml !== currentXML && modelerRef.current) {
      modelerRef.current.importXML(xml).catch((err: Error) => {
        console.error('Failed to import XML:', err);
      });
    }
  }, [xml, currentXML]);

  // 保存
  const handleSave = useCallback(() => {
    if (onSave) {
      onSave(currentXML);
    }
  }, [currentXML, onSave]);

  // 部署
  const handleDeploy = useCallback(() => {
    if (onDeploy) {
      onDeploy(currentXML);
    }
  }, [currentXML, onDeploy]);

  // 导出图片
  const handleExportSVG = useCallback(async () => {
    if (!modelerRef.current) return;
    try {
      const result = await modelerRef.current.saveSVG();
      if (result.svg) {
        const blob = new Blob([result.svg], { type: 'image/svg+xml' });
        const url = URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        link.download = 'workflow.svg';
        link.click();
        URL.revokeObjectURL(url);
        message.success('SVG 已导出');
      }
    } catch {
      message.error('导出失败');
    }
  }, []);

  // 导出XML
  const handleExportXML = useCallback(() => {
    const blob = new Blob([currentXML], { type: 'application/xml' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'workflow.bpmn';
    link.click();
    URL.revokeObjectURL(url);
    message.success('BPMN 文件已导出');
  }, [currentXML]);

  // 导入XML
  const handleImportXML = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      modelerRef.current?.importXML(content).then(() => {
        message.success('导入成功');
      }).catch((err: Error) => {
        message.error('导入失败: ' + err.message);
      });
    };
    reader.readAsText(file);
  }, []);

  // 缩放
  const handleZoom = useCallback((delta: number) => {
    if (!modelerRef.current) return;
    const canvas = modelerRef.current.get('canvas') as { zoom: (level: number) => void } | undefined;
    if (canvas) {
      const newZoom = Math.min(Math.max(zoom + delta, 0.2), 4);
      canvas.zoom(newZoom);
      setZoom(newZoom);
    }
  }, [zoom]);

  // 重置缩放
  const handleZoomReset = useCallback(() => {
    if (!modelerRef.current) return;
    const canvas = modelerRef.current.get('canvas') as { zoom: (level: string) => void } | undefined;
    if (canvas) {
      canvas.zoom('fit-viewport');
      setZoom(1);
    }
  }, []);

  // 撤销
  const handleUndo = useCallback(() => {
    if (!modelerRef.current || historyIndex <= 0) return;
    const newIndex = historyIndex - 1;
    modelerRef.current.importXML(history[newIndex]);
    setHistoryIndex(newIndex);
  }, [history, historyIndex]);

  // 重做
  const handleRedo = useCallback(() => {
    if (!modelerRef.current || historyIndex >= history.length - 1) return;
    const newIndex = historyIndex + 1;
    modelerRef.current.importXML(history[newIndex]);
    setHistoryIndex(newIndex);
  }, [history, historyIndex]);

  // 删除选中
  const handleDelete = useCallback(() => {
    if (!modelerRef.current) return;
    const selectionObj = modelerRef.current.get('selection') as { get: () => string[] } | undefined;
    const modelingObj = modelerRef.current.get('modeling') as { removeElements: (elements: string[]) => void } | undefined;
    const selection = selectionObj?.get?.() || [];
    if (selection.length > 0 && modelingObj) {
      modelingObj.removeElements(selection);
    }
  }, []);

  return (
    <div style={{ display: 'flex', height, border: '1px solid #d9d9d9', borderRadius: '6px' }}>
      {/* 工具栏 */}
      <div style={{
        width: 48,
        borderRight: '1px solid #d9d9d9',
        background: '#f5f5f5',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        padding: '8px 0',
        gap: 4,
      }}>
        <Tooltip title="保存" placement="right">
          <Button type="text" icon={<Save size={18} />} onClick={handleSave} disabled={readOnly} />
        </Tooltip>
        <Tooltip title="部署" placement="right">
          <Button type="text" icon={<PlayCircle size={18} />} onClick={handleDeploy} disabled={readOnly} />
        </Tooltip>
        <Tooltip title="撤销" placement="right">
          <Button
            type="text"
            icon={<Undo size={18} />}
            onClick={handleUndo}
            disabled={readOnly || historyIndex <= 0}
          />
        </Tooltip>
        <Tooltip title="重做" placement="right">
          <Button
            type="text"
            icon={<Redo size={18} />}
            onClick={handleRedo}
            disabled={readOnly || historyIndex >= history.length - 1}
          />
        </Tooltip>
        <div style={{ flex: 1 }} />
        <Tooltip title="导出SVG" placement="right">
          <Button type="text" icon={<Download size={18} />} onClick={handleExportSVG} />
        </Tooltip>
        <Tooltip title="导出BPMN" placement="right">
          <Button type="text" icon={<FileJson size={18} />} onClick={handleExportXML} />
        </Tooltip>
        <label>
          <input type="file" accept=".bpmn,.xml" style={{ display: 'none' }} onChange={handleImportXML} />
          <Tooltip title="导入BPMN" placement="right">
            <Button type="text" icon={<Upload size={18} />} />
          </Tooltip>
        </label>
      </div>

      {/* BPMN 图 */}
      <div ref={containerRef} style={{ flex: 1, position: 'relative' }} />

      {/* 缩放控制 */}
      <div style={{
        position: 'absolute',
        bottom: 16,
        right: 16,
        display: 'flex',
        gap: 4,
        background: 'white',
        padding: 4,
        borderRadius: 6,
        boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
      }}>
        <Tooltip title="缩小">
          <Button type="text" size="small" icon={<ZoomOut size={16} />} onClick={() => handleZoom(-0.1)} />
        </Tooltip>
        <span style={{ minWidth: 50, textAlign: 'center', lineHeight: '28px' }}>
          {Math.round(zoom * 100)}%
        </span>
        <Tooltip title="放大">
          <Button type="text" size="small" icon={<ZoomIn size={16} />} onClick={() => handleZoom(0.1)} />
        </Tooltip>
        <Tooltip title="适应屏幕">
          <Button type="text" size="small" icon={<Maximize size={16} />} onClick={handleZoomReset} />
        </Tooltip>
      </div>
    </div>
  );
};

export default BPMNDesigner;
