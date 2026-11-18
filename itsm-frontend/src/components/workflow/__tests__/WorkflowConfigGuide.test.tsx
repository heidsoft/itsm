import React from 'react';
import { render, screen } from '@testing-library/react';
import { WorkflowConfigGuide } from '../WorkflowConfigGuide';

describe('WorkflowConfigGuide', () => {
  it('marks the current surface and links to the other two', () => {
    render(<WorkflowConfigGuide current="approval" />);

    expect(screen.getByText('你在这里：配置审批链')).toBeInTheDocument();
    expect(screen.getByText('当前')).toBeInTheDocument();

    const workflows = screen.getByRole('link', { name: /流程定义/ });
    expect(workflows).toHaveAttribute('href', '/admin/workflows');
    const types = screen.getByRole('link', { name: /工单类型/ });
    expect(types).toHaveAttribute('href', '/tickets/types');

    // 当前面（审批链配置）不是跳转链接
    expect(screen.queryByRole('link', { name: /审批链配置/ })).not.toBeInTheDocument();
  });

  it('highlights the binding surface when current=binding', () => {
    render(<WorkflowConfigGuide current="binding" />);
    expect(screen.getByText('你在这里：绑定工单类型')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /流程定义/ })).toHaveAttribute('href', '/admin/workflows');
    expect(screen.getByRole('link', { name: /审批链配置/ })).toHaveAttribute('href', '/admin/approval-chains');
    expect(screen.queryByRole('link', { name: /工单类型/ })).not.toBeInTheDocument();
  });

  it('renders all three steps in the navigator', () => {
    render(<WorkflowConfigGuide current="designer" />);
    expect(screen.getByText('设计流程')).toBeInTheDocument();
    expect(screen.getByText('配置审批链')).toBeInTheDocument();
    expect(screen.getByText('绑定工单类型')).toBeInTheDocument();
  });
});