/**
 * The MCP tool surface.
 *
 * 205 CLI commands exist; advertising all of them costs more context than the
 * work they enable and makes tool selection worse (the model drowns in near
 * duplicates like `ticket.update` vs `ticket.subtasks.update`). So the surface
 * is two-tier:
 *
 *   Tier 1 — a curated set of hot-path tools with full JSON Schema, so the
 *            common requests (list/get/create/comment/transition, CMDB impact,
 *            knowledge search, SLA stats, my tasks) are one call away.
 *   Tier 2 — `itsm_discover` / `itsm_describe` / `itsm_call`, which reach every
 *            remaining command without bloating `tools/list`.
 *
 * Schemas are derived from the CLI's own command registry (`CommandSpec.input`)
 * rather than restated here, so a new command or a changed flag reaches MCP for
 * free and cannot drift from the CLI contract.
 */
import { registry } from '../agent/registry.js';
import { describeCommand } from '../agent/spec.js';
/**
 * Curated hot path. Anything not here stays reachable through `itsm_call` —
 * the list is about discovery cost, not about capability.
 */
const CURATED = [
    {
        command: 'infra.me',
        tool: 'itsm_whoami',
        title: '当前身份与租户',
        description: '返回当前 MCP 连接使用的身份、角色、权限、租户以及 baseURL —— 排查 403 的第一步。',
    },
    {
        command: 'infra.doctor',
        tool: 'itsm_health',
        title: '实例健康自检',
        description: '对后端做一次端到端自检：可达性、版本、身份、各域读权限探针。冷启动或报错前先跑这个。',
    },
    {
        command: 'infra.search',
        tool: 'itsm_search',
        title: '全局搜索',
        description: '跨工单/事件/配置项/知识库的统一搜索。不知道资源属于哪个域时先用它定位。',
    },
    {
        command: 'ticket.list',
        tool: 'itsm_ticket_list',
        title: '工单列表',
        description: '按状态、优先级分页查询工单。用 limit/fields 控制返回体积。',
    },
    {
        command: 'ticket.get',
        tool: 'itsm_ticket_get',
        title: '工单详情',
        description: '读取单个工单的完整字段。',
    },
    {
        command: 'ticket.comments.list',
        tool: 'itsm_ticket_comments',
        title: '工单评论',
        description: '读取工单的评论/沟通记录。',
    },
    {
        command: 'ticket.create',
        tool: 'itsm_ticket_create',
        title: '创建工单',
        description: '创建工单（标题、描述、优先级必填）。写操作，成功后返回新工单。',
    },
    {
        command: 'ticket.update',
        tool: 'itsm_ticket_update',
        title: '更新工单',
        description: 'PATCH 局部更新工单。data 内字段遵循后端 DTO；若携带 version 则作为乐观锁，版本不符返回 409，需重新读取后再改。',
    },
    {
        command: 'ticket.comments.add',
        tool: 'itsm_ticket_comment_add',
        title: '添加工单评论',
        description: '在工单下追加评论（1–5000 字）。internal=true 为内部备注，不对请求人可见。',
    },
    {
        command: 'ticket.status',
        tool: 'itsm_ticket_transition',
        title: '工单状态流转',
        description: '按服务端状态机流转工单（如 in_progress / resolved / closed）。服务端会拒绝非法流转并返回 409，不要绕过状态机直接 PATCH status。',
    },
    {
        command: 'incident.list',
        tool: 'itsm_incident_list',
        title: '事件列表',
        description: '按状态、优先级、关键字分页查询事件。',
    },
    {
        command: 'incident.get',
        tool: 'itsm_incident_get',
        title: '事件详情',
        description: '读取单个事件的完整字段。',
    },
    {
        command: 'incident.triage',
        tool: 'itsm_incident_triage',
        title: '事件 AI 分诊',
        description: '基于标题+描述给出 AI 分诊建议（分类、优先级、影响面）。只做决策支持，不会自动处置任何对象。',
    },
    {
        command: 'change.list',
        tool: 'itsm_change_list',
        title: '变更列表',
        description: '按状态、风险等级查询变更单。',
    },
    {
        command: 'change.approvals',
        tool: 'itsm_change_approvals',
        title: '变更审批链',
        description: '读取变更的审批链状态（当前层级、各决策人、是否已满足会签阈值）。审批前必看，避免在未达阈值时误判通过。',
    },
    {
        command: 'cmdb.search',
        tool: 'itsm_cmdb_search',
        title: '配置项搜索',
        description: '搜索 CMDB 配置项。简单查询传 q；结构化条件（类型、属性、关系）用 data 传后端搜索体。',
    },
    {
        command: 'cmdb.impact',
        tool: 'itsm_cmdb_impact',
        title: '配置项影响分析',
        description: '从某个配置项出发做有界关系遍历，回答“改动它会影响谁”。变更风险评估的核心依据。',
    },
    {
        command: 'cloud.providers',
        tool: 'itsm_cloud_providers',
        title: '云厂商发现能力清单',
        description: '列出全部云厂商及其自动发现能力（notImplemented=true 表示可以建账号、手工登记资源，但不会自动发现）。建云账号前先查，避免为不支持的厂商配半天凭据。',
    },
    {
        command: 'cloud.accounts.list',
        tool: 'itsm_cloud_accounts_list',
        title: '云账号列表',
        description: '列出云账号（不分页，是 CI 表单与云资源页的选择器数据源）。可按 provider 过滤。',
    },
    {
        command: 'cloud.resources.list',
        tool: 'itsm_cloud_resources_list',
        title: '云资源列表',
        description: '分页列出云资源，可按 provider / cloudAccountId / serviceId / region / status / search 过滤。',
    },
    {
        command: 'cloud.discovery.jobs.list',
        tool: 'itsm_discovery_jobs_list',
        title: '发现任务列表',
        description: '列出云发现任务，可按 status 过滤。查某次发现的进度与失败原因用 itsm_call 调 cloud.discovery.jobs.get。',
    },
    {
        command: 'knowledge.search',
        tool: 'itsm_knowledge_search',
        title: '知识库检索',
        description: '检索知识文章（RAG 检索遵循权限与时效）。',
    },
    {
        command: 'sla.stats',
        tool: 'itsm_sla_stats',
        title: 'SLA 概览',
        description: 'SLA 看板汇总指标（达标率、违约数、各优先级分布）。',
    },
    {
        command: 'workflow.tasks',
        tool: 'itsm_workflow_tasks',
        title: '我的待办任务',
        description: '列出分配给我（或我作为候选人/候选组）的工作流人工任务。默认只返待处理（status=created）；要查已完成传 status=completed。',
        // GET /bpmn/tasks returns completed tasks too when status is omitted, so a
        // tool titled 待办 would hand back history mixed with work. Default the
        // filter here instead of silently renaming the endpoint's semantics.
        defaults: { status: 'created' },
    },
];
const META_TOOLS = [
    {
        name: 'itsm_discover',
        title: '检索可用能力',
        description: '在 {count} 条 ITSM 命令中按关键字或域检索，返回紧凑清单（命令名、摘要、风险、必填参数）。找不到需要的工具时先调用它，再用 itsm_describe 取完整入参契约，最后用 itsm_call 执行。',
        inputSchema: {
            type: 'object',
            properties: {
                query: { type: 'string', description: '关键字，匹配命令名与摘要（中英文均可），如 审批 / approval / 影响 / impact' },
                domain: {
                    type: 'string',
                    description: '限定域，如 ticket / incident / change / cmdb / cloud / knowledge / sla / workflow / admin / request / release / problem',
                },
                limit: { type: 'integer', minimum: 1, maximum: 100, description: '最多返回多少条，默认 25' },
            },
            required: [],
            additionalProperties: false,
        },
        annotations: { readOnlyHint: true, openWorldHint: false },
    },
    {
        name: 'itsm_describe',
        title: '查看命令入参契约',
        description: '返回单条命令的完整契约：usage、参数（类型/必填/枚举）、是否接受 data 请求体、后端端点、风险级别、示例。调用 itsm_call 之前用它消除参数猜测。',
        inputSchema: {
            type: 'object',
            properties: {
                command: { type: 'string', description: '命令名，如 ticket.create / change.approvals / cmdb.impact' },
            },
            required: ['command'],
            additionalProperties: false,
        },
        annotations: { readOnlyHint: true, openWorldHint: false },
    },
    {
        name: 'itsm_call',
        title: '执行任意 ITSM 命令',
        description: '执行任意注册表中的命令。高风险写操作（删除/终止/回滚/批量等）在 confirm=false 时不会执行，而是返回风险详情与“再次调用并置 confirm=true”的指令 —— 必须先取得用户同意。params 的键名与 itsm_describe 返回的参数名一致。',
        inputSchema: {
            type: 'object',
            properties: {
                command: { type: 'string', description: '命令名，如 ticket.tags-add / incident.major-incident / workflow.complete' },
                params: {
                    type: 'object',
                    description: '命令参数，键名与 itsm_describe 的 input.params 一致；positional 参数按 usage 顺序传入',
                    additionalProperties: true,
                },
                confirm: { type: 'boolean', description: '仅在用户明确同意高风险写操作后置 true' },
                preview: { type: 'boolean', description: '置 true 只返回将要调用的端点、风险与必填参数，不发请求' },
                limit: { type: 'integer', minimum: 1, description: '列表结果最多返回多少条' },
                fields: { type: 'array', items: { type: 'string' }, description: '只保留这些字段（支持 a.b 点号路径）' },
            },
            required: ['command'],
            additionalProperties: false,
        },
        annotations: { readOnlyHint: false, destructiveHint: true, idempotentHint: false, openWorldHint: false },
    },
];
/** Commands whose payload is a paged collection — they accept page/pageSize. */
function isListish(spec) {
    const verb = spec.name.split('.').pop() ?? '';
    if (/^(list|search|tasks|history|logs|versions|violations|overdue)$/.test(verb))
        return true;
    if (/^(stats|report|monitoring|performance|overview|metrics)$/.test(verb))
        return false;
    return spec.endpoints.some((e) => e.startsWith('GET ') && !/:id(\/|$)/.test(e));
}
/** Explicit array-of-scalars parameters (comma lists on the CLI). */
const CSV_PARAM = /(Ids|Users|recipients|tags|Cc)$/;
/** Paging params carry a fixed meaning, so they get a fixed description. */
const PAGING_DESCRIPTION = {
    page: '页码，从 1 开始',
    pageSize: '每页条数',
    limit: '最多返回多少条（客户端裁剪；meta 会给出 total 与 returned）',
};
/**
 * Usage strings name a slot (`<s>`, `<N>`) rather than describe it. Handing
 * that straight to a model produces `description: "<N>"`, which says nothing —
 * drop the slot marker and keep only real hints (`<new status>` → `new status`).
 */
