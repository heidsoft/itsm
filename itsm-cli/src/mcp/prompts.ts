/**
 * MCP prompts — reusable, parameterised operating procedures.
 *
 * A prompt is the counterpart of a tool: the tool performs one call, the prompt
 * describes the *sequence* an operator would follow, including where the data
 * lives and what must not be assumed. They exist because the most common ITSM
 * failures are procedural, not technical: closing a ticket without its SLA
 * clock, approving a change nobody assessed, writing a postmortem from memory
 * instead of from correlated events.
 */

export interface McpPrompt {
  name: string;
  title: string;
  description: string;
  arguments: Array<{ name: string; description: string; required?: boolean }>;
}

export interface PromptResult {
  description: string;
  messages: Array<{ role: 'user'; content: { type: 'text'; text: string } }>;
}

export const PROMPTS: McpPrompt[] = [
  {
    name: 'triage_my_queue',
    title: '整理我的待办队列',
    description: '按 SLA 风险与优先级梳理当前身份的待处理任务，产出可执行的处置顺序。',
    arguments: [
      { name: 'focus', description: '只看某一类：ticket / incident / task，缺省全看', required: false },
    ],
  },
  {
    name: 'incident_postmortem',
    title: '事件复盘',
    description: '基于事件本体、关联告警/事件、CMDB 影响面与知识库，产出可评审的复盘报告。',
    arguments: [{ name: 'incidentId', description: '事件 ID', required: true }],
  },
  {
    name: 'change_risk_review',
    title: '变更风险评审',
    description: '在审批前把变更内容、影响面、审批链状态与历史依据放在一起评审，给出批准/驳回建议。',
    arguments: [{ name: 'changeId', description: '变更 ID 或编号（如 CHG-1）', required: true }],
  },
];

export function getPrompt(name: string, args: Record<string, string> | undefined): PromptResult {
  const prompt = PROMPTS.find((p) => p.name === name);
  if (!prompt) throw new Error(`未知提示词 '${name}'。可用：${PROMPTS.map((p) => p.name).join(', ')}`);

  const missing = prompt.arguments.filter((a) => a.required && !args?.[a.name]?.trim());
  if (missing.length) throw new Error(`提示词 ${name} 缺少必填参数：${missing.map((m) => m.name).join(', ')}`);

  return { description: prompt.description, messages: [{ role: 'user', content: { type: 'text', text: render(name, args ?? {}) } }] };
}

function render(name: string, args: Record<string, string>): string {
  switch (name) {
    case 'triage_my_queue':
      return [
        '请帮我梳理当前身份在 ITSM 里的待办队列，并给出处置顺序。',
        '',
        args.focus ? `只看 ${args.focus} 这一类。` : '工单、事件、工作流任务都要看。',
        '',
        '按这个顺序取数据（每一步都用工具拿真实数据，不要凭空推断）：',
        '1. `itsm_workflow_tasks` —— BPMN 里等我处理的人工任务（最有时效性）。',
        '2. `itsm_ticket_list` —— status 为 open/pending 的工单，关注 priority 与 SLA。',
        '3. `itsm_incident_list` —— status 为 open 的事件，关注 severity。',
        '4. 对前 5 个最紧急的对象用 `itsm_ticket_get` / `itsm_incident_get` 看细节，必要时用 `itsm_call` 调 `ticket.sla` 看 SLA 计时。',
        '',
        '输出要求：一张表（对象、编号、优先级、SLA 状态、为什么紧急、下一步动作），然后给出今天建议的处理顺序。',
        '只做读取与建议。需要变更状态或派人时先说明理由，等我确认后再调用写工具。',
      ].join('\n');

    case 'incident_postmortem':
      return [
        `请为事件 ${args['incidentId']} 写一份复盘报告。`,
        '',
        '取数步骤（全部用工具取真实数据）：',
        `1. \`itsm_incident_get\`（id=${args['incidentId']}）—— 事件本体：标题、严重度、时间线、影响范围。`,
        '2. `itsm_call` 调 `incident.events` 与 `incident.alerts` —— 关联事件与告警，确定“什么时候开始不对”。',
        '3. `itsm_call` 调 `incident.root-cause-get` 与 `incident.impact-assessment` —— 已有的根因与影响评估，不要覆盖它们，只补全。',
        '4. `itsm_cmdb_search` + `itsm_cmdb_impact` —— 从受影响配置项出发做影响面遍历，说明波及的业务链路。',
        '5. `itsm_knowledge_search` —— 是否有同类历史事件/已知错误，避免重复归因。',
        '',
        '报告结构：摘要（3 行内）→ 时间线（含数据来源）→ 直接原因与根本原因（区分）→ 影响面（配置项与业务）→ 已有的处置动作 → 改进项（每条要有负责人角色与验收标准）→ 未解问题。',
        '明确区分“数据支持的结论”与“推测”。推测必须标注为推测。',
      ].join('\n');

    case 'change_risk_review':
      return [
        `请对变更 ${args['changeId']} 做审批前风险评审。`,
        '',
        '取数步骤：',
        `1. \`itsm_call\` 调 \`change.get\`（id=${args['changeId']}）—— 变更内容、计划窗口、实施人、回滚方案。`,
        '2. `itsm_change_approvals` —— 审批链状态：当前层级、已决策人、是否满足会签阈值。',
        '3. `itsm_call` 调 `change.risk` 与 `change.pir` —— 风险评估与历史复盘（如有）。',
        '4. `itsm_cmdb_impact` —— 对变更涉及的配置项做影响分析；配置项 ID 从变更详情里取。',
        '5. `itsm_knowledge_search` —— 是否有同类变更的失败记录或已知坑。',
        '',
        '评审输出：风险等级建议（低/中/高）及依据 → 缺失信息清单（回滚方案？验证方案？影响面？）→ 批准/有条件批准/驳回的建议与条件。',
        '注意：不要自行调用审批工具。审批是人的决定，你只提供判断依据；如果审批链未达阈值，明确说明这一点。',
      ].join('\n');

    default:
      throw new Error(`提示词 '${name}' 没有内容模板`);
  }
}
