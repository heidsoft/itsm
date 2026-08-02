'use client';

import { useEffect } from 'react';
import { useTheme } from '@/lib/design-system/theme';

/**
 * 将主题上下文中的 isDark 状态同步到 <html> 上的 .dark 类，
 * 让 globals.css 中 .dark { ... } 变量能即时生效。
 *
 * 同时设置 color-scheme 以影响浏览器原生滚动条/表单控件。
 */
export const ThemeHtmlClassSync: React.FC = () => {
  const { isDark } = useTheme();

  useEffect(() => {
    if (typeof document === 'undefined') return;
    const root = document.documentElement;
    if (isDark) {
      root.classList.add('dark');
      root.style.colorScheme = 'dark';
    } else {
      root.classList.remove('dark');
      root.style.colorScheme = 'light';
    }
  }, [isDark]);

  return null;
};
