import React from 'react';
import { render, screen, fireEvent, act, waitFor } from '@testing-library/react';
import BPMNDiagramPreview from '@/components/workflow/BPMNDiagramPreview';

const mockImportXML = jest.fn();
let currentZoomLevel = 1;
let zoomLevels: (string | number | undefined)[] = [];
const mockZoom = jest.fn((level?: string | number) => {
  zoomLevels.push(level);
  if (typeof level === 'number') currentZoomLevel = level;
  return currentZoomLevel;
});
const mockOn = jest.fn();
const mockDestroy = jest.fn();
const mockAddMarker = jest.fn();
const mockScrollToElement = jest.fn();
let viewerInstances = 0;
// 图上实际存在的元素 ID；测试按需改写，用于验证「高亮不存在的元素必须跳过」
let registryElementIds: string[] = [];

jest.mock('bpmn-js/lib/NavigatedViewer', () =>
  jest.fn().mockImplementation(() => {
    viewerInstances += 1;
    return {
      importXML: (...args: unknown[]) => mockImportXML(...args),
      on: mockOn,
      destroy: mockDestroy,
      get: (name: string) => {
        if (name === 'canvas') {
          return { zoom: mockZoom, addMarker: mockAddMarker, scrollToElement: mockScrollToElement };
        }
        if (name === 'elementRegistry') {
          return {
            get: (id: string) => (registryElementIds.includes(id) ? { id } : undefined),
          };
        }
        return {};
      },
    };
  })
);

