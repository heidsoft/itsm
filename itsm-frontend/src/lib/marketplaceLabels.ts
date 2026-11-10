/**
 * 应用市场的展示文案映射。
 *
 * 后端返回的是英文枚举（type: connector/skill/plugin；category: im/ai-skill/integration；
 * tag: plugin/sync/triage 等）。列表页与详情页必须用同一套映射，否则同一件商品在
 * 两个页面会显示不同语言。
 */

export type MarketplaceItemType = 'connector' | 'skill' | 'plugin';

export const MARKETPLACE_TYPE_LABELS: Record<MarketplaceItemType, string> = {
  connector: 'Connectors',
  skill: 'AI skills',
  plugin: 'Plug-in',
};

const CATEGORY_LABELS: Record<string, string> = {
  im: 'Instant messaging',
  'ai-skill': 'AI skills',
  integration: 'Integrated',
  notification: 'Notification',
};

const TAG_LABELS: Record<string, string> = {
  plugin: 'Plug-in',
  skill: 'Skill',
  sync: 'Synchronization',
  triage: 'Intelligent triage',
};

export const marketplaceCategoryLabel = (code?: string | null): string => {
  if (!code) return 'Uncategorized';
  return CATEGORY_LABELS[code] ?? code;
};

export const marketplaceTagLabel = (tag: string): string => TAG_LABELS[tag.toLowerCase()] ?? tag;
