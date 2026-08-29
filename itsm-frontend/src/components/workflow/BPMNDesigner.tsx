'use client';

import React, { useEffect, useRef, useState, useCallback } from 'react';
import type { MenuProps } from 'antd';
import { Button, Tooltip, App, Input, AutoComplete, Dropdown } from 'antd';
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
  PanelTop,
  Rows2,
  PanelBottom,
  AlignHorizontalDistributeCenter,
  AlignVerticalDistributeCenter,
  Grid,
  Copy,
  ClipboardPaste,
  ListChecks,
  Settings,
  Bug
} from 'lucide-react';
import BpmnModeler from 'bpmn-js/lib/Modeler';
import itsmModdleDescriptor from './itsm-moddle-descriptor';
import gridModule from 'diagram-js/lib/features/grid-snapping';
import { useI18n } from '@/lib/i18n/useI18n';


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

// 搜索结果项
interface SearchMatch {
  id: string;
  name: string;
  type: string;
}

/** 判断事件目标是否为可编辑元素（输入框/文本域），快捷键不应劫持 */
const isEditableTarget = (target: EventTarget | null): boolean => {
  const el = target as HTMLElement | null;
  if (!el) return false;
  return el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.isContentEditable;
};

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
  const { t } = useI18n();
  const containerRef = useRef<HTMLDivElement>(null);
  const modelerRef = useRef<BpmnModeler | null>(null);
  const initAttemptedRef = useRef(false);
  const [currentXML, setCurrentXML] = useState(xml);
  const [zoom, setZoom] = useState(1);
  const [canUndo, setCanUndo] = useState(false);
  const [canRedo, setCanRedo] = useState(false);
  const [showGrid, setShowGrid] = useState(true);
  const [snapToGrid, setSnapToGrid] = useState(true);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [searchMatches, setSearchMatches] = useState<SearchMatch[]>([]);
  const [selectedElements, setSelectedElements] = useState<string[]>([]);

  // ---------------------------------------------------------------------------
  // refs：保持回调/配置的最新引用，保证 modeler 只初始化一次（不随编辑重建）
  // ---------------------------------------------------------------------------
  const onChangeRef = useRef(onChange);
  const onSaveRef = useRef(onSave);
  const onSelectionChangeRef = useRef(onSelectionChange);
  const messageRef = useRef(message);
  const tRef = useRef(t);
  const snapToGridRef = useRef(snapToGrid);
  const showGridRef = useRef(showGrid);
  const selectedElementsRef = useRef<string[]>([]);
  const currentXMLRef = useRef(currentXML);

  useEffect(() => {
    onChangeRef.current = onChange;
    onSaveRef.current = onSave;
    onSelectionChangeRef.current = onSelectionChange;
    messageRef.current = message;
    tRef.current = t;
  }, [onChange, onSave, onSelectionChange, message, t]);

  useEffect(() => {
    snapToGridRef.current = snapToGrid;
    showGridRef.current = showGrid;
    selectedElementsRef.current = selectedElements;
    currentXMLRef.current = currentXML;
  }, [snapToGrid, showGrid, selectedElements, currentXML]);

  // 防抖同步 XML：命令触发后延迟序列化，避免连续操作时反复 saveXML
  const xmlSyncTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // 最近一次通过 onChange 向外发出的 XML，用于识别父组件回传的"回环"更新
  const emittedXmlRef = useRef<string | null>(xml || null);
  // import 并发令牌：仅接受最新一次导入的结果
  const importTokenRef = useRef(0);

  /** 撤销/重做可用状态（来自 bpmn-js 原生命令栈） */
  const syncCommandStackState = useCallback(() => {
    const modeler = modelerRef.current;
    if (!modeler) return;
    try {
      const commandStack = modeler.get('commandStack') as {
        canUndo: () => boolean;
        canRedo: () => boolean;
      };
      setCanUndo(commandStack.canUndo());
      setCanRedo(commandStack.canRedo());
    } catch {
      // commandStack 不可用时保持现状
    }
  }, []);

  const scheduleXmlSync = useCallback((delay = 150) => {
    if (xmlSyncTimerRef.current) clearTimeout(xmlSyncTimerRef.current);
    xmlSyncTimerRef.current = setTimeout(async () => {
      const modeler = modelerRef.current;
      if (!modeler) return;
      try {
        const result = await modeler.saveXML({ format: true });
        if (result.xml) {
          setCurrentXML(result.xml);
          emittedXmlRef.current = result.xml;
          onChangeRef.current?.(result.xml);
        }
      } catch (err) {
        console.error('Failed to serialize BPMN XML:', err);
      }
    }, delay);
  }, []);

  // 等待容器具有有效尺寸后再初始化 BPMN Modeler
  // bpmn-js 在容器尺寸为 0 时会抛出 "Cannot read properties of undefined (reading 'root-0')"
  const initializeModeler = useCallback(() => {
    if (!containerRef.current || modelerRef.current || initAttemptedRef.current) {
      return;
    }

    const rect = containerRef.current.getBoundingClientRect();
    if (rect.width <= 0 || rect.height <= 0) {
      // 容器尚未布局完成，由外部重试逻辑处理
      return;
    }

    try {
      const additionalModules = [gridModule];

      const modeler = new BpmnModeler({
        container: containerRef.current,
        additionalModules,
        moddleExtensions: { itsm: itsmModdleDescriptor },
        grid: {
          size: 10,
          visible: showGridRef.current,
        },
      });

      modelerRef.current = modeler;
      initAttemptedRef.current = true;

      // 加载初始 XML（仅挂载时使用当时的 prop，后续变化由 [xml] effect 处理）
      const importInitial = async () => {
        try {
          if (xml) {
            await modeler.importXML(xml);
          } else {
            await modeler.createDiagram();
          }
        } catch (err) {
          console.error('Failed to import initial XML:', err);
          messageRef.current.error(tRef.current('bpmnDesigner.messages.loadFailed'));
          return;
        }
        // 避免 React StrictMode 卸载后渲染丢失
        if (modelerRef.current !== modeler) return;
        try {
          const canvas = modeler.get('canvas') as { zoom: (level?: string | number) => number };
          canvas.zoom('fit-viewport');
        } catch (renderErr) {
          console.error('Error after XML import:', renderErr);
        }
      };
      void importInitial();

      // 命令栈变化：同步撤销/重做状态 + 防抖序列化 XML 推给父组件
      modeler.on('commandStack.changed', () => {
        syncCommandStackState();
        scheduleXmlSync();
      });

      // 视口变化（滚轮缩放、fit-viewport 等）时同步缩放比例显示
      modeler.on('canvas.viewbox.changed', (event: { viewbox?: { scale?: number } }) => {
        const scale = event?.viewbox?.scale;
        if (typeof scale === 'number' && scale > 0) {
          setZoom(scale);
        }
      });

      // 选中变化：更新本地选中集合，并通知父组件（驱动节点属性面板）
      modeler.on('selection.changed', () => {
        const selection = modeler.get('selection') as { get: () => any[] };
        const elements = selection.get() || [];
        setSelectedElements(elements.map(e => e.id));

        const cb = onSelectionChangeRef.current;
        if (!cb) return;
        if (elements.length === 0) {
          cb(null);
          return;
        }
        const el = elements[0];
        if (!el) {
          cb(null);
          return;
        }
        const nodeSelection: BpmnNodeSelection = {
          id: el.id,
          type: el.type || (el.businessObject && el.businessObject.$type) || 'unknown',
          businessObject: el.businessObject || {},
        };
        if (el.businessObject?.name) {
          nodeSelection.name = el.businessObject.name;
        }
        cb(nodeSelection);
      });

      // 初始网格吸附状态
      try {
        const gridSnapping = modeler.get('gridSnapping') as
          | { setActive?: (active: boolean) => void }
          | undefined;
        gridSnapping?.setActive?.(snapToGridRef.current);
      } catch {
        // gridSnapping 服务不可用时忽略
      }
    } catch (err) {
      console.error('Failed to initialize BPMN Modeler:', err);
      initAttemptedRef.current = false;
    }
  }, [syncCommandStackState, scheduleXmlSync]);

  // 初始化 BPMN Modeler - 等待容器布局完成（initializeModeler 引用稳定，effect 仅在挂载时执行）
  useEffect(() => {
    if (!containerRef.current) return;

    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let retryCount = 0;
    const MAX_RETRIES = 30;
    const RETRY_DELAY = 100;

    const tryInit = () => {
      if (modelerRef.current) return;
      if (!containerRef.current) return;
      const rect = containerRef.current.getBoundingClientRect();
      // 仅当容器显示状态为可见时才初始化（避免被其他面板遮挡时启动）
      const isVisible = rect.width > 100 && rect.height > 100;
      if (isVisible) {
        initializeModeler();
      } else if (retryCount < MAX_RETRIES) {
        retryCount++;
        retryTimer = setTimeout(tryInit, RETRY_DELAY);
      } else {
        // 最后一次重试时强制尝试初始化
        initializeModeler();
      }
    };

    // 使用 setTimeout 确保 DOM 已完成布局
    retryTimer = setTimeout(tryInit, 200);

    return () => {
      if (retryTimer !== null) {
        clearTimeout(retryTimer);
      }
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

  // 同步 xml prop 变化（版本切换、模板选择、AI 生成等外部来源）
  // 通过 emittedXmlRef 识别"编辑回环"：父组件把 onChange 发出的 XML 原样传回时不重新导入，
  // 否则每次编辑都会触发 importXML 导致视口重置、选中丢失、命令栈被清空。
  const lastPropXmlRef = useRef(xml);
  useEffect(() => {
    if (xml === lastPropXmlRef.current) return;
    lastPropXmlRef.current = xml;
    if (xml && xml === emittedXmlRef.current) return;

    const modeler = modelerRef.current;
    if (!modeler || !xml) return;

    const token = ++importTokenRef.current;
    modeler
      .importXML(xml)
      .then(() => {
        if (token !== importTokenRef.current || modelerRef.current !== modeler) return;
        try {
          const canvas = modeler.get('canvas') as { zoom: (level?: string) => number };
          canvas.zoom('fit-viewport');
        } catch (err) {
          console.error('Failed to fit viewport after import:', err);
        }
        setCurrentXML(xml);
        // 导入新流程会重置命令栈
        setCanUndo(false);
        setCanRedo(false);
      })
      .catch((err: Error) => {
        if (token !== importTokenRef.current) return;
        console.error('Failed to import XML:', err);
        message.error(t('bpmnDesigner.messages.importFailed') + ': ' + err.message);
      });
  }, [xml, t]);

  // 卸载时清理防抖定时器
  useEffect(() => {
    return () => {
      if (xmlSyncTimerRef.current) clearTimeout(xmlSyncTimerRef.current);
    };
  }, []);

  // 保存：序列化最新 XML 后交给父组件（由父组件统一反馈保存结果）
  const handleSave = useCallback(async () => {
    if (readOnly) return;
    const modeler = modelerRef.current;
    let xmlToSave = currentXMLRef.current;
    if (modeler) {
      try {
        const result = await modeler.saveXML({ format: true });
        if (result.xml) {
          xmlToSave = result.xml;
          setCurrentXML(result.xml);
          emittedXmlRef.current = result.xml;
        }
      } catch (err) {
        console.error('Failed to save XML:', err);
      }
    }
    if (xmlToSave) {
      onSaveRef.current?.(xmlToSave);
    }
  }, [readOnly]);

  // 部署
  const handleDeploy = useCallback(() => {
    if (onDeploy && !readOnly) {
      onDeploy(currentXMLRef.current);
    }
  }, [onDeploy, readOnly]);

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
        message.success(t('bpmnDesigner.messages.exportSvg'));
      }
    } catch {
      message.error(t('bpmnDesigner.messages.exportFailed'));
    }
  }, [message, t]);

  // 导出XML
  const handleExportXML = useCallback(() => {
    const blob = new Blob([currentXMLRef.current], { type: 'application/xml' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'workflow.bpmn';
    link.click();
    URL.revokeObjectURL(url);
    message.success(t('bpmnDesigner.messages.exportBpmn'));
  }, [message, t]);

  // 导入XML
  const handleImportXML = useCallback(
    (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (!file) return;

      const reader = new FileReader();
      reader.onload = e => {
        const content = e.target?.result as string;
        modelerRef.current
          ?.importXML(content)
          .then(() => {
            message.success(t('bpmnDesigner.messages.importSuccess'));
            setCurrentXML(content);
            setCanUndo(false);
            setCanRedo(false);
            // 导入视为外部流程变更，同步给父组件
            emittedXmlRef.current = content;
            lastPropXmlRef.current = content;
            onChangeRef.current?.(content);
          })
          .catch((err: Error) => {
            message.error(t('bpmnDesigner.messages.importFailed') + ': ' + err.message);
          });
      };
      reader.readAsText(file);
      // 允许重复选择同一文件
      event.target.value = '';
    },
    [message, t]
  );

  // 缩放
  const handleZoom = useCallback((delta: number) => {
    if (!modelerRef.current) return;
    try {
      const canvas = modelerRef.current.get('canvas') as {
        zoom: (level: number) => void;
        viewbox: () => { scale: number };
      };
      const currentScale = canvas.viewbox().scale;
      const newZoom = Math.min(Math.max(currentScale + delta, 0.2), 4);
      canvas.zoom(newZoom);
    } catch (err) {
      console.error('Zoom failed:', err);
    }
  }, []);

  // 适应画布
  const handleZoomReset = useCallback(() => {
    if (!modelerRef.current) return;
    try {
      const canvas = modelerRef.current.get('canvas') as { zoom: (level: string) => void };
      canvas.zoom('fit-viewport');
    } catch (err) {
      console.error('Fit viewport failed:', err);
    }
  }, []);

  // 撤销/重做：直接使用 bpmn-js 原生命令栈，保证视口/选中和性能稳定
  const handleUndo = useCallback(() => {
    if (readOnly) return;
    const modeler = modelerRef.current;
    if (!modeler) return;
    try {
      const commandStack = modeler.get('commandStack') as { undo: () => void; canUndo: () => boolean };
      if (!commandStack.canUndo()) return;
      commandStack.undo();
    } catch (err) {
      console.error('Undo failed:', err);
    }
  }, [readOnly]);

  const handleRedo = useCallback(() => {
    if (readOnly) return;
    const modeler = modelerRef.current;
    if (!modeler) return;
    try {
      const commandStack = modeler.get('commandStack') as { redo: () => void; canRedo: () => boolean };
      if (!commandStack.canRedo()) return;
      commandStack.redo();
    } catch (err) {
      console.error('Redo failed:', err);
    }
  }, [readOnly]);

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
      message.success(t('bpmnDesigner.messages.elementsDeleted', { count: selection.length }));
    }
  }, [readOnly, message, t]);

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
    message.success(t('bpmnDesigner.messages.elementsSelected', { count: allElements.length }));
  }, [message, t]);

  // 复制
  const handleCopy = useCallback(() => {
    if (!modelerRef.current || readOnly) return;
    const ids = selectedElementsRef.current;
    if (ids.length === 0) return;
    try {
      const copyPaste = modelerRef.current.get('copyPaste') as {
        copy: (elements: any[]) => void;
      };
      const elementRegistry = modelerRef.current.get('elementRegistry') as {
        get: (id: string) => any;
      };

      const elements = ids.map(id => elementRegistry.get(id)).filter(Boolean);
      if (elements.length > 0) {
        copyPaste.copy(elements);
        message.success(t('bpmnDesigner.messages.elementsCopied', { count: elements.length }));
      }
    } catch (err) {
      console.error('Copy failed:', err);
    }
  }, [readOnly, message, t]);

  // 粘贴
  const handlePaste = useCallback(() => {
    if (!modelerRef.current || readOnly) return;
    try {
      const copyPaste = modelerRef.current.get('copyPaste') as {
        paste: (options?: { point?: { x: number; y: number } }) => any[];
        isClipboardEmpty: () => boolean;
      };

      if (copyPaste.isClipboardEmpty()) {
        message.warning(t('bpmnDesigner.messages.clipboardEmpty'));
        return;
      }

      const pastedElements = copyPaste.paste();
      if (pastedElements.length > 0) {
        message.success(t('bpmnDesigner.messages.elementsPasted', { count: pastedElements.length }));
      }
    } catch (err) {
      console.error('Paste failed:', err);
    }
  }, [readOnly, message, t]);

  // 对齐操作
  const handleAlign = useCallback(
    (type: 'left' | 'center' | 'right' | 'top' | 'middle' | 'bottom') => {
      if (!modelerRef.current || readOnly) return;
      if (selectedElementsRef.current.length < 2) return;
      const alignElements = modelerRef.current.get('alignElements') as {
        trigger: (type: string) => void;
      };

      try {
        alignElements.trigger(type);
        const alignMsg =
          type === 'left' ? t('bpmnDesigner.messages.alignLeftSuccess') :
          type === 'center' ? t('bpmnDesigner.messages.alignCenterSuccess') :
          type === 'right' ? t('bpmnDesigner.messages.alignRightSuccess') :
          type === 'top' ? t('bpmnDesigner.messages.alignTopSuccess') :
          type === 'middle' ? t('bpmnDesigner.messages.alignMiddleSuccess') :
          t('bpmnDesigner.messages.alignBottomSuccess');
        message.success(alignMsg);
      } catch (err) {
        console.error('Align failed:', err);
        message.error(t('bpmnDesigner.messages.alignFailed'));
      }
    },
    [readOnly, message, t]
  );

  // 分布操作
  const handleDistribute = useCallback(
    (direction: 'horizontal' | 'vertical') => {
      if (!modelerRef.current || readOnly) return;
      if (selectedElementsRef.current.length < 3) return;
      const distributeElements = modelerRef.current.get('distributeElements') as {
        trigger: (type: string) => void;
      };

      try {
        distributeElements.trigger(direction);
        message.success(
          direction === 'horizontal'
            ? t('bpmnDesigner.messages.distributeHorizontalSuccess')
            : t('bpmnDesigner.messages.distributeVerticalSuccess')
        );
      } catch (err) {
        console.error('Distribute failed:', err);
        message.error(t('bpmnDesigner.messages.distributeFailed'));
      }
    },
    [readOnly, message, t]
  );

  // 切换网格显示
  const toggleGrid = useCallback(() => {
    const newShowGrid = !showGridRef.current;
    setShowGrid(newShowGrid);
    showGridRef.current = newShowGrid;

    try {
      const grid = modelerRef.current?.get('grid') as { toggle?: () => void } | undefined;
      grid?.toggle?.();
    } catch (err) {
      console.warn('Grid toggle not supported:', err);
    }
  }, []);

  // 切换网格吸附
  const toggleSnapToGrid = useCallback(() => {
    const newSnapToGrid = !snapToGridRef.current;
    setSnapToGrid(newSnapToGrid);
    snapToGridRef.current = newSnapToGrid;

    try {
      const gridSnapping = modelerRef.current?.get('gridSnapping') as
        | { setActive?: (active: boolean) => void }
        | undefined;
      gridSnapping?.setActive?.(newSnapToGrid);
    } catch (err) {
      console.warn('Grid snapping toggle not supported:', err);
    }

    message.success(
      newSnapToGrid ? t('bpmnDesigner.messages.gridSnapEnabled') : t('bpmnDesigner.messages.gridSnapDisabled')
    );
  }, [message, t]);

  // 搜索：输入时仅过滤联想列表，选中/回车后再定位（避免每击键全选+缩放跳变）
  const handleSearchChange = useCallback((value: string) => {
    setSearchKeyword(value);
    const modeler = modelerRef.current;
    const keyword = value.trim().toLowerCase();
    if (!modeler || !keyword) {
      setSearchMatches([]);
      return;
    }

    try {
      const elementRegistry = modeler.get('elementRegistry') as {
        filter: (fn: (element: any) => boolean) => any[];
      };
      const matches = elementRegistry
        .filter(element => {
          if (element.hidden || element.type === 'label') return false;
          const name = (element.businessObject?.name || '').toLowerCase();
          const id = (element.id || '').toLowerCase();
          return name.includes(keyword) || id.includes(keyword);
        })
        .slice(0, 12)
        .map(element => ({
          id: element.id,
          name: element.businessObject?.name || element.id,
          type: (element.type || '').replace('bpmn:', ''),
        }));
      setSearchMatches(matches);
    } catch (err) {
      console.error('Search failed:', err);
    }
  }, []);

  // 定位元素：选中并平移视口居中（不改变缩放比例）
  const focusElement = useCallback((elementId: string) => {
    const modeler = modelerRef.current;
    if (!modeler) return;
    try {
      const elementRegistry = modeler.get('elementRegistry') as { get: (id: string) => any };
      const selection = modeler.get('selection') as { select: (elements: any[]) => void };
      const canvas = modeler.get('canvas') as { viewbox: (vb?: unknown) => { x: number; y: number; width: number; height: number } };

      const el = elementRegistry.get(elementId);
      if (!el) return;
      selection.select([el]);

      const viewbox = canvas.viewbox();
      canvas.viewbox({
        x: el.x + el.width / 2 - viewbox.width / 2,
        y: el.y + el.height / 2 - viewbox.height / 2,
        width: viewbox.width,
        height: viewbox.height,
      });
    } catch (err) {
      console.error('Focus element failed:', err);
    }
  }, []);

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
      errors.push({ type: 'error', message: t('bpmnDesigner.messages.missingStartEvent') });
    }

    if (endEvents.length === 0) {
      errors.push({ type: 'error', message: t('bpmnDesigner.messages.missingEndEvent') });
    }

    // 检查用户任务是否有配置
    const userTasks = elementRegistry.filter(el => el.type === 'bpmn:UserTask');
    userTasks.forEach(task => {
      const bo = task.businessObject;
      if (!bo.assignee && !bo.candidateUsers && !bo.candidateGroups) {
        errors.push({
          type: 'warning',
          message: t('bpmnDesigner.messages.userTaskNoConfig', { name: bo.name || task.id }),
          elementId: task.id,
          elementType: task.type,
          elementName: bo.name || task.id,
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
          message: t('bpmnDesigner.messages.serviceTaskNoConfig', { name: bo.name || task.id }),
          elementId: task.id,
          elementType: task.type,
          elementName: bo.name || task.id,
        });
      }
      const implementation = bo.implementation || bo.operationRef;
      const supported = new Set(['webhook', 'cc_handler', 'ticket_handler', 'change_handler', 'incident_handler', 'service_request_handler', 'notification_handler', 'approval_handler', 'generic_handler']);
      if (implementation && !supported.has(implementation)) {
        errors.push({
          type: 'error',
          message: `服务任务 ${bo.name || task.id} 的处理器“${implementation}”未注册，当前不可发布`,
          elementId: task.id,
          elementType: task.type,
          elementName: bo.name || task.id,
        });
      }
    });

    elementRegistry.filter(el => el.type === 'bpmn:ScriptTask').forEach(task => {
      errors.push({
        type: 'error',
        message: `脚本任务 ${task.businessObject?.name || task.id} 尚无隔离执行器，当前不可发布`,
        elementId: task.id,
        elementType: task.type,
        elementName: task.businessObject?.name || task.id,
      });
    });

    // 检查网关是否有默认分支和条件
    const gateways = elementRegistry.filter(el => el.type.includes('Gateway'));
    gateways.forEach(gateway => {
      const bo = gateway.businessObject;
      const outgoing = gateway.outgoing || [];

      if (bo.$type === 'bpmn:ExclusiveGateway' && outgoing.length > 1 && !bo.default) {
        errors.push({
          type: 'warning',
          message: t('bpmnDesigner.messages.gatewayNoDefault', { name: bo.name || gateway.id }),
          elementId: gateway.id,
          elementType: gateway.type,
          elementName: bo.name || gateway.id,
        });
      }

      // 检查输出流是否有条件
      outgoing.forEach((flow: any) => {
        if (!flow.conditionExpression && outgoing.length > 1) {
          errors.push({
            type: 'warning',
            message: t('bpmnDesigner.messages.flowNoCondition', { name: bo.name || gateway.id, flowId: flow.id }),
            elementId: flow.id,
            elementType: flow.type,
            elementName: flow.name || flow.id,
          });
        }
      });
    });

    // 显示验证结果
    if (errors.length === 0) {
      message.success(t('bpmnDesigner.messages.validationPassed'));
    } else {
      const errorCount = errors.filter(e => e.type === 'error').length;
      const warningCount = errors.filter(e => e.type === 'warning').length;
      message.warning(t('bpmnDesigner.messages.validationResult', { errors: errorCount, warnings: warningCount }));
    }

    return errors;
  }, [message, t]);

  /**
   * 供父组件调用：修改当前 BPMN 元素的属性。
   * 通过 modeling.updateProperties 走命令栈，会触发 commandStack.changed，
   * 进而通过 saveXML 把最新的 XML 推回父组件。
   */
  const updateElementProperties = useCallback((elementId: string, properties: Record<string, unknown>) => {
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
  }, []);

  /**
   * 供父组件调用：触发 fit-viewport（节点改变或初始加载后可用）
   */
  const fitViewport = useCallback(() => {
    if (!modelerRef.current) return;
    try {
      const canvas = modelerRef.current.get('canvas') as { zoom: (level: string) => void };
      canvas?.zoom('fit-viewport');
    } catch (err) {
      console.error('Fit viewport failed:', err);
    }
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

  // 键盘快捷键处理
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (readOnly) return;

      const mod = e.ctrlKey || e.metaKey;
      const key = e.key.toLowerCase();

      // Ctrl/Cmd + S: 保存（任何焦点下都生效，避免浏览器默认"保存网页"）
      if (mod && key === 's') {
        e.preventDefault();
        void handleSave();
        return;
      }

      // 输入框内不劫持其余快捷键（避免影响正常输入/复制粘贴）
      if (isEditableTarget(e.target)) return;

      // Ctrl/Cmd + Z: 撤销
      if (mod && key === 'z' && !e.shiftKey) {
        e.preventDefault();
        handleUndo();
        return;
      }

      // Ctrl/Cmd + Y / Ctrl/Cmd + Shift + Z: 重做
      if ((mod && key === 'y') || (mod && e.shiftKey && key === 'z')) {
        e.preventDefault();
        handleRedo();
        return;
      }

      // Delete / Backspace: 删除选中元素
      if (e.key === 'Delete' || e.key === 'Backspace') {
        if (selectedElementsRef.current.length === 0) return;
        e.preventDefault();
        handleDelete();
        return;
      }

      // Ctrl/Cmd + A: 全选
      if (mod && key === 'a') {
        e.preventDefault();
        handleSelectAll();
        return;
      }

      // Ctrl/Cmd + C: 复制
      if (mod && key === 'c') {
        if (selectedElementsRef.current.length === 0) return;
        e.preventDefault();
        handleCopy();
        return;
      }

      // Ctrl/Cmd + V: 粘贴
      if (mod && key === 'v') {
        e.preventDefault();
        handlePaste();
        return;
      }

      // Ctrl/Cmd + F: 搜索
      if (mod && key === 'f') {
        e.preventDefault();
        const searchInput = document.getElementById('bpmn-search-input');
        searchInput?.focus();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [readOnly, handleSave, handleUndo, handleRedo, handleDelete, handleSelectAll, handleCopy, handlePaste]);

  // 对齐菜单
  const alignMenuItems: MenuProps['items'] = [
    {
      key: 'left',
      icon: <AlignLeft size={14} />,
      label: t('bpmnDesigner.buttons.alignLeft'),
      onClick: () => handleAlign('left'),
      disabled: selectedElements.length < 2 || readOnly
    },
    {
      key: 'center',
      icon: <AlignCenter size={14} />,
      label: t('bpmnDesigner.buttons.alignCenter'),
      onClick: () => handleAlign('center'),
      disabled: selectedElements.length < 2 || readOnly
    },
    {
      key: 'right',
      icon: <AlignRight size={14} />,
      label: t('bpmnDesigner.buttons.alignRight'),
      onClick: () => handleAlign('right'),
      disabled: selectedElements.length < 2 || readOnly
    },
    {
      type: 'divider'
    },
    {
      key: 'top',
      icon: <PanelTop size={14} />,
      label: t('bpmnDesigner.buttons.alignTop'),
      onClick: () => handleAlign('top'),
      disabled: selectedElements.length < 2 || readOnly
    },
    {
      key: 'middle',
      icon: <Rows2 size={14} />,
      label: t('bpmnDesigner.buttons.alignMiddle'),
      onClick: () => handleAlign('middle'),
      disabled: selectedElements.length < 2 || readOnly
    },
    {
      key: 'bottom',
      icon: <PanelBottom size={14} />,
      label: t('bpmnDesigner.buttons.alignBottom'),
      onClick: () => handleAlign('bottom'),
      disabled: selectedElements.length < 2 || readOnly
    }
  ];

  // 分布菜单
  const distributeMenuItems: MenuProps['items'] = [
    {
      key: 'horizontal',
      icon: <AlignHorizontalDistributeCenter size={14} />,
      label: t('bpmnDesigner.buttons.distributeHorizontal'),
      onClick: () => handleDistribute('horizontal'),
      disabled: selectedElements.length < 3 || readOnly
    },
    {
      key: 'vertical',
      icon: <AlignVerticalDistributeCenter size={14} />,
      label: t('bpmnDesigner.buttons.distributeVertical'),
      onClick: () => handleDistribute('vertical'),
      disabled: selectedElements.length < 3 || readOnly
    }
  ];

  // 设置菜单
  const settingsMenuItems: MenuProps['items'] = [
    {
      key: 'grid',
      icon: <Grid size={14} />,
      label: showGrid ? t('bpmnDesigner.buttons.hideGrid') : t('bpmnDesigner.buttons.showGrid'),
      onClick: toggleGrid
    },
    {
      key: 'snap',
      icon: <Grid size={14} />,
      label: snapToGrid ? t('bpmnDesigner.buttons.disableSnap') : t('bpmnDesigner.buttons.enableSnap'),
      onClick: toggleSnapToGrid
    },
    {
      type: 'divider'
    },
    {
      key: 'validate',
      icon: <Bug size={14} />,
      label: t('bpmnDesigner.buttons.validateFlow'),
      onClick: handleValidate
    }
  ];

  // 搜索下拉选项
  const searchOptions = searchMatches.map(match => ({
    value: match.id,
    label: (
      <div className="flex items-center justify-between gap-2">
        <span className="truncate">{match.name}</span>
        <span className="text-xs text-gray-400 shrink-0">{match.type}</span>
      </div>
    ),
  }));

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
        <Tooltip title={t('bpmnDesigner.buttons.saveShortcut')} placement="right">
          <Button type="text" icon={<Save size={18} />} onClick={handleSave} disabled={readOnly} />
        </Tooltip>
        {onDeploy && (
          <Tooltip title={t('bpmnDesigner.buttons.deploy')} placement="right">
            <Button
              type="text"
              icon={<PlayCircle size={18} />}
              onClick={handleDeploy}
              disabled={readOnly}
            />
          </Tooltip>
        )}
        <Tooltip title={t('bpmnDesigner.buttons.undoShortcut')} placement="right">
          <Button
            type="text"
            icon={<Undo size={18} />}
            onClick={handleUndo}
            disabled={readOnly || !canUndo}
          />
        </Tooltip>
        <Tooltip title={t('bpmnDesigner.buttons.redoShortcut')} placement="right">
          <Button
            type="text"
            icon={<Redo size={18} />}
            onClick={handleRedo}
            disabled={readOnly || !canRedo}
          />
        </Tooltip>

        <div style={{ height: 1, width: '80%', background: '#e8e8e8', margin: '8px 0' }} />

        <Tooltip title={t('bpmnDesigner.buttons.copyShortcut')} placement="right">
          <Button
            type="text"
            icon={<Copy size={18} />}
            onClick={handleCopy}
            disabled={readOnly || selectedElements.length === 0}
          />
        </Tooltip>
        <Tooltip title={t('bpmnDesigner.buttons.pasteShortcut')} placement="right">
          <Button
            type="text"
            icon={<ClipboardPaste size={18} />}
            onClick={handlePaste}
            disabled={readOnly}
          />
        </Tooltip>
        <Tooltip title={t('bpmnDesigner.buttons.selectAllShortcut')} placement="right">
          <Button
            type="text"
            icon={<ListChecks size={18} />}
            onClick={handleSelectAll}
            disabled={readOnly}
          />
        </Tooltip>
        <Tooltip title={t('bpmnDesigner.buttons.deleteShortcut')} placement="right">
          <Button
            type="text"
            icon={<Trash2 size={18} />}
            onClick={handleDelete}
            disabled={readOnly || selectedElements.length === 0}
            danger
          />
        </Tooltip>

        <div style={{ height: 1, width: '80%', background: '#e8e8e8', margin: '8px 0' }} />

        <Dropdown menu={{ items: alignMenuItems }} placement="bottomRight" trigger={['click']}>
          <Tooltip title={t('bpmnDesigner.buttons.align')} placement="right">
            <Button type="text" icon={<AlignLeft size={18} />} disabled={selectedElements.length < 2 || readOnly} />
          </Tooltip>
        </Dropdown>

        <Dropdown menu={{ items: distributeMenuItems }} placement="bottomRight" trigger={['click']}>
          <Tooltip title={t('bpmnDesigner.buttons.distribute')} placement="right">
          <Button type="text" icon={<AlignHorizontalDistributeCenter size={18} />} disabled={selectedElements.length < 3 || readOnly} />
          </Tooltip>
        </Dropdown>

        <div style={{ flex: 1 }} />

        <Dropdown menu={{ items: settingsMenuItems }} placement="bottomRight" trigger={['click']}>
          <Tooltip title={t('bpmnDesigner.buttons.settings')} placement="right">
            <Button type="text" icon={<Settings size={18} />} />
          </Tooltip>
        </Dropdown>

        <Tooltip title={t('bpmnDesigner.buttons.exportSvg')} placement="right">
          <Button type="text" icon={<Download size={18} />} onClick={handleExportSVG} />
        </Tooltip>
        <Tooltip title={t('bpmnDesigner.buttons.exportBpmn')} placement="right">
          <Button type="text" icon={<FileJson size={18} />} onClick={handleExportXML} />
        </Tooltip>
        <label>
          <input
            type="file"
            accept=".bpmn,.xml"
            style={{ display: 'none' }}
            onChange={handleImportXML}
          />
          <Tooltip title={t('bpmnDesigner.buttons.importBpmn')} placement="right">
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
        <AutoComplete
          value={searchKeyword}
          options={searchOptions}
          onSearch={handleSearchChange}
          onSelect={value => focusElement(String(value))}
          onChange={(value: string) => {
            if (!value) {
              setSearchKeyword('');
              setSearchMatches([]);
            }
          }}
          popupMatchSelectWidth={280}
          defaultActiveFirstOption
          size="small"
          style={{ width: '100%' }}
        >
          <Input
            id="bpmn-search-input"
            placeholder={t('bpmnDesigner.buttons.searchPlaceholder')}
            prefix={<Search size={14} />}
            allowClear
            size="small"
          />
        </AutoComplete>
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
        <Tooltip title={t('bpmnDesigner.buttons.zoomOut')}>
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
        <Tooltip title={t('bpmnDesigner.buttons.zoomIn')}>
          <Button
            type="text"
            size="small"
            icon={<ZoomIn size={16} />}
            onClick={() => handleZoom(0.1)}
          />
        </Tooltip>
        <Tooltip title={t('bpmnDesigner.buttons.fit')}>
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
          {t('bpmnDesigner.messages.elementsSelected', { count: selectedElements.length })}
        </div>
      )}
    </div>
  );
};

export default BPMNDesigner;
