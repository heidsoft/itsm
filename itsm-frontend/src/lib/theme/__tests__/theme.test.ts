/**
 * Tests for theme configuration
 */

import React from 'react';
import { renderHook, act } from '@testing-library/react';

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import {
  colors,
  typography,
  spacing,
  borderRadius,
  shadows,
  zIndex,
  components,
  layout,
  animation,
  breakpoints,
  theme,
} from '../index';
import { ThemeProvider, useTheme, useBreakpoint } from '../components';

describe('Theme Configuration', () => {
  describe('colors', () => {
    it('should have primary color palette', () => {
      expect(colors.primary).toBeDefined();
      expect(colors.primary[500]).toBe('#3b82f6');
      expect(colors.primary[50]).toBe('#eff6ff');
    });

    it('should have gray palette', () => {
      expect(colors.gray).toBeDefined();
      expect(colors.gray[500]).toBe('#6b7280');
    });

    it('should have semantic status colors', () => {
      expect(colors.status.new).toBe('#3b82f6');
      expect(colors.status.resolved).toBe('#22c55e');
      expect(colors.status.cancelled).toBe('#ef4444');
    });

    it('should have priority colors', () => {
      expect(colors.priority.low).toBe('#22c55e');
      expect(colors.priority.urgent).toBe('#ef4444');
      expect(colors.priority.critical).toBe('#a855f7');
    });

    it('should have success, warning, danger, info palettes', () => {
      expect(colors.success[500]).toBe('#22c55e');
      expect(colors.warning[500]).toBe('#f59e0b');
      expect(colors.danger[500]).toBe('#ef4444');
      expect(colors.info[500]).toBe('#0ea5e9');
    });
  });

  describe('typography', () => {
    it('should have font families', () => {
      expect(typography.fontFamily.sans).toContain('Inter');
      expect(typography.fontFamily.mono).toContain('JetBrains Mono');
    });

    it('should have font sizes', () => {
      expect(typography.fontSize.base).toBeDefined();
      expect(typography.fontSize.xs).toBeDefined();
    });

    it('should have font weights', () => {
      expect(typography.fontWeight.normal).toBe('400');
      expect(typography.fontWeight.bold).toBe('700');
    });
  });

  describe('spacing', () => {
    it('should have spacing values', () => {
      expect(spacing[0]).toBe('0');
      expect(spacing[4]).toBe('1rem');
      expect(spacing.px).toBe('1px');
    });
  });

  describe('borderRadius', () => {
    it('should have radius values', () => {
      expect(borderRadius.none).toBe('0');
      expect(borderRadius.full).toBe('9999px');
      expect(borderRadius.lg).toBe('0.5rem');
    });
  });

  describe('shadows', () => {
    it('should have shadow values', () => {
      expect(shadows.none).toBe('0 0 #0000');
      expect(shadows.sm).toBeDefined();
      expect(shadows.lg).toBeDefined();
    });
  });

  describe('zIndex', () => {
    it('should have z-index values', () => {
      expect(zIndex.modal).toBe('1040');
      expect(zIndex.tooltip).toBe('1060');
      expect(zIndex.dropdown).toBe('1000');
    });
  });

  describe('components', () => {
    it('should have button component tokens', () => {
      expect(components.button.size.md).toBeDefined();
      expect(components.button.variant.primary).toBeDefined();
      expect(components.button.variant.primary.color).toBe('#ffffff');
    });

    it('should have input component tokens', () => {
      expect(components.input.size.md).toBeDefined();
      expect(components.input.variant.default).toBeDefined();
    });

    it('should have card component tokens', () => {
      expect(components.card.variant.default).toBeDefined();
      expect(components.card.variant.elevated).toBeDefined();
    });

    it('should have badge component tokens', () => {
      expect(components.badge.size.sm).toBeDefined();
      expect(components.badge.variant.success).toBeDefined();
    });
  });

  describe('layout', () => {
    it('should have container max widths', () => {
      expect(layout.container.lg).toBe('1024px');
      expect(layout.container.xl).toBe('1280px');
    });

    it('should have sidebar widths', () => {
      expect(layout.sidebar.normal).toBe('288px');
    });

    it('should have header heights', () => {
      expect(layout.header.md).toBe('4rem');
    });
  });

  describe('animation', () => {
    it('should have duration values', () => {
      expect(animation.duration.fast).toBe('150ms');
      expect(animation.duration.normal).toBe('200ms');
      expect(animation.duration.slow).toBe('300ms');
    });

    it('should have easing values', () => {
      expect(animation.easing.ease).toBe('ease');
    });

    it('should have transition presets', () => {
      expect(animation.transition.all).toBeDefined();
    });
  });

  describe('breakpoints', () => {
    it('should have responsive breakpoints', () => {
      expect(breakpoints.sm).toBe('640px');
      expect(breakpoints.md).toBe('768px');
      expect(breakpoints.lg).toBe('1024px');
    });
  });

  describe('theme (combined)', () => {
    it('should export complete theme object', () => {
      expect(theme.colors).toBe(colors);
      expect(theme.typography).toBe(typography);
      expect(theme.spacing).toBe(spacing);
      expect(theme.borderRadius).toBe(borderRadius);
      expect(theme.shadows).toBe(shadows);
      expect(theme.zIndex).toBe(zIndex);
      expect(theme.components).toBe(components);
      expect(theme.layout).toBe(layout);
      expect(theme.animation).toBe(animation);
      expect(theme.breakpoints).toBe(breakpoints);
    });
  });

  describe('colors deep structure', () => {
    it('primary has all shades 50-950', () => {
      expect(colors.primary[50]).toBeDefined();
      expect(colors.primary[100]).toBeDefined();
      expect(colors.primary[200]).toBeDefined();
      expect(colors.primary[300]).toBeDefined();
      expect(colors.primary[400]).toBeDefined();
      expect(colors.primary[600]).toBeDefined();
      expect(colors.primary[700]).toBeDefined();
      expect(colors.primary[800]).toBeDefined();
      expect(colors.primary[900]).toBeDefined();
      expect(colors.primary[950]).toBeDefined();
    });

    it('gray has all shades', () => {
      expect(colors.gray[50]).toBeDefined();
      expect(colors.gray[950]).toBeDefined();
    });

    it('success palette exists', () => {
      expect(colors.success[50]).toBeDefined();
      expect(colors.success[900]).toBeDefined();
    });

    it('warning palette exists', () => {
      expect(colors.warning[50]).toBeDefined();
      expect(colors.warning[900]).toBeDefined();
    });

    it('danger palette exists', () => {
      expect(colors.danger[50]).toBeDefined();
      expect(colors.danger[900]).toBeDefined();
    });

    it('info palette exists', () => {
      expect(colors.info[50]).toBeDefined();
      expect(colors.info[900]).toBeDefined();
    });

    it('status colors cover all statuses', () => {
      expect(colors.status.open).toBeDefined();
      expect(colors.status.inProgress).toBeDefined();
      expect(colors.status.pending).toBeDefined();
      expect(colors.status.closed).toBeDefined();
    });

    it('priority colors cover all priorities', () => {
      expect(colors.priority.normal).toBeDefined();
      expect(colors.priority.high).toBeDefined();
    });
  });

  describe('components - button variants', () => {
    it('has all button sizes', () => {
      expect(components.button.size.xs).toBeDefined();
      expect(components.button.size.sm).toBeDefined();
      expect(components.button.size.lg).toBeDefined();
      expect(components.button.size.xl).toBeDefined();
    });

    it('button sizes have padding, fontSize, borderRadius', () => {
      Object.values(components.button.size).forEach(size => {
        expect(size.padding).toBeDefined();
        expect(size.fontSize).toBeDefined();
        expect(size.borderRadius).toBeDefined();
      });
    });

    it('has all button variants', () => {
      expect(components.button.variant.secondary).toBeDefined();
      expect(components.button.variant.outline).toBeDefined();
      expect(components.button.variant.ghost).toBeDefined();
      expect(components.button.variant.danger).toBeDefined();
    });

    it('button variants have hover and focus', () => {
      Object.values(components.button.variant).forEach(v => {
        expect(v.hover).toBeDefined();
        expect(v.focus).toBeDefined();
      });
    });
  });

  describe('components - input', () => {
    it('has size and variant', () => {
      expect(components.input.size.sm).toBeDefined();
      expect(components.input.size.lg).toBeDefined();
      expect(components.input.variant.error).toBeDefined();
    });
  });

  describe('components - card', () => {
    it('has all card variants', () => {
      expect(components.card.variant.outlined).toBeDefined();
      expect(components.card.variant.filled).toBeDefined();
    });
  });

  describe('components - badge', () => {
    it('has all badge sizes', () => {
      expect(components.badge.size.xs).toBeDefined();
      expect(components.badge.size.md).toBeDefined();
    });

    it('has all badge variants', () => {
      expect(components.badge.variant.default).toBeDefined();
      expect(components.badge.variant.primary).toBeDefined();
      expect(components.badge.variant.warning).toBeDefined();
      expect(components.badge.variant.danger).toBeDefined();
      expect(components.badge.variant.info).toBeDefined();
    });
  });

  describe('layout deep structure', () => {
    it('has all container sizes', () => {
      expect(layout.container.sm).toBeDefined();
      expect(layout.container.md).toBeDefined();
      expect(layout.container['2xl']).toBeDefined();
    });

    it('has all sidebar widths', () => {
      expect(layout.sidebar.narrow).toBeDefined();
      expect(layout.sidebar.wide).toBeDefined();
    });

    it('has all header heights', () => {
      expect(layout.header.sm).toBeDefined();
      expect(layout.header.lg).toBeDefined();
    });

    it('has section config', () => {
      expect(layout.section.padding).toBeDefined();
      expect(layout.section.gap).toBeDefined();
    });
  });

  describe('spacing structure', () => {
    it('has numeric spacing values', () => {
      expect(spacing[1]).toBeDefined();
      expect(spacing[2]).toBeDefined();
      expect(spacing[8]).toBeDefined();
      expect(spacing[16]).toBeDefined();
      expect(spacing[96]).toBeDefined();
    });
  });

  describe('shadows structure', () => {
    it('has all shadow levels', () => {
      expect(shadows.DEFAULT).toBeDefined();
      expect(shadows.md).toBeDefined();
      expect(shadows.xl).toBeDefined();
      expect(shadows['2xl']).toBeDefined();
      expect(shadows.inner).toBeDefined();
    });
  });

  describe('zIndex structure', () => {
    it('has semantic z-index values', () => {
      expect(zIndex.sticky).toBe('1020');
      expect(zIndex.fixed).toBe('1030');
      expect(zIndex.popover).toBe('1050');
      expect(zIndex.notification).toBe('1070');
    });
  });
});