function describeParam(p) {
    const canonical = PAGING_DESCRIPTION[p.name];
    const raw = p.description?.replace(/^value:\s*/, '').trim();
    const placeholder = raw?.match(/^<([^>]+)>$/);
    const hint = placeholder ? placeholder[1].trim() : raw;
    if (!hint || (placeholder && hint.length <= 2))
        return canonical;
    return canonical ? `${canonical}（${hint}）` : hint;
}
function paramSchema(p) {
    const description = describeParam(p);
    if (p.type === 'json') {
        // The derived description is English ("JSON request body; field names follow
        // the backend DTO") because it is shared with the CLI's `--help`. The tool
        // description is what the model reads most often, so keep it in the language
        // the operator writes in.
        return {
            type: 'object',
            additionalProperties: true,
            description: p.kind === 'body' ? '请求体：字段名遵循后端 DTO（camelCase），与 itsm_describe 的 bodyFields 对应' : 'JSON 参数',
        };
    }
    if (CSV_PARAM.test(p.name)) {
        return {
            type: ['string', 'array'],
            items: { type: ['string', 'integer'] },
            description: `逗号分隔字符串或数组${description ? `；${description}` : ''}`,
        };
    }
    if (p.type === 'boolean')
        return { type: 'boolean', ...(description ? { description } : {}) };
    if (p.type === 'number') {
        const integer = /Id$|Ids$|count$|page$|pageSize$|limit$|Level$/.test(p.name);
        return { type: integer ? 'integer' : 'number', ...(description ? { description } : {}) };
    }
    if (p.type === 'enum') {
        return { type: 'string', ...(p.values?.length ? { enum: p.values } : {}), ...(description ? { description } : {}) };
    }
    return { type: 'string', ...(p.required ? { minLength: 1 } : {}), ...(description ? { description } : {}) };
}
function buildInputSchema(spec, opts) {
    const properties = {};
    const required = [];
    for (const p of spec.input.params) {
        properties[p.name] = paramSchema(p);
        if (p.required)
            required.push(p.name);
    }
    if (opts.listish) {
        if (!properties['page'])
            properties['page'] = { type: 'integer', minimum: 1, description: '页码，从 1 开始' };
        if (!properties['pageSize'])
            properties['pageSize'] = { type: 'integer', minimum: 1, maximum: 200, description: '每页条数' };
        properties['limit'] = { type: 'integer', minimum: 1, description: '最多返回多少条（客户端裁剪；meta 会给出 total 与 returned）' };
        properties['fields'] = { type: 'array', items: { type: 'string' }, description: '只保留这些字段（支持 a.b 点号路径），压缩上下文' };
    }
    return { type: 'object', properties, required, additionalProperties: false };
}
function riskAnnotations(spec) {
    if (spec.risk === 'high-risk-write') {
        return { readOnlyHint: false, destructiveHint: true, idempotentHint: false, openWorldHint: false };
    }
    if (spec.risk === 'write') {
        return { readOnlyHint: false, destructiveHint: false, idempotentHint: false, openWorldHint: false };
    }
    return { readOnlyHint: true, openWorldHint: false };
}
let cachedSpecs = null;
export function specIndex() {
    if (!cachedSpecs) {
        cachedSpecs = new Map(registry().map((s) => [s.name, s]));
    }
    return cachedSpecs;
}
export function toolNameFor(command) {
    return `itsm_${command.replace(/[.-]/g, '_')}`;
}
/**
 * Resolve a caller-supplied identifier to a command.
 * Accepts `ticket.list`, `ticket list`, `itsm_ticket_list`, `Ticket.List`.
 */
