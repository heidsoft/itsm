/**
 * 色彩对比度和可访问性工具
 * 提供WCAG 2.1标准对比度检查
 */

// RGB转相对亮度
export const rgbToLuminance = (r: number, g: number, b: number): number => {
  const [rs, gs, bs] = [r, g, b].map(val => {
    val = val / 255;
    return val <= 0.03928 ? val / 12.92 : Math.pow((val + 0.055) / 1.055, 2.4);
  });
  return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs;
};

// HEX转RGB
export const hexToRgb = (hex: string): { r: number; g: number; b: number } | null => {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return result ? {
    r: parseInt(result[1], 16),
    g: parseInt(result[2], 16),
    b: parseInt(result[3], 16)
  } : null;
};

// 计算对比度
export const getContrastRatio = (color1: string, color2: string): number => {
  const rgb1 = hexToRgb(color1);
  const rgb2 = hexToRgb(color2);
  
  if (!rgb1 || !rgb2) return 1;
  
  const lum1 = rgbToLuminance(rgb1.r, rgb1.g, rgb1.b);
  const lum2 = rgbToLuminance(rgb2.r, rgb2.g, rgb2.b);
  
  const brightest = Math.max(lum1, lum2);
  const darkest = Math.min(lum1, lum2);
  
  return (brightest + 0.05) / (darkest + 0.05);
};

// WCAG等级检查
export const getWCAGLevel = (contrastRatio: number): {
  AA: { normal: boolean; large: boolean };
  AAA: { normal: boolean; large: boolean };
  level: 'Fail' | 'AA' | 'AAA';
} => {
  const AA = {
    normal: contrastRatio >= 4.5,
    large: contrastRatio >= 3.0,
  };
  
  const AAA = {
    normal: contrastRatio >= 7.0,
    large: contrastRatio >= 4.5,
  };
  
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const _unused = AAA;

  let level: 'Fail' | 'AA' | 'AAA' = 'Fail';
  if (AA.normal) {
    if (AAA.normal) {
      level = 'AAA';
    } else {
      level = 'AA';
    }
  }
  
  return { AA, AAA, level };
};

// 获取可访问性建议
export const getAccessibilityRecommendation = (
  foreground: string,
  background: string
): {
  passes: boolean;
  ratio: number;
  level: string;
  recommendation?: string;
  alternative?: {
    foreground?: string;
    background?: string;
  };
} => {
  const ratio = getContrastRatio(foreground, background);
  const { AA, AAA, level } = getWCAGLevel(ratio);
  const passes = AA.normal;
  
  let recommendation: string | undefined;
  let alternative: { foreground?: string; background?: string } | undefined;
  
  if (!passes) {
    recommendation = `对比度 ${ratio.toFixed(2)} 不符合WCAG 2.1 AA标准(需要≥4.5:1)`;
    
    // 提供颜色建议
    const brightnessRatio = getContrastRatio(foreground, '#ffffff');
    if (brightnessRatio < 4.5) {
      // 前景色太浅
      alternative = {
        foreground: '#000000', // 使用纯黑
        background: background
      };
    } else {
      // 背景色太浅
      alternative = {
        foreground: foreground,
        background: '#000000' // 使用纯黑
      };
    }
  } else if (level === 'AA') {
    recommendation = '符合AA标准，建议考虑AAA标准以获得更好的可访问性';
  }
  
  return {
    passes,
    ratio,
    level,
    recommendation,
    alternative
  };
};

// 色盲友好颜色检查
export const isColorBlindFriendly = (
  foreground: string,
  background: string,
  type: 'protanopia' | 'deuteranopia' | 'tritanopia' = 'deuteranopia'
): boolean => {
  // 简化的色盲模拟 - 检查色相差异
  const rgb1 = hexToRgb(foreground);
  const rgb2 = hexToRgb(background);
  
  if (!rgb1 || !rgb2) return false;
  
  // 计算色相差异
  const hue1 = Math.atan2(Math.sqrt(3) * (rgb1.g - rgb1.b), 2 * rgb1.r - rgb1.g - rgb1.b);
  const hue2 = Math.atan2(Math.sqrt(3) * (rgb2.g - rgb2.b), 2 * rgb2.r - rgb2.g - rgb2.b);
  
  const hueDifference = Math.abs(hue1 - hue2);
  const normalizedHueDifference = Math.min(hueDifference, 2 * Math.PI - hueDifference);
  
  // 色盲用户需要更大的色相差异
  const minHueDifference = type === 'tritanopia' ? Math.PI / 6 : Math.PI / 4;
  
  return normalizedHueDifference >= minHueDifference;
};

// 获取可访问的文本颜色
export const getAccessibleTextColor = (
  backgroundColor: string,
  preferredColor: string = '#000000'
): string => {
  const lightColors = ['#ffffff', '#f8f9fa', '#e9ecef', '#dee2e6', '#ced4da'];
  const darkColors = ['#000000', '#212529', '#343a40', '#495057', '#6c757d'];
  
  // 检查首选颜色是否合适
  if (getContrastRatio(preferredColor, backgroundColor) >= 4.5) {
    return preferredColor;
  }
  
  // 尝试找到一个合适的颜色
  const colors = preferredColor === '#000000' ? lightColors : darkColors;
  
  for (const color of colors) {
    if (getContrastRatio(color, backgroundColor) >= 4.5) {
      return color;
    }
  }
  
  // 如果都不合适，返回最高对比度的颜色
  return getContrastRatio('#000000', backgroundColor) > 
         getContrastRatio('#ffffff', backgroundColor) ? '#000000' : '#ffffff';
};

// 可访问调色板
export const accessibleColors = {
  // 主色调 - 足够对比度的变体
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
  
  // 辅助色
  success: {
    50: '#f0fdf4',
    100: '#dcfce7',
    200: '#bbf7d0',
    300: '#86efac',
    400: '#4ade80',
    500: '#22c55e', // 主色
    600: '#16a34a',
    700: '#15803d',
    800: '#166534',
    900: '#14532d',
    950: '#052e16',
  },
  
  warning: {
    50: '#fffbeb',
    100: '#fef3c7',
    200: '#fde68a',
    300: '#fcd34d',
    400: '#fbbf24',
    500: '#f59e0b', // 主色
    600: '#d97706',
    700: '#b45309',
    800: '#92400e',
    900: '#78350f',
    950: '#451a03',
  },
  
  error: {
    50: '#fef2f2',
    100: '#fee2e2',
    200: '#fecaca',
    300: '#fca5a5',
    400: '#f87171',
    500: '#ef4444', // 主色
    600: '#dc2626',
    700: '#b91c1c',
    800: '#991b1b',
    900: '#7f1d1d',
    950: '#450a0a',
  },
  
  // 中性色 - 优化对比度
  gray: {
    0: '#ffffff',
    50: '#fafafa',
    100: '#f5f5f5',
    200: '#e5e5e5',
    300: '#d4d4d4',
    400: '#a3a3a3',
    500: '#737373',
    600: '#525252',
    700: '#404040',
    800: '#262626',
    900: '#171717',
    950: '#0a0a0a',
  }
};

export default {
  getContrastRatio,
  getWCAGLevel,
  getAccessibilityRecommendation,
  isColorBlindFriendly,
  getAccessibleTextColor,
  accessibleColors
};