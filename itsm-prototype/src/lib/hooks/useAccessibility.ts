'use client';

import { useEffect, useCallback, useRef } from 'react';
import { message } from 'antd';

/**
 * 键盘快捷键配置
 */
export interface KeyboardShortcut {
  key: string;
  ctrl?: boolean;
  meta?: boolean;
  shift?: boolean;
  alt?: boolean;
  description: string;
  handler: () => void;
}

/**
 * 键盘快捷键Hook
 * 
 * @example
 * ```tsx
 * useKeyboardShortcuts([
 *   {
 *     key: 'k',
 *     ctrl: true,
 *     meta: true,
 *     description: '打开搜索',
 *     handler: () => openSearch()
 *   }
 * ]);
 * ```
 */
export const useKeyboardShortcuts = (shortcuts: KeyboardShortcut[]) => {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      shortcuts.forEach((shortcut) => {
        const ctrlOrMeta = (shortcut.ctrl && e.ctrlKey) || (shortcut.meta && e.metaKey);
        const shiftMatch = shortcut.shift ? e.shiftKey : !e.shiftKey;
        const altMatch = shortcut.alt ? e.altKey : !e.altKey;
        const keyMatch = e.key.toLowerCase() === shortcut.key.toLowerCase();

        if (ctrlOrMeta && shiftMatch && altMatch && keyMatch) {
          e.preventDefault();
          shortcut.handler();
        }
      });
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [shortcuts]);

  // 显示快捷键帮助
  const showShortcutsHelp = useCallback(() => {
    const isMac = typeof window !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.platform);
    const modifierKey = isMac ? '⌘' : 'Ctrl';

    const helpText = shortcuts
      .map((s) => {
        const keys = [];
        if (s.ctrl || s.meta) keys.push(modifierKey);
        if (s.shift) keys.push('Shift');
        if (s.alt) keys.push('Alt');
        keys.push(s.key.toUpperCase());
        return `${keys.join(' + ')}: ${s.description}`;
      })
      .join('\n');

    message.info({
      content: (
        <div className="text-left whitespace-pre-line">
          <div className="font-semibold mb-2">快捷键说明</div>
          {helpText}
        </div>
      ),
      duration: 5,
    });
  }, [shortcuts]);

  return { showShortcutsHelp };
};

/**
 * 焦点陷阱Hook
 * 
 * 用于模态框等场景，将焦点限制在指定元素内
 * 
 * @example
 * ```tsx
 * const trapRef = useFocusTrap<HTMLDivElement>(isModalOpen);
 * return <div ref={trapRef}>...</div>
 * ```
 */
export const useFocusTrap = <T extends HTMLElement>(active: boolean = true) => {
  const ref = useRef<T>(null);

  useEffect(() => {
    if (!active || !ref.current) return;

    const element = ref.current;
    const focusableElements = element.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );

    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];

    // 聚焦到第一个元素
    firstElement?.focus();

    const handleTabKey = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return;

      if (e.shiftKey) {
        // Shift + Tab
        if (document.activeElement === firstElement) {
          e.preventDefault();
          lastElement?.focus();
        }
      } else {
        // Tab
        if (document.activeElement === lastElement) {
          e.preventDefault();
          firstElement?.focus();
        }
      }
    };

    element.addEventListener('keydown', handleTabKey);
    return () => element.removeEventListener('keydown', handleTabKey);
  }, [active]);

  return ref;
};

/**
 * 屏幕阅读器公告Hook
 * 
 * 用于向屏幕阅读器用户公告重要信息
 * 
 * @example
 * ```tsx
 * const announce = useScreenReaderAnnounce();
 * announce('表单提交成功', 'polite');
 * ```
 */
export const useScreenReaderAnnounce = () => {
  const announceRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    // 创建隐藏的公告区域
    if (typeof window === 'undefined') return;

    const announcer = document.createElement('div');
    announcer.setAttribute('role', 'status');
    announcer.setAttribute('aria-live', 'polite');
    announcer.setAttribute('aria-atomic', 'true');
    announcer.style.position = 'absolute';
    announcer.style.left = '-10000px';
    announcer.style.width = '1px';
    announcer.style.height = '1px';
    announcer.style.overflow = 'hidden';
    
    document.body.appendChild(announcer);
    announceRef.current = announcer;

    return () => {
      if (announcer && document.body.contains(announcer)) {
        document.body.removeChild(announcer);
      }
    };
  }, []);

  const announce = useCallback((message: string, priority: 'polite' | 'assertive' = 'polite') => {
    if (!announceRef.current) return;

    announceRef.current.setAttribute('aria-live', priority);
    announceRef.current.textContent = message;

    // 清空消息，以便下次相同消息也能被读取
    setTimeout(() => {
      if (announceRef.current) {
        announceRef.current.textContent = '';
      }
    }, 1000);
  }, []);

  return announce;
};

