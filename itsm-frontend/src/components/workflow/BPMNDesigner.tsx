'use client';

import React, { useEffect, useRef, useState, useCallback, useMemo } from 'react';
import { Button, Tooltip, App, Input, Space, Dropdown, MenuProps } from 'antd';
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
  Search,
  AlignLeft,
  AlignCenter,
  AlignRight,
  AlignTop,
  AlignMiddle,
  AlignBottom,
  DistributeHorizontal,
  DistributeVertical,
  Grid,
  Copy,
  Paste,
  SelectAll,
  Settings,
  Bug
} from 'lucide-react';
import BpmnModeler from 'bpmn-js/lib/Modeler';
import gridModule from 'diagram-js/lib/features/grid-snapping';


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
  getXML: () => Promise<string | null>;
  validate: () => Promise<any[]>;
  selectElement: (elementId: string) => void;
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

// 历史记录项
interface HistoryItem {
  xml: string;
  timestamp: number;
  description?: string;
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
  const [history, setHistory] = useState<HistoryItem[]>([{ xml, timestamp: Date.now(), description: '初始' }]);
  const [historyIndex, setHistoryIndex] = useState(0);
  const [showGrid, setShowGrid] = useState(true);
  const [snapToGrid, setSnapToGrid] = useState(true);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [selectedElements, setSelectedElements] = useState<string[]>([]);
  const [isDragging, setIsDragging] = useState(false);

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
      const additionalModules = [
        gridModule,
        alignToOriginModule
      ];

      const modeler = new BpmnModeler({
        container: containerRef.current,
        additionalModules,
        grid: {
          size: 10,
          visible: showGrid
        },
        alignToOrigin: {
          enabled: true
        },
        keyboard: {
          bindTo: document
        }
      });

      modelerRef.current = modeler;

