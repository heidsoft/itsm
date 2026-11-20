/**
 * 顶栏「AI 助手」可见性判定（2026-11-21 方案 A）。
 *
 * 与侧栏 AI 能力页共用同一套 capability 门控（isCapabilityVisible +
 * capabilityForPath）：AI 未申报就绪时，顶栏按钮与侧栏入口一起隐藏，
 * 不允许「侧栏整条消失、顶栏按钮却永远亮着」的双口径。
 *
 * 权限码只回答「这个人能不能用」，capability 回答「这个部署形态有没有」，
 * 两者是交集；能力矩阵加载中按 fail-closed 处理，避免先闪一下再消失。
 */

import { isCapabilityVisible, type ProductCapabilityState } from '@/lib/api/capability-api';
import { capabilityForPath } from '@/components/layout/sidebar/menu-config';

export const AI_ASSISTANT_PATH = '/ai/chat';

export interface AIAssistantVisibility {
  capabilities: ProductCapabilityState[];
  capabilitiesLoading: boolean;
  permissionGranted: boolean;
}

export function shouldShowAIAssistant({
  capabilities,
  capabilitiesLoading,
  permissionGranted,
}: AIAssistantVisibility): boolean {
  if (capabilitiesLoading) return false;
  if (!permissionGranted) return false;

  const key = capabilityForPath(AI_ASSISTANT_PATH);
  // 门控规则被误删时同样 fail-closed：宁可不显示，也不能让未就绪能力露出入口。
  if (!key) return false;

  return isCapabilityVisible(capabilities.find(item => item.key === key));
}