export function resolveCommand(input) {
    const specs = specIndex();
    const raw = input.trim();
    const candidates = [
        raw,
        raw.toLowerCase(),
        raw.replace(/^itsm_/, '').replace(/_/g, '.'),
        raw.replace(/\s+/g, '.'),
    ];
    for (const candidate of candidates) {
        const exact = specs.get(candidate);
        if (exact)
            return exact;
    }
    const lower = raw.toLowerCase().replace(/^itsm_/, '').replace(/[\s_]+/g, '.');
    for (const [name, spec] of specs) {
        if (name.toLowerCase() === lower)
            return spec;
    }
    return undefined;
}
/** Cheap suggestion for an unknown command: same domain, shared verb prefix. */
export function suggestCommand(input) {
    const tokens = input.replace(/^itsm_/, '').replace(/[.\-_\s]+/g, ' ').trim().split(/\s+/);
    const domain = tokens[0]?.toLowerCase();
    if (!domain)
        return undefined;
    const names = [...specIndex().keys()];
    const inDomain = names.filter((n) => n.startsWith(`${domain}.`));
    if (inDomain.length && tokens[1]) {
        const verb = tokens[1].toLowerCase();
        const hit = inDomain.find((n) => n.slice(domain.length + 1).startsWith(verb.slice(0, 3)));
        if (hit)
            return hit;
    }
    // No verb-level match: suggest the closest domain's first command so the
    // caller still lands in the right neighbourhood instead of getting nothing.
    const domainHit = [...new Set(names.map((n) => n.split('.')[0]))].find((d) => d.startsWith(domain.slice(0, 3)));
    return inDomain[0] ?? (domainHit ? names.find((n) => n.startsWith(`${domainHit}.`)) : undefined);
}
/** Full tool list: curated first (stable order), then the three meta tools. */
export function buildTools() {
    const specs = specIndex();
    const tools = [];
    for (const entry of CURATED) {
        const spec = specs.get(entry.command);
        // A curated entry pointing at a deleted command must not silently vanish
        // from the surface: surface it at startup instead (checked by the test).
        if (!spec)
            throw new Error(`MCP catalog references unknown command '${entry.command}'`);
        const listish = isListish(spec);
        tools.push({
            name: entry.tool,
            title: entry.title,
            description: `${entry.description}${entry.note ? ` ${entry.note}` : ''}（命令 ${spec.name}${spec.risk !== 'normal' ? `，风险 ${spec.risk}` : ''}）`,
            inputSchema: buildInputSchema(spec, { listish }),
            annotations: { title: entry.title, ...riskAnnotations(spec) },
            command: spec.name,
            ...(entry.defaults ? { defaults: entry.defaults } : {}),
        });
    }
    // The registry grows; a hardcoded count in a description becomes a lie that
    // no test can catch. Substitute the live size at assembly time.
    for (const meta of META_TOOLS) {
        tools.push({ ...meta, description: meta.description.replace('{count}', String(specs.size)) });
    }
    return tools;
}
/**
 * Chinese → English needles.
 *
 * Command summaries are English (`Approve transition`, `Impact analysis`); an
 * operator querying in Chinese (`审批`, `影响面`) would match nothing and
 * conclude the capability is missing. Expanding the query instead of translating
 * the registry keeps one canonical vocabulary (the backend's) while making
 * discovery work in the language the operator actually types.
 */
