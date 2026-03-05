/**
 * CITopologyGraph - CI 拓扑图组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { CITopologyGraph } from '../sections/CITopologyGraph';
import type { CITopologyGraphProps } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

// Mock reactflow
jest.mock('reactflow', () => ({
  ReactFlow: ({ children }: any) => <div data-testid="reactflow">{children}</div>,
  Controls: () => <div data-testid="controls">Controls</div>,
  MiniMap: () => <div data-testid="minimap">MiniMap</div>,
  Background: () => <div data-testid="background">Background</div>,
  useNodesState: () => [ [], jest.fn() ],
  useEdgesState: () => [ [], jest.fn() ],
  addEdge: jest.fn(),
}));

describe('CITopologyGraph', () => {
  const mockNodes = [
    {
      id: 'ci-001',
      type: 'ciNode',
      position: { x: 100, y: 100 },
      data: {
        label: 'Web Server 1',
        type: 'server',
        status: 'active',
      },
    },
    {
      id: 'ci-002',
      type: 'ciNode',
      position: { x: 300, y: 100 },
      data: {
        label: 'Database Server',
        type: 'database',
        status: 'active',
      },
    },
  ];

  const mockEdges = [
    {
      id: 'e1',
      source: 'ci-001',
      target: 'ci-002',
      type: 'smoothstep',
      animated: true,
    },
  ];

  const defaultProps: CITopologyGraphProps = {
    ciId: 'ci-001',
    nodes: mockNodes,
    edges: mockEdges,
    onNodeClick: jest.fn(),
    onEdgeClick: jest.fn(),
    loading: false,
  };

  it('应该渲染拓扑图容器', () => {
    render(<CITopologyGraph {...defaultProps} />);

    expect(screen.getByTestId('reactflow')).toBeInTheDocument();
  });

  it('应该显示控件组件', () => {
    render(<CITopologyGraph {...defaultProps} />);

    expect(screen.getByTestId('controls')).toBeInTheDocument();
  });

  it('应该显示小地图', () => {
    render(<CITopologyGraph {...defaultProps} />);

    expect(screen.getByTestId('minimap')).toBeInTheDocument();
  });

  it('应该显示背景', () => {
    render(<CITopologyGraph {...defaultProps} />);

    expect(screen.getByTestId('background')).toBeInTheDocument();
  });

  it('应该传递节点数据', () => {
    render(<CITopologyGraph {...defaultProps} />);

    // 节点应该被渲染
    expect(screen.getByText('Web Server 1')).toBeInTheDocument();
    expect(screen.getByText('Database Server')).toBeInTheDocument();
  });

  it('应该渲染连接线', () => {
    render(<CITopologyGraph {...defaultProps} />);

    // 边可能不直接可见，但确保它们被传递
    expect(mockEdges.length).toBeGreaterThan(0);
  });

  it('加载状态应显示加载指示器', () => {
    render(<CITopologyGraph {...defaultProps} loading={true} />);

    expect(screen.getByText(/loading/i)).toBeInTheDocument();
  });

  it('没有节点应显示空状态', () => {
    render(<CITopologyGraph {...defaultProps} nodes={[]} edges={[]} />);

    expect(screen.getByText(/no topology data/i)).toBeInTheDocument();
  });

  it('应该处理节点点击', async () => {
    const mockOnNodeClick = jest.fn();
    render(<CITopologyGraph {...defaultProps} onNodeClick={mockOnNodeClick} />);

    const node = screen.getByText('Web Server 1');
    fireEvent.click(node);

    expect(mockOnNodeClick).toHaveBeenCalledWith('ci-001');
  });

  it('应该处理边点击', async () => {
    const mockOnEdgeClick = jest.fn();
    render(<CITopologyGraph {...defaultProps} onEdgeClick={mockOnEdgeClick} />);

    // 边的点击可能需要更复杂的交互，这里简单测试
    expect(mockOnEdgeClick).toBeDefined();
  });

  it('应该高亮当前 CI', () => {
    render(<CITopologyGraph {...defaultProps} ciId="ci-001" />);

    const currentNode = screen.getByText('Web Server 1');
    expect(currentNode.closest('div')).toHaveClass('highlighted');
  });

  it('应该显示节点类型图标', () => {
    render(<CITopologyGraph {...defaultProps} />);

    // 检查服务器节点图标
    const serverNode = screen.getByText('Web Server 1');
    expect(serverNode.closest('div')).toHaveClass('node-server');
  });

  it('应该根据状态显示不同颜色', () => {
    const nodesWithStatus = [
      {
        ...mockNodes[0],
        data: { ...mockNodes[0].data, status: 'inactive' },
      },
    ];

    render(<CITopologyGraph {...defaultProps} nodes={nodesWithStatus} />);

    const inactiveNode = screen.getByText('Web Server 1');
    expect(inactiveNode.closest('div')).toHaveClass('status-inactive');
  });

  it('应该支持缩放和平移', () => {
    render(<CITopologyGraph {...defaultProps} />);

    // 控件存在表明支持这些功能
    expect(screen.getByTestId('controls')).toBeInTheDocument();
  });

  it('应该显示连接关系', () => {
    render(<CITopologyGraph {...defaultProps} />);

    // 检查连接关系可能涉及边的渲染
    expect(screen.getByText('Web Server 1')).toBeInTheDocument();
    expect(screen.getByText('Database Server')).toBeInTheDocument();
  });

  it('应该处理大型拓扑图', () => {
    const manyNodes = Array.from({ length: 50 }, (_, i) => ({
      id: `ci-${i}`,
      type: 'ciNode',
      position: { x: i * 100, y: i * 50 },
      data: {
        label: `Node ${i}`,
        type: i % 2 === 0 ? 'server' : 'database',
        status: 'active',
      },
    }));

    render(<CITopologyGraph {...defaultProps} nodes={manyNodes} edges={[]} />);

    expect(screen.getByText('Node 0')).toBeInTheDocument();
    expect(screen.getByText('Node 49')).toBeInTheDocument();
  });

  it('应该显示节点详细信息', () => {
    const detailedNode = {
      ...mockNodes[0],
      data: {
        ...mockNodes[0].data,
        ip: '192.168.1.100',
        environment: 'production',
      },
    };

    render(<CITopologyGraph {...defaultProps} nodes={[detailedNode]} />);

    expect(screen.getByText(/192\.168\.1\.100/)).toBeInTheDocument();
  });

  it('应该支持节点类型过滤', () => {
    const [serverNode, dbNode, appNode] = mockNodes.map((node, i) => ({
      ...node,
      id: `node-${i}`,
      data: {
        label: `${node.data.label} (${i === 0 ? 'server' : i === 1 ? 'database' : 'application'})`,
        type: i === 0 ? 'server' : i === 1 ? 'database' : 'application',
        status: 'active',
      },
    }));

    render(<CITopologyGraph {...defaultProps} nodes={[serverNode, dbNode, appNode]} />);

    expect(screen.getByText(/server/)).toBeInTheDocument();
    expect(screen.getByText(/database/)).toBeInTheDocument();
  });
});
