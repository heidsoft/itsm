/**
 * 响应式设计 Hook
 * Responsive Design Hook
 */

'use client';

import { useState, useEffect } from 'react';

// 断点定义
const breakpoints = {
  xs: 0,      // < 576px (mobile)
  sm: 576,    // >= 576px (large mobile)
  md: 768,    // >= 768px (tablet)
  lg: 992,    // >= 992px (desktop)
  xl: 1200,   // >= 1200px (large desktop)
  xxl: 1600,  // >= 1600px (xlarge desktop)
};

export type Breakpoint = keyof typeof breakpoints;

export interface ResponsiveInfo {
  // 当前断点
  currentBreakpoint: Breakpoint;
  // 当前宽度
  width: number;
  // 是否为移动端
  isMobile: boolean;
  // 是否为平板
  isTablet: boolean;
  // 是否为桌面端
  isDesktop: boolean;
  // 各断点状态
  xs: boolean;
  sm: boolean;
  md: boolean;
  lg: boolean;
  xl: boolean;
  xxl: boolean;
}

/**
 * 获取当前断点
 */
const getCurrentBreakpoint = (width: number): Breakpoint => {
  if (width >= breakpoints.xxl) return 'xxl';
  if (width >= breakpoints.xl) return 'xl';
  if (width >= breakpoints.lg) return 'lg';
  if (width >= breakpoints.md) return 'md';
  if (width >= breakpoints.sm) return 'sm';
  return 'xs';
};

/**
 * 响应式Hook
 */
export const useResponsive = (): ResponsiveInfo => {
  const [responsiveInfo, setResponsiveInfo] = useState<ResponsiveInfo>(() => {
    // SSR 时的默认值
    if (typeof window === 'undefined') {
      return {
        currentBreakpoint: 'lg',
        width: 1200,
        isMobile: false,
        isTablet: false,
        isDesktop: true,
        xs: false,
        sm: false,
        md: false,
        lg: true,
        xl: false,
        xxl: false,
      };
    }

    const width = window.innerWidth;
    const currentBreakpoint = getCurrentBreakpoint(width);

    return {
      currentBreakpoint,
      width,
      isMobile: width < breakpoints.md,
      isTablet: width >= breakpoints.md && width < breakpoints.lg,
      isDesktop: width >= breakpoints.lg,
      xs: width < breakpoints.sm,
      sm: width >= breakpoints.sm && width < breakpoints.md,
      md: width >= breakpoints.md && width < breakpoints.lg,
      lg: width >= breakpoints.lg && width < breakpoints.xl,
      xl: width >= breakpoints.xl && width < breakpoints.xxl,
      xxl: width >= breakpoints.xxl,
    };
  });

  useEffect(() => {
    const handleResize = () => {
      const width = window.innerWidth;
      const currentBreakpoint = getCurrentBreakpoint(width);

      setResponsiveInfo({
        currentBreakpoint,
        width,
        isMobile: width < breakpoints.md,
        isTablet: width >= breakpoints.md && width < breakpoints.lg,
        isDesktop: width >= breakpoints.lg,
        xs: width < breakpoints.sm,
        sm: width >= breakpoints.sm && width < breakpoints.md,
        md: width >= breakpoints.md && width < breakpoints.lg,
        lg: width >= breakpoints.lg && width < breakpoints.xl,
        xl: width >= breakpoints.xl && width < breakpoints.xxl,
        xxl: width >= breakpoints.xxl,
      });
    };

    // 添加防抖
    let timeoutId: NodeJS.Timeout;
    const debouncedHandleResize = () => {
      clearTimeout(timeoutId);
      timeoutId = setTimeout(handleResize, 150);
    };

    window.addEventListener('resize', debouncedHandleResize);
    handleResize(); // 初始化

    return () => {
      window.removeEventListener('resize', debouncedHandleResize);
      clearTimeout(timeoutId);
    };
  }, []);

  return responsiveInfo;
};

/**
 * 检查当前是否匹配指定断点
 */
export const useBreakpoint = (breakpoint: Breakpoint): boolean => {
  const { currentBreakpoint } = useResponsive();
  
  const breakpointOrder: Breakpoint[] = ['xs', 'sm', 'md', 'lg', 'xl', 'xxl'];
  const currentIndex = breakpointOrder.indexOf(currentBreakpoint);
  const targetIndex = breakpointOrder.indexOf(breakpoint);
  
  return currentIndex >= targetIndex;
};

/**
 * 媒体查询Hook
 */
export const useMediaQuery = (query: string): boolean => {
  const [matches, setMatches] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false;
    return window.matchMedia(query).matches;
  });

  useEffect(() => {
    const mediaQuery = window.matchMedia(query);
    
    const handleChange = (e: MediaQueryListEvent) => {
      setMatches(e.matches);
    };

    // 现代浏览器
    if (mediaQuery.addEventListener) {
      mediaQuery.addEventListener('change', handleChange);
    } else {
      // 旧版浏览器兼容
      mediaQuery.addListener(handleChange);
    }

    // 初始化
    setMatches(mediaQuery.matches);

    return () => {
      if (mediaQuery.removeEventListener) {
        mediaQuery.removeEventListener('change', handleChange);
      } else {
        mediaQuery.removeListener(handleChange);
      }
    };
  }, [query]);

  return matches;
};

/**
 * 获取屏幕方向
 */
export const useOrientation = (): 'portrait' | 'landscape' => {
  const [orientation, setOrientation] = useState<'portrait' | 'landscape'>(() => {
    if (typeof window === 'undefined') return 'landscape';
    return window.innerHeight > window.innerWidth ? 'portrait' : 'landscape';
  });

  useEffect(() => {
    const handleOrientationChange = () => {
      setOrientation(window.innerHeight > window.innerWidth ? 'portrait' : 'landscape');
    };

    window.addEventListener('resize', handleOrientationChange);
    handleOrientationChange();

    return () => {
      window.removeEventListener('resize', handleOrientationChange);
    };
  }, []);

  return orientation;
};

/**
 * 检测触摸设备
 */
export const useIsTouchDevice = (): boolean => {
  const [isTouchDevice, setIsTouchDevice] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false;
    return 'ontouchstart' in window || navigator.maxTouchPoints > 0;
  });

  useEffect(() => {
    setIsTouchDevice('ontouchstart' in window || navigator.maxTouchPoints > 0);
  }, []);

  return isTouchDevice;
};

