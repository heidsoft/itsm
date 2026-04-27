import { render, screen } from '@testing-library/react';
import { StatCard } from './StatCard';
import { FileText } from 'lucide-react';

describe('StatCard', () => {
  // 基础渲染测试
  it('应该正确渲染标题和数值', () => {
    render(<StatCard title="总工单" value={100} />);
    
    expect(screen.getByText('总工单')).toBeInTheDocument();
    expect(screen.getByText('100')).toBeInTheDocument();
  });

  // Props测试
  describe('Props', () => {
    it('应该支持自定义颜色', () => {
      render(<StatCard title="测试" value={50} color="#ff0000" />);
      const valueElement = screen.getByText('50');
      expect(valueElement).toBeInTheDocument();
    });

    it('应该显示图标', () => {
      render(<StatCard title="测试" value={50} icon={<FileText data-testid="icon" />} />);
      expect(screen.getByTestId('icon')).toBeInTheDocument();
    });

    it('应该显示前缀和后缀', () => {
      render(<StatCard title="增长" value={75} prefix="$" suffix="%" />);
      expect(screen.getByText('$')).toBeInTheDocument();
      expect(screen.getByText('%')).toBeInTheDocument();
    });
  });

  // 加载状态测试
  describe('加载状态', () => {
    it('应该显示加载状态', () => {
      render(<StatCard title="加载中" value={0} loading />);
      expect(screen.getByRole('status')).toBeInTheDocument();
    });
  });

  // 可访问性测试
  describe('可访问性', () => {
    it('应该有正确的结构', () => {
      render(<StatCard title="测试" value={100} />);
      
      // 验证标题和数值都存在
      expect(screen.getByText('测试')).toBeInTheDocument();
      expect(screen.getByText('100')).toBeInTheDocument();
    });
  });

  // 边界情况测试
  describe('边界情况', () => {
    it('应该处理0值', () => {
      render(<StatCard title="零值" value={0} />);
      expect(screen.getByText('0')).toBeInTheDocument();
    });

    it('应该处理大数值', () => {
      render(<StatCard title="大数值" value={1000000} />);
      expect(screen.getByText('1,000,000')).toBeInTheDocument();
    });

    it('应该处理负数', () => {
      render(<StatCard title="负数" value={-50} />);
      expect(screen.getByText('-50')).toBeInTheDocument();
    });
  });
});
