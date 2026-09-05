/**
 * CIRelationshipsTab - CI 关系标签页测试
 * 注意：CIRelationshipsTab 是一个包装器组件，实际功能在 CIRelationshipManager 中
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { CIRelationshipsTab } from '../sections/CIRelationshipsTab';

const mockRelationshipManager = jest.fn((_props: unknown) => <div data-testid="relationship-manager" />);

jest.mock('../../CIRelationshipManager', () => ({
  __esModule: true,
  default: (props: unknown) => mockRelationshipManager(props),
}));

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('CIRelationshipsTab', () => {
  const defaultProps = {
    ciId: 123,
    ciName: 'Web Server 1',
    onRefresh: jest.fn(),
  };

  it('应该正确渲染组件', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    expect(screen.getByTestId('relationship-manager')).toBeInTheDocument();
  });

  it('应该传递正确的 ciId 和 ciName 给子组件', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    expect(mockRelationshipManager).toHaveBeenLastCalledWith(
      expect.objectContaining({ ciId: 123, ciName: 'Web Server 1' })
    );
  });

  it('应该接收必需的 props', () => {
    const { container } = render(<CIRelationshipsTab {...defaultProps} />);

    expect(container).not.toBeEmptyDOMElement();
  });

  it('onRefresh 函数应该被传递', () => {
    const mockOnRefresh = jest.fn();
    render(<CIRelationshipsTab {...defaultProps} onRefresh={mockOnRefresh} />);

    expect(mockRelationshipManager).toHaveBeenLastCalledWith(
      expect.objectContaining({ onRefresh: mockOnRefresh })
    );
  });

  it('应该处理空状态', () => {
    render(<CIRelationshipsTab {...defaultProps} />);

    expect(screen.getByTestId('relationship-manager')).toBeInTheDocument();
  });
});
