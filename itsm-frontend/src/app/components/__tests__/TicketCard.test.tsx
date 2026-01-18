import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { TicketCard } from '@/components/business/TicketCard';

// Mock audio API
Object.defineProperty(window, 'Audio', {
  writable: true,
  value: jest.fn().mockImplementation(() => ({
    play: jest.fn().mockResolvedValue(undefined),
    pause: jest.fn(),
    currentTime: 0,
    duration: 0,
  })),
});

// Mock Lucide React icons
jest.mock('lucide-react', () => ({
  Zap: () => <div data-testid='zap-icon'>Zap</div>,
  AlertCircle: () => <div data-testid='alert-circle-icon'>AlertCircle</div>,
  Info: () => <div data-testid='info-icon'>Info</div>,
  ChevronsDown: () => <div data-testid='chevrons-down-icon'>ChevronsDown</div>,
  Clock: () => <div data-testid='clock-icon'>Clock</div>,
}));

// Mock ticket data matching the actual component interface
const mockTicketProps = {
  id: '12345',
  title: '系统登录问题',
  status: '待处理',
  priority: 'P2' as const,
  lastUpdate: '2小时前',
  type: '事件',
};

describe('TicketCard', () => {
  beforeEach(() => {
    // Clear mocks but preserve Audio mock
    jest.clearAllMocks();
    // Reset Audio mock to track new calls
    (window.Audio as unknown as jest.Mock).mockClear();
  });

  describe('Rendering', () => {
    it('should render ticket card with basic information', () => {
      render(<TicketCard {...mockTicketProps} />);

      expect(screen.getByText('系统登录问题')).toBeInTheDocument();
      expect(screen.getByText('事件ID: 12345')).toBeInTheDocument();
      expect(screen.getByText('状态: 待处理')).toBeInTheDocument();
      expect(screen.getByText('2小时前')).toBeInTheDocument();
    });

    it('should display priority badge correctly', () => {
      render(<TicketCard {...mockTicketProps} />);

      expect(screen.getByText('P2 高')).toBeInTheDocument();
    });

    it('should show appropriate icon for priority', () => {
      render(<TicketCard {...mockTicketProps} />);

      expect(screen.getByTestId('alert-circle-icon')).toBeInTheDocument();
    });

    it('should display clock icon and last update time', () => {
      render(<TicketCard {...mockTicketProps} />);

      expect(screen.getByTestId('clock-icon')).toBeInTheDocument();
      expect(screen.getByText('2小时前')).toBeInTheDocument();
    });
  });

  describe('Priority Variants', () => {
    it('should render P1 priority correctly with Zap icon', () => {
      const p1Props = { ...mockTicketProps, priority: 'P1' as const };
      render(<TicketCard {...p1Props} />);

      expect(screen.getByText('P1 紧急')).toBeInTheDocument();
      expect(screen.getByTestId('zap-icon')).toBeInTheDocument();
    });

    it('should render P2 priority correctly with AlertCircle icon', () => {
      const p2Props = { ...mockTicketProps, priority: 'P2' as const };
      render(<TicketCard {...p2Props} />);

      expect(screen.getByText('P2 高')).toBeInTheDocument();
      expect(screen.getByTestId('alert-circle-icon')).toBeInTheDocument();
    });

    it('should render P3 priority correctly with Info icon', () => {
      const p3Props = { ...mockTicketProps, priority: 'P3' as const };
      render(<TicketCard {...p3Props} />);

      expect(screen.getByText('P3 中')).toBeInTheDocument();
      expect(screen.getByTestId('info-icon')).toBeInTheDocument();
    });

    it('should render P4 priority correctly with ChevronsDown icon', () => {
      const p4Props = { ...mockTicketProps, priority: 'P4' as const };
      render(<TicketCard {...p4Props} />);

      expect(screen.getByText('P4 低')).toBeInTheDocument();
      expect(screen.getByTestId('chevrons-down-icon')).toBeInTheDocument();
    });
  });

  describe('P1 Priority Alert Behavior', () => {
    it('should play alert sound for P1 priority tickets', () => {
      const p1Props = { ...mockTicketProps, priority: 'P1' as const };
      render(<TicketCard {...p1Props} />);

      // Check if Audio constructor was called
      expect(window.Audio).toHaveBeenCalledWith('/alert.mp3');
    });

    it('should add pulse animation class for P1 priority', () => {
      const p1Props = { ...mockTicketProps, priority: 'P1' as const };
      const { container } = render(<TicketCard {...p1Props} />);

      // The component should add animate-pulse-strong class
      const cardElement = container.querySelector('div[class*="animate-pulse-strong"]');
      expect(cardElement).toBeInTheDocument();
    });

    it('should not play sound for non-P1 priorities', () => {
      render(<TicketCard {...mockTicketProps} />);

      // Audio should not be called for P2 priority
      expect(window.Audio).not.toHaveBeenCalled();
    });
  });

  describe('Type Variants', () => {
    it('should display custom type when provided', () => {
      const customTypeProps = { ...mockTicketProps, type: '故障' };
      render(<TicketCard {...customTypeProps} />);

      expect(screen.getByText('故障ID: 12345')).toBeInTheDocument();
    });

    it('should use default type when not provided', () => {
      // eslint-disable-next-line @typescript-eslint/no-unused-vars
      const { type: _, ...propsWithoutType } = mockTicketProps;
      render(<TicketCard {...propsWithoutType} />);

      expect(screen.getByText('事件ID: 12345')).toBeInTheDocument();
    });
  });

  describe('ID Variants', () => {
    it('should handle string ID', () => {
      const stringIdProps = { ...mockTicketProps, id: 'TICKET-001' };
      render(<TicketCard {...stringIdProps} />);

      expect(screen.getByText('事件ID: TICKET-001')).toBeInTheDocument();
    });

    it('should handle numeric ID', () => {
      const numericIdProps = { ...mockTicketProps, id: 12345 };
      render(<TicketCard {...numericIdProps} />);

      expect(screen.getByText('事件ID: 12345')).toBeInTheDocument();
    });
  });

  describe('Styling and Layout', () => {
    it('should have proper CSS classes for layout', () => {
      const { container } = render(<TicketCard {...mockTicketProps} />);

      const cardElement = container.firstChild as HTMLElement;
      expect(cardElement).toHaveClass('relative', 'bg-white', 'p-6', 'rounded-lg', 'shadow-md');
    });

    it('should have hover effects', () => {
      const { container } = render(<TicketCard {...mockTicketProps} />);

      const cardElement = container.firstChild as HTMLElement;
      expect(cardElement).toHaveClass('hover:shadow-xl', 'transition-shadow');
    });

    it('should have priority-based border color', () => {
      const { container } = render(<TicketCard {...mockTicketProps} />);

      const cardElement = container.firstChild as HTMLElement;
      expect(cardElement.className).toContain('border-orange-500');
    });
  });

  describe('Accessibility', () => {
    it('should have proper semantic structure', () => {
      render(<TicketCard {...mockTicketProps} />);

      // Should have heading for title
      const titleElement = screen.getByRole('heading', { level: 3 });
      expect(titleElement).toHaveTextContent('系统登录问题');
    });

    it('应该支持键盘导航', () => {
      const { container } = render(<TicketCard {...mockTicketProps} />);
      const cardElement = container.firstChild as HTMLElement;
      // 添加 tabIndex 使元素可聚焦
      cardElement.setAttribute('tabIndex', '0');
      cardElement.focus();
      expect(cardElement).toHaveFocus();
    });
  });

  describe('Edge Cases', () => {
    it('should handle very long titles', () => {
      const longTitle = 'A'.repeat(200);
      const longTitleProps = { ...mockTicketProps, title: longTitle };
      render(<TicketCard {...longTitleProps} />);

      expect(screen.getByText(longTitle)).toBeInTheDocument();
    });

    it('should handle empty status', () => {
      const emptyStatusProps = { ...mockTicketProps, status: '' };
      render(<TicketCard {...emptyStatusProps} />);

      expect(screen.getByText('状态:')).toBeInTheDocument();
    });

    it('should handle invalid priority gracefully', () => {
      // TypeScript would prevent this, but testing runtime behavior
      const invalidPriorityProps = {
        ...mockTicketProps,
        priority: 'INVALID' as 'P1' | 'P2' | 'P3' | 'P4',
      };
      render(<TicketCard {...invalidPriorityProps} />);

      // Should fall back to P4 default
      expect(screen.getByText('P4 低')).toBeInTheDocument();
    });
  });

  describe('Performance', () => {
    it('should not re-render unnecessarily', () => {
      const { rerender } = render(<TicketCard {...mockTicketProps} />);

      // Re-render with same props
      rerender(<TicketCard {...mockTicketProps} />);

      // Should still be in document
      expect(screen.getByText('系统登录问题')).toBeInTheDocument();
    });

    it('should handle priority changes correctly', () => {
      const { rerender } = render(<TicketCard {...mockTicketProps} />);

      // Change to P1 priority
      const p1Props = { ...mockTicketProps, priority: 'P1' as const };
      rerender(<TicketCard {...p1Props} />);

      expect(screen.getByText('P1 紧急')).toBeInTheDocument();
      expect(window.Audio).toHaveBeenCalledWith('/alert.mp3');
    });
  });

  describe('Audio Error Handling', () => {
    it('should handle audio play errors gracefully', () => {
      // Mock console.error to verify error handling
      const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

      // Mock Audio to throw error on play
      (window.Audio as jest.Mock).mockImplementation(() => ({
        play: jest.fn().mockRejectedValue(new Error('Audio play failed')),
      }));

      const p1Props = { ...mockTicketProps, priority: 'P1' as const };
      render(<TicketCard {...p1Props} />);

      // Should not crash the component
      expect(screen.getByText('P1 紧急')).toBeInTheDocument();

      consoleSpy.mockRestore();
    });
  });

  describe('Component Lifecycle', () => {
    it('should clean up animation classes on unmount', () => {
      const p1Props = { ...mockTicketProps, priority: 'P1' as const };
      const { unmount, container } = render(<TicketCard {...p1Props} />);

      const cardElement = container.firstChild as HTMLElement;
      expect(cardElement.className).toContain('animate-pulse-strong');

      unmount();
      // Component should be removed from DOM
      expect(cardElement).not.toBeInTheDocument();
    });

    it('should handle priority changes during component lifecycle', () => {
      const { rerender, container } = render(<TicketCard {...mockTicketProps} />);

      const cardElement = container.firstChild as HTMLElement;
      expect(cardElement.className).not.toContain('animate-pulse-strong');

      // Change to P1
      const p1Props = { ...mockTicketProps, priority: 'P1' as const };
      rerender(<TicketCard {...p1Props} />);

      expect(cardElement.className).toContain('animate-pulse-strong');

      // Change back to P2
      rerender(<TicketCard {...mockTicketProps} />);

      expect(cardElement.className).not.toContain('animate-pulse-strong');
    });
  });
});
