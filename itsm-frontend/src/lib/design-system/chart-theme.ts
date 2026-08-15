'use client';

import { useTheme } from '@/lib/design-system/theme';

/**
 * Single source of truth for chart theming (Recharts / reactflow / any SVG chart).
 *
 * The design system already defines a complete `.dark` token set in globals.css,
 * but Recharts and reactflow render their own SVG with hardcoded colors
 * (`#8884d8` fills, near-black axis text, white tooltips), so they ignore the
 * theme. This helper mirrors the design-system tokens into the chart layer so
 * charts stay legible in dark mode.
 */

export interface ChartTheme {
  isDark: boolean;
  /** Axis tick + label text */
  axisText: string;
  /** Axis line + tick line stroke */
  axisLine: string;
  /** CartesianGrid / divider stroke */
  grid: string;
  /** Tooltip surface */
  tooltipBg: string;
  tooltipBorder: string;
  tooltipText: string;
  /** Legend text */
  legendText: string;
  /** Sequential categorical palette (bars / series) */
  palette: string[];
}

// Categorical palette kept close to the brand primary/success/warning/error/info
// ramp so charts read consistently with badges and tags across the app.
const LIGHT_PALETTE = ['#2563eb', '#16a34a', '#d97706', '#dc2626', '#0891b2', '#7c3aed', '#db2777'];
const DARK_PALETTE = ['#60a5fa', '#4ade80', '#fbbf24', '#f87171', '#22d3ee', '#a78bfa', '#f472b6'];

export function getChartTheme(isDark: boolean): ChartTheme {
  if (isDark) {
    return {
      isDark: true,
      axisText: '#cbd5e1',
      axisLine: '#475569',
      grid: '#334155',
      tooltipBg: '#1e293b',
      tooltipBorder: '#475569',
      tooltipText: '#f1f5f9',
      legendText: '#cbd5e1',
      palette: DARK_PALETTE,
    };
  }
  return {
    isDark: false,
    axisText: '#475569',
    axisLine: '#cbd5e1',
    grid: '#e2e8f0',
    tooltipBg: '#ffffff',
    tooltipBorder: '#e2e8f0',
    tooltipText: '#0f172a',
    legendText: '#475569',
    palette: LIGHT_PALETTE,
  };
}

export function useChartTheme(): ChartTheme {
  const { isDark } = useTheme();
  return getChartTheme(isDark);
}
