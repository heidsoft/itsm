import { render, screen } from '@testing-library/react';
import { LoadingSpinner } from './LoadingSpinner';

describe('LoadingSpinner', () => {
  it('应该正确渲染', () => {
    render(<LoadingSpinner />);
    expect(screen.getByRole('status')).toBeInTheDocument();
  });

  it('应该支持不同尺寸', () => {
    const { container } = render(<LoadingSpinner size="lg" />);
    expect(container.firstChild).toBeInTheDocument();
  });

  it('应该支持自定义提示文本', () => {
    render(<LoadingSpinner tip="加载中..." />);
    expect(screen.getByText('加载中...')).toBeInTheDocument();
  });

  it('应该有正确的ARIA标签', () => {
    render(<LoadingSpinner />);
    const spinner = screen.getByRole('status');
    expect(spinner).toHaveAttribute('aria-label', '加载中');
  });
});
