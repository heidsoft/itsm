import { renderHook, act } from '@testing-library/react';
import { useResponsive, useViewportSize, useMediaQuery, BREAKPOINTS } from '../useResponsive';

describe('useResponsive', () => {
  const originalInnerWidth = window.innerWidth;

  afterEach(() => {
    Object.defineProperty(window, 'innerWidth', {
      writable: true,
      configurable: true,
      value: originalInnerWidth,
    });
  });

  it('should return desktop state for large screens', () => {
    Object.defineProperty(window, 'innerWidth', { writable: true, configurable: true, value: 1200 });

    const { result } = renderHook(() => useResponsive());

    expect(result.current.isDesktop).toBe(true);
    expect(result.current.isMobile).toBe(false);
    expect(result.current.isTablet).toBe(false);
  });

  it('should return mobile state for small screens', () => {
    Object.defineProperty(window, 'innerWidth', { writable: true, configurable: true, value: 400 });

    const { result } = renderHook(() => useResponsive());

    expect(result.current.isMobile).toBe(true);
    expect(result.current.isDesktop).toBe(false);
    expect(result.current.isTablet).toBe(false);
    expect(result.current.breakpoint).toBe('xs');
  });

  it('should return tablet state for medium screens', () => {
    Object.defineProperty(window, 'innerWidth', { writable: true, configurable: true, value: 900 });

    const { result } = renderHook(() => useResponsive());

    expect(result.current.isTablet).toBe(true);
    expect(result.current.isMobile).toBe(false);
    expect(result.current.isDesktop).toBe(false);
    expect(result.current.breakpoint).toBe('md');
  });

  it('should respond to resize events', () => {
    Object.defineProperty(window, 'innerWidth', { writable: true, configurable: true, value: 1200 });

    const { result } = renderHook(() => useResponsive());

    expect(result.current.isDesktop).toBe(true);

    act(() => {
      Object.defineProperty(window, 'innerWidth', { writable: true, configurable: true, value: 400 });
      window.dispatchEvent(new Event('resize'));
    });

    expect(result.current.isMobile).toBe(true);
  });

  it('should expose BREAKPOINTS', () => {
    const { result } = renderHook(() => useResponsive());

    expect(result.current.BREAKPOINTS).toBe(BREAKPOINTS);
    expect(result.current.BREAKPOINTS.xs).toBe(480);
    expect(result.current.BREAKPOINTS.lg).toBe(1024);
  });
});

describe('useViewportSize', () => {
  it('should return current viewport size', () => {
    Object.defineProperty(window, 'innerWidth', { writable: true, configurable: true, value: 1024 });
    Object.defineProperty(window, 'innerHeight', { writable: true, configurable: true, value: 768 });

    const { result } = renderHook(() => useViewportSize());

    expect(result.current.width).toBe(1024);
    expect(result.current.height).toBe(768);
  });

  it('should update on resize', () => {
    const { result } = renderHook(() => useViewportSize());

    act(() => {
      Object.defineProperty(window, 'innerWidth', { writable: true, configurable: true, value: 500 });
      Object.defineProperty(window, 'innerHeight', { writable: true, configurable: true, value: 800 });
      window.dispatchEvent(new Event('resize'));
    });

    expect(result.current.width).toBe(500);
    expect(result.current.height).toBe(800);
  });
});

describe('useMediaQuery', () => {
  beforeEach(() => {
    // Mock window.matchMedia
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: jest.fn().mockImplementation(query => ({
        matches: query === '(min-width: 1024px)',
        media: query,
        onchange: null,
        addEventListener: jest.fn(),
        removeEventListener: jest.fn(),
        dispatchEvent: jest.fn(),
      })),
    });
  });

  it('should return true for matching query', () => {
    const { result } = renderHook(() => useMediaQuery('(min-width: 1024px)'));
    expect(result.current).toBe(true);
  });

  it('should return false for non-matching query', () => {
    const { result } = renderHook(() => useMediaQuery('(max-width: 767px)'));
    expect(result.current).toBe(false);
  });
});
