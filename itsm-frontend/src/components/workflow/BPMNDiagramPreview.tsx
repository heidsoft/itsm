'use client';

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Spin, Tooltip, Typography } from 'antd';
import { Maximize, RefreshCw, ZoomIn, ZoomOut } from 'lucide-react';
import BpmnNavigatedViewer from 'bpmn-js/lib/NavigatedViewer';
import itsmModdleDescriptor from './itsm-moddle-descriptor';
import { ensureBpmnDI, hasBpmnDiagramWithShapes } from './bpmnAutoLayout';
import { useI18n } from '@/lib/i18n/useI18n';

import 'bpmn-js/dist/assets/diagram-js.css';
import 'bpmn-js/dist/assets/bpmn-font/css/bpmn.css';

interface BPMNDiagramPreviewProps {
  /** 待预览的 BPMN 2.0 XML；空串视为无内容，展示空态而不是报错 */
  xml: string;
  height?: number | string;
  /** 并排对比时标注这是哪一个版本 */
  label?: string;
  /**
   * 需要高亮的元素 ID（通常是实例的 currentActivityId）。
   *
   * 只读预览此前只能看到「图长什么样」，看不到「我的单子现在卡在哪」——
   * 用户必须自己在 21 个节点里找那个名字。传入本字段后导入完成即给该元素
   * 打上 bpmn-current-activity 标记并滚动到视野中央。
   *
   * 传 undefined 时行为与改造前完全一致（不注册任何 marker），
   * 定义预览、版本对比等既有调用方零影响。
   */
  activeElementId?: string;
  /** 高亮元素的补充说明（如「当前节点：自动分配」），仅在 activeElementId 命中时展示 */
  activeElementLabel?: string;
  /**
   * 已走过的元素 ID 列表（通常是实例 history 里的 activityId）。
   *
   * 只高亮当前节点时，用户仍然只能看到「现在在哪」，看不到「已经走过哪些」——
   * 而「走到这一步之前经过了谁」是排查卡单、复盘流转路径的核心信息。传入本字段后
   * 这些元素会被打上 bpmn-visited 标记（浅色描边，不与当前节点的橙色争视觉焦点）。
   *
   * 排序语义由调用方负责：本组件只按传入顺序打标记，不做「谁是最后一步」的推断。
   */
  visitedElementIds?: string[];
}

interface CanvasService {
  zoom: (level?: string | number) => number;
  addMarker?: (elementId: string, marker: string) => void;
  removeMarker?: (elementId: string, marker: string) => void;
  scrollToElement?: (elementId: string) => void;
}

interface ElementRegistryService {
  get?: (elementId: string) => unknown;
}

type PreviewStatus = 'loading' | 'ready' | 'empty' | 'error';

const SIZE_RETRY_INTERVAL = 120;
const SIZE_RETRY_LIMIT = 15;

/** 当前节点高亮标记。样式见 src/app/globals.css 的 .djs-element.bpmn-current-activity */
const CURRENT_ACTIVITY_MARKER = 'bpmn-current-activity';
/** 已走过节点标记。浅色，不与当前节点的橙色争视觉焦点 */
const VISITED_MARKER = 'bpmn-visited';

/**
 * 取 canvas 与 elementRegistry，二者任一不可用即返回 null。
 *
 * 注册表不可用时不赌运气：宁可不高亮，也不让 exception 冒到 import 流程里。
 * canvas.addMarker 内部会对元素对象直接读写 markers 字段，拿到 undefined 会抛
 * TypeError —— 这个异常会顺着 importXML 的调用栈把整张图打成错误态。
 */
function resolveMarkerTargets(
  viewer: BpmnNavigatedViewer
): { canvas: CanvasService; registry: ElementRegistryService } | null {
  const canvas = viewer.get('canvas') as CanvasService | undefined;
  if (!canvas?.addMarker) return null;
  const registry = viewer.get('elementRegistry') as ElementRegistryService | undefined;
  if (!registry?.get) return null;
  return { canvas, registry };
}

