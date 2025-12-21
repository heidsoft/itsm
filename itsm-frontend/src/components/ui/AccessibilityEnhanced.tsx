'use client';

import React, { useState, useEffect, useRef } from 'react';

export interface AccessibilityEnhancedProps {
  children: React.ReactNode;
  /** 高对比度模式 */
  highContrast?: boolean;
  /** 大字体模式 */
  largeText?: boolean;
  /** 减少动画模式 */
  reducedMotion?: boolean;
  /** 屏幕阅读器优化 */
  screenReaderOptimized?: boolean;
  /** 键盘导航优化 */
  keyboardNavigation?: boolean;
  /** 颜色盲友好模式 */
  colorBlindFriendly?: boolean;
}

/**
 * 可访问性增强组件
 * 提供多种无障碍访问支持
 */
export const AccessibilityEnhanced: React.FC<AccessibilityEnhancedProps> = ({
  children,
  highContrast = false,
  largeText = false,
  reducedMotion = false,
  screenReaderOptimized = true,
  keyboardNavigation = true,
  colorBlindFriendly = false,
}) => {
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false);
  const [prefersHighContrast, setPrefersHighContrast] = useState(false);
  const [prefersLargeText, setPrefersLargeText] = useState(false);

  // 检测系统偏好
  useEffect(() => {
    // 检测减少动画偏好
    const motionQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    setPrefersReducedMotion(motionQuery.matches);
    
    const handleMotionChange = (e: MediaQueryListEvent) => {
      setPrefersReducedMotion(e.matches);
    };
    
    // 检测高对比度偏好
    const contrastQuery = window.matchMedia('(prefers-contrast: high)');
    setPrefersHighContrast(contrastQuery.matches);
    
    const handleContrastChange = (e: MediaQueryListEvent) => {
      setPrefersHighContrast(e.matches);
    };

    // 检测大字体偏好
    const textQuery = window.matchMedia('(prefers-reduced-data: reduce)');
    setPrefersLargeText(textQuery.matches);

    if (motionQuery.addEventListener) {
      motionQuery.addEventListener('change', handleMotionChange);
    } else {
      motionQuery.addListener(handleMotionChange);
    }

    if (contrastQuery.addEventListener) {
      contrastQuery.addEventListener('change', handleContrastChange);
    } else {
      contrastQuery.addListener(handleContrastChange);
    }

    return () => {
      if (motionQuery.removeEventListener) {
        motionQuery.removeEventListener('change', handleMotionChange);
      } else {
        motionQuery.removeListener(handleMotionChange);
      }
      if (contrastQuery.removeEventListener) {
        contrastQuery.removeEventListener('change', handleContrastChange);
      } else {
        contrastQuery.removeListener(handleContrastChange);
      }
    };
  }, []);

  // 计算最终状态
  const shouldReduceMotion = reducedMotion || prefersReducedMotion;
  const shouldHighContrast = highContrast || prefersHighContrast;
  const shouldLargeText = largeText || prefersLargeText;

  // 构建可访问性类名
  const accessibilityClasses = [
    shouldHighContrast && 'high-contrast',
    shouldLargeText && 'large-text',
    shouldReduceMotion && 'reduce-motion',
    colorBlindFriendly && 'colorblind-friendly',
    screenReaderOptimized && 'sr-optimized',
    keyboardNavigation && 'keyboard-nav',
  ].filter(Boolean).join(' ');

  // 添加CSS变量
  const accessibilityStyles: React.CSSProperties = {
    // 高对比度模式
    ...(shouldHighContrast && {
      '--text-color': '#000000',
      '--bg-color': '#ffffff',
      '--border-color': '#000000',
      '--link-color': '#0000ff',
    }),
    // 大字体模式
    ...(shouldLargeText && {
      '--font-size-base': '18px',
      '--font-size-large': '24px',
    }),
    // 减少动画
    ...(shouldReduceMotion && {
      '--transition-duration': '0ms',
      '--animation-duration': '0ms',
    }),
  } as React.CSSProperties;

  return (
    <div 
      className={`accessibility-enhanced ${accessibilityClasses}`}
      style={accessibilityStyles}
      data-accessibility={{
        highContrast: shouldHighContrast,
        largeText: shouldLargeText,
        reducedMotion: shouldReduceMotion,
        colorBlindFriendly,
        screenReaderOptimized,
        keyboardNavigation,
      }}
    >
      {children}
    </div>
  );
};

/**
 * 屏幕阅读器专用文本组件
 */
export const ScreenReaderOnly: React.FC<{ children: React.ReactNode; className?: string }> = ({
  children,
  className = '',
}) => (
  <span className={`
    sr-only
    absolute w-px h-px p-0 -m-px overflow-hidden
    whitespace-nowrap border-0
    ${className}
  `}>
    {children}
  </span>
);

/**
 * 焦点指示器组件
 */
export const FocusIndicator: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <div className="focus-within:ring-2 focus-within:ring-blue-500 focus-within:ring-offset-2">
    {children}
  </div>
);

/**
 * 跳转链接组件
 */
export const SkipLink: React.FC<{
  href: string;
  children: React.ReactNode;
  className?: string;
}> = ({ href, children, className = '' }) => (
  <a
    href={href}
    className={`
      absolute top-0 left-0 -m-full p-2 
      bg-blue-600 text-white
      focus:m-0 focus:z-50 focus:static
      rounded transition-transform
      ${className}
    `}
  >
    {children}
  </a>
);

/**
 * ARIA实时区域组件
 */
export const AriaLiveRegion: React.FC<{
  politeness?: 'polite' | 'assertive' | 'off';
  atomic?: boolean;
  children: React.ReactNode;
}> = ({ politeness = 'polite', atomic = false, children }) => (
  <div
    aria-live={politeness}
    aria-atomic={atomic}
    className="sr-only"
  >
    {children}
  </div>
);

/**
 * 键盘导航提示组件
 */
export const KeyboardNavigationHint: React.FC<{
  shortcuts: Array<{ key: string; description: string }>;
  show?: boolean;
}> = ({ shortcuts, show = false }) => {
  if (!show) return null;

  return (
    <div 
      className="fixed bottom-4 right-4 bg-gray-900 text-white p-4 rounded-lg shadow-lg z-50"
      role="tooltip"
    >
      <h3 className="font-semibold mb-2">键盘快捷键</h3>
      <ul className="space-y-1 text-sm">
        {shortcuts.map((shortcut, index) => (
          <li key={index} className="flex justify-between gap-4">
            <kbd className="px-2 py-1 bg-gray-700 rounded text-xs">
              {shortcut.key}
            </kbd>
            <span>{shortcut.description}</span>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default AccessibilityEnhanced;