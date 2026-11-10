/**
 * 设计系统颜色配置
 * 提供统一的颜色变量和主题支持
 */

// 基础颜色定义
export const colors = {
  // 主色调
  primary: {
    50: '#eff6ff',
    100: '#dbeafe',
    200: '#bfdbfe',
    300: '#93c5fd',
    400: '#60a5fa',
    500: '#3b82f6', // 主色
    600: '#2563eb',
    700: '#1d4ed8',
    800: '#1e40af',
    900: '#1e3a8a',
    950: '#172554',
  },

  // 中性色
  neutral: {
    50: '#f8fafc',
    100: '#f1f5f9',
    200: '#e2e8f0',
    300: '#cbd5e1',
    400: '#94a3b8',
    500: '#64748b',
    600: '#475569',
    700: '#334155',
    800: '#1e293b',
    900: '#0f172a',
    950: '#020617',
  },

  // 语义色
  semantic: {
    success: {
      50: '#f0fdf4',
      100: '#dcfce7',
      200: '#bbf7d0',
      300: '#86efac',
      400: '#4ade80',
      500: '#22c55e',
      600: '#16a34a',
      700: '#15803d',
      800: '#166534',
      900: '#14532d',
    },
    warning: {
      50: '#fffbeb',
      100: '#fef3c7',
      200: '#fde68a',
      300: '#fcd34d',
      400: '#fbbf24',
      500: '#f59e0b',
      600: '#d97706',
      700: '#b45309',
      800: '#92400e',
      900: '#78350f',
    },
    error: {
      50: '#fef2f2',
      100: '#fee2e2',
      200: '#fecaca',
      300: '#fca5a5',
      400: '#f87171',
      500: '#ef4444',
      600: '#dc2626',
      700: '#b91c1c',
      800: '#991b1b',
      900: '#7f1d1d',
    },
    info: {
      50: '#f0f9ff',
      100: '#e0f2fe',
      200: '#bae6fd',
      300: '#7dd3fc',
      400: '#38bdf8',
      500: '#0ea5e9',
      600: '#0284c7',
      700: '#0369a1',
      800: '#075985',
      900: '#0c4a6e',
    },
  },

  // 功能色
  functional: {
    background: {
      primary: '#ffffff',
      secondary: '#f8fafc',
      tertiary: '#f1f5f9',
      elevated: '#ffffff',
    },
    surface: {
      primary: '#ffffff',
      secondary: '#f8fafc',
      tertiary: '#f1f5f9',
      elevated: '#ffffff',
    },
    border: {
      primary: '#e2e8f0',
      secondary: '#cbd5e1',
      tertiary: '#94a3b8',
      focus: '#3b82f6',
    },
    text: {
      primary: '#0f172a',
      secondary: '#475569',
      tertiary: '#64748b',
      disabled: '#94a3b8',
      inverse: '#ffffff',
    },
  },
} as const;

// 暗色主题颜色
export const darkColors = {
  // 主色调（暗色主题下调整）
  primary: {
    50: '#172554',
    100: '#1e3a8a',
    200: '#1e40af',
    300: '#1d4ed8',
    400: '#2563eb',
    500: '#3b82f6', // 主色
    600: '#60a5fa',
    700: '#93c5fd',
    800: '#bfdbfe',
    900: '#dbeafe',
    950: '#eff6ff',
  },

  // 中性色（暗色主题）
  neutral: {
    50: '#020617',
    100: '#0f172a',
    200: '#1e293b',
    300: '#334155',
    400: '#475569',
    500: '#64748b',
    600: '#94a3b8',
    700: '#cbd5e1',
    800: '#e2e8f0',
    900: '#f1f5f9',
    950: '#f8fafc',
  },

  // 功能色（暗色主题）
  functional: {
    background: {
      primary: '#0f172a',
      secondary: '#1e293b',
      tertiary: '#334155',
      elevated: '#1e293b',
    },
    surface: {
      primary: '#1e293b',
      secondary: '#334155',
      tertiary: '#475569',
      elevated: '#334155',
    },
    border: {
      primary: '#334155',
      secondary: '#475569',
      tertiary: '#64748b',
      focus: '#3b82f6',
    },
    text: {
      primary: '#f8fafc',
      secondary: '#cbd5e1',
      tertiary: '#94a3b8',
      disabled: '#64748b',
      inverse: '#0f172a',
    },
  },
} as const;