/**
 * 给指定元素打标记；元素不在图上（如流程改版后旧实例引用了已删除节点）时跳过。
 *
 * prod 实测存在这种情形：实例的 currentActivityId 指向的元素在当前版本 XML 里
 * 已不存在，此时必须提前判掉而不是依赖 addMarker 自己容错。
 */
function markIfPresent(
  targets: { canvas: CanvasService; registry: ElementRegistryService },
  elementId: string,
  marker: string
): boolean {
  if (!targets.registry.get?.(elementId)) return false;
  targets.canvas.addMarker?.(elementId, marker);
  return true;
}

/**
 * 应用三类标记（已走过 / 当前节点 / 滚动定位）。
 *
 * 全部步骤 best-effort：高亮只是「锦上添花」，任何一步失败都不能让整张图变空白——
 * 后者正是上一批要修的反面（预览失败必须能看见图）。
 *
 * 顺序固定：先 visited 再 current，最后滚动到 current。当前节点永远压在最上层，
 * 否则「已走过」的浅色描边会盖住「现在就停在这」的强调。
 */
function applyActivityMarkers(
  viewer: BpmnNavigatedViewer,
  activeElementId: string | undefined,
  visitedElementIds: string[] | undefined
): void {
  if (!activeElementId && !visitedElementIds?.length) return;
  try {
    const targets = resolveMarkerTargets(viewer);
    if (!targets) return;

    // 去重：history 里同一节点可能因回退/重入出现多次，重复 addMarker 无意义
    for (const id of new Set(visitedElementIds ?? [])) {
      if (id && id !== activeElementId) markIfPresent(targets, id, VISITED_MARKER);
    }

    if (activeElementId && markIfPresent(targets, activeElementId, CURRENT_ACTIVITY_MARKER)) {
      targets.canvas.scrollToElement?.(activeElementId);
    }
  } catch {
    // 标记失败不影响图形本身，静默跳过
  }
}

/**
 * 只读 BPMN 预览：供流程列表、版本详情、模板详情、实例进度使用。
 * 用 NavigatedViewer 而不是 Modeler，预览面不该带出调色板、命令栈和属性写入能力。
 */
