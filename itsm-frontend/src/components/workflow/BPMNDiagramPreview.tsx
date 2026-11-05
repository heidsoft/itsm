'use client';

import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button, Spin, Tooltip, Typography } from 'antd';
import { Maximize, RefreshCw, ZoomIn, ZoomOut } from 'lucide-react';
import BpmnNavigatedViewer from 'bpmn-js/lib/NavigatedViewer';
import itsmModdleDescriptor from './itsm-moddle-descriptor';
import { ensureBpmnDI, hasBpmnDiagram } from './bpmnAutoLayout';
import { useI18n } from '@/lib/i18n/useI18n';

import 'bpmn-js/dist/assets/diagram-js.css';
import 'bpmn-js/dist/assets/bpmn-font/css/bpmn.css';

interface BPMNDiagramPreviewProps {
  /** 待预览的 BPMN 2.0 XML；空串视为无内容，展示空态而不是报错 */
  xml: string;
  height?: number | string;
  /** 并排对比时标注这是哪一个版本 */
  label?: string;
}

interface CanvasService {
  zoom: (level?: string | number) => number;
}

type PreviewStatus = 'loading' | 'ready' | 'empty' | 'error';

const SIZE_RETRY_INTERVAL = 120;
const SIZE_RETRY_LIMIT = 15;

/**
 * 只读 BPMN 预览：供流程列表、版本详情、模板详情使用。
 * 用 NavigatedViewer 而不是 Modeler，预览面不该带出调色板、命令栈和属性写入能力。
 */
const BPMNDiagramPreview: React.FC<BPMNDiagramPreviewProps> = ({ xml, height = 360, label }) => {
  const { t } = useI18n();
  const containerRef = useRef<HTMLDivElement>(null);
  const viewerRef = useRef<BpmnNavigatedViewer | null>(null);
  // 只接受最新一次导入结果，避免快速切换预览对象时旧图覆盖新图
  const importTokenRef = useRef(0);
  const sizeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [status, setStatus] = useState<PreviewStatus>('loading');
  const [errorMessage, setErrorMessage] = useState('');
  const [zoom, setZoom] = useState(1);

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
    viewer.on('import.parse.complete', 1500, (event: { definitions?: unknown; error?: unknown }) => {
      if (event?.error || !event.definitions) return;
      const definitions = event.definitions as Parameters<typeof hasBpmnDiagram>[0];
      if (hasBpmnDiagram(definitions)) return;
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
        setStatus('ready');
      } catch (err) {
        if (token !== importTokenRef.current) return;
        setErrorMessage(err instanceof Error ? err.message : String(err));
        setStatus('error');
      }
    };

    await attempt(0);
  }, [ensureViewer, readZoom, t, xml]);

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
    </div>
  );
};

export default BPMNDiagramPreview;