// 颜色使用指南
export const colorUsage = {
  // 主色调使用
  primary: {
    description: 'The main color of the brand, used for main operation buttons, links, selected states, etc.',
    usage: ['Main button', 'Link', 'selected state', 'Progress bar', 'Brand Identity'],
    avoid: ['Large area background', 'Text color', 'Border color'],
  },

  // 中性色使用
  neutral: {
    description: 'Used for basic elements such as text, borders, and backgrounds',
    usage: ['Text color', 'Border color', 'background color', 'dividing line'],
    avoid: ['Emphasis on elements', 'Interactive status'],
  },

  // 语义色使用
  semantic: {
    success: {
      description: 'Success status, used for success prompts, completion status, etc.',
      usage: ['Success prompt', 'Complete status', 'Confirm button', 'Progress completed'],
      avoid: ['Error status', 'Warning status'],
    },
    warning: {
      description: 'Warning status, used for warning tips, precautions, etc.',
      usage: ['Warning prompt', 'Precautions', 'Pending status'],
      avoid: ['Error status', 'Success status'],
    },
    error: {
      description: 'Error status, used for error prompts, dangerous operations, etc.',
      usage: ['Error Message', 'Dangerous operation', 'Delete button', 'Authentication failed'],
      avoid: ['Success status', 'Warning status'],
    },
    info: {
      description: 'Information status, used for information prompts, help instructions, etc.',
      usage: ['Information prompt', 'Help instructions', 'Neutral operation'],
      avoid: ['Emphasize action', 'Status indication'],
    },
  },
} as const;

// 颜色对比度检查工具
export const contrastChecker = {
  // 计算颜色对比度
  getContrastRatio(color1: string, color2: string): number {
    const getLuminance = (color: string): number => {
      const rgb = this.hexToRgb(color);
      if (!rgb) return 0;

      const { r, g, b } = rgb;
      const [rs, gs, bs] = [r, g, b].map(c => {
        c = c / 255;
        return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4);
      });

      return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
    };

    const l1 = getLuminance(color1);
    const l2 = getLuminance(color2);
    const lighter = Math.max(l1, l2);
    const darker = Math.min(l1, l2);

    return (lighter + 0.05) / (darker + 0.05);
  },

  // 检查对比度是否满足WCAG标准
  checkWCAGCompliance(
    color1: string,
    color2: string
  ): {
    AA: boolean;
    AAA: boolean;
    ratio: number;
  } {
    const ratio = this.getContrastRatio(color1, color2);
    return {
      AA: ratio >= 4.5, // WCAG AA标准
      AAA: ratio >= 7, // WCAG AAA标准
      ratio,
    };
  },

  // 十六进制颜色转RGB
  hexToRgb(hex: string): { r: number; g: number; b: number } | null {
    const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
    return result
      ? {
          r: parseInt(result[1], 16),
          g: parseInt(result[2], 16),
          b: parseInt(result[3], 16),
        }
      : null;
  },

  // 获取推荐的颜色组合
  getRecommendedCombinations(): Array<{
    background: string;
    text: string;
    ratio: number;
    compliance: 'AA' | 'AAA';
  }> {
    const combinations = [
      { background: colors.functional.background.primary, text: colors.functional.text.primary },
      { background: colors.functional.background.secondary, text: colors.functional.text.primary },
      { background: colors.primary[500], text: colors.functional.text.inverse },
      { background: colors.semantic.success[500], text: colors.functional.text.inverse },
      { background: colors.semantic.warning[500], text: colors.functional.text.inverse },
      { background: colors.semantic.error[500], text: colors.functional.text.inverse },
    ];

    return combinations.map(combo => {
      const ratio = this.getContrastRatio(combo.background, combo.text);
      return {
        ...combo,
        ratio,
        compliance: ratio >= 7 ? 'AAA' : ratio >= 4.5 ? 'AA' : 'AA',
      };
    });
  },
};

// 主题配置
export const themeConfig = {
  light: {
    colors: colors,
    name: 'light',
    description: 'Light theme, suitable for daytime use',
  },
  dark: {
    colors: darkColors,
    name: 'dark',
    description: 'Dark theme, suitable for night use',
  },
} as const;

// 导出类型
export type ColorTheme = keyof typeof themeConfig;
export type ColorPalette = typeof colors;
export type SemanticColor = keyof typeof colors.semantic;
export type FunctionalColor = keyof typeof colors.functional;

export default colors;