const BPMNDiagramPreview: React.FC<BPMNDiagramPreviewProps> = ({
  xml,
  height = 360,
  label,
  activeElementId,
  activeElementLabel,
  visitedElementIds,
}) => {
  const { t } = useI18n();
  const containerRef = useRef<HTMLDivElement>(null);
  const viewerRef = useRef<BpmnNavigatedViewer | null>(null);
  // 只接受最新一次导入结果，避免快速切换预览对象时旧图覆盖新图
  const importTokenRef = useRef(0);
  const sizeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [status, setStatus] = useState<PreviewStatus>('loading');
  const [errorMessage, setErrorMessage] = useState('');
  const [zoom, setZoom] = useState(1);

  // 用内容指纹代替数组引用做依赖（见 runImport 依赖数组处的注释）。
  // 顺序有意义（历史按时间推进），故不做排序归一化。
  const visitedKey = (visitedElementIds ?? []).join('\u0001');

  const readZoom = useCallback((): number => {
    const viewer = viewerRef.current;
    if (!viewer) return 1;
    try {
      return (viewer.get('canvas') as CanvasService).zoom();
    } catch {
      return 1;
    }
  }, []);

  const applyZoom = useCallback(
    (next: number | 'fit-viewport') => {
      const viewer = viewerRef.current;
      if (!viewer) return;
      try {
        (viewer.get('canvas') as CanvasService).zoom(next);
        setZoom(next === 'fit-viewport' ? readZoom() : next);
      } catch {
        // 视图尚未导入完成时缩放不可用，状态区已经给出提示
      }
    },
    [readZoom]
  );

  const ensureViewer = useCallback((container: HTMLElement): BpmnNavigatedViewer => {
    if (viewerRef.current) return viewerRef.current;

    const viewer = new BpmnNavigatedViewer({
      container,
      moddleExtensions: { itsm: itsmModdleDescriptor },
    });
    viewerRef.current = viewer;

    // 模板/AI 产出的 XML 常常没有 <bpmndi:BPMNDiagram>，Viewer 没有自动布局能力，
    // 不补 DI 只会得到 "no diagram to display"。与设计器复用同一注入点。
    // 判据用 hasBpmnDiagramWithShapes 而非 hasBpmnDiagram：prod 有 3 个定义带的是
    // 零 shape/edge 的空壳 plane，只看「有没有 diagram」会让它们静默渲染成空白画布。
    viewer.on('import.parse.complete', 1500, (event: { definitions?: unknown; error?: unknown }) => {
      if (event?.error || !event.definitions) return;
      const definitions = event.definitions as Parameters<typeof hasBpmnDiagramWithShapes>[0];
      if (hasBpmnDiagramWithShapes(definitions)) return;
      try {
        ensureBpmnDI(
          viewer.get('moddle') as unknown as Parameters<typeof ensureBpmnDI>[0],
          definitions as Parameters<typeof ensureBpmnDI>[1]
        );
      } catch {
        // 补全失败交给 importXML 的错误分支统一提示
      }
    });

    return viewer;
  }, []);

  const runImport = useCallback(async () => {
    if (!xml.trim()) {
      setStatus('empty');
      setErrorMessage('');
      return;
    }

    const attempt = async (retryCount: number): Promise<void> => {
      const container = containerRef.current;
      if (!container) return;

      // Modal 动画期间容器尺寸为 0，bpmn-js 会在零尺寸画布上抛错，等布局完成再导入
      const rect = container.getBoundingClientRect();
      if (rect.width <= 0 || rect.height <= 0) {
        if (retryCount >= SIZE_RETRY_LIMIT) {
          setErrorMessage(t('bpmnDesigner.preview.containerNotReady'));
          setStatus('error');
          return;
        }
        if (sizeTimerRef.current) clearTimeout(sizeTimerRef.current);
        sizeTimerRef.current = setTimeout(() => void attempt(retryCount + 1), SIZE_RETRY_INTERVAL);
        return;
      }

      const token = ++importTokenRef.current;
      setStatus('loading');
      setErrorMessage('');

      try {
        const viewer = ensureViewer(container);
        await viewer.importXML(xml);
        if (token !== importTokenRef.current || viewerRef.current !== viewer) return;
        (viewer.get('canvas') as CanvasService).zoom('fit-viewport');
        setZoom(readZoom());
        // 先 fit-viewport 再标记：marker 自身不改视口，滚动到元素必须在
        // 「全图已在视野内」之后执行，否则会被 fit 的缩放重置掉目标位置。
        applyActivityMarkers(viewer, activeElementId, visitedElementIds);
        setStatus('ready');
      } catch (err) {
        if (token !== importTokenRef.current) return;
        setErrorMessage(err instanceof Error ? err.message : String(err));
        setStatus('error');
      }
    };

    await attempt(0);
    // activeElementId 必须在依赖里：同一份 XML 下切换实例时若不重跑导入，
    // 旧实例的高亮会一直挂在图上，用户会以为自己的单卡在别人那个节点。
    //
    // visitedElementIds 走序列化后的值而不是数组本身：数组是对象引用，调用方
    // 每次渲染新建一个（如 history.map(...)）会让依赖恒不等，导致每帧都重新
    // importXML，画布持续闪烁。用稳定字符串比较才是真正的「内容是否变化」。
  }, [ensureViewer, readZoom, t, xml, activeElementId, visitedKey]);

  useEffect(() => {
    void runImport();
  }, [runImport]);

  useEffect(() => {
    return () => {
      if (sizeTimerRef.current) clearTimeout(sizeTimerRef.current);
      importTokenRef.current += 1;
      viewerRef.current?.destroy();
      viewerRef.current = null;
    };
  }, []);

  return (
    <div style={{ position: 'relative', border: '1px solid #d9d9d9', borderRadius: 6, overflow: 'hidden' }}>
      {label && (
        <div
          style={{
            padding: '4px 10px',
            borderBottom: '1px solid #d9d9d9',
            background: '#fafafa',
            fontSize: 12,
          }}
        >
          {label}
        </div>
      )}
      <div ref={containerRef} style={{ height, width: '100%' }} />

      {status !== 'ready' && (
        <div
          style={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: 16,
            background: '#fafafa',
          }}
        >
          {status === 'loading' && <Spin size="large" description={t('bpmnDesigner.preview.loading')} />}
          {status === 'empty' && (
            <Typography.Text type="secondary">{t('bpmnDesigner.preview.empty')}</Typography.Text>
          )}
          {status === 'error' && (
            <Alert
              type="error"
              showIcon
              title={t('bpmnDesigner.preview.failed')}
              description={
                <div>
                  <div>{errorMessage || t('bpmnDesigner.preview.unknownError')}</div>
                  <Button
                    size="small"
                    style={{ marginTop: 8 }}
                    icon={<RefreshCw size={14} />}
                    onClick={() => void runImport()}
                  >
                    {t('bpmnDesigner.preview.retry')}
                  </Button>
                </div>
              }
            />
          )}
        </div>
      )}

      {status === 'ready' && (
        <div
          style={{
            position: 'absolute',
            right: 8,
            bottom: 8,
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
              aria-label={t('bpmnDesigner.buttons.zoomOut')}
              type="text"
              size="small"
              icon={<ZoomOut size={16} />}
              onClick={() => applyZoom(Math.max(0.4, Number((readZoom() * 0.8).toFixed(2))))}
            />
          </Tooltip>
          <span style={{ minWidth: 46, textAlign: 'center', lineHeight: '24px', fontSize: 12 }}>
            {Math.round(zoom * 100)}%
          </span>
          <Tooltip title={t('bpmnDesigner.buttons.zoomIn')}>
            <Button
              aria-label={t('bpmnDesigner.buttons.zoomIn')}
              type="text"
              size="small"
              icon={<ZoomIn size={16} />}
              onClick={() => applyZoom(Math.min(3, Number((readZoom() * 1.2).toFixed(2))))}
            />
          </Tooltip>
          <Tooltip title={t('bpmnDesigner.buttons.fit')}>
            <Button
              aria-label={t('bpmnDesigner.buttons.fit')}
              type="text"
              size="small"
              icon={<Maximize size={16} />}
              onClick={() => applyZoom('fit-viewport')}
            />
          </Tooltip>
        </div>
      )}

      {/* 图例：光有描边，用户不知道那是什么意思。仅在确实传了标记且图已就绪时
          展示，避免在纯定义预览里凭空出现一个空图例。
          已走过与当前节点必须同时出图例，否则两种颜色的差异只是「看起来不一样」。 */}
      {status === 'ready' && (activeElementId || visitedKey) && (
        <div
          style={{
            position: 'absolute',
            left: 8,
            bottom: 8,
            display: 'flex',
            alignItems: 'center',
            flexWrap: 'wrap',
            gap: 10,
            background: 'white',
            padding: '3px 8px',
            borderRadius: 6,
            boxShadow: '0 2px 8px rgba(0,0,0,0.15)',
            fontSize: 12,
          }}
        >
          {activeElementId && (
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
              <span
                aria-hidden
                style={{
                  width: 10,
                  height: 10,
                  borderRadius: 2,
                  border: '2px solid #fa8c16',
                  background: '#fff7e6',
                  flex: '0 0 auto',
                }}
              />
              <span>{activeElementLabel ? `当前节点：${activeElementLabel}` : '当前节点'}</span>
            </span>
          )}
          {visitedKey && (
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
              <span
                aria-hidden
                style={{
                  width: 10,
                  height: 10,
                  borderRadius: 2,
                  border: '1.5px dashed #52c41a',
                  background: 'transparent',
                  flex: '0 0 auto',
                }}
              />
              <span>已走过</span>
            </span>
          )}
        </div>
      )}
    </div>
  );
};

export default BPMNDiagramPreview;
