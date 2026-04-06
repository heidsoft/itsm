/**
 * Design System Tokens
 * Centralized design constants for consistent styling across the application
 */

export const colors = {
  primary: '#0f172a',
  accent: '#3b82f6',
  success: '#10b981',
  warning: '#f59e0b',
  danger: '#ef4444',
  surface: '#ffffff',
  border: '#e2e8f0',
  text: '#1e293b',
  textMuted: '#64748b',
  bgSubtle: '#f8fafc',
} as const;

export const shadows = {
  dropdown: '0 10px 40px -10px rgba(0,0,0,0.15)',
  card: '0 1px 3px 0 rgba(0, 0, 0, 0.1)',
  cardHover: '0 4px 6px -1px rgba(0, 0, 0, 0.1)',
  glow: (color: string) => `0 0 20px ${color}20`,
} as const;

export const radius = {
  sm: '8px',
  md: '12px',
  lg: '16px',
  full: '9999px',
} as const;

export const spacing = {
  xs: '4px',
  sm: '8px',
  md: '16px',
  lg: '24px',
  xl: '32px',
} as const;

export const typography = {
  fontFamily: {
    sans: 'Inter, Noto Sans SC, system-ui, sans-serif',
    mono: 'JetBrains Mono, Menlo, monospace',
  },
  fontSize: {
    xs: '0.75rem',
    sm: '0.875rem',
    base: '1rem',
    lg: '1.125rem',
    xl: '1.25rem',
    '2xl': '1.5rem',
  },
} as const;

export const transitions = {
  fast: '150ms ease',
  normal: '300ms ease',
  slow: '500ms ease',
} as const;

// Legacy DESIGN object for backward compatibility with existing components
export const DESIGN = {
  colors,
  shadows,
  radius,
} as const;

export type ColorKey = keyof typeof colors;
export type ShadowKey = keyof typeof shadows;
export type RadiusKey = keyof typeof radius;
