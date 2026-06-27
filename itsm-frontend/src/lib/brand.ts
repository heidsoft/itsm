/**
 * AI-Native ITSM · 品牌常量
 *
 * 品牌策略（v1.0 GA）：
 * - 英文名：AI-Native ITSM
 * - 中文名：智能服务管理
 * - 中文标语：让 IT 服务更高效
 * - 英文标语：LLM-first Service Management
 *
 * 使用规则：
 * 1. 用户可见文案统一使用 BRAND.NAME
 * 2. Footer / About / Login 等核心位置必须中英双语
 * 3. 不允许在代码中硬编码「ITSM」单独作为产品名出现
 */

export const BRAND = {
  NAME: 'AI-Native ITSM',
  NAME_CN: '智能服务管理',
  SLOGAN: 'LLM-first Service Management',
  SLOGAN_CN: '让 IT 服务更高效',
  COPYRIGHT: `© ${new Date().getFullYear()} AI-Native ITSM. Licensed under Apache 2.0.`,
  COPYRIGHT_CN: `© ${new Date().getFullYear()} 智能服务管理 · Apache 2.0 开源协议`,
  HOMEPAGE: 'https://cloudmesh.top/',
  REPO: 'https://github.com/heidsoft/itsm',
} as const;

export type Brand = typeof BRAND;
