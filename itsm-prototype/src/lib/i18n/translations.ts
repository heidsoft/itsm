// 翻译文本定义
export const translations = {
  'zh-CN': {
    dashboard: {
      title: '仪表板',
      description: '系统概览和关键指标',
    },
  },
  'en-US': {
    dashboard: {
      title: 'Dashboard',
      description: 'System overview and key metrics',
    },
  },
} as const;

export type Language = keyof typeof translations;
export type TranslationKey = keyof typeof translations['zh-CN'];