describe('BPMNDiagramPreview', () => {
  const setContainerSize = (width: number, height: number) => {
    Element.prototype.getBoundingClientRect = jest.fn(() => ({
      width,
      height,
      x: 0,
      y: 0,
      top: 0,
      right: width,
      bottom: height,
      left: 0,
      toJSON: () => ({}),
    }));
  };

  beforeEach(() => {
    jest.clearAllMocks();
    viewerInstances = 0;
    zoomLevels = [];
    currentZoomLevel = 1;
    registryElementIds = [];
    setContainerSize(800, 600);
  });

  it('空白 XML 展示空态而不是报错，也不创建 Viewer', async () => {
    render(<BPMNDiagramPreview xml="   " />);

    expect(await screen.findByText('该流程暂无可预览的 BPMN 图形')).toBeInTheDocument();
    expect(viewerInstances).toBe(0);
    expect(screen.queryByRole('button', { name: '适应画布' })).not.toBeInTheDocument();
  });

  it('导入成功后显示缩放控件并适应画布', async () => {
    mockImportXML.mockResolvedValueOnce({});

    render(<BPMNDiagramPreview xml="<definitions />" label="版本 2" />);

    await waitFor(() => expect(mockZoom).toHaveBeenCalledWith('fit-viewport'));
    expect(screen.getByText('版本 2')).toBeInTheDocument();
    expect(screen.getByText('100%')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '缩小' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '放大' })).toBeInTheDocument();
  });

  it('注册 BPMNDI 补全钩子，复用设计器的同一注入点', async () => {
    mockImportXML.mockResolvedValueOnce({});

    render(<BPMNDiagramPreview xml="<definitions />" />);
    await waitFor(() => expect(mockZoom).toHaveBeenCalledWith('fit-viewport'));

    const registration = mockOn.mock.calls.find(call => call[0] === 'import.parse.complete');
    expect(registration?.[1]).toBe(1500);
    expect(typeof registration?.[2]).toBe('function');
  });

  it('导入失败时给出原因并允许重试', async () => {
    mockImportXML
      .mockRejectedValueOnce(new Error('no diagram to display'))
      .mockResolvedValueOnce({});

    render(<BPMNDiagramPreview xml="<definitions />" />);

    expect(await screen.findByText('流程图预览失败')).toBeInTheDocument();
    expect(screen.getByText('no diagram to display')).toBeInTheDocument();

    act(() => {
      fireEvent.click(screen.getByRole('button', { name: '重试' }));
    });

    await waitFor(() => expect(mockZoom).toHaveBeenCalledWith('fit-viewport'));
    expect(screen.queryByText('流程图预览失败')).not.toBeInTheDocument();
    expect(mockImportXML).toHaveBeenCalledTimes(2);
  });

  it('容器尚未布局时重试，超过上限后提示未就绪而不是静默空白', () => {
    jest.useFakeTimers();
    setContainerSize(0, 0);

    render(<BPMNDiagramPreview xml="<definitions />" />);

    act(() => {
      jest.advanceTimersByTime(120 * 16);
    });

    expect(screen.getByText('预览容器尚未就绪，请重试')).toBeInTheDocument();
    expect(viewerInstances).toBe(0);

    jest.useRealTimers();
  });

  it('快速切换预览对象时旧导入不覆盖新结果', async () => {
    const pending: { resolve: (value: unknown) => void }[] = [];
    mockImportXML.mockImplementation(
      () => new Promise(resolve => pending.push({ resolve }))
    );

    const { rerender } = render(<BPMNDiagramPreview xml="<definitions>A</definitions>" />);
    rerender(<BPMNDiagramPreview xml="<definitions>B</definitions>" />);

    expect(pending).toHaveLength(2);

    // 新导入先完成
    await act(async () => {
      pending[1].resolve({});
    });
    await waitFor(() => expect(mockZoom).toHaveBeenCalledWith('fit-viewport'));

    // 旧导入后完成，不能再次改写视口
    await act(async () => {
      pending[0].resolve({});
    });
    expect(zoomLevels.filter(level => level === 'fit-viewport')).toHaveLength(1);
  });

  it('卸载时销毁 Viewer', async () => {
    mockImportXML.mockResolvedValueOnce({});

    const { unmount } = render(<BPMNDiagramPreview xml="<definitions />" />);
    await waitFor(() => expect(mockZoom).toHaveBeenCalledWith('fit-viewport'));

    unmount();
    expect(mockDestroy).toHaveBeenCalled();
  });

  it('缩放按钮按当前比例逐步放大缩小', async () => {
    mockImportXML.mockResolvedValueOnce({});

    render(<BPMNDiagramPreview xml="<definitions />" />);
    await waitFor(() => expect(screen.getByText('100%')).toBeInTheDocument());

    act(() => {
      fireEvent.click(screen.getByRole('button', { name: '放大' }));
    });
    expect(mockZoom).toHaveBeenCalledWith(1.2);
    expect(screen.getByText('120%')).toBeInTheDocument();

    act(() => {
      fireEvent.click(screen.getByRole('button', { name: '缩小' }));
    });
    expect(mockZoom).toHaveBeenLastCalledWith(0.96);
    expect(screen.getByText('96%')).toBeInTheDocument();
  });

  describe('当前节点高亮', () => {
    it('传入 activeElementId 且元素存在时打标记并滚动到元素', async () => {
      mockImportXML.mockResolvedValueOnce({});
      registryElementIds = ['Activity_AutoAssign'];

      render(
        <BPMNDiagramPreview
          xml="<definitions />"
          activeElementId="Activity_AutoAssign"
          activeElementLabel="自动分配"
        />
      );

      await waitFor(() => expect(mockAddMarker).toHaveBeenCalledWith('Activity_AutoAssign', 'bpmn-current-activity'));
      expect(mockScrollToElement).toHaveBeenCalledWith('Activity_AutoAssign');
      // 图例要告诉用户橙色描边是什么，否则高亮等于没解释
      expect(screen.getByText('当前节点：自动分配')).toBeInTheDocument();
    });

    it('不传 activeElementId 时完全不注册 marker（既有调用方零影响）', async () => {
      mockImportXML.mockResolvedValueOnce({});

      render(<BPMNDiagramPreview xml="<definitions />" />);

      await waitFor(() => expect(mockZoom).toHaveBeenCalledWith('fit-viewport'));
      expect(mockAddMarker).not.toHaveBeenCalled();
      expect(mockScrollToElement).not.toHaveBeenCalled();
      expect(screen.queryByText(/当前节点/)).not.toBeInTheDocument();
    });

    it('元素不在图上（流程改版后旧实例）时不打标记也不抛错，图仍正常渲染', async () => {
      mockImportXML.mockResolvedValueOnce({});
      // 实例的 currentActivityId 指向已被新版定义删除的节点
      registryElementIds = ['Activity_Assign'];

      render(<BPMNDiagramPreview xml="<definitions />" activeElementId="Activity_Gone" />);

      await waitFor(() => expect(screen.getByText('100%')).toBeInTheDocument());
      expect(mockAddMarker).not.toHaveBeenCalled();
      expect(mockScrollToElement).not.toHaveBeenCalled();
      // 关键：图必须仍然是 ready（不能因高亮失败让整张图变成错误/空白）
      expect(screen.queryByText('流程图预览失败')).not.toBeInTheDocument();
    });

    it('高亮发生在 fit-viewport 之后（否则滚动位置会被 fit 重置）', async () => {
      mockImportXML.mockResolvedValueOnce({});
      registryElementIds = ['Task_1'];

      render(<BPMNDiagramPreview xml="<definitions />" activeElementId="Task_1" />);

      await waitFor(() => expect(mockAddMarker).toHaveBeenCalled());
      const fitIndex = mockZoom.mock.calls.findIndex(call => call[0] === 'fit-viewport');
      const markerOrder = mockAddMarker.mock.invocationCallOrder[0];
      expect(markerOrder).toBeGreaterThan(mockZoom.mock.invocationCallOrder[fitIndex]);
    });

    it('同一份 XML 切换 activeElementId 时重新导入，旧实例高亮不残留', async () => {
      mockImportXML.mockResolvedValue({});
      registryElementIds = ['A', 'B'];

      const { rerender } = render(<BPMNDiagramPreview xml="<definitions />" activeElementId="A" />);
      await waitFor(() => expect(mockAddMarker).toHaveBeenCalledWith('A', 'bpmn-current-activity'));

      rerender(<BPMNDiagramPreview xml="<definitions />" activeElementId="B" />);

      await waitFor(() => expect(mockAddMarker).toHaveBeenCalledWith('B', 'bpmn-current-activity'));
      // 旧标记由 importXML 重建 DOM 自然清除；关键是必须重新导入而不是只换 marker
      expect(mockImportXML.mock.calls.length).toBeGreaterThanOrEqual(2);
      expect(mockAddMarker).not.toHaveBeenCalledWith('B', undefined);
    });
  });
});