/**
 * 跳过导航链接Hook
 * 
 * 用于创建"跳到主内容"链接
 * 
 * @example
 * ```tsx
 * const { SkipLink } = useSkipNavigation('main-content');
 * return (
 *   <>
 *     <SkipLink />
 *     <main id="main-content">...</main>
 *   </>
 * );
 * ```
 */
export const useSkipNavigation = (targetId: string) => {
  const handleSkip = (e: React.MouseEvent<HTMLAnchorElement>) => {
    e.preventDefault();
    const target = document.getElementById(targetId);
    if (target) {
      target.focus();
      target.scrollIntoView({ behavior: 'smooth' });
    }
  };

  const SkipLink = () => (
    <a
      href={`#${targetId}`}
      onClick={handleSkip}
      className="sr-only focus:not-sr-only focus:absolute focus:top-2 focus:left-2 focus:z-50 focus:px-4 focus:py-2 focus:bg-blue-600 focus:text-white focus:rounded"
    >
      跳到主内容
    </a>
  );

  return { SkipLink, handleSkip };
};

/**
 * 焦点可见性Hook
 * 
 * 检测用户是否使用键盘导航
 * 
 * @example
 * ```tsx
 * const isKeyboardUser = useFocusVisible();
 * return <div className={isKeyboardUser ? 'show-focus-ring' : ''}>...</div>
 * ```
 */
export const useFocusVisible = () => {
  const [isKeyboardUser, setIsKeyboardUser] = React.useState(false);

  useEffect(() => {
    const handleKeyDown = () => setIsKeyboardUser(true);
    const handleMouseDown = () => setIsKeyboardUser(false);

    window.addEventListener('keydown', handleKeyDown);
    window.addEventListener('mousedown', handleMouseDown);

    return () => {
      window.removeEventListener('keydown', handleKeyDown);
      window.removeEventListener('mousedown', handleMouseDown);
    };
  }, []);

  return isKeyboardUser;
};

/**
 * 减少动画偏好检测Hook
 * 
 * 检测用户是否偏好减少动画
 * 
 * @example
 * ```tsx
 * const prefersReducedMotion = usePrefersReducedMotion();
 * return <div className={!prefersReducedMotion ? 'animate' : ''}>...</div>
 * ```
 */
export const usePrefersReducedMotion = () => {
  const [prefersReducedMotion, setPrefersReducedMotion] = React.useState(false);

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)');
    setPrefersReducedMotion(mediaQuery.matches);

    const handleChange = (e: MediaQueryListEvent) => {
      setPrefersReducedMotion(e.matches);
    };

    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  return prefersReducedMotion;
};

/**
 * ARIA实时区域Hook
 * 
 * 管理ARIA实时区域的更新
 * 
 * @example
 * ```tsx
 * const { liveRegionRef, updateLiveRegion } = useAriaLiveRegion();
 * 
 * useEffect(() => {
 *   updateLiveRegion(`加载了 ${data.length} 条记录`);
 * }, [data]);
 * 
 * return <div ref={liveRegionRef} />
 * ```
 */
export const useAriaLiveRegion = (politeness: 'polite' | 'assertive' = 'polite') => {
  const liveRegionRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (liveRegionRef.current) {
      liveRegionRef.current.setAttribute('role', 'status');
      liveRegionRef.current.setAttribute('aria-live', politeness);
      liveRegionRef.current.setAttribute('aria-atomic', 'true');
      // 视觉隐藏
      liveRegionRef.current.className = 'sr-only';
    }
  }, [politeness]);

  const updateLiveRegion = useCallback((message: string) => {
    if (liveRegionRef.current) {
      liveRegionRef.current.textContent = message;
    }
  }, []);

  return { liveRegionRef, updateLiveRegion };
};

// React import for useState
import React from 'react';

export default {
  useKeyboardShortcuts,
  useFocusTrap,
  useScreenReaderAnnounce,
  useSkipNavigation,
  useFocusVisible,
  usePrefersReducedMotion,
  useAriaLiveRegion,
};