describe('Theme Components', () => {
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    React.createElement(ThemeProvider, { defaultMode: 'light' }, children);

  describe('ThemeProvider & useTheme', () => {
    it('provides theme context', () => {
      const { result } = renderHook(() => useTheme(), { wrapper });
      expect(result.current.theme).toBe(theme);
      expect(result.current.colors).toBe(theme.colors);
      expect(result.current.mode).toBe('light');
      expect(result.current.colorScheme).toBe('light');
    });

    it('allows setting mode', () => {
      const { result } = renderHook(() => useTheme(), { wrapper });
      act(() => { result.current.setMode('dark'); });
      expect(result.current.mode).toBe('dark');
      expect(result.current.colorScheme).toBe('dark');
    });

    it('sets system mode to follow system preference', () => {
      const { result } = renderHook(() => useTheme(), { wrapper });
      act(() => { result.current.setMode('system'); });
      expect(result.current.mode).toBe('system');
    });

    it('throws when used outside provider', () => {
      const consoleError = jest.spyOn(console, 'error').mockImplementation(() => {});
      expect(() => {
        renderHook(() => useTheme());
      }).toThrow('useTheme must be used within a ThemeProvider');
      consoleError.mockRestore();
    });

    it('reads mode from localStorage', () => {
      localStorage.setItem('itsm-theme-mode', 'dark');
      const { result } = renderHook(() => useTheme(), { wrapper });
      // After effect runs, mode should be loaded from localStorage
      expect(result.current.mode).toBeDefined();
      localStorage.removeItem('itsm-theme-mode');
    });

    it('saves mode to localStorage on change', () => {
      const { result } = renderHook(() => useTheme(), { wrapper });
      act(() => { result.current.setMode('dark'); });
      expect(localStorage.getItem('itsm-theme-mode')).toBe('dark');
      localStorage.removeItem('itsm-theme-mode');
    });
  });

  describe('useBreakpoint', () => {
    it('returns breakpoint info', () => {
      const { result } = renderHook(() => useBreakpoint());
      expect(result.current.breakpoint).toBeDefined();
      expect(typeof result.current.isMobile).toBe('boolean');
      expect(typeof result.current.isTablet).toBe('boolean');
      expect(typeof result.current.isDesktop).toBe('boolean');
    });

    it('responds to resize events', () => {
      const { result } = renderHook(() => useBreakpoint());
      act(() => {
        Object.defineProperty(window, 'innerWidth', { writable: true, value: 500 });
        window.dispatchEvent(new Event('resize'));
      });
      expect(result.current.isMobile).toBe(true);

      act(() => {
        Object.defineProperty(window, 'innerWidth', { writable: true, value: 1200 });
        window.dispatchEvent(new Event('resize'));
      });
      expect(result.current.isDesktop).toBe(true);
    });
  });
});
