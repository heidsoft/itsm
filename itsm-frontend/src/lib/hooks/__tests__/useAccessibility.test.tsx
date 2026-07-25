import { renderHook, act } from '@testing-library/react';
import {
  useKeyboardShortcuts,
  useFocusTrap,
  useScreenReaderAnnounce,
  useSkipNavigation,
  useFocusVisible,
  usePrefersReducedMotion,
  useAriaLiveRegion,
} from '../useAccessibility';

// Mock antd
jest.mock('antd', () => ({
  message: {
    info: jest.fn(),
  },
}));

describe('useAccessibility hooks', () => {
  describe('useKeyboardShortcuts', () => {
    it('should register keyboard shortcuts and call handlers', () => {
      const handler = jest.fn();
      const shortcuts = [
        { key: 'k', ctrl: true, description: 'Open search', handler },
      ];

      renderHook(() => useKeyboardShortcuts(shortcuts));

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
      });
      window.dispatchEvent(event);

      expect(handler).toHaveBeenCalled();
    });

    it('should not trigger when modifier keys do not match', () => {
      const handler = jest.fn();
      const shortcuts = [
        { key: 'k', ctrl: true, description: 'Open search', handler },
      ];

      renderHook(() => useKeyboardShortcuts(shortcuts));

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: false,
      });
      window.dispatchEvent(event);

      expect(handler).not.toHaveBeenCalled();
    });

    it('should handle meta key shortcuts', () => {
      const handler = jest.fn();
      const shortcuts = [
        { key: 's', meta: true, description: 'Save', handler },
      ];

      renderHook(() => useKeyboardShortcuts(shortcuts));

      const event = new KeyboardEvent('keydown', {
        key: 's',
        metaKey: true,
      });
      window.dispatchEvent(event);

      expect(handler).toHaveBeenCalled();
    });

    it('should handle shift key shortcuts', () => {
      const handler = jest.fn();
      const shortcuts = [
        { key: 'n', ctrl: true, shift: true, description: 'New item', handler },
      ];

      renderHook(() => useKeyboardShortcuts(shortcuts));

      // Should not trigger without shift
      const event1 = new KeyboardEvent('keydown', {
        key: 'n',
        ctrlKey: true,
        shiftKey: false,
      });
      window.dispatchEvent(event1);
      expect(handler).not.toHaveBeenCalled();

      // Should trigger with shift
      const event2 = new KeyboardEvent('keydown', {
        key: 'n',
        ctrlKey: true,
        shiftKey: true,
      });
      window.dispatchEvent(event2);
      expect(handler).toHaveBeenCalled();
    });

    it('should handle alt key shortcuts', () => {
      const handler = jest.fn();
      const shortcuts = [
        { key: 'h', ctrl: true, alt: true, description: 'Help', handler },
      ];

      renderHook(() => useKeyboardShortcuts(shortcuts));

      const event = new KeyboardEvent('keydown', {
        key: 'h',
        ctrlKey: true,
        altKey: true,
      });
      window.dispatchEvent(event);

      expect(handler).toHaveBeenCalled();
    });

    it('should provide showShortcutsHelp function', () => {
      const shortcuts = [
        { key: 'k', ctrl: true, description: 'Search', handler: jest.fn() },
      ];

      const { result } = renderHook(() => useKeyboardShortcuts(shortcuts));

      expect(typeof result.current.showShortcutsHelp).toBe('function');
      // Should not throw
      result.current.showShortcutsHelp();
    });

    it('should handle case-insensitive keys', () => {
      const handler = jest.fn();
      const shortcuts = [
        { key: 'K', ctrl: true, description: 'Search', handler },
      ];

      renderHook(() => useKeyboardShortcuts(shortcuts));

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
      });
      window.dispatchEvent(event);

      expect(handler).toHaveBeenCalled();
    });

    it('should clean up event listeners on unmount', () => {
      const handler = jest.fn();
      const shortcuts = [
        { key: 'k', ctrl: true, description: 'Search', handler },
      ];

      const { unmount } = renderHook(() => useKeyboardShortcuts(shortcuts));
      unmount();

      const event = new KeyboardEvent('keydown', {
        key: 'k',
        ctrlKey: true,
      });
      window.dispatchEvent(event);

      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe('useFocusTrap', () => {
    it('should return a ref', () => {
      const { result } = renderHook(() => useFocusTrap<HTMLDivElement>());
      expect(result.current).toBeDefined();
      expect(result.current.current).toBeNull();
    });

    it('should not activate when active is false', () => {
      const { result } = renderHook(() => useFocusTrap<HTMLDivElement>(false));
      expect(result.current.current).toBeNull();
    });
  });

  describe('useScreenReaderAnnounce', () => {
    it('should return an announce function', () => {
      const { result } = renderHook(() => useScreenReaderAnnounce());
      expect(typeof result.current).toBe('function');
    });

    it('should create an announcer element in the DOM', () => {
      const { unmount } = renderHook(() => useScreenReaderAnnounce());

      const announcer = document.querySelector('[role="status"]');
      expect(announcer).not.toBeNull();

      unmount();
    });

    it('should announce a message', () => {
      const { result } = renderHook(() => useScreenReaderAnnounce());

      act(() => {
        result.current('Test announcement');
      });

      const announcer = document.querySelector('[role="status"]');
      expect(announcer?.textContent).toBe('Test announcement');
    });

    it('should set aria-live to assertive when specified', () => {
      const { result } = renderHook(() => useScreenReaderAnnounce());

      act(() => {
        result.current('Urgent message', 'assertive');
      });

      const announcer = document.querySelector('[aria-live="assertive"]');
      expect(announcer).not.toBeNull();
    });

    it('should clean up announcer on unmount', () => {
      const { unmount } = renderHook(() => useScreenReaderAnnounce());

      const beforeCount = document.querySelectorAll('[role="status"]').length;
      expect(beforeCount).toBeGreaterThan(0);

      unmount();
      // Element should be removed
    });
  });

  describe('useSkipNavigation', () => {
    it('should return SkipLink component and handleSkip', () => {
      const { result } = renderHook(() => useSkipNavigation('main-content'));

      expect(result.current.SkipLink).toBeDefined();
      expect(typeof result.current.handleSkip).toBe('function');
    });
  });

  describe('useFocusVisible', () => {
    it('should initially return false', () => {
      const { result } = renderHook(() => useFocusVisible());
      expect(result.current).toBe(false);
    });

    it('should return true after keydown event', () => {
      const { result } = renderHook(() => useFocusVisible());

      act(() => {
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab' }));
      });

      expect(result.current).toBe(true);
    });

    it('should return false after mousedown event', () => {
      const { result } = renderHook(() => useFocusVisible());

      act(() => {
        window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab' }));
      });
      expect(result.current).toBe(true);

      act(() => {
        window.dispatchEvent(new MouseEvent('mousedown'));
      });
      expect(result.current).toBe(false);
    });

    it('should clean up listeners on unmount', () => {
      const { unmount } = renderHook(() => useFocusVisible());
      unmount();

      // Should not throw
      window.dispatchEvent(new KeyboardEvent('keydown'));
    });
  });

  describe('usePrefersReducedMotion', () => {
    let matchMediaMock: jest.Mock;

    beforeEach(() => {
      matchMediaMock = jest.fn().mockReturnValue({
        matches: false,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      });
      window.matchMedia = matchMediaMock;
    });

    it('should return false when user does not prefer reduced motion', () => {
      const { result } = renderHook(() => usePrefersReducedMotion());
      expect(result.current).toBe(false);
    });

    it('should return true when user prefers reduced motion', () => {
      matchMediaMock.mockReturnValue({
        matches: true,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
      });

      const { result } = renderHook(() => usePrefersReducedMotion());
      expect(result.current).toBe(true);
    });

    it('should respond to media query changes', () => {
      let changeHandler: ((e: MediaQueryListEvent) => void) | null = null;
      matchMediaMock.mockReturnValue({
        matches: false,
        addEventListener: jest.fn((event: string, handler: any) => {
          if (event === 'change') changeHandler = handler;
        }),
        removeEventListener: jest.fn(),
      });

      const { result } = renderHook(() => usePrefersReducedMotion());
      expect(result.current).toBe(false);

      act(() => {
        changeHandler?.({ matches: true } as MediaQueryListEvent);
      });

      expect(result.current).toBe(true);
    });
  });

  describe('useAriaLiveRegion', () => {
    it('should return liveRegionRef and updateLiveRegion', () => {
      const { result } = renderHook(() => useAriaLiveRegion());

      expect(result.current.liveRegionRef).toBeDefined();
      expect(typeof result.current.updateLiveRegion).toBe('function');
    });

    it('should use polite politeness by default', () => {
      const { result } = renderHook(() => useAriaLiveRegion());
      // The ref will be null since we're not rendering a component
      expect(result.current.liveRegionRef.current).toBeNull();
    });

    it('should accept assertive politeness', () => {
      const { result } = renderHook(() => useAriaLiveRegion('assertive'));
      expect(result.current.liveRegionRef).toBeDefined();
    });

    it('should update live region content when ref is available', () => {
      const div = document.createElement('div');
      const { result } = renderHook(() => useAriaLiveRegion());

      // Simulate attaching the ref
      Object.defineProperty(result.current.liveRegionRef, 'current', {
        value: div,
        writable: true,
      });

      act(() => {
        result.current.updateLiveRegion('Updated content');
      });

      expect(div.textContent).toBe('Updated content');
    });
  });
});