      // 加载初始 XML
      if (xml) {
        modeler
          .importXML(xml)
          .then(() => {
            const canvas = modeler.get('canvas') as
              | { zoom: (level: string | number) => void; getZoom: () => number }
              | undefined;
            canvas?.zoom('fit-viewport');
            setZoom(canvas?.getZoom() || 1);
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
              | { zoom: (level: string) => void; getZoom: () => number }
              | undefined;
            canvas?.zoom('fit-viewport');
            setZoom(canvas?.getZoom() || 1);
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

            // 更新历史记录，避免连续重复添加
            setHistory(prev => {
              // 如果当前不是在最后位置，先截断后面的历史
              const newHistory = prev.slice(0, historyIndex + 1);
              
              // 避免完全相同的内容重复添加
              const lastItem = newHistory[newHistory.length - 1];
              if (lastItem && lastItem.xml === result.xml) {
                return prev;
              }

              // 添加新的历史项，最多保留50步
              newHistory.push({ 
                xml: result.xml, 
                timestamp: Date.now(),
                description: '修改流程'
              });
              return newHistory.slice(-50);
            });
            setHistoryIndex(prev => Math.min(prev + 1, history.length));
          }
        });
      });

      // 监听选中节点变化，向父组件传递当前节点
      if (onSelectionChange) {
        const selection = modeler.get('selection') as {
          on: (event: string, handler: (event: any) => void) => void;
          get: () => any[];
          select: (elements: any[]) => void;
        };
        const elementRegistry = modeler.get('elementRegistry') as {
          get: (id: string) => any;
          filter: (fn: (element: any) => boolean) => any[];
        };

        const notifySelection = () => {
          const elements = selection.get();
          setSelectedElements(elements.map(e => e.id));
          
          if (elements.length === 0) {
            onSelectionChange(null);
            return;
          }
          
          // 只返回第一个选中的元素给父组件（保持向后兼容）
          const el = elements[0];
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

      // 监听拖拽事件
      modeler.on('drag.start', () => setIsDragging(true));
      modeler.on('drag.end', () => setIsDragging(false));

      // 设置网格吸附
      const gridSnapping = modeler.get('gridSnapping') as {
        setActive: (active: boolean) => void;
      };
      if (gridSnapping) {
        gridSnapping.setActive(snapToGrid);
      }

    } catch (err) {
      console.error('Failed to initialize BPMN Modeler:', err);
      initAttemptedRef.current = false;
    }
  }, [xml, historyIndex, message, onChange, onSelectionChange, showGrid, snapToGrid]);

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
            | { zoom: (level: string) => void; getZoom: () => number }
            | undefined;
          canvas?.zoom('fit-viewport');
          setZoom(canvas?.getZoom() || 1);
          setCurrentXML(xml);
          // 重置历史记录
          setHistory([{ xml, timestamp: Date.now(), description: '加载流程' }]);
          setHistoryIndex(0);
        })
        .catch((err: Error) => {
          console.error('Failed to import XML:', err);
        });
    }
  }, [xml]);

  // 键盘快捷键处理
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (readOnly) return;
      
      // Ctrl/Cmd + S: 保存
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault();
        handleSave();
      }
      
      // Ctrl/Cmd + Z: 撤销
      if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey) {
        e.preventDefault();
        handleUndo();
      }
      
      // Ctrl/Cmd + Y / Ctrl/Cmd + Shift + Z: 重做
      if (((e.ctrlKey || e.metaKey) && e.key === 'y') || ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === 'Z')) {
        e.preventDefault();
        handleRedo();
      }
      
      // Delete / Backspace: 删除选中元素
      if ((e.key === 'Delete' || e.key === 'Backspace') && !['INPUT', 'TEXTAREA'].includes((e.target as HTMLElement).tagName)) {
        e.preventDefault();
        handleDelete();
      }
      
      // Ctrl/Cmd + A: 全选
      if ((e.ctrlKey || e.metaKey) && e.key === 'a') {
        e.preventDefault();
        handleSelectAll();
      }
      
      // Ctrl/Cmd + C: 复制
      if ((e.ctrlKey || e.metaKey) && e.key === 'c') {
        e.preventDefault();
        handleCopy();
      }
      
      // Ctrl/Cmd + V: 粘贴
      if ((e.ctrlKey || e.metaKey) && e.key === 'v') {
        e.preventDefault();
        handlePaste();
      }
      
      // Ctrl/Cmd + F: 搜索
      if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
        e.preventDefault();
        // 聚焦搜索框
        const searchInput = document.getElementById('bpmn-search-input');
        if (searchInput) {
          searchInput.focus();
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [readOnly, historyIndex, history]);

  // 保存
  const handleSave = useCallback(() => {
    if (onSave) {
      onSave(currentXML);
      message.success('流程已保存');
    }
  }, [currentXML, onSave, message]);

  // 部署
  const handleDeploy = useCallback(() => {
    if (onDeploy) {
      onDeploy(currentXML);
      message.success('流程已部署');
    }
  }, [currentXML, onDeploy, message]);

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
  }, [message]);

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
  }, [currentXML, message]);

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
          // 更新历史记录
          setHistory([{ xml: content, timestamp: Date.now(), description: '导入流程' }]);
          setHistoryIndex(0);
        })
        .catch((err: Error) => {
          message.error('导入失败: ' + err.message);
        });
    };
    reader.readAsText(file);
  }, [message]);

  // 缩放
  const handleZoom = useCallback(
    (delta: number) => {
      if (!modelerRef.current) return;
      const canvas = modelerRef.current.get('canvas') as
        | { zoom: (level: number) => void; getZoom: () => number }
        | undefined;
      if (canvas) {
        const newZoom = Math.min(Math.max(canvas.getZoom() + delta, 0.2), 4);
        canvas.zoom(newZoom);
        setZoom(newZoom);
      }
    },
    []
  );

  // 重置缩放
  const handleZoomReset = useCallback(() => {
    if (!modelerRef.current) return;
    const canvas = modelerRef.current.get('canvas') as
      | { zoom: (level: string) => void; getZoom: () => number }
      | undefined;
    if (canvas) {
      canvas.zoom('fit-viewport');
      setZoom(canvas.getZoom() || 1);
    }
  }, []);

  // 撤销
  const handleUndo = useCallback(() => {
    if (!modelerRef.current || historyIndex <= 0) return;
    const newIndex = historyIndex - 1;
    const historyItem = history[newIndex];
    if (historyItem) {
      modelerRef.current.importXML(historyItem.xml)
        .then(() => {
          setHistoryIndex(newIndex);
          setCurrentXML(historyItem.xml);
          message.info('已撤销');
        })
        .catch(err => {
          console.error('撤销失败:', err);
          message.error('撤销失败');
        });
    }
  }, [history, historyIndex, message]);

  // 重做
  const handleRedo = useCallback(() => {
    if (!modelerRef.current || historyIndex >= history.length - 1) return;
    const newIndex = historyIndex + 1;
    const historyItem = history[newIndex];
    if (historyItem) {
      modelerRef.current.importXML(historyItem.xml)
        .then(() => {
          setHistoryIndex(newIndex);
          setCurrentXML(historyItem.xml);
          message.info('已重做');
        })
        .catch(err => {
          console.error('重做失败:', err);
          message.error('重做失败');
        });
    }
  }, [history, historyIndex, message]);

  // 删除选中
  const handleDelete = useCallback(() => {
    if (!modelerRef.current || readOnly) return;
    const selectionObj = modelerRef.current.get('selection') as { get: () => any[] } | undefined;
    const modelingObj = modelerRef.current.get('modeling') as
      | { removeElements: (elements: any[]) => void }
      | undefined;
    const selection = selectionObj?.get?.() || [];
    if (selection.length > 0 && modelingObj) {
      modelingObj.removeElements(selection);
      message.success(`已删除 ${selection.length} 个元素`);
    }
  }, [readOnly, message]);

  // 全选
  const handleSelectAll = useCallback(() => {
    if (!modelerRef.current) return;
    const elementRegistry = modelerRef.current.get('elementRegistry') as {
      filter: (fn: (element: any) => boolean) => any[];
    };
    const selection = modelerRef.current.get('selection') as {
      select: (elements: any[]) => void;
    };
    
    // 选择所有可见元素
    const allElements = elementRegistry.filter(element => !element.hidden && element.type !== 'label');
    selection.select(allElements);
    message.success(`已选中 ${allElements.length} 个元素`);
  }, [message]);

  // 复制
  const handleCopy = useCallback(() => {
    if (!modelerRef.current || selectedElements.length === 0 || readOnly) return;
    const copyPaste = modelerRef.current.get('copyPaste') as {
      copy: (elements: any[]) => void;
    };
    const elementRegistry = modelerRef.current.get('elementRegistry') as {
      get: (id: string) => any;
    };
    
    const elements = selectedElements.map(id => elementRegistry.get(id)).filter(Boolean);
    if (elements.length > 0) {
      copyPaste.copy(elements);
      message.success(`已复制 ${elements.length} 个元素`);
    }
  }, [selectedElements, readOnly, message]);

  // 粘贴
  const handlePaste = useCallback(() => {
    if (!modelerRef.current || readOnly) return;
    const copyPaste = modelerRef.current.get('copyPaste') as {
      paste: (options?: { point?: { x: number; y: number } }) => any[];
      isClipboardEmpty: () => boolean;
    };
    
    if (copyPaste.isClipboardEmpty()) {
      message.warning('剪贴板为空');
      return;
    }
    
    const pastedElements = copyPaste.paste();
    if (pastedElements.length > 0) {
      message.success(`已粘贴 ${pastedElements.length} 个元素`);
    }
  }, [readOnly, message]);

  // 对齐操作
  const handleAlign = useCallback((type: 'left' | 'center' | 'right' | 'top' | 'middle' | 'bottom') => {
    if (!modelerRef.current || selectedElements.length < 2 || readOnly) return;
    const alignElements = modelerRef.current.get('alignElements') as {
      trigger: (type: string) => void;
    };
    
    try {
      alignElements.trigger(type);
      message.success(`已${type === 'left' ? '左对齐' :
        type === 'center' ? '水平居中对齐' :
        type === 'right' ? '右对齐' :
        type === 'top' ? '顶部对齐' :
        type === 'middle' ? '垂直居中对齐' :
        '底部对齐'}`);
    } catch (err) {
      console.error('对齐失败:', err);
      message.error('对齐操作失败');
    }
  }, [selectedElements.length, readOnly, message]);

  // 分布操作
  const handleDistribute = useCallback((direction: 'horizontal' | 'vertical') => {
    if (!modelerRef.current || selectedElements.length < 3 || readOnly) return;
    const distributeElements = modelerRef.current.get('distributeElements') as {
      trigger: (type: string) => void;
    };
    
    try {
      distributeElements.trigger(direction);
      message.success(`已${direction === 'horizontal' ? '水平' : '垂直'}分布元素`);
    } catch (err) {
      console.error('分布失败:', err);
      message.error('分布操作失败');
    }
  }, [selectedElements.length, readOnly, message]);

  // 切换网格显示
  const toggleGrid = useCallback(() => {
    if (!modelerRef.current) return;
    const newShowGrid = !showGrid;
    setShowGrid(newShowGrid);
    
    const grid = modelerRef.current.get('grid') as {
      toggle: () => void;
    };
    
    if (grid) {
      grid.toggle();
    }
  }, [showGrid]);

  // 切换网格吸附
  const toggleSnapToGrid = useCallback(() => {
    if (!modelerRef.current) return;
    const newSnapToGrid = !snapToGrid;
    setSnapToGrid(newSnapToGrid);
    
    const gridSnapping = modelerRef.current.get('gridSnapping') as {
      setActive: (active: boolean) => void;
    };
    
    if (gridSnapping) {
      gridSnapping.setActive(newSnapToGrid);
    }
    
    message.success(`网格吸附已${newSnapToGrid ? '开启' : '关闭'}`);
  }, [snapToGrid, message]);

  // 搜索节点
  const handleSearch = useCallback((value: string) => {
    setSearchKeyword(value);
    if (!modelerRef.current || !value) return;
    
    const elementRegistry = modelerRef.current.get('elementRegistry') as {
      filter: (fn: (element: any) => boolean) => any[];
    };
    const selection = modelerRef.current.get('selection') as {
      select: (elements: any[]) => void;
      focus: (element: any) => void;
    };
    const canvas = modelerRef.current.get('canvas') as {
      zoom: (level: number, point?: { x: number; y: number }) => void;
    };
    
    // 搜索名称或ID包含关键词的元素
    const keyword = value.toLowerCase();
    const matchedElements = elementRegistry.filter(element => {
      const name = (element.businessObject?.name || '').toLowerCase();
      const id = element.id.toLowerCase();
      const type = (element.type || '').toLowerCase();
      return name.includes(keyword) || id.includes(keyword) || type.includes(keyword);
    });
    
    if (matchedElements.length > 0) {
      selection.select(matchedElements);
      // 缩放并聚焦到第一个匹配的元素
      const firstElement = matchedElements[0];
      const center = {
        x: firstElement.x + firstElement.width / 2,
        y: firstElement.y + firstElement.height / 2
      };
      canvas.zoom(1.5, center);
      message.success(`找到 ${matchedElements.length} 个匹配元素`);
    } else {
      message.info('未找到匹配元素');
    }
  }, [message]);

  // 验证流程
  const handleValidate = useCallback(async () => {
    if (!modelerRef.current) return [];
    const elementRegistry = modelerRef.current.get('elementRegistry') as {
      filter: (fn: (element: any) => boolean) => any[];
    };
    
    const errors: any[] = [];
    
    // 检查流程是否有开始和结束事件
    const startEvents = elementRegistry.filter(el => el.type === 'bpmn:StartEvent');
    const endEvents = elementRegistry.filter(el => el.type === 'bpmn:EndEvent');
    
    if (startEvents.length === 0) {
      errors.push({ type: 'error', message: '流程缺少开始事件' });
    }
    
    if (endEvents.length === 0) {
      errors.push({ type: 'error', message: '流程缺少结束事件' });
    }
    
    // 检查用户任务是否有配置
    const userTasks = elementRegistry.filter(el => el.type === 'bpmn:UserTask');
    userTasks.forEach(task => {
      const bo = task.businessObject;
      if (!bo.assignee && !bo.candidateUsers && !bo.candidateGroups) {
        errors.push({ 
          type: 'warning', 
          message: `用户任务 "${bo.name || task.id}" 未配置受理人或候选人` 
        });
      }
    });
    
    // 检查服务任务是否有配置
    const serviceTasks = elementRegistry.filter(el => el.type === 'bpmn:ServiceTask');
    serviceTasks.forEach(task => {
      const bo = task.businessObject;
      if (!bo.implementation && !bo.operationRef) {
        errors.push({ 
          type: 'warning', 
          message: `服务任务 "${bo.name || task.id}" 未配置实现类型或操作引用` 
        });
      }
    });
    
    // 检查网关是否有默认分支和条件
    const gateways = elementRegistry.filter(el => el.type.includes('Gateway'));
    gateways.forEach(gateway => {
      const bo = gateway.businessObject;
      const outgoing = gateway.outgoing || [];
      
      if (bo.$type === 'bpmn:ExclusiveGateway' && outgoing.length > 1 && !bo.default) {
        errors.push({ 
          type: 'warning', 
          message: `排他网关 "${bo.name || gateway.id}" 有多个输出流但未配置默认分支` 
        });
      }
      
      // 检查输出流是否有条件
      outgoing.forEach((flow: any) => {
        if (!flow.conditionExpression && outgoing.length > 1) {
          errors.push({ 
            type: 'warning', 
            message: `网关 "${bo.name || gateway.id}" 的输出流 "${flow.id}" 未配置条件表达式` 
          });
        }
      });
    });
    
    // 显示验证结果
    if (errors.length === 0) {
      message.success('流程验证通过，未发现问题');
    } else {
      const errorCount = errors.filter(e => e.type === 'error').length;
      const warningCount = errors.filter(e => e.type === 'warning').length;
      message.warning(`验证完成，发现 ${errorCount} 个错误，${warningCount} 个警告`);
      console.log('验证结果:', errors);
    }
    
    return errors;
  }, [message]);

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

  /**
   * 供父组件调用：获取当前XML
   */
  const getXML = useCallback(async () => {
    if (!modelerRef.current) return null;
    try {
      const result = await modelerRef.current.saveXML({ format: true });
      return result.xml || null;
    } catch (err) {
      console.error('Failed to get XML:', err);
      return null;
    }
  }, []);

  /**
   * 供父组件调用：验证流程
   */
  const validate = useCallback(async () => {
    return handleValidate();
  }, [handleValidate]);

  /**
   * 供父组件调用：选中指定元素
   */
  const selectElement = useCallback((elementId: string) => {
    if (!modelerRef.current) return;
    const elementRegistry = modelerRef.current.get('elementRegistry') as {
      get: (id: string) => any;
    };
    const selection = modelerRef.current.get('selection') as {
      select: (elements: any[]) => void;
    };
    
    const element = elementRegistry.get(elementId);
    if (element) {
      selection.select([element]);
    }
  }, []);

  // 暴露命令式 API 给父组件
  useEffect(() => {
    if (apiRef) {
      apiRef.current = { updateElementProperties, fitViewport, getXML, validate, selectElement };
    }
  }, [apiRef, updateElementProperties, fitViewport, getXML, validate, selectElement]);

  // 对齐菜单
  const alignMenuItems: MenuProps['items'] = [
    {
      key: 'left',
      icon: <AlignLeft size={14} />,
      label: '左对齐',
      onClick: () => handleAlign('left'),
      disabled: selectedElements.length < 2 || readOnly
    },
    {
      key: 'center',
      icon: <AlignCenter size={14} />,
      label: '水平居中',
      onClick: () => handleAlign('center'),
      disabled: selectedElements.length < 2 || readOnly
    },
    {
      key: 'right',
      icon: <AlignRight size={14} />,
      label: '右对齐',
      onClick: () => handleAlign('right'),
      disabled: selectedElements.length < 2 || readOnly
    },
    {
      type: 'divider'
    },
    {
      key: 'top',
      icon: <AlignTop size={14} />,
      label: '顶部对齐',
      onClick: () => handleAlign('top'),
      disabled: selectedElements.length < 2 || readOnly
    },
    {
      key: 'middle',
      icon: <AlignMiddle size={14} />,
      label: '垂直居中',
      onClick: () => handleAlign('middle'),
      disabled: selectedElements.length < 2 || readOnly
    },
    {
      key: 'bottom',
      icon: <AlignBottom size={14} />,
      label: '底部对齐',
      onClick: () => handleAlign('bottom'),
      disabled: selectedElements.length < 2 || readOnly
    }
  ];

  // 分布菜单
  const distributeMenuItems: MenuProps['items'] = [
    {
      key: 'horizontal',
      icon: <DistributeHorizontal size={14} />,
      label: '水平分布',
      onClick: () => handleDistribute('horizontal'),
      disabled: selectedElements.length < 3 || readOnly
    },
    {
      key: 'vertical',
      icon: <DistributeVertical size={14} />,
      label: '垂直分布',
      onClick: () => handleDistribute('vertical'),
      disabled: selectedElements.length < 3 || readOnly
    }
  ];

  // 设置菜单
  const settingsMenuItems: MenuProps['items'] = [
    {
      key: 'grid',
      icon: <Grid size={14} />,
      label: showGrid ? '隐藏网格' : '显示网格',
      onClick: toggleGrid
    },
    {
      key: 'snap',
      icon: <Grid size={14} />,
      label: snapToGrid ? '关闭网格吸附' : '开启网格吸附',
      onClick: toggleSnapToGrid
    },
    {
      type: 'divider'
    },
    {
      key: 'validate',
      icon: <Bug size={14} />,
      label: '验证流程',
      onClick: handleValidate
    }
  ];

  return (
    <div style={{ display: 'flex', height, border: '1px solid #d9d9d9', borderRadius: '6px', position: 'relative' }}>
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
        <Tooltip title="保存 (Ctrl+S)" placement="right">
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
        <Tooltip title="撤销 (Ctrl+Z)" placement="right">
          <Button
            type="text"
            icon={<Undo size={18} />}
            onClick={handleUndo}
            disabled={readOnly || historyIndex <= 0}
          />
        </Tooltip>
        <Tooltip title="重做 (Ctrl+Y)" placement="right">
          <Button
            type="text"
            icon={<Redo size={18} />}
            onClick={handleRedo}
            disabled={readOnly || historyIndex >= history.length - 1}
          />
        </Tooltip>

        <div style={{ height: 1, width: '80%', background: '#e8e8e8', margin: '8px 0' }} />

        <Tooltip title="复制 (Ctrl+C)" placement="right">
          <Button
            type="text"
            icon={<Copy size={18} />}
            onClick={handleCopy}
            disabled={readOnly || selectedElements.length === 0}
          />
        </Tooltip>
        <Tooltip title="粘贴 (Ctrl+V)" placement="right">
          <Button
            type="text"
            icon={<Paste size={18} />}
            onClick={handlePaste}
            disabled={readOnly}
          />
        </Tooltip>
        <Tooltip title="全选 (Ctrl+A)" placement="right">
          <Button
            type="text"
            icon={<SelectAll size={18} />}
            onClick={handleSelectAll}
            disabled={readOnly}
          />
        </Tooltip>
        <Tooltip title="删除 (Delete)" placement="right">
          <Button
            type="text"
            icon={<Trash2 size={18} />}
            onClick={handleDelete}
            disabled={readOnly || selectedElements.length === 0}
            danger
          />
        </Tooltip>

        <div style={{ height: 1, width: '80%', background: '#e8e8e8', margin: '8px 0' }} />

        <Dropdown menu={{ items: alignMenuItems }} placement="right" trigger={['click']}>
          <Tooltip title="对齐" placement="right">
            <Button type="text" icon={<AlignLeft size={18} />} disabled={selectedElements.length < 2 || readOnly} />
          </Tooltip>
        </Dropdown>

        <Dropdown menu={{ items: distributeMenuItems }} placement="right" trigger={['click']}>
          <Tooltip title="分布" placement="right">
            <Button type="text" icon={<DistributeHorizontal size={18} />} disabled={selectedElements.length < 3 || readOnly} />
          </Tooltip>
        </Dropdown>

        <div style={{ flex: 1 }} />

        <Dropdown menu={{ items: settingsMenuItems }} placement="right" trigger={['click']}>
          <Tooltip title="设置" placement="right">
            <Button type="text" icon={<Settings size={18} />} />
          </Tooltip>
        </Dropdown>

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

      {/* 顶部搜索栏 */}
      <div style={{
        position: 'absolute',
        top: 16,
        left: '50%',
        transform: 'translateX(-50%)',
        width: 300,
        zIndex: 10,
        background: 'white',
        borderRadius: '6px',
        boxShadow: '0 2px 8px rgba(0,0,0,0.15)'
      }}>
        <Input
          id="bpmn-search-input"
          placeholder="搜索节点名称/ID/类型..."
          prefix={<Search size={14} />}
          value={searchKeyword}
          onChange={e => handleSearch(e.target.value)}
          allowClear
          size="small"
        />
      </div>

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
        <span style={{ minWidth: 50, textAlign: 'center', lineHeight: '28px', fontSize: '12px' }}>
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

      {/* 选中信息 */}
      {selectedElements.length > 0 && (
        <div style={{
          position: 'absolute',
          bottom: 16,
          left: 16,
          background: 'rgba(0, 0, 0, 0.6)',
          color: 'white',
          padding: '4px 12px',
          borderRadius: '4px',
          fontSize: '12px'
        }}>
          已选中 {selectedElements.length} 个元素
        </div>
      )}
    </div>
  );
};

export default BPMNDesigner;