const ZH_SYNONYMS = [
    [/工单|报障|服务台/, ['ticket']],
    [/事件|故障|事故/, ['incident']],
    [/问题|故障根因/, ['problem']],
    [/变更|改动/, ['change']],
    [/发布|上线/, ['release']],
    [/服务请求|申请|目录/, ['request', 'catalog']],
    [/配置项|资产|拓扑|关系/, ['cmdb', 'ci ', 'topology', 'relationship']],
    [/云|多云|云厂商|云账号|云资源|发现|厂商/, ['cloud', 'provider', 'discovery']],
    [/知识|文章|文档库|faq/i, ['knowledge', 'article']],
    [/sla|服务水平|时效|违约|合规/i, ['sla', 'violation', 'compliance']],
    [/工作流|流程|待办|任务|会签/, ['workflow', 'process', 'task', 'approv']],
    [/审批|审核|批准|驳回|决策/, ['approv', 'decision', 'reject']],
    [/影响|波及|关联/, ['impact', 'depend']],
    [/分诊|分派|派单|指派/, ['triage', 'assign']],
    [/评论|回复|沟通/, ['comment']],
    [/标签/, ['tag']],
    [/评分|满意度/, ['rating']],
    [/通知|消息|告警/, ['notification', 'alert']],
    [/连接器|集成|飞书|钉钉|企业微信/, ['connector']],
    [/审计|日志/, ['audit', 'log']],
    [/用户|成员|账号/, ['user', 'member']],
    [/团队|部门|组织/, ['team', 'department']],
    [/租户|多租户|msp/i, ['tenant', 'msp']],
    [/统计|看板|报表|趋势|指标/, ['stats', 'report', 'monitor', 'metric']],
    [/根因|复盘|回顾/, ['root-cause', 'pir']],
    [/风险/, ['risk']],
    [/回滚/, ['rollback']],
    [/重试/, ['retry']],
    [/搜索|检索|查找/, ['search', 'list']],
    [/创建|新建|提交|发起/, ['create', 'submit']],
    [/更新|修改|编辑/, ['update']],
    [/删除|移除/, ['delete', 'remove']],
    [/关闭|结束/, ['close']],
    [/恢复|重开/, ['reopen', 'resume', 'restore']],
    [/挂起|暂停/, ['suspend', 'pause']],
    [/终止|取消/, ['terminate', 'cancel']],
    [/版本/, ['version']],
    [/健康|自检|就绪/, ['health', 'doctor', 'ready']],
];
function expandQuery(query) {
    const needles = [query.toLowerCase()];
    for (const [pattern, terms] of ZH_SYNONYMS) {
        if (pattern.test(query))
            needles.push(...terms);
    }
    return needles;
}
/** `itsm_discover` payload: compact enough to stay inside a small context budget. */
export function discoverTools(query, domain, limit) {
    const specs = registry();
    const q = query?.trim();
    const needles = q ? expandQuery(q) : [];
    const matched = specs.filter((s) => {
        if (domain && s.name.split('.')[0] !== domain.toLowerCase())
            return false;
        if (!needles.length)
            return true;
        const haystack = `${s.name} ${s.summary} ${s.usage}`.toLowerCase();
        return needles.some((needle) => haystack.includes(needle));
    });
    // A synonym like `reject` also appears in unrelated summaries ("backend
    // rejects an empty note"). Rank by *where* the hit landed, not by how many
    // synonyms fired: a command whose name matches is the capability being asked
    // for, a summary that merely mentions it is context. Counting every needle
    // would rank a long summary ("approve | reject | delegate") above the actual
    // approval commands.
    const ranked = q
        ? matched
            .map((spec) => {
            const name = spec.name.toLowerCase();
            const summary = `${spec.summary} ${spec.usage}`.toLowerCase();
            // A hit that lands on a name boundary (`knowledge.*` for "知识") is the
            // capability being asked for. A hit buried inside another word is
            // noise: "knowledge" is a substring of "acknowledge", so searching
            // 知识 used to rank `incident.acknowledge` above every knowledge
            // command, and the tie-break (alphabetical) made it stick.
            const boundaryHit = needles.some((needle) => name === needle || name.startsWith(`${needle}.`) || name.includes(`.${needle}`) || name.startsWith(`${needle}-`));
            const nameHit = needles.some((needle) => name.includes(needle));
            const summaryHit = needles.some((needle) => summary.includes(needle));
            return { spec, score: (boundaryHit ? 5 : 0) + (nameHit ? 3 : 0) + (summaryHit ? 1 : 0) };
        })
            .sort((a, b) => b.score - a.score || a.spec.name.localeCompare(b.spec.name))
            .map((entry) => entry.spec)
        : matched;
    const curated = new Set(CURATED.map((c) => c.command));
    return {
        query: q ?? null,
        domain: domain ?? null,
        matched: matched.length,
        returned: Math.min(ranked.length, limit),
        commands: ranked.slice(0, limit).map((s) => ({
            command: s.name,
            summary: s.summary,
            risk: s.risk,
            required: s.input.params.filter((p) => p.required).map((p) => p.name),
            ...(curated.has(s.name) ? { hasNativeTool: true } : {}),
        })),
        next: matched.length > limit
            ? '结果被截断：加 domain 或更具体的关键字收窄，或用 limit 提高上限（最大 100）。'
            : '取完整入参契约：itsm_describe { command }；执行：itsm_call { command, params }。',
    };
}
/** `itsm_describe` payload: the machine-readable contract for one command. */
export function describeTool(command) {
    const spec = resolveCommand(command);
    if (!spec) {
        const suggestion = suggestCommand(command);
        throw new Error(`未知命令 '${command}'${suggestion ? `，是否要 '${suggestion}'？` : ''}。先用 itsm_discover 检索可用命令。`);
    }
    // Delegate to the CLI's own describeCommand: `itsm --json schema <cmd>` and
    // `itsm_describe` must return ONE contract shape. Hand-rolling a second
    // projection here previously dropped `input.deniedFlags` — the record that
    // tells a host `--password` is refused and which channel to use instead.
    return {
        ...describeCommand(spec),
        nativeTool: CURATED.find((c) => c.command === spec.name)?.tool,
    };
}
