/**
 * 顶栏「AI 助手」可见性回归锁（2026-11-21 方案 A）。
 *
 * 反模式：侧栏 AI 页受 ai capability fail-closed 门控，顶栏按钮却只按权限码
 * 永远可见。这里锁「权限 ∩ 能力」交集，并逐项验证 fail-closed 方向；
 * 组合判定放在纯函数里，避免依赖整枚 Header 的渲染环境。
 */
import { AI_ASSISTANT_PATH, shouldShowAIAssistant } from '../ai-assistant-visibility';
import { capabilityForPath } from '@/components/layout/sidebar/menu-config';
import type { ProductCapabilityState } from '@/lib/api/capability-api';

const aiReady = (patch: Partial<ProductCapabilityState> = {}): ProductCapabilityState => ({
  key: 'ai',
  maturity: 'ga',
  buildAvailable: true,
  deploymentReady: true,
  tenantReady: true,
  allowedActions: ['read'],
  dependencies: [],
  acceptanceVersion: 'test',
  ...patch,
});

const base = { capabilitiesLoading: false, permissionGranted: true };

describe('顶栏 AI 助手可见性（统一 ai capability 门控）', () => {
  it('能力申报就绪且持码时显示', () => {
    expect(shouldShowAIAssistant({ ...base, capabilities: [aiReady()] })).toBe(true);
  });

  it.each([
    { name: '能力矩阵未下发', capabilities: [] as ProductCapabilityState[] },
    { name: 'maturity=disabled', capabilities: [aiReady({ maturity: 'disabled' })] },
    { name: 'buildAvailable=false', capabilities: [aiReady({ buildAvailable: false })] },
    {
      name: 'deploymentReady=false（ITSM_AI_READY 未申报）',
      capabilities: [aiReady({ deploymentReady: false })],
    },
    { name: 'tenantReady=false', capabilities: [aiReady({ tenantReady: false })] },
    { name: '只下发了别的 capability', capabilities: [aiReady({ key: 'cmdb' })] },
  ])('$name 时隐藏（fail-closed）', ({ capabilities }) => {
    expect(shouldShowAIAssistant({ ...base, capabilities })).toBe(false);
  });

  it('能力矩阵加载中不提前露出按钮', () => {
    expect(
      shouldShowAIAssistant({ ...base, capabilities: [aiReady()], capabilitiesLoading: true })
    ).toBe(false);
  });

  it('无 ai:read 权限时隐藏（admin 兜底之外）', () => {
    expect(
      shouldShowAIAssistant({ ...base, capabilities: [aiReady()], permissionGranted: false })
    ).toBe(false);
  });

  it('门控 key 与侧栏同源，且覆盖三个归位后的 AI 能力页', () => {
    expect(AI_ASSISTANT_PATH).toBe('/ai/chat');
    expect(capabilityForPath(AI_ASSISTANT_PATH)).toBe('ai');
    expect(capabilityForPath('/tickets/ai-create')).toBe('ai');
    expect(capabilityForPath('/ai/approval')).toBe('ai');
    expect(capabilityForPath('/ai/audit')).toBe('ai');
  });
});
