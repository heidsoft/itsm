'use client';

import React, { useEffect, useRef, useState, useCallback } from 'react';
import { Button, Tooltip, App } from 'antd';
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
  onSelectionChange?: (selection: BpmnNodeSelection | null) => void;
  /**
   * 命令式 API 容器。父组件传入 ref-like 对象，
   * 组件内部会把 { updateElementProperties, fitViewport } 写入 ref.current
   */
  apiRef?: { current: BpmnDesignerApi | null };
}

/**
 * 暴露给父组件的命令式 API
 */
export interface BpmnDesignerApi {
  updateElementProperties: (elementId: string, properties: Record<string, unknown>) => boolean;
  fitViewport: () => void;
}

/**
 * BPMN 节点当前选中信息。
 * null 表示当前未选中任何元素。
 */
export interface BpmnNodeSelection {
  id: string;
  type: string; // bpmn:UserTask / bpmn:ServiceTask / bpmn:ExclusiveGateway 等
  name?: string;
  businessObject?: Record<string, unknown>;
}

const BPMNDesigner: React.FC<BPMNDesignerProps> = ({
  xml = '',
  onSave,
  onChange,
  onDeploy,
  readOnly = false,
  height = 600,
  onSelectionChange,
  apiRef,
}) => {
  const { message } = App.useApp();
  const containerRef = useRef<HTMLDivElement>(null);
  const modelerRef = useRef<BpmnModeler | null>(null);
  const initAttemptedRef = useRef(false);
  const [currentXML, setCurrentXML] = useState(xml);
  const [zoom, setZoom] = useState(1);
  const [history, setHistory] = useState<string[]>([xml]);
  const [historyIndex, setHistoryIndex] = useState(0);

  // 等待容器具有有效尺寸后再初始化 BPMN Modeler
  // bpmn-js 在容器尺寸为 0 时会抛出 "Cannot read properties of undefined (reading 'root-0')"
  const initializeModeler = useCallback(() => {
    if (!containerRef.current || modelerRef.current || initAttemptedRef.current) {
      return;
    }

    const rect = containerRef.current.getBoundingClientRect();
    if (rect.width <= 0 || rect.height <= 0) {
      // 容器尚未布局完成，等待下一帧重试
      return;
    }

    initAttemptedRef.current = true;

    try {
      const modeler = new BpmnModeler({
        container: containerRef.current,
      });

      modelerRef.current = modeler;

      // 加载初始 XML
      if (xml) {
        modeler
          .importXML(xml)
          .then(() => {
            const canvas = modeler.get('canvas') as
              | { zoom: (level: string) => void }
              | undefined;
            canvas?.zoom('fit-viewport');
          })
          .catch((err: Error) => {
            console.error('Failed to import XML:', err);
            message.error('加载流程图失败');
          });
      } else {
        modeler
          .createDiagram()
          .then(() => {
            const canvas = modeler.get('canvas') as
              | { zoom: (level: string) => void }
              | undefined;
            canvas?.zoom('fit-viewport');
          })
          .catch((err: Error) => {
            console.error('Failed to create blank diagram:', err);
          });
      }

      // 监听变化
      modeler.on('commandStack.changed', () => {
        modeler.saveXML({ format: true }).then(result => {
          if (result.xml) {
            setCurrentXML(result.xml);
            onChange?.(result.xml);

            // 更新历史记录
            setHistory(prev => {
              const newHistory = prev.slice(0, historyIndex + 1);
              newHistory.push(result.xml!);
              return newHistory.slice(-20); // 最多保留20步
            });
          }
        });
      });

      // 监听选中节点变化，向父组件传递当前节点
      if (onSelectionChange) {
        const selection = modeler.get('selection') as {
          on: (event: string, handler: () => void) => void;
          get: () => string[];
        };
        const elementRegistry = modeler.get('elementRegistry') as {
          get: (id: string) => any;
        };
        const notifySelection = () => {
          const ids = selection.get();
          if (ids.length === 0) {
            onSelectionChange(null);
            return;
          }
          const el = elementRegistry.get(ids[0]);
          if (!el) {
            onSelectionChange(null);
            return;
          }
          onSelectionChange({
            id: el.id,
            type: el.type || (el.businessObject && el.businessObject.$type) || 'unknown',
            name: el.businessObject?.name,
            businessObject: el.businessObject || {},
          });
        };
        selection.on('selection.changed', notifySelection);
      }
    } catch (err) {
      console.error('Failed to initialize BPMN Modeler:', err);
      initAttemptedRef.current = false;
    }
  }, [xml, historyIndex, message, onChange, onSelectionChange]);

  // 初始化 BPMN Modeler - 等待容器布局完成
  useEffect(() => {
    if (!containerRef.current) return;

    // 使用 requestAnimationFrame 确保 DOM 已完成布局
    let rafId: number | null = null;
    let resizeObserver: ResizeObserver | null = null;

    const tryInit = () => {
      if (!containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      if (rect.width > 0 && rect.height > 0) {
        initializeModeler();
      } else {
        // 容器仍未布局，使用 ResizeObserver 等待
        resizeObserver = new ResizeObserver(entries => {
          for (const entry of entries) {
            const { width, height } = entry.contentRect;
            if (width > 0 && height > 0) {
              initializeModeler();
              resizeObserver?.disconnect();
              break;
            }
          }
        });
        resizeObserver.observe(containerRef.current);
      }
    };

    rafId = requestAnimationFrame(tryInit);

    return () => {
      if (rafId !== null) {
        cancelAnimationFrame(rafId);
      }
      resizeObserver?.disconnect();
      if (modelerRef.current) {
        try {
          modelerRef.current.destroy();
        } catch (err) {
          console.warn('BPMN Modeler destroy failed:', err);
        }
        modelerRef.current = null;
      }
      initAttemptedRef.current = false;
    };
  }, [initializeModeler]);

  // 同步 xml prop 变化 - 仅在父组件传入新的 XML 时才更新
  // 使用 ref 跟踪上次 prop 避免命令栈变化触发的回环
  const lastPropXmlRef = useRef(xml);
  useEffect(() => {
    if (xml === lastPropXmlRef.current) return;
    lastPropXmlRef.current = xml;

    if (modelerRef.current && xml) {
      modelerRef.current
        .importXML(xml)
        .then(() => {
          const canvas = modelerRef.current!.get('canvas') as
            | { zoom: (level: string) => void }
            | undefined;
          canvas?.zoom('fit-viewport');
          setCurrentXML(xml);
        })
        .catch((err: Error) => {
          console.error('Failed to import XML:', err);
        });
    }
  }, [xml]);

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
    reader.onload = e => {
      const content = e.target?.result as string;
      modelerRef.current
        ?.importXML(content)
        .then(() => {
          message.success('导入成功');
        })
        .catch((err: Error) => {
          message.error('导入失败: ' + err.message);
        });
    };
    reader.readAsText(file);
  }, []);

  // 缩放
  const handleZoom = useCallback(
    (delta: number) => {
      if (!modelerRef.current) return;
      const canvas = modelerRef.current.get('canvas') as
        | { zoom: (level: number) => void }
        | undefined;
      if (canvas) {
        const newZoom = Math.min(Math.max(zoom + delta, 0.2), 4);
        canvas.zoom(newZoom);
        setZoom(newZoom);
      }
    },
    [zoom]
  );

  // 重置缩放
  const handleZoomReset = useCallback(() => {
    if (!modelerRef.current) return;
    const canvas = modelerRef.current.get('canvas') as
      | { zoom: (level: string) => void }
      | undefined;
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
    const modelingObj = modelerRef.current.get('modeling') as
      | { removeElements: (elements: string[]) => void }
      | undefined;
    const selection = selectionObj?.get?.() || [];
    if (selection.length > 0 && modelingObj) {
      modelingObj.removeElements(selection);
    }
  }, []);

  /**
   * 供父组件调用：修改当前 BPMN 元素的属性。
   * 通过 modeling.updateProperties 走命令栈，会触发 commandStack.changed，
   * 进而通过 saveXML 把最新的 XML 推回父组件。
   */
  const updateElementProperties = useCallback(
    (elementId: string, properties: Record<string, unknown>) => {
      if (!modelerRef.current) {
        console.warn('Modeler not ready');
        return false;
      }
      try {
        const modeling = modelerRef.current.get('modeling') as
          | { updateProperties: (el: any, props: Record<string, unknown>) => void }
          | undefined;
        const elementRegistry = modelerRef.current.get('elementRegistry') as {
          get: (id: string) => any;
        };
        const element = elementRegistry.get(elementId);
        if (!element || !modeling) {
          console.warn('Element or modeling not found:', elementId);
          return false;
        }
        modeling.updateProperties(element, properties);
        return true;
      } catch (err) {
        console.error('Failed to update element properties:', err);
        return false;
      }
    },
    []
  );

  /**
   * 供父组件调用：触发 fit-viewport（节点改变或初始加载后可用）
   */
  const fitViewport = useCallback(() => {
    if (!modelerRef.current) return;
    const canvas = modelerRef.current.get('canvas') as
      | { zoom: (level: string) => void }
      | undefined;
    canvas?.zoom('fit-viewport');
  }, []);

  // 暴露命令式 API 给父组件
  useEffect(() => {
    if (apiRef) {
      apiRef.current = { updateElementProperties, fitViewport };
    }
  }, [apiRef, updateElementProperties, fitViewport]);

  return (
    <div style={{ display: 'flex', height, border: '1px solid #d9d9d9', borderRadius: '6px' }}>
      {/* 工具栏 */}
      <div
        style={{
          width: 48,
          borderRight: '1px solid #d9d9d9',
          background: '#f5f5f5',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          padding: '8px 0',
          gap: 4,
        }}
      >
        <Tooltip title="保存" placement="right">
          <Button type="text" icon={<Save size={18} />} onClick={handleSave} disabled={readOnly} />
        </Tooltip>
        <Tooltip title="部署" placement="right">
          <Button
            type="text"
            icon={<PlayCircle size={18} />}
            onClick={handleDeploy}
            disabled={readOnly}
          />
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
          <input
            type="file"
            accept=".bpmn,.xml"
            style={{ display: 'none' }}
            onChange={handleImportXML}
          />
          <Tooltip title="导入BPMN" placement="right">
            <Button type="text" icon={<Upload size={18} />} />
          </Tooltip>
        </label>
      </div>

      {/* BPMN 图 */}
      <div ref={containerRef} style={{ flex: 1, position: 'relative' }} />

      {/* 缩放控制 */}
      <div
        style={{
          position: 'absolute',
          bottom: 16,
          right: 16,
          display: 'flex',
          gap: 4,
          background: 'white',
          padding: 4,
          borderRadius: 6,
          boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
        }}
      >
        <Tooltip title="缩小">
          <Button
            type="text"
            size="small"
            icon={<ZoomOut size={16} />}
            onClick={() => handleZoom(-0.1)}
          />
        </Tooltip>
        <span style={{ minWidth: 50, textAlign: 'center', lineHeight: '28px' }}>
          {Math.round(zoom * 100)}%
        </span>
        <Tooltip title="放大">
          <Button
            type="text"
            size="small"
            icon={<ZoomIn size={16} />}
            onClick={() => handleZoom(0.1)}
          />
        </Tooltip>
        <Tooltip title="适应屏幕">
          <Button
            type="text"
            size="small"
            icon={<Maximize size={16} />}
            onClick={handleZoomReset}
          />
        </Tooltip>
      </div>
    </div>
  );
};

export default BPMNDesigner;
