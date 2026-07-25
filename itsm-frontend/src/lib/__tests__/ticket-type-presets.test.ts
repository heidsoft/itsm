/**
 * Tests for src/lib/ticket-type-presets.ts
 */

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { get: jest.fn(), post: jest.fn(), put: jest.fn(), delete: jest.fn(), patch: jest.fn() },
}));

import { ticketTypePresets } from '../ticket-type-presets';
import type { TicketTypePreset } from '../ticket-type-presets';

describe('ticket-type-presets', () => {
  it('exports ticketTypePresets array', () => {
    expect(Array.isArray(ticketTypePresets)).toBe(true);
    expect(ticketTypePresets.length).toBeGreaterThan(0);
  });

  it('each preset has required fields', () => {
    ticketTypePresets.forEach((preset: TicketTypePreset) => {
      expect(preset.id).toBeDefined();
      expect(typeof preset.id).toBe('string');
      expect(preset.code).toBeDefined();
      expect(typeof preset.code).toBe('string');
      expect(preset.name).toBeDefined();
      expect(preset.description).toBeDefined();
      expect(preset.icon).toBeDefined();
      expect(preset.color).toBeDefined();
      expect(preset.category).toBeDefined();
      expect(['low', 'medium', 'high', 'urgent']).toContain(preset.priority);
    });
  });

  it('has k8s-scale preset', () => {
    const preset = ticketTypePresets.find(p => p.id === 'k8s-scale');
    expect(preset).toBeDefined();
    expect(preset!.code).toBe('k8s_scale');
    expect(preset!.priority).toBe('high');
  });

  it('presets have unique ids and codes', () => {
    const ids = ticketTypePresets.map(p => p.id);
    const codes = ticketTypePresets.map(p => p.code);
    expect(new Set(ids).size).toBe(ids.length);
    expect(new Set(codes).size).toBe(codes.length);
  });

  it('presets with fields have valid field definitions', () => {
    ticketTypePresets
      .filter(p => p.fields && p.fields.length > 0)
      .forEach(preset => {
        preset.fields!.forEach(field => {
          expect(field.name).toBeDefined();
          expect(field.label).toBeDefined();
          expect(['text', 'textarea', 'select', 'number', 'date']).toContain(field.type);
        });
      });
  });

  it('select fields have options', () => {
    ticketTypePresets
      .filter(p => p.fields)
      .forEach(preset => {
        preset.fields!
          .filter(f => f.type === 'select')
          .forEach(field => {
            expect(field.options).toBeDefined();
            expect(field.options!.length).toBeGreaterThan(0);
            field.options!.forEach(opt => {
              expect(opt.label).toBeDefined();
              expect(opt.value).toBeDefined();
            });
          });
      });
  });
});
