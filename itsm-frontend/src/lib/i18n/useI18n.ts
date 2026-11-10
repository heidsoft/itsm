'use client';

import { useCallback } from 'react';
import { userPreferences } from '@/lib/user-preferences';
import type { Language } from './translations';
import { translations } from './translations';

/** English-only translations, including the first server/client render. */
export function useI18n() {
  const t = useCallback((key: string, params?: Record<string, string | number>): string => {
    let value: unknown = translations['en-US'];
    for (const part of key.split('.')) {
      if (
        !value ||
        typeof value !== 'object' ||
        !Object.prototype.hasOwnProperty.call(value, part)
      ) {
        return key;
      }
      value = (value as Record<string, unknown>)[part];
    }
    const result = typeof value === 'string' ? value : key;
    return params
      ? result.replace(/\{([^{}]+)\}/g, (placeholder, name: string) =>
          Object.prototype.hasOwnProperty.call(params, name) ? String(params[name]) : placeholder
        )
      : result;
  }, []);

  const changeLanguage = useCallback((_language: Language) => {
    userPreferences.update({ language: 'en-US' });
  }, []);

  return { t, language: 'en-US' as const, changeLanguage };
}
