'use client';

import { useEffect, useState, useCallback } from 'react';
import { userPreferences } from '@/app/lib/user-preferences';
import { translations, Language } from './translations';

/**
 * 国际化 Hook
 * 根据用户偏好设置返回对应的翻译文本
 */
export function useI18n() {
  const [language, setLanguage] = useState<Language>('zh-CN');

  // 初始化语言设置
  useEffect(() => {
    const prefs = userPreferences.get();
    const currentLang = prefs.language === 'en-US' ? 'en-US' : 'zh-CN';
    setLanguage(currentLang);

    // 订阅语言变化
    const unsubscribe = userPreferences.subscribe((prefs) => {
      const newLang = prefs.language === 'en-US' ? 'en-US' : 'zh-CN';
      setLanguage(newLang);
    });

    return unsubscribe;
  }, []);

  /**
   * 获取翻译文本
   * @param key 翻译键，支持嵌套路径，如 'dashboard.title'
   */
  const t = useCallback(
    (key: string): string => {
      const keys = key.split('.');
      let value: any = translations[language];

      for (const k of keys) {
        if (value && typeof value === 'object' && k in value) {
          value = value[k as keyof typeof value];
        } else {
          // 如果找不到翻译，尝试使用中文作为后备
          value = translations['zh-CN'];
          for (const fallbackKey of keys) {
            if (value && typeof value === 'object' && fallbackKey in value) {
              value = value[fallbackKey as keyof typeof value];
            } else {
              return key; // 如果还是找不到，返回原始键
            }
          }
          break;
        }
      }

      return typeof value === 'string' ? value : key;
    },
    [language]
  );

  /**
   * 切换语言
   */
  const changeLanguage = useCallback((lang: Language) => {
    userPreferences.update({ language: lang });
  }, []);

  return {
    t,
    language,
    changeLanguage,
  };
}

