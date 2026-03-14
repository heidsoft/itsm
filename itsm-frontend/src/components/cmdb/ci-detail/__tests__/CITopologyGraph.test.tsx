/**
 * CI 拓扑图标签页组件测试
 * CITopologyGraph 是一个包装器，只接收 ciId，内部加载拓扑数据
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { CITopologyGraph } from '../sections/CITopologyGraph';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

// Mock TopologyGraph 组件
jest.mock('../../TopologyGraph', () => {
  return function MockTopologyGraph({ ciId, height }: { ciId: number; height: number }) {
    return (
      <div data-testid="topology-graph">
        <span>Topology Graph for CI: {ciId}</span>
        <span>Height: {height}</span>
      </div>
    );
  };
});

describe('CITopologyGraph', () => {
  const defaultProps = {
    ciId: 123,
  };

  it('应该渲染拓扑图组件', () => {
    render(<CITopologyGraph {...defaultProps} />);

    expect(screen.getByTestId('topology-graph')).toBeInTheDocument();
  });

  it('应该传递正确的 ciId 给子组件', () => {
    render(<CITopologyGraph {...defaultProps} />);

    expect(screen.getByText(/Topology Graph for CI: 123/)).toBeInTheDocument();
  });

  it('应该设置正确的高度', () => {
    render(<CITopologyGraph {...defaultProps} />);

    expect(screen.getByText(/Height: 500/)).toBeInTheDocument();
  });

  it('应该接收必需的 ciId prop', () => {
    const { container } = render(<CITopologyGraph ciId={123} />);

    expect(container).toBeTruthy();
  });

  it('ciId 变化时应重新渲染', () => {
    const { rerender } = render(<CITopologyGraph ciId={123} />);

    expect(screen.getByText(/CI: 123/)).toBeInTheDocument();

    rerender(<CITopologyGraph ciId={456} />);

    expect(screen.getByText(/CI: 456/)).toBeInTheDocument();
  });
});
