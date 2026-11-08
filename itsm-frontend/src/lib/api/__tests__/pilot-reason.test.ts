import { splitPilotReason } from '../pilot-reason';

/**
 * 预览态三段式切分的契约测试。
 *
 * 切分只服务于展示：必须能区分新旧实现（整段平铺 vs 结构化），
 * 且任何情况下都不能吞掉接口原文——解析不出来时由页面如实标「契约不完整」。
 */
describe('splitPilotReason', () => {
  const canonical =
    '执行链与调度已闭环，五厂商适配器已装配。剩余阻断：每轮全量分页列举，真实账号下的发现未验收。晋升条件：增量水位线 + 真实账号发现验收。';

  it('把理由切成进展 / 剩余阻断 / 晋升条件三段，并保留原文', () => {
    const parts = splitPilotReason(canonical);
    expect(parts).not.toBeNull();
    expect(parts?.structured).toBe(true);
    expect(parts?.progress).toContain('执行链与调度已闭环');
    expect(parts?.blockers).toContain('真实账号下的发现未验收');
    expect(parts?.criteria).toBe('增量水位线 + 真实账号发现验收');
    expect(parts?.full).toBe(canonical);
  });

  it('没有「剩余阻断：」时用转折词切出阻断段', () => {
    const parts = splitPilotReason(
      'SSO 的 11 个端点与服务齐备，但没有与真实 IdP 的联调证据。晋升条件：至少一家真实 IdP 的登录验收。'
    );
    expect(parts?.structured).toBe(true);
    expect(parts?.progress).toContain('11 个端点');
    expect(parts?.blockers).toContain('没有与真实 IdP 的联调证据');
    expect(parts?.criteria).toContain('真实 IdP');
  });

  it('没有显式阻断标记时把「晋升条件：」之前的整段当作阻断，不丢信息', () => {
    const parts = splitPilotReason('回调验签只有单测，未与真实租户联调。晋升条件：真实租户收发 + 验签验收。');
    expect(parts?.structured).toBe(true);
    expect(parts?.progress).toBe('');
    expect(parts?.blockers).toContain('回调验签只有单测');
    expect(parts?.criteria).toBe('真实租户收发 + 验签验收');
  });

  it('缺「晋升条件：」时标记契约不完整，全文仍可回看', () => {
    const reason = '只剩一段描述，没有给出任何可验证的晋升线索。';
    const parts = splitPilotReason(reason);
    expect(parts?.structured).toBe(false);
    expect(parts?.criteria).toBe('');
    expect(parts?.full).toBe(reason);
    expect(parts?.blockers).toContain('只剩一段描述');
  });

  it('空理由返回 null，不给页面造出假的预览条目', () => {
    expect(splitPilotReason()).toBeNull();
    expect(splitPilotReason('   ')).toBeNull();
  });

  it('重复段时取最后一次晋升条件，且重复内容留在原文与阻断段里', () => {
    const duplicated = '进展。剩余阻断：X 未验收。晋升条件：条件一。剩余阻断：X 未验收。晋升条件：条件二。';
    const parts = splitPilotReason(duplicated);
    expect(parts?.criteria).toBe('条件二');
    expect(parts?.full).toBe(duplicated);
    expect(parts?.progress).toContain('条件一');
    expect(parts?.blockers).toContain('X 未验收');
  });

  it('只清理切分留下的尾部句号，不改写句子内容', () => {
    const parts = splitPilotReason('进展内容。剩余阻断：缺真实语料验收；晋升条件：跑一次质量验收。');
    expect(parts?.progress).toBe('进展内容');
    expect(parts?.blockers).toBe('缺真实语料验收');
    expect(parts?.criteria).toBe('跑一次质量验收');
  });
});
