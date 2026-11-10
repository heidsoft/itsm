import React from 'react';
import { renderHook } from '@testing-library/react';
import { useI18n } from '../useI18n';
import { translations } from '../translations';

jest.mock('@/lib/user-preferences', () => ({
  userPreferences: {
    get: () => ({ language: 'zh-CN' }),
    subscribe: () => () => {},
    update: jest.fn(),
  },
}));

describe('English-only application', () => {
  it('renders English even when a previous Chinese preference is stored', () => {
    const { result } = renderHook(() => useI18n());
    expect(result.current.language).toBe('en-US');
    expect(result.current.t('common.confirm')).toBe('Confirm');
    expect(result.current.t('header.slaWarningContent', { ticketId: 'INC-123' })).toBe(
      'Ticket INC-123 is about to breach SLA'
    );
    expect(result.current.t('unknown.key')).toBe('unknown.key');
  });

  it('ships only English translations without Chinese fallback text', () => {
    expect(Object.keys(translations)).toEqual(['en-US']);
    expect(JSON.stringify(translations['en-US'])).not.toMatch(/\p{Script=Han}/u);
  });
});
