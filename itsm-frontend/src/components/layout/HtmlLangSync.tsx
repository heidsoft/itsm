'use client';

import { useEffect } from 'react';
import { useI18n } from '@/lib/i18n/useI18n';

/**
 * 监听 i18n language 变化，同步 `<html lang>` 属性。
 *
 * 缺陷 O6-C：`app/layout.tsx` 的 `<html lang="zh-CN">` 是硬编码的 SSR 初值，
 * 全站没有任何补偿同步点，切到 English 后 `html lang` 仍停在 zh-CN——
 * 影响屏幕阅读器发音与浏览器翻译判定。
 *
 * `<html lang="zh-CN">` 保留为服务端渲染初值（避免 SSR/CSR 首帧不一致），
 * 客户端语言偏好变化后由此组件对齐。写法对齐既有的 `DayjsLocaleSync`。
 */
export const HtmlLangSync: React.FC = () => {
  const { language } = useI18n();

  useEffect(() => {
    document.documentElement.lang = language;
  }, [language]);

  return null;
};
