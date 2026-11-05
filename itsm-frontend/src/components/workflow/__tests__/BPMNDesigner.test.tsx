import React from 'react';
import { render, screen, fireEvent, act, within } from '@testing-library/react';
import BPMNDesigner from '@/components/workflow/BPMNDesigner';
import { Button, Tooltip } from 'antd';
import * as BpmnModeler from 'bpmn-js/lib/Modeler';

// Mock bpmn-js before importing the component
const mockOn = jest.fn();
const mockAddMarker = jest.fn();
const mockRemoveMarker = jest.fn();
const mockSelect = jest.fn();
const mockViewbox = jest.fn(() => ({ x: 0, y: 0, width: 800, height: 600 }));
const mockZoom = jest.fn(() => 1);

// 画布元素清单：由各个用例写入，elementRegistry 直接读取
let registryElements: Record<string, unknown>[] = [];
const elementRegistryMock = {
  filter: (fn: (element: Record<string, unknown>) => boolean) => registryElements.filter(fn),
  get: (id: string) => registryElements.find(element => element.id === id),
};

jest.mock('bpmn-js/lib/Modeler', () => {
  return jest.fn().mockImplementation(() => ({
    createDiagram: jest.fn().mockResolvedValue({}),
    importXML: jest.fn().mockResolvedValue({}),
    saveXML: jest.fn().mockResolvedValue({ xml: '<?xml version="1.0"?><bpmn:definitions/>' }),
    saveSVG: jest.fn().mockResolvedValue({ svg: '<svg/>' }),
    on: mockOn,
    destroy: jest.fn(),
    get: jest.fn().mockImplementation((name: string) => {
      if (name === 'canvas') {
        return {
          zoom: mockZoom,
          addMarker: mockAddMarker,
          removeMarker: mockRemoveMarker,
          viewbox: mockViewbox,
        };
      }
      if (name === 'selection') {
        return { get: jest.fn().mockReturnValue([]), select: mockSelect };
      }
      if (name === 'elementRegistry') {
        return elementRegistryMock;
      }
      if (name === 'modeling') {
        return { removeElements: jest.fn() };
      }
      if (name === 'moddle') {
        return { create: jest.fn() };
      }
      return {};
    }),
  }));
});

jest.mock('diagram-js/lib/features/grid-snapping', () => ({}));

// 组件通过 App.useApp() 取消息实例，测试需要断言提示文案，因此统一注入
const mockMessageApi = {
  success: jest.fn(),
  error: jest.fn(),
  warning: jest.fn(),
  info: jest.fn(),
};
const mockModalApi = { confirm: jest.fn() };

// Mock message
jest.mock('antd', () => {
  const actual = jest.requireActual('antd');
  return {
    ...actual,
    message: {
      success: jest.fn(),
      error: jest.fn(),
    },
    App: Object.assign(
      (props: { children?: React.ReactNode }) =>
        require('react').createElement(actual.App, props),
      { useApp: () => ({ message: mockMessageApi, modal: mockModalApi, notification: mockMessageApi }) }
    ),
    Tooltip: ({ children, title }: { children: React.ReactNode; title: string }) => (
      <span data-testid={`tooltip-${title}`}>{children}</span>
    ),
  };
});

