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
let viewerInstances = 0;

jest.mock('bpmn-js/lib/NavigatedViewer', () =>
  jest.fn().mockImplementation(() => {
    viewerInstances += 1;
    return {
      importXML: (...args: unknown[]) => mockImportXML(...args),
      on: mockOn,
      destroy: mockDestroy,
      get: (name: string) => (name === 'canvas' ? { zoom: mockZoom } : {}),
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
});
