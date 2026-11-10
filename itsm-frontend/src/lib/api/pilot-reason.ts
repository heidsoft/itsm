/**
 * 预览态理由的三段式视图。
 *
 * 后端 `pilotReason` 是一整段自然语言，把「已闭环进展 / 剩余阻断 / 晋升条件」
 * 写在一起（AI 条目一度长达 2300 字，阻断压在段尾）。控制台若整段平铺，
 * 读者无法回答「这项还差什么、怎么晋升」，页面注释里写的「预览态可核对」
 * 就落不了地。这里只做展示层切分：不丢原文、不改 API 契约，解析不出来时
 * 由调用方回退显示全文并如实标记契约不完整。
 *
 * 独立成模块而不是挂在 capability-api 上：能力矩阵页面测试会把 API 客户端
 * 整个 mock 掉，纯解析工具不该随网络层一起消失。
 */
export interface PilotReasonParts {
  /** 「晋升条件：」之前的已闭环进展，可能为空。 */
  progress: string;
  /** 仍成立的阻断项；没有显式标记时取「晋升条件：」之前的整段。 */
  blockers: string;
  /** 「晋升条件：」之后的原文，保持原样不拆分，避免把一句话切成伪条件。 */
  criteria: string;
  /** 是否切出了实质性的晋升条件。false 表示契约不完整，必须如实暴露。 */
  structured: boolean;
  /** 未改动的原文，供详情面板完整回看。 */
  full: string;
}

// Legacy markers remain readable for saved reasons from earlier releases.
const CRITERIA_MARKS = ['Promotion criteria:', '晋升条件：'];
const BLOCKER_MARKS = ['Remaining blockers:', 'Remaining blocker:', '剩余阻断：'];
const TRANSITION_MARKS = [' However, ', ' but ', '但'];

const lastMarker = (text: string, markers: string[]) =>
  markers.reduce(
    (latest, marker) => {
      const at = text.lastIndexOf(marker);
      return at > latest.at ? { at, length: marker.length } : latest;
    },
    { at: -1, length: 0 }
  );

export const splitPilotReason = (reason?: string): PilotReasonParts | null => {
  const full = (reason ?? '').trim();
  if (!full) return null;

  // 取最后一次出现：若后端又出现重复段，仍能把完整的晋升条件切出来，
  // 重复内容留在 blockers 里而不吞掉任何原文（后端另有守卫禁止重复）。
  const criteriaMarker = lastMarker(full, CRITERIA_MARKS);
  const criteriaAt = criteriaMarker.at;
  const criteria = criteriaAt >= 0 ? full.slice(criteriaAt + criteriaMarker.length).trim() : '';
  const head = (criteriaAt >= 0 ? full.slice(0, criteriaAt) : full).trim();

  let progress = '';
  let blockers = head;
  let blockerMarker = lastMarker(head, BLOCKER_MARKS);
  if (blockerMarker.at < 0) blockerMarker = lastMarker(head, TRANSITION_MARKS);
  if (blockerMarker.at >= 0) {
    progress = head.slice(0, blockerMarker.at).trim();
    blockers = head.slice(blockerMarker.at + blockerMarker.length).trim();
  }

  // 清理切分留下的尾部句号，但不改写句子内容。
  const tidy = (value: string) => value.replace(/[。；;]\s*$/, '').trim();
  return {
    progress: tidy(progress),
    blockers: tidy(blockers),
    criteria: tidy(criteria),
    structured: criteria.length > 0,
    full,
  };
};
