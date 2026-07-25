/**
 * Tests for Design System - Colors & Spacing
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import { colors, darkColors, colorUsage, contrastChecker, themeConfig } from '../colors';
import { spacing, semanticSpacing, grid, layout, borderRadius, boxShadow, fontSize, lineHeight, fontWeight, animation, responsive, spacingSystem } from '../spacing';

describe('Design System Colors', () => {
  describe('colors', () => {
    it('should have primary palette with all shades', () => {
      expect(colors.primary[50]).toBe('#eff6ff');
      expect(colors.primary[500]).toBe('#3b82f6');
      expect(colors.primary[950]).toBe('#172554');
    });

    it('should have neutral palette', () => {
      expect(colors.neutral[50]).toBe('#f8fafc');
      expect(colors.neutral[900]).toBe('#0f172a');
    });

    it('should have semantic colors', () => {
      expect(colors.semantic.success[500]).toBe('#22c55e');
      expect(colors.semantic.warning[500]).toBe('#f59e0b');
      expect(colors.semantic.error[500]).toBe('#ef4444');
      expect(colors.semantic.info[500]).toBe('#0ea5e9');
    });

    it('should have functional colors', () => {
      expect(colors.functional.background.primary).toBe('#ffffff');
      expect(colors.functional.text.primary).toBe('#0f172a');
      expect(colors.functional.border.focus).toBe('#3b82f6');
    });
  });

  describe('darkColors', () => {
    it('should have dark theme primary colors', () => {
      expect(darkColors.primary[500]).toBe('#3b82f6');
    });

    it('should have dark theme functional colors', () => {
      expect(darkColors.functional.background.primary).toBe('#0f172a');
      expect(darkColors.functional.text.primary).toBe('#f8fafc');
    });
  });

  describe('colorUsage', () => {
    it('should have usage descriptions', () => {
      expect(colorUsage.primary.description).toBeDefined();
      expect(colorUsage.primary.usage.length).toBeGreaterThan(0);
      expect(colorUsage.semantic.success.description).toBeDefined();
    });
  });

  describe('contrastChecker', () => {
    it('should calculate contrast ratio', () => {
      const ratio = contrastChecker.getContrastRatio('#000000', '#ffffff');
      expect(ratio).toBeCloseTo(21, 0);
    });

    it('should return 1 for identical colors', () => {
      const ratio = contrastChecker.getContrastRatio('#ffffff', '#ffffff');
      expect(ratio).toBeCloseTo(1, 0);
    });

    it('should check WCAG compliance', () => {
      const result = contrastChecker.checkWCAGCompliance('#000000', '#ffffff');
      expect(result.AA).toBe(true);
      expect(result.AAA).toBe(true);
      expect(result.ratio).toBeGreaterThan(7);
    });

    it('should fail WCAG for low contrast', () => {
      const result = contrastChecker.checkWCAGCompliance('#cccccc', '#ffffff');
      expect(result.AA).toBe(false);
    });

    it('should convert hex to RGB', () => {
      const rgb = contrastChecker.hexToRgb('#ff0000');
      expect(rgb).toEqual({ r: 255, g: 0, b: 0 });
    });

    it('should return null for invalid hex', () => {
      const rgb = contrastChecker.hexToRgb('invalid');
      expect(rgb).toBeNull();
    });

    it('should get recommended combinations', () => {
      const combinations = contrastChecker.getRecommendedCombinations();
      expect(combinations.length).toBeGreaterThan(0);
      expect(combinations[0]).toHaveProperty('ratio');
      expect(combinations[0]).toHaveProperty('compliance');
    });
  });

  describe('themeConfig', () => {
    it('should have light and dark themes', () => {
      expect(themeConfig.light.name).toBe('light');
      expect(themeConfig.dark.name).toBe('dark');
      expect(themeConfig.light.colors).toBe(colors);
      expect(themeConfig.dark.colors).toBe(darkColors);
    });
  });
});

describe('Design System Spacing', () => {
  describe('spacing', () => {
    it('should have base spacing values', () => {
      expect(spacing[0]).toBe('0px');
      expect(spacing[4]).toBe('16px');
      expect(spacing[8]).toBe('32px');
      expect(spacing.px).toBe('1px');
    });
  });

  describe('semanticSpacing', () => {
    it('should have padding values', () => {
      expect(semanticSpacing.padding.sm).toBe('8px');
      expect(semanticSpacing.padding.md).toBe('16px');
    });

    it('should have margin values', () => {
      expect(semanticSpacing.margin.lg).toBe('24px');
    });

    it('should have component spacing', () => {
      expect(semanticSpacing.component.md).toBe('12px');
    });

    it('should have layout spacing', () => {
      expect(semanticSpacing.layout.md).toBe('32px');
    });
  });

  describe('grid', () => {
    it('should have breakpoints', () => {
      expect(grid.breakpoints.sm).toBe('640px');
      expect(grid.breakpoints.lg).toBe('1024px');
    });

    it('should have columns configuration', () => {
      expect(grid.columns.lg).toBe(12);
      expect(grid.columns.xs).toBe(4);
    });

    it('should have gap values', () => {
      expect(grid.gap.md).toBe('16px');
    });
  });

  describe('layout', () => {
    it('should have page layout config', () => {
      expect(layout.page.maxWidth).toBe('1280px');
    });

    it('should have sidebar widths', () => {
      expect(layout.sidebar.width.md).toBe('280px');
      expect(layout.sidebar.collapsedWidth).toBe('64px');
    });

    it('should have header heights', () => {
      expect(layout.header.height.md).toBe('64px');
    });
  });

  describe('borderRadius', () => {
    it('should have radius values', () => {
      expect(borderRadius.none).toBe('0px');
      expect(borderRadius.full).toBe('9999px');
      expect(borderRadius.lg).toBe('8px');
    });
  });

  describe('boxShadow', () => {
    it('should have shadow values', () => {
      expect(boxShadow.none).toBe('none');
      expect(boxShadow.sm).toBeDefined();
    });
  });

  describe('fontSize', () => {
    it('should have font size values', () => {
      expect(fontSize.base).toBe('16px');
      expect(fontSize.sm).toBe('14px');
      expect(fontSize.xs).toBe('12px');
    });
  });

  describe('lineHeight', () => {
    it('should have line height values', () => {
      expect(lineHeight.normal).toBe('1.5');
      expect(lineHeight.tight).toBe('1.25');
    });
  });

  describe('fontWeight', () => {
    it('should have font weight values', () => {
      expect(fontWeight.normal).toBe('400');
      expect(fontWeight.bold).toBe('700');
    });
  });

  describe('animation', () => {
    it('should have duration values', () => {
      expect(animation.duration.fast).toBe('150ms');
      expect(animation.duration.normal).toBe('300ms');
    });

    it('should have easing values', () => {
      expect(animation.easing.ease).toBe('ease');
      expect(animation.easing.bounce).toBeDefined();
    });

    it('should have animation presets', () => {
      expect(animation.presets.fadeIn).toBeDefined();
      expect(animation.presets.slideUp).toBeDefined();
      expect(animation.presets.scaleIn).toBeDefined();
    });
  });

  describe('responsive', () => {
    it('should have media queries', () => {
      expect(responsive.mediaQuery.sm).toContain('640px');
      expect(responsive.mediaQuery.lg).toContain('1024px');
    });

    it('should check breakpoint', () => {
      expect(responsive.isBreakpoint('sm', 700)).toBe(true);
      expect(responsive.isBreakpoint('lg', 500)).toBe(false);
    });

    it('should get current breakpoint', () => {
      expect(responsive.getCurrentBreakpoint(1200)).toBe('lg');
      expect(responsive.getCurrentBreakpoint(500)).toBe('xs');
      expect(responsive.getCurrentBreakpoint(1600)).toBe('2xl');
    });
  });

  describe('spacingSystem', () => {
    it('should export all spacing modules', () => {
      expect(spacingSystem.spacing).toBe(spacing);
      expect(spacingSystem.grid).toBe(grid);
      expect(spacingSystem.responsive).toBe(responsive);
    });
  });
});

describe('Design System - Theme', () => {
  // Mock localStorage
  const mockLocalStorage: Record<string, string> = {};
  beforeEach(() => {
    Object.keys(mockLocalStorage).forEach(key => delete mockLocalStorage[key]);
    jest.spyOn(Storage.prototype, 'getItem').mockImplementation(key => mockLocalStorage[key] || null);
    jest.spyOn(Storage.prototype, 'setItem').mockImplementation((key, value) => { mockLocalStorage[key] = value; });
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('getAntdTheme returns light theme config', async () => {
    const { getAntdTheme } = await import('../theme');
    const themeConfig = getAntdTheme(false);
    expect(themeConfig).toBeDefined();
    expect(themeConfig.token).toBeDefined();
    expect(themeConfig.token.colorPrimary).toBe('#3b82f6');
    expect(themeConfig.token.colorBgContainer).toBe('#ffffff');
    expect(themeConfig.components).toBeDefined();
  });

  it('getAntdTheme returns dark theme config', async () => {
    const { getAntdTheme } = await import('../theme');
    const themeConfig = getAntdTheme(true);
    expect(themeConfig.token.colorBgContainer).toBe('#0f172a');
    expect(themeConfig.token.colorText).toBe('#f8fafc');
  });

  it('generateCSSVariables returns light CSS variables', async () => {
    const { generateCSSVariables } = await import('../theme');
    const vars = generateCSSVariables(false);
    expect(vars['--color-primary-500']).toBe('#3b82f6');
    expect(vars['--color-background-primary']).toBe('#ffffff');
    expect(vars['--color-text-primary']).toBe('#0f172a');
    expect(vars['--spacing-md']).toBeDefined();
    expect(vars['--border-radius-lg']).toBeDefined();
    expect(vars['--font-size-base']).toBeDefined();
  });

  it('generateCSSVariables returns dark CSS variables', async () => {
    const { generateCSSVariables } = await import('../theme');
    const vars = generateCSSVariables(true);
    expect(vars['--color-background-primary']).toBe('#0f172a');
    expect(vars['--color-text-primary']).toBe('#f8fafc');
  });

  it('applyCSSVariables applies variables to document', async () => {
    const { applyCSSVariables } = await import('../theme');
    const setPropertySpy = jest.spyOn(document.documentElement.style, 'setProperty').mockImplementation(() => {});
    const addSpy = jest.spyOn(document.documentElement.classList, 'toggle').mockImplementation(() => false);
    
    applyCSSVariables(false);
    expect(setPropertySpy).toHaveBeenCalled();
    expect(addSpy).toHaveBeenCalledWith('dark', false);
    expect(addSpy).toHaveBeenCalledWith('light', true);

    applyCSSVariables(true);
    expect(addSpy).toHaveBeenCalledWith('dark', true);
    expect(addSpy).toHaveBeenCalledWith('light', false);
    
    setPropertySpy.mockRestore();
    addSpy.mockRestore();
  });
});
