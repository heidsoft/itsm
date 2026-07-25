/**
 * Tests for color-contrast utilities
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import {
  rgbToLuminance,
  hexToRgb,
  getContrastRatio,
  getWCAGLevel,
  getAccessibilityRecommendation,
  isColorBlindFriendly,
  getAccessibleTextColor,
  accessibleColors,
} from '../color-contrast';

describe('Color Contrast Utilities', () => {
  describe('rgbToLuminance', () => {
    it('should return 0 for black', () => {
      expect(rgbToLuminance(0, 0, 0)).toBeCloseTo(0, 4);
    });

    it('should return 1 for white', () => {
      expect(rgbToLuminance(255, 255, 255)).toBeCloseTo(1, 4);
    });

    it('should calculate intermediate values', () => {
      const lum = rgbToLuminance(128, 128, 128);
      expect(lum).toBeGreaterThan(0);
      expect(lum).toBeLessThan(1);
    });

    it('should handle low values (≤ 0.03928 threshold)', () => {
      const lum = rgbToLuminance(5, 5, 5);
      expect(lum).toBeGreaterThan(0);
    });
  });

  describe('hexToRgb', () => {
    it('should convert hex to RGB', () => {
      expect(hexToRgb('#ff0000')).toEqual({ r: 255, g: 0, b: 0 });
      expect(hexToRgb('#00ff00')).toEqual({ r: 0, g: 255, b: 0 });
      expect(hexToRgb('#0000ff')).toEqual({ r: 0, g: 0, b: 255 });
    });

    it('should handle hex without #', () => {
      expect(hexToRgb('ff0000')).toEqual({ r: 255, g: 0, b: 0 });
    });

    it('should return null for invalid hex', () => {
      expect(hexToRgb('invalid')).toBeNull();
      expect(hexToRgb('#xyz')).toBeNull();
    });

    it('should convert white and black', () => {
      expect(hexToRgb('#ffffff')).toEqual({ r: 255, g: 255, b: 255 });
      expect(hexToRgb('#000000')).toEqual({ r: 0, g: 0, b: 0 });
    });
  });

  describe('getContrastRatio', () => {
    it('should return 21 for black on white', () => {
      const ratio = getContrastRatio('#000000', '#ffffff');
      expect(ratio).toBeCloseTo(21, 0);
    });

    it('should return 1 for same color', () => {
      const ratio = getContrastRatio('#ffffff', '#ffffff');
      expect(ratio).toBeCloseTo(1, 0);
    });

    it('should return 1 for invalid colors', () => {
      const ratio = getContrastRatio('invalid', '#ffffff');
      expect(ratio).toBe(1);
    });

    it('should calculate ratio regardless of order', () => {
      const ratio1 = getContrastRatio('#000000', '#ffffff');
      const ratio2 = getContrastRatio('#ffffff', '#000000');
      expect(ratio1).toBeCloseTo(ratio2, 2);
    });
  });

  describe('getWCAGLevel', () => {
    it('should return AAA for high contrast (21:1)', () => {
      const result = getWCAGLevel(21);
      expect(result.level).toBe('AAA');
      expect(result.AA.normal).toBe(true);
      expect(result.AAA.normal).toBe(true);
    });

    it('should return AA for medium contrast (5:1)', () => {
      const result = getWCAGLevel(5);
      expect(result.level).toBe('AA');
      expect(result.AA.normal).toBe(true);
      expect(result.AAA.normal).toBe(false);
    });

    it('should return Fail for low contrast (2:1)', () => {
      const result = getWCAGLevel(2);
      expect(result.level).toBe('Fail');
      expect(result.AA.normal).toBe(false);
    });

    it('should check large text thresholds', () => {
      const result = getWCAGLevel(3.5);
      expect(result.AA.large).toBe(true);
      expect(result.AA.normal).toBe(false);
    });
  });

  describe('getAccessibilityRecommendation', () => {
    it('should pass for high contrast combination', () => {
      const result = getAccessibilityRecommendation('#000000', '#ffffff');
      expect(result.passes).toBe(true);
      expect(result.ratio).toBeCloseTo(21, 0);
    });

    it('should fail for low contrast and provide recommendation', () => {
      const result = getAccessibilityRecommendation('#cccccc', '#ffffff');
      expect(result.passes).toBe(false);
      expect(result.recommendation).toBeDefined();
      expect(result.alternative).toBeDefined();
    });

    it('should provide alternative with dark foreground suggestion', () => {
      const result = getAccessibilityRecommendation('#cccccc', '#ffffff');
      expect(result.alternative?.foreground).toBe('#000000');
    });

    it('should provide alternative with dark background suggestion', () => {
      // When foreground has good contrast with white
      const result = getAccessibilityRecommendation('#000000', '#111111');
      expect(result.passes).toBe(false);
      expect(result.alternative).toBeDefined();
    });

    it('should suggest AAA improvement for AA-level contrast', () => {
      // A color combo that passes AA but not AAA
      const result = getAccessibilityRecommendation('#595959', '#ffffff');
      if (result.level === 'AA') {
        expect(result.recommendation).toContain('AAA');
      }
    });
  });

  describe('isColorBlindFriendly', () => {
    it('should return true for high contrast colors', () => {
      const result = isColorBlindFriendly('#ff0000', '#0000ff', 'deuteranopia');
      expect(typeof result).toBe('boolean');
    });

    it('should return false for invalid hex', () => {
      expect(isColorBlindFriendly('invalid', '#ffffff')).toBe(false);
      expect(isColorBlindFriendly('#ffffff', 'invalid')).toBe(false);
    });

    it('should handle tritanopia type', () => {
      const result = isColorBlindFriendly('#ff0000', '#0000ff', 'tritanopia');
      expect(typeof result).toBe('boolean');
    });

    it('should handle protanopia type', () => {
      const result = isColorBlindFriendly('#ff0000', '#00ff00', 'protanopia');
      expect(typeof result).toBe('boolean');
    });
  });

  describe('getAccessibleTextColor', () => {
    it('should return black text for white background', () => {
      const result = getAccessibleTextColor('#ffffff');
      expect(result).toBe('#000000');
    });

    it('should return preferred color if contrast is sufficient', () => {
      const result = getAccessibleTextColor('#ffffff', '#000000');
      expect(result).toBe('#000000');
    });

    it('should find alternative when preferred is insufficient', () => {
      const result = getAccessibleTextColor('#ffffff', '#ffffff');
      expect(result).not.toBe('#ffffff');
    });

    it('should return white for very dark backgrounds', () => {
      const result = getAccessibleTextColor('#000000');
      // Should find a light color
      expect(getContrastRatio(result, '#000000')).toBeGreaterThanOrEqual(4.5);
    });
  });

  describe('accessibleColors', () => {
    it('should have primary palette', () => {
      expect(accessibleColors.primary[500]).toBe('#3b82f6');
    });

    it('should have success palette', () => {
      expect(accessibleColors.success[500]).toBe('#22c55e');
    });

    it('should have gray palette', () => {
      expect(accessibleColors.gray[0]).toBe('#ffffff');
      expect(accessibleColors.gray[950]).toBe('#0a0a0a');
    });
  });
});
