'use client';

import { useEffect } from 'react';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import 'dayjs/locale/en';
import { useI18n } from '@/lib/i18n/useI18n';

/**
 * 监听 i18n language 变化，动态切换 dayjs 的 locale，
 * 确保所有 dayjs 格式化输出跟随当前语言。
 */
export const DayjsLocaleSync: React.FC = () => {
  const { language } = useI18n();

  useEffect(() => {
    const locale = language === 'en-US' ? 'en' : 'zh-cn';
    dayjs.locale(locale);
  }, [language]);

  return null;
};
