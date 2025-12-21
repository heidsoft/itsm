"use client";

import { useState, useEffect } from 'react';

// 断点定义
export const BREAKPOINTS = {
  xs: 480,
  sm: 640,
  md: 768,
  lg: 1024,
  xl: 1280,
  '2xl': 1536,
} as const;

export type Breakpoint = keyof typeof BREAKPOINTS;

// 响应式 Hook
export const useResponsive = () => {
  const [breakpoint, setBreakpoint] = useState<Breakpoint>('lg');
  const [isMobile, setIsMobile] = useState(false);
  const [isTablet, setIsTablet] = useState(false);
  const [isDesktop, setIsDesktop] = useState(false);

  useEffect(() => {
    const updateBreakpoint = () => {
      const width = window.innerWidth;
      
      if (width < BREAKPOINTS.sm) {
        setBreakpoint('xs');
        setIsMobile(true);
        setIsTablet(false);
        setIsDesktop(false);
      } else if (width < BREAKPOINTS.md) {
        setBreakpoint('sm');
        setIsMobile(true);
        setIsTablet(false);
        setIsDesktop(false);
      } else if (width < BREAKPOINTS.lg) {
        setBreakpoint('md');
        setIsMobile(false);
        setIsTablet(true);
        setIsDesktop(false);
      } else if (width < BREAKPOINTS.xl) {
        setBreakpoint('lg');
        setIsMobile(false);
        setIsTablet(false);
        setIsDesktop(true);
      } else {
        setBreakpoint('2xl');
        setIsMobile(false);
        setIsTablet(false);
        setIsDesktop(true);
      }
    };

    // 初始化
    updateBreakpoint();

    // 监听窗口大小变化
    window.addEventListener('resize', updateBreakpoint);

    // 清理
    return () => window.removeEventListener('resize', updateBreakpoint);
  }, []);

  // 检查是否大于等于指定断点
  const isGreaterThan = (bp: Breakpoint) => {
    return window.innerWidth >= BREAKPOINTS[bp];
  };

  // 检查是否小于指定断点
  const isLessThan = (bp: Breakpoint) => {
    return window.innerWidth < BREAKPOINTS[bp];
  };

  // 检查是否在指定断点范围内
  const isBetween = (min: Breakpoint, max: Breakpoint) => {
    const width = window.innerWidth;
    return width >= BREAKPOINTS[min] && width < BREAKPOINTS[max];
  };

  return {
    breakpoint,
    isMobile,
    isTablet,
    isDesktop,
    isGreaterThan,
    isLessThan,
    isBetween,
    BREAKPOINTS,
  };
};

// 设备类型 Hook
export const useDeviceType = () => {
  const { isMobile, isTablet, isDesktop } = useResponsive();

  return {
    isMobile,
    isTablet,
    isDesktop,
    isTouch: typeof window !== 'undefined' && 'ontouchstart' in window,
    isLandscape: typeof window !== 'undefined' && window.innerWidth > window.innerHeight,
  };
};

// 视口尺寸 Hook
export const useViewportSize = () => {
  const [size, setSize] = useState({
    width: typeof window !== 'undefined' ? window.innerWidth : 0,
    height: typeof window !== 'undefined' ? window.innerHeight : 0,
  });

  useEffect(() => {
    const updateSize = () => {
      setSize({
        width: window.innerWidth,
        height: window.innerHeight,
      });
    };

    updateSize();
    window.addEventListener('resize', updateSize);
    return () => window.removeEventListener('resize', updateSize);
  }, []);

  return size;
};

// 滚动位置 Hook
export const useScrollPosition = () => {
  const [scrollPosition, setScrollPosition] = useState({
    x: 0,
    y: 0,
  });

  useEffect(() => {
    const updatePosition = () => {
      setScrollPosition({
        x: window.pageXOffset,
        y: window.pageYOffset,
      });
    };

    updatePosition();
    window.addEventListener('scroll', updatePosition);
    return () => window.removeEventListener('scroll', updatePosition);
  }, []);

  return scrollPosition;
};

// 媒体查询 Hook
export const useMediaQuery = (query: string) => {
  const [matches, setMatches] = useState(false);

  useEffect(() => {
    const mediaQuery = window.matchMedia(query);
    setMatches(mediaQuery.matches);

    const handler = (event: MediaQueryListEvent) => {
      setMatches(event.matches);
    };

    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [query]);

  return matches;
};

// 预定义的媒体查询
export const MEDIA_QUERIES = {
  mobile: '(max-width: 767px)',
  tablet: '(min-width: 768px) and (max-width: 1023px)',
  desktop: '(min-width: 1024px)',
  darkMode: '(prefers-color-scheme: dark)',
  lightMode: '(prefers-color-scheme: light)',
  reducedMotion: '(prefers-reduced-motion: reduce)',
  highContrast: '(prefers-contrast: high)',
} as const;
