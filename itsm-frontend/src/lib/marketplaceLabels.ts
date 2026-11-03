/**
 * 应用市场的展示文案映射。
 *
 * 后端返回的是英文枚举（type: connector/skill/plugin；category: im/ai-skill/integration；
 * tag: plugin/sync/triage 等）。列表页与详情页必须用同一套映射，否则同一件商品在
 * 两个页面会显示不同语言。
 */

export type MarketplaceItemType = 'connector' | 'skill' | 'plugin';

export const MARKETPLACE_TYPE_LABELS: Record<MarketplaceItemType, string> = {
  connector: '连接器',
  skill: 'AI 技能',
  plugin: '插件',
};

const CATEGORY_LABELS: Record<string, string> = {
  im: '即时通讯',
  'ai-skill': 'AI 技能',
  integration: '集成',
  notification: '通知',
};

const TAG_LABELS: Record<string, string> = {
  plugin: '插件',
  skill: '技能',
  sync: '同步',
  triage: '智能分诊',
};

export const marketplaceCategoryLabel = (code?: string | null): string => {
  if (!code) return '未分类';
  return CATEGORY_LABELS[code] ?? code;
};

export const marketplaceTagLabel = (tag: string): string => TAG_LABELS[tag.toLowerCase()] ?? tag;