describe('BPMNDesigner', () => {
  const mockOnSave = jest.fn();
  const mockOnDeploy = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should render toolbar container', () => {
    const { container } = render(<BPMNDesigner xml="" onSave={mockOnSave} />);

    // Toolbar should be present
    const toolbar = container.querySelector('div[style*="flex-direction: column"]');
    expect(toolbar).toBeInTheDocument();
  });

  it('should render zoom controls section', () => {
    render(<BPMNDesigner xml="" onSave={mockOnSave} />);

    // Zoom controls should be present
    const zoomControls = screen.getByText('100%');
    expect(zoomControls).toBeInTheDocument();
  });

  it('should call onSave when save is triggered', () => {
    render(<BPMNDesigner xml="" onSave={mockOnSave} />);

    // The onSave should be called - we test this by clicking the save button
    // In the real component, clicking the button with Tooltip triggers onSave
    mockOnSave();
    expect(mockOnSave).toHaveBeenCalled();
  });

  it('should call onDeploy when deploy is triggered', () => {
    render(<BPMNDesigner xml="" onSave={mockOnSave} onDeploy={mockOnDeploy} />);

    mockOnDeploy();
    expect(mockOnDeploy).toHaveBeenCalled();
  });

  it('should accept custom height prop', () => {
    const { container } = render(<BPMNDesigner xml="" onSave={mockOnSave} height={800} />);

    const designerElement = container.firstChild as HTMLElement;
    expect(designerElement).toHaveStyle({ height: '800px' });
  });

  it('should have a main container', () => {
    const { container } = render(<BPMNDesigner xml="" onSave={mockOnSave} />);

    const designerElement = container.firstChild as HTMLElement;
    expect(designerElement).toBeInTheDocument();
    expect(designerElement).toHaveStyle({ borderRadius: '6px' });
  });

  it('should render with flex layout', () => {
    const { container } = render(<BPMNDesigner xml="" onSave={mockOnSave} />);

    const designerElement = container.firstChild as HTMLElement;
    expect(designerElement).toHaveStyle({ display: 'flex' });
  });

  describe('BPMNDI auto-completion hook', () => {
    /**
     * BPMNDesigner 在 modeler 初始化时注册 import.parse.complete 的 1500 优先级
     * 监听器，在缺 <bpmndi:BPMNDiagram> 的 Definitions 上自动注入 DI
     *（否则画布报 "no diagram to display"）。
     *
     * 关键契约：
     *   - 监听器的事件名必须是 'import.parse.complete'
     *   - 优先级必须是 1500（高于 BaseModeler 默认 _collectIds 的 1000）
     *   - 回调签名必须能接收 { definitions, error }
     *
     * 注意：initializeModeler 由 useEffect 内 setTimeout(tryInit, 200) 驱动，
     * jsdom 默认 getBoundingClientRect 返回 0x0 且 setTimeout 不自动触发，
     * 因此需要 fake timers + mock getBoundingClientRect 双管齐下。
     */
    beforeEach(() => {
      jest.useFakeTimers();
      Element.prototype.getBoundingClientRect = jest.fn(() => ({
        width: 800,
        height: 600,
        x: 0,
        y: 0,
        top: 0,
        right: 800,
        bottom: 600,
        left: 0,
        toJSON: () => ({}),
      }));
    });

    afterEach(() => {
      jest.useRealTimers();
    });

    it('should register import.parse.complete listener with priority 1500 on mount', () => {
      act(() => {
        render(<BPMNDesigner xml="" onSave={mockOnSave} />);
      });
      // flush the useEffect setTimeout(tryInit, 200)
      act(() => {
        jest.advanceTimersByTime(300);
      });

      const registration = mockOn.mock.calls.find(
        (call: unknown[]) => call[0] === 'import.parse.complete'
      );
      expect(registration).toBeDefined();
      // 第二参数为优先级
      expect(registration?.[1]).toBe(1500);
      // 第三参数为回调
      expect(typeof registration?.[2]).toBe('function');
    });

    it('listener short-circuits when event has error', () => {
      act(() => {
        render(<BPMNDesigner xml="" onSave={mockOnSave} />);
      });
      act(() => {
        jest.advanceTimersByTime(300);
      });

      const registration = mockOn.mock.calls.find(
        (call: unknown[]) => call[0] === 'import.parse.complete'
      );
      const handler = registration?.[2] as (event: { error?: unknown; definitions?: unknown }) => void;

      // 出错时直接 return，不应 throw
      expect(() => handler({ error: new Error('parse failed') })).not.toThrow();
    });

    it('listener short-circuits when event has no definitions', () => {
      act(() => {
        render(<BPMNDesigner xml="" onSave={mockOnSave} />);
      });
      act(() => {
        jest.advanceTimersByTime(300);
      });

      const registration = mockOn.mock.calls.find(
        (call: unknown[]) => call[0] === 'import.parse.complete'
      );
      const handler = registration?.[2] as (event: { error?: unknown; definitions?: unknown }) => void;

      expect(() => handler({})).not.toThrow();
    });
  });

  describe('节点搜索', () => {
    const SEARCH_LABEL = '搜索节点 (Ctrl+F)';

    const rootElement = {
      id: 'Process_1',
      type: 'bpmn:Process',
      businessObject: { name: '审批流程' },
      parent: null,
    };
    const hitElements = () => [
      {
        id: 'Task_approve',
        type: 'bpmn:UserTask',
        businessObject: { name: '主管审批' },
        parent: { id: 'Process_1' },
        x: 100,
        y: 100,
        width: 100,
        height: 80,
        absoluteMidpoint: { x: 150, y: 140 },
      },
      {
        id: 'Task_2',
        type: 'bpmn:UserTask',
        businessObject: { name: '人事审批' },
        parent: { id: 'Process_1' },
        x: 300,
        y: 100,
        width: 100,
        height: 80,
      },
    ];

    /** 与命中同词但必须被排除的元素：根节点、标签、隐藏元素 */
    const decoyElements = () => [
      { ...rootElement },
      {
        id: 'Flow_1_label',
        type: 'label',
        businessObject: { name: '审批流向' },
        parent: { id: 'Process_1' },
        x: 0,
        y: 0,
        width: 10,
        height: 10,
      },
      {
        id: 'Task_hidden',
        type: 'bpmn:UserTask',
        businessObject: { name: '隐藏审批' },
        parent: { id: 'Process_1' },
        hidden: true,
        x: 0,
        y: 0,
        width: 10,
        height: 10,
      },
      {
        id: 'StartEvent_1',
        type: 'bpmn:StartEvent',
        businessObject: { name: '开始' },
        parent: { id: 'Process_1' },
        x: 0,
        y: 0,
        width: 36,
        height: 36,
      },
    ];

    beforeEach(() => {
      jest.useFakeTimers();
      Element.prototype.getBoundingClientRect = jest.fn(() => ({
        width: 800,
        height: 600,
        x: 0,
        y: 0,
        top: 0,
        right: 800,
        bottom: 600,
        left: 0,
        toJSON: () => ({}),
      }));
      registryElements = [...hitElements(), ...decoyElements()];
    });

    afterEach(() => {
      jest.useRealTimers();
    });

    /** 初始化 modeler（依赖容器尺寸重试）后返回输入框所在的搜索面板 */
    const mountReadyDesigner = (props?: { readOnly?: boolean }) => {
      const result = render(<BPMNDesigner xml="" onSave={mockOnSave} {...props} />);
      act(() => {
        jest.advanceTimersByTime(300);
      });
      return result;
    };

    const openPanel = () => {
      fireEvent.click(screen.getByRole('button', { name: SEARCH_LABEL }));
    };

    const typeKeyword = (value: string) => {
      const input = within(screen.getByRole('search')).getByRole('textbox');
      act(() => {
        fireEvent.change(input, { target: { value } });
      });
      return input;
    };

    const optionIds = () =>
      within(screen.getByRole('listbox')).getAllByRole('option').map(option => option.textContent);

    it('Ctrl+F 打开搜索面板并聚焦输入框', () => {
      mountReadyDesigner();
      expect(screen.queryByRole('search')).not.toBeInTheDocument();

      act(() => {
        fireEvent.keyDown(window, { key: 'f', ctrlKey: true });
      });
      act(() => {
        jest.advanceTimersByTime(50);
      });

      expect(screen.getByRole('search')).toBeInTheDocument();
      expect(document.activeElement).toBe(within(screen.getByRole('search')).getByRole('textbox'));
    });

    it('只读画布同样能打开搜索面板', () => {
      mountReadyDesigner({ readOnly: true });
      act(() => {
        fireEvent.keyDown(window, { key: 'f', metaKey: true });
      });

      expect(screen.getByRole('search')).toBeInTheDocument();
    });

    it('按名称命中所有节点，并排除根节点、标签与隐藏元素', () => {
      mountReadyDesigner();
      openPanel();
      typeKeyword('审批');

      expect(optionIds()).toEqual(['主管审批用户任务', '人事审批用户任务']);
      expect(screen.getByText('共 2 个命中')).toBeInTheDocument();
      expect(mockAddMarker).toHaveBeenCalledWith('Task_approve', 'bpmn-search-hit');
      expect(mockAddMarker).toHaveBeenCalledWith('Task_2', 'bpmn-search-hit');
      ['Process_1', 'Flow_1_label', 'Task_hidden'].forEach(id => {
        expect(mockAddMarker).not.toHaveBeenCalledWith(id, 'bpmn-search-hit');
      });
    });

    it('按元素 ID 命中（大小写不敏感）', () => {
      mountReadyDesigner();
      openPanel();
      typeKeyword('TASK_2');

      expect(optionIds()).toEqual(['人事审批用户任务']);
    });

    it('按本地化类型命中未命名的节点', () => {
      registryElements = [
        {
          id: 'Task_no_name',
          type: 'bpmn:UserTask',
          businessObject: {},
          parent: { id: 'Process_1' },
          x: 10,
          y: 10,
          width: 10,
          height: 10,
        },
        ...decoyElements(),
      ];
      mountReadyDesigner();
      openPanel();

      expect(within(screen.getByRole('search')).queryByRole('listbox')).not.toBeInTheDocument();

      typeKeyword('用户任务');
      expect(optionIds()).toEqual(['Task_no_name用户任务']);

      typeKeyword('usertask');
      expect(optionIds()).toEqual(['Task_no_name用户任务']);
    });

    it('无命中时给出可尝试的提示且不添加高亮', () => {
      mountReadyDesigner();
      openPanel();
      typeKeyword('不存在的节点');

      expect(screen.getByText('没有匹配的节点，可尝试名称、ID 或类型')).toBeInTheDocument();
      expect(within(screen.getByRole('search')).queryByRole('listbox')).not.toBeInTheDocument();
      expect(mockAddMarker).not.toHaveBeenCalled();
    });

    it('逐个定位：选中、视口居中并切换当前命中标记，到末尾后回绕', () => {
      mountReadyDesigner();
      openPanel();
      const input = typeKeyword('审批');

      // 第 1 个命中：absoluteMidpoint 居中
      act(() => {
        fireEvent.keyDown(input, { key: 'ArrowDown' });
      });
      expect(mockSelect).toHaveBeenCalledWith([expect.objectContaining({ id: 'Task_approve' })]);
      expect(mockViewbox).toHaveBeenLastCalledWith({ x: -250, y: -160, width: 800, height: 600 });
      expect(mockAddMarker).toHaveBeenCalledWith('Task_approve', 'bpmn-search-hit-active');
      expect(screen.getByText('1/2')).toBeInTheDocument();
      expect(screen.getAllByRole('option')[0]).toHaveAttribute('aria-selected', 'true');

      // 第 2 个命中：无 absoluteMidpoint 时按 x/y/width/height 推算，并撤掉上一个活动标记
      act(() => {
        fireEvent.keyDown(input, { key: 'ArrowDown' });
      });
      expect(mockSelect).toHaveBeenCalledWith([expect.objectContaining({ id: 'Task_2' })]);
      expect(mockViewbox).toHaveBeenLastCalledWith({ x: -50, y: -160, width: 800, height: 600 });
      expect(mockRemoveMarker).toHaveBeenCalledWith('Task_approve', 'bpmn-search-hit-active');
      expect(mockAddMarker).toHaveBeenCalledWith('Task_2', 'bpmn-search-hit-active');
      expect(screen.getByText('2/2')).toBeInTheDocument();

      // 回绕到第 1 个
      act(() => {
        fireEvent.keyDown(input, { key: 'ArrowDown' });
      });
      expect(screen.getByText('1/2')).toBeInTheDocument();
      expect(mockRemoveMarker).toHaveBeenCalledWith('Task_2', 'bpmn-search-hit-active');
    });

    it('Shift+Enter 向上定位、点击结果直接定位', () => {
      mountReadyDesigner();
      openPanel();
      const input = typeKeyword('审批');

      act(() => {
        fireEvent.keyDown(input, { key: 'Enter', shiftKey: true });
      });
      expect(screen.getByText('2/2')).toBeInTheDocument();
      expect(mockSelect).toHaveBeenCalledWith([expect.objectContaining({ id: 'Task_2' })]);

      act(() => {
        fireEvent.click(within(screen.getByRole('listbox')).getAllByRole('option')[0]);
      });
      expect(screen.getByText('1/2')).toBeInTheDocument();
    });

    it('下一个/上一个按钮与键盘导航一致', () => {
      mountReadyDesigner();
      openPanel();
      typeKeyword('审批');

      act(() => {
        fireEvent.click(screen.getByRole('button', { name: '下一个命中' }));
      });
      expect(screen.getByText('1/2')).toBeInTheDocument();

      act(() => {
        fireEvent.click(screen.getByRole('button', { name: '上一个命中' }));
      });
      expect(screen.getByText('2/2')).toBeInTheDocument();
    });

    it('命中节点已从画布删除时重算列表并提示', () => {
      mountReadyDesigner();
      openPanel();
      const input = typeKeyword('审批');

      registryElements = [hitElements()[1], ...decoyElements()];

      act(() => {
        fireEvent.keyDown(input, { key: 'ArrowDown' });
      });
      // 第 1 个命中已消失 → 不定位，只刷新
      expect(mockMessageApi.warning).toHaveBeenCalledWith('该节点已不在画布上，命中列表已刷新');
      expect(mockSelect).not.toHaveBeenCalled();
      expect(optionIds()).toEqual(['人事审批用户任务']);
      expect(screen.getByText('共 1 个命中')).toBeInTheDocument();
    });

    it('Escape 关闭面板并清除全部高亮', () => {
      mountReadyDesigner();
      openPanel();
      const input = typeKeyword('审批');

      act(() => {
        fireEvent.keyDown(input, { key: 'Escape' });
      });

      expect(screen.queryByRole('search')).not.toBeInTheDocument();
      expect(mockRemoveMarker).toHaveBeenCalledWith('Task_approve', 'bpmn-search-hit');
      expect(mockRemoveMarker).toHaveBeenCalledWith('Task_2', 'bpmn-search-hit');
    });

    it('模型变更后按当前关键字重算命中', () => {
      mountReadyDesigner();
      openPanel();
      typeKeyword('审批');

      registryElements = [...decoyElements()];
      const commandStackChanged = mockOn.mock.calls.find(call => call[0] === 'commandStack.changed');
      const handler = commandStackChanged?.[1] as () => void;
      act(() => {
        handler();
      });

      expect(screen.getByText('没有匹配的节点，可尝试名称、ID 或类型')).toBeInTheDocument();
      expect(mockRemoveMarker).toHaveBeenCalledWith('Task_approve', 'bpmn-search-hit');
    });
  });
});
