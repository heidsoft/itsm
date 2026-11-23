'use client';

import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
  Card,
  Table,
  Select,
  Space,
  Tag,
  Typography,
  Button,
  Empty,
  App,
  Input,
  Modal,
} from 'antd';
import { CheckCircle2, XCircle, RefreshCw, ShieldAlert } from 'lucide-react';

import {
  aiGetToolApprovals,
  aiApproveTool,
  aiGetRunInvocations,
  aiReviewToolInvocation,
  aiGetControlPlane,
  type ToolApproval,
  type ToolApprovalListResponse,
  type ToolRunInvocationsResponse,
  type ToolDryRunResult,
  type AgentControlPlaneOverview,
} from '@/lib/api/ai-api';
import { useAuthStoreHydration } from '@/lib/store/auth-store';
import { usePermissions } from '@/lib/hooks/use-permissions';
import { PERM } from '@/lib/permissions';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text } = Typography;

const STATE_LABELS: Record<string, string> = {
  pending: '待审批',
  approved: '已通过',
  rejected: '已驳回',
  auto: '自动执行',
  // AI 主体（agent）专属状态：审批人数由权限码策略的 minApprovals 决定，
  // 且后端强制「发起人不可自批」。
  pending_agent: 'AI 待审批',
  pending_agent_two: 'AI 待双人审批',
  awaiting_second: 'AI 待二审',
};

const stateColor = (s: string): string => {
  switch (s) {
    case 'pending':
      return 'gold';
    case 'approved':
      return 'green';
    case 'rejected':
      return 'red';
    case 'pending_agent':
      return 'gold';
    case 'pending_agent_two':
    case 'awaiting_second':
      return 'volcano';
    default:
      return 'default';
  }
};

// AI 发起的双重审批进度：awaiting_second 表示首审已通过、待第二位审批人。
const approvalProgress = (s: string, minApprovals?: number): string | null => {
  if (s === 'pending_agent') return minApprovals && minApprovals > 1 ? '0/2' : '0/1';
  if (s === 'pending_agent_two') return '0/2';
  if (s === 'awaiting_second') return '1/2';
  return null;
};

const RISK_LABELS: Record<string, string> = {
  low: '低风险',
  medium: '中风险',
  high: '高风险',
  critical: '危急',
  unknown: '未知',
};

/** 判定主体的人类可读名。不暴露 principal 的引擎取值（human/agent/service）。 */
const PRINCIPAL_LABELS: Record<string, string> = {
  human: '人类',
  agent: 'AI',
  service: '服务',
};

/**
 * 策略比对四态的展示口径。
 *
 * 四态**不可合并**，这是本组件存在的核心理由：
 * - identical     → 「判据未变」，原结论可沿用（安心点通过）
 * - changed       → 「策略已改」，原结论不能沿用（须按新策略重判）
 * - unknown       → 「读不出策略」，不能断言未变（提示运维，别当作没变）
 * - not_applicable→ 「本就不涉及策略」，比较无意义（而非「未变」）
 *
 * 把 unknown 显示成 identical 是最危险的假阴性：审批人会以为策略没动过，
 * 实际上系统根本没读到策略。因此 unknown 必须显式给出红色提示且文案
 * 不得出现「未变」二字。
 */
const POLICY_COMPARISON_LABELS: Record<
  NonNullable<ToolDryRunResult['policyComparison']>,
  { label: string; color: string; hint: string }
> = {
  identical: {
    label: '策略未变',
    color: 'green',
    hint: '两次判定的策略指纹相同。指纹取自风险级/允许主体/是否需审批/审批人数/是否启用，'
      + '相同即表示放行判据未变，本次复核结论与原判定一致。',
  },
  changed: {
    label: '策略已变更',
    color: 'red',
    hint: '排队期间该权限码的策略被修改（风险级、允许主体或审批人数任一变化都会改指纹）。'
      + '原判定结论不能直接沿用，请按本次复核结论判断。',
  },
  unknown: {
    label: '策略版本不可读',
    color: 'red',
    hint: '至少一侧未能取得策略指纹，因此**不能**断言策略未变。'
      + '这不等于「没变化」——请检查权限目录是否可用后再决定。',
  },
  not_applicable: {
    label: '不涉及策略比对',
    color: 'default',
    hint: '两次判定均无策略参与（人类主体走角色 RBAC，不查策略信封），没有可比对的对象。'
      + '既不表示「未变」也不表示「已变」。',
  },
};

const PolicyComparisonNote: React.FC<{
  comparison: NonNullable<ToolDryRunResult['policyComparison']>;
  baseline?: string;
  current?: string;
}> = ({ comparison, baseline, current }) => {
  // 未知取值不静默渲染成「未变」：后端新增态时前端必须显式暴露，而不是
  // 让它悄悄落进一个看似安全的默认分支。
  const view = POLICY_COMPARISON_LABELS[comparison];
  if (!view) {
    return (
      <Text type="danger" style={{ fontSize: 12 }}>
        策略比对结果无法识别（{comparison}），请勿据此判断策略是否变更。
      </Text>
    );
  }
  return (
    <TooltipWrapper text={view.hint}>
      <Space size={6} wrap>
        <Text type="secondary" style={{ fontSize: 12 }}>
          策略比对
        </Text>
        <Tag color={view.color}>{view.label}</Tag>
        <Text type="secondary" style={{ fontSize: 12 }}>
          记录时 {baseline ? baseline.slice(0, 8) : '（无）'} → 当前{' '}
          {current ? current.slice(0, 8) : '（无）'}
        </Text>
      </Space>
    </TooltipWrapper>
  );
};

const riskColor = (r?: string): string => {
  switch (r) {
    case 'low':
      return 'green';
    case 'medium':
      return 'gold';
    case 'high':
      return 'volcano';
    case 'critical':
      return 'red';
    default:
      return 'default';
  }
};

const permissionColor = (p?: string): string => {
  if (p === 'passed') return 'green';
  if (p === 'denied') return 'red';
  if (p === 'skipped') return 'default';
  return 'default';
};

const prettyArgs = (raw?: string): string => {
  if (!raw) return '-';
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
};

/** 展开行持有的「整轮证据链」状态。runId 与 items 必须成对存，防止串台。 */
interface ToolRunChain {
  runId: string;
  loading: boolean;
  items: ToolApproval[];
  error?: string;
}

/**
 * 展开行持有的「决策复核」状态。
 *
 * 与 ToolRunChain 一样按记录 id 分桶，理由相同且更紧迫：复核是**逐条**的
 * （每条调用按各自的工具、参数与原发起主体判定），单值 state 下同时展开
 * A、B 两行，后发请求落地时会把 A 的复核结论画到 B 上——而复核结论直接
 * 影响「这个人要不要点通过」，串台等于让审批人按别人的结论做决定。
 *
 * key 用记录 id而不是 runId：同一轮里可能有多条待批调用，各有各的判定输入。
 * 这也是按 id 而非 runId 的**硬理由**：复核接口的入参就是记录 id。
 */
interface ToolPreflight {
  toolName: string;
  loading: boolean;
  result?: ToolDryRunResult;
  error?: string;
}

/**
 * 控制面概览的加载状态。
 *
 * `error` 必须与「确实没有任何待批」分开：两者在界面上都会让表格是空的，
 * 但含义完全相反——前者是「读不到」，后者是「读到了、队列干净」。
 * 把读失败渲染成空态，运维会以为队列已清空而不再处理；
 * 而真正的失败（后端 500、权限不足、超时）就这样被掩盖了。
 */
interface ControlPlaneState {
  loading: boolean;
  data?: AgentControlPlaneOverview;
  error?: string;
}

/**
 * 策略版本三态的展示口径。**不在展示层归一化**：
 * 空串（历史行）与 unknown（读不出）含义不同，前者是「当时没这概念」，
 * 后者是「有这概念但这次没取到」——后者意味着运维该查一下策略目录。
 * 把两者都显示成「-」会让后一种故障永远不被发现。
 */
const policyVersionView = (v?: string): { label: string; hint: string; color: string } => {
  if (!v) {
    return {
      label: '历史记录',
      hint: '该调用发生在策略版本字段上线之前，因此未记录放行依据的版本。',
      color: 'default',
    };
  }
  if (v === 'unknown') {
    return {
      label: '未取到',
      hint: '本次放行时未能读取策略目录，无法追溯依据哪一版策略。需检查权限目录是否可用。',
      color: 'red',
    };
  }
  return {
    label: v.slice(0, 8),
    hint: `完整指纹 ${v}；相同指纹 ⟺ 放行判据（风险级/允许主体/是否需审批/审批人数/是否启用）相同。`,
    color: 'blue',
  };
};

/**
 * 控制面概览面板：把「队列 / 策略覆盖 / 工具可见面 / 最近拒绝」四个事实面
 * 放在审批队列上方，一次看清。
 *
 * 为什么放在这一页而不是新开一页：审批人打开这一页的真实问题是
 * 「我现在该处理什么、还有多少没处理、为什么有些调用一直被我挡」。
 * 这三问的答案分散在四个接口里，运维只能靠人工拼装——
 * 于是一个最常见的问题「昨天改了策略到底生效没有」没有确定答案。
 *
 * 展示纪律：
 * - 策略三类**分开显示**（已配置 / 未配置 / 配置损坏）。合并任意两类都会让
 *   风险面被低估，尤其是「配置损坏」——它看起来配了策略，鉴权时却全拒。
 * - 队列四态**分开显示**，`awaitingSecond` 标出「已有一票待二审」，
 *   因为那批已经批了一半、再不处理就卡住了。
 * - 工具面显示 `visible / total` 并回显**当前角色**：同一租户同一时刻，
 *   管理员看到 14 个、普通技术员看到 6 个，不说明按谁算的数字没有含义。
 */
const ControlPlanePanel: React.FC<{ state: ControlPlaneState; onRefresh: () => void }> = ({
  state,
  onRefresh,
}) => {
  const { data, loading, error } = state;

  // 读失败必须显式告知，且**不落成空态**：后端任一真源失败即整单 500
  // （fail-closed）。若这里渲染成「0 待批」，运维会以为队列已清空而不再处理。
  if (error) {
    return (
      <Card size="small" style={{ borderColor: '#ffccc7' }}>
        <Space align="center">
          <Text type="danger" strong>
            控制面状态读取失败
          </Text>
          <Text type="secondary">
            {error}——这不代表队列为空，请重试或检查后端
          </Text>
          <a onClick={onRefresh}>
            <RefreshCw size={14} /> 重试
          </a>
        </Space>
      </Card>
    );
  }

  return (
    <Card
      size="small"
      loading={loading && !data}
      title={
        <Space size={8}>
          <ShieldAlert size={16} color="#1677ff" />
          <Text strong>控制面状态</Text>
          {data && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              采集于 {formatDateTime(data.gatheredAt)}
            </Text>
          )}
        </Space>
      }
      extra={
        <a onClick={onRefresh}>
          <RefreshCw size={14} /> 刷新
        </a>
      }
    >
      {data && (
        <div className="grid gap-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))' }}>
          {/* 待批队列：四态分开。awaitingSecond 单独高亮因为首票不放行、最容易卡住。 */}
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>待批队列</Text>
            <div style={{ marginTop: 4 }}>
              <Space size={6} wrap>
                <Tag color={data.queue.pending > 0 ? 'gold' : 'default'}>
                  待审批 {data.queue.pending}
                </Tag>
                <Tag color={data.queue.pendingAgent > 0 ? 'gold' : 'default'}>
                  AI 待批 {data.queue.pendingAgent}
                </Tag>
                <Tag color={data.queue.pendingAgentTwo > 0 ? 'volcano' : 'default'}>
                  AI 待双人 {data.queue.pendingAgentTwo}
                </Tag>
                <Tag color={data.queue.awaitingSecond > 0 ? 'red' : 'default'}>
                  已一票待二审 {data.queue.awaitingSecond}
                </Tag>
              </Space>
              <div style={{ marginTop: 4 }}>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  合计 {data.queue.total} 条
                  {data.queue.awaitingSecond > 0 && '（含已批一半的，需尽快安排第二人复审）'}
                </Text>
              </div>
            </div>
          </div>

          {/* 策略覆盖：三类分开，unreadable 用红色因为它是 fail-closed 全拒。 */}
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>策略覆盖</Text>
            <div style={{ marginTop: 4 }}>
              <Space size={6} wrap>
                <Tag color="blue">已配置 {data.policies.withPolicy}</Tag>
                <Tag color="default">未配置 {data.policies.policyFree}</Tag>
                <Tag color={data.policies.unreadable > 0 ? 'red' : 'default'}>
                  配置损坏 {data.policies.unreadable}
                </Tag>
              </Space>
              <div style={{ marginTop: 4 }}>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  共 {data.policies.total} 个权限点
                  {data.policies.unreadable > 0 &&
                    '；「配置损坏」表示策略解不出，这类权限点当前一律拒绝'}
                </Text>
              </div>
            </div>
          </div>

          {/* 工具可见面：必须回显角色，否则数字没有确定含义。 */}
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>工具可见性</Text>
            <div style={{ marginTop: 4 }}>
              <Text strong>
                {data.visibleTools} / {data.totalTools}
              </Text>
              <Text type="secondary" style={{ fontSize: 12 }}> 个工具当前角色可用</Text>
              <div style={{ marginTop: 4 }}>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  按角色「{data.role}」计算；被挡住的工具也列出，便于判断是否策略过严
                </Text>
              </div>
            </div>
          </div>

          {/* 最近拒绝：不含预演行，只列真实被拦的调用。 */}
          <div>
            <Text type="secondary" style={{ fontSize: 12 }}>最近被拒</Text>
            <div style={{ marginTop: 4 }}>
              {data.recentDenial.length === 0 ? (
                <Text type="secondary" style={{ fontSize: 12 }}>
                  近期没有真实调用被拒
                </Text>
              ) : (
                <Space orientation="vertical" size={2} style={{ width: '100%' }}>
                  {data.recentDenial.slice(0, 3).map(d => (
                    <Text key={d.id} style={{ fontSize: 12 }} ellipsis>
                      <Tag color="red">{d.toolName}</Tag>
                      {d.reason || '策略不允许'}
                    </Text>
                  ))}
                  {data.recentDenial.length > 3 && (
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      另有 {data.recentDenial.length - 3} 条
                    </Text>
                  )}
                </Space>
              )}
            </div>
          </div>
        </div>
      )}
    </Card>
  );
};

const AIApprovalQueue: React.FC = () => {
  const { message } = App.useApp();
  useAuthStoreHydration();
  const { hasPermission } = usePermissions();
  // 通过/驳回按钮与后端审批路由同码：router/ai_routes.go 的
  // POST /agent/tools/:id/approve 挂 RequirePermission("ai","approve")，
  // 即 ai:approve。前端若用 ai:write，会与后端错位——持有 ai:approve 的
  // 审批人（一线/管理）看不到按钮，而只有 ai:write 的账号看到按钮却会被 403。
  const canApprove = hasPermission(PERM.ai.approve);

  const [items, setItems] = useState<ToolApproval[]>([]);
  // 列表初值必须为 true：mount 必定发起首次拉取，初值 false 会让首帧先渲染
  // 空表格，再被真实数据覆盖。
  const [listLoading, setListLoading] = useState(true);
  const [state, setState] = useState<string>('pending');
  const [rejectId, setRejectId] = useState<number | null>(null);
  const [rejectReason, setRejectReason] = useState('');
  // 展开某条调用时按运行轮次取回的整轮证据链，**按 runId 分桶**。
  //
  // 为什么不是单个 chain 变量：antd Table 的 expandable 默认允许多行同时展开，
  // 单值状态下后展开的行会把先展开行的内容顶掉，先展开那行则永远停在「加载中」。
  // 这是「一次只放一个」的假实现——按下两行就能复现。
  const [runChains, setRunChains] = useState<Record<string, ToolRunChain>>({});
  // 每个 runId 一条序号：同一轮被重复展开（收起再展开）时，只有最后一次响应能落地。
  const runSeqRef = useRef<Record<string, number>>({});
  // 决策复核结果，按记录 id 分桶（理由见 ToolPreflight 注释）。
  const [preflights, setPreflights] = useState<Record<number, ToolPreflight>>({});
  // 每条记录一条序号，规则同 runSeqRef。
  const preflightSeqRef = useRef<Record<number, number>>({});

  // 控制面概览：初值 loading=true（mount 必定发起首次拉取）。
  const [controlPlane, setControlPlane] = useState<ControlPlaneState>({ loading: true });
  // 概览请求的 generation 序号。
  //
  // 为什么需要：概览会被「刷新」按钮与队列状态切换重复触发，两次请求并发时
  // 先发后到的响应会把新数据顶掉（界面上表现为「点了刷新，数字反而回到旧的」）。
  // 单值 state + 无守卫是这类竞态的典型形态，且它不会报错，只会让运维
  // 看到一个过期快照——比读失败更难发现。
  const controlPlaneSeqRef = useRef(0);

  const fetchControlPlane = useCallback(async () => {
    const seq = ++controlPlaneSeqRef.current;
    setControlPlane(prev => ({ ...prev, loading: true, error: undefined }));
    try {
      const data = await aiGetControlPlane();
      // 只有最后一次请求能落地：中途被超越的响应直接丢弃，
      // 否则先发后到会覆盖新结果（本仓踩过的深链竞态同一形态）。
      if (seq !== controlPlaneSeqRef.current) return;
      setControlPlane({ loading: false, data });
    } catch (e) {
      if (seq !== controlPlaneSeqRef.current) return;
      // 失败不吞掉、也不落成空态：把错误留在 state 里由面板显式渲染。
      setControlPlane({ loading: false, error: (e as Error).message });
    }
  }, []);

  const fetchList = useCallback(async () => {
    setListLoading(true);
    try {
      const res: ToolApprovalListResponse = await aiGetToolApprovals(state);
      setItems(res.items ?? []);
    } catch (e) {
      message.error(`加载审批队列失败：${(e as Error).message}`);
    } finally {
      setListLoading(false);
    }
  }, [state, message]);

  useEffect(() => {
    fetchList();
  }, [fetchList]);

  useEffect(() => {
    fetchControlPlane();
  }, [fetchControlPlane]);

  // 「刷新」按钮：两个读面必须一起刷。
  //
  // 只刷其中一个会让面板与表格互相矛盾——例如队列已批完（表格空了），
  // 而概览还显示「待双人 3」。运维无法判断是哪个过期了。
  const refreshAll = useCallback(() => {
    fetchControlPlane();
    fetchList();
  }, [fetchControlPlane, fetchList]);

  // 展开行时拉取该轮证据链。收起不拉取（审批人只想收起时不该产生请求）。
  const handleExpand = useCallback(async (expanded: boolean, record: ToolApproval) => {
    if (!expanded) return;
    const rid = (record.runId ?? '').trim();

    // 复核：与轮次链**并列**发起，互不阻塞。
    //
    // 无 runId 的记录同样要复核：历史行没有轮次，但仍是一条待批调用，
    // 审批人依旧需要知道「批了会发生什么」。若跟着 runId 一起跳过，
    // 恰好在最缺少上下文的历史行上不给复核。
    const rid4Preflight = record.id;
    const pSeq = (preflightSeqRef.current[rid4Preflight] ?? 0) + 1;
    preflightSeqRef.current[rid4Preflight] = pSeq;
    setPreflights(prev => ({
      ...prev,
      [rid4Preflight]: { toolName: record.toolName, loading: true },
    }));
    void (async () => {
      try {
        // 复核既有调用，而不是「按我自己预演一次」。
        //
        // 这个区别决定了这个功能有没有意义：
        //   aiDryRunTool(record.toolName, args) 按**当前登录者**（审批人）判定，
        //   而审批人通常持有该工具权限 → 结论永远是 allowed=true。
        //   审批人要回答的是「这条 **AI 发起**的调用现在还合不合规」，
        //   所以必须按原发起主体判定：aiReviewToolInvocation 用落库的
        //   role_snapshot 与 approval_state 推导原主体，并带回两份策略指纹的比对。
        //
        // 参数不需要再传：后端从被复核记录的 arguments 读取，读不出就拒绝复核
        // （拿一份读不懂的输入去判定，比拒绝更危险）。
        const res = await aiReviewToolInvocation(record.id);
        if (preflightSeqRef.current[rid4Preflight] !== pSeq) return;
        setPreflights(prev => ({
          ...prev,
          [rid4Preflight]: { toolName: record.toolName, loading: false, result: res },
        }));
      } catch (e) {
        // 先校验序号再判定是否是过期响应——并发展开两行时，后到响应
        // 不得覆盖先到结论（这条纪律对预演比轮次链更关键：预演结论直接
        // 决定审批人点不点通过）。
        if (preflightSeqRef.current[rid4Preflight] !== pSeq) return;
        // 与轮次链同一纪律：失败不伪装成「没有结论」。
        setPreflights(prev => ({
          ...prev,
          [rid4Preflight]: {
            toolName: record.toolName,
            loading: false,
            error: (e as Error).message,
          },
        }));
      }
    })();

    // 无轮次的记录不发起请求：这是历史行的正常形态，不是错误，由渲染层说明。
    if (!rid) return;

    // 同步自增序号，await 之前占位——await 之后再取号会让后发请求先落地。
    const seq = (runSeqRef.current[rid] ?? 0) + 1;
    runSeqRef.current[rid] = seq;
    setRunChains(prev => ({ ...prev, [rid]: { runId: rid, loading: true, items: [] } }));
    try {
      const res: ToolRunInvocationsResponse = await aiGetRunInvocations(rid);
      if (runSeqRef.current[rid] !== seq) return; // 同一轮已有更新的一次展开，丢弃过期响应
      setRunChains(prev => ({ ...prev, [rid]: { runId: rid, loading: false, items: res.items ?? [] } }));
    } catch (e) {
      if (runSeqRef.current[rid] !== seq) return;
      // 加载失败与「这一轮确实没动作」必须分流：都返回空数组会让审批人
      // 以为「前面什么都没有」，从而低估这次调用的风险。
      setRunChains(prev => ({
        ...prev,
        [rid]: { runId: rid, loading: false, items: [], error: (e as Error).message },
      }));
    }
  }, []);

  const handleApprove = async (id: number) => {
    try {
      await aiApproveTool(id, { approve: true });
      message.success('已通过');
      fetchList();
    } catch (e) {
      message.error(`操作失败：${(e as Error).message}`);
    }
  };

  const handleRejectOk = async () => {
    if (rejectId == null) return;
    try {
      await aiApproveTool(rejectId, { approve: false, reason: rejectReason });
      message.success('已驳回');
      setRejectId(null);
      setRejectReason('');
      fetchList();
    } catch (e) {
      message.error(`操作失败：${(e as Error).message}`);
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 70,
    },
    {
      title: '工具',
      dataIndex: 'toolName',
      key: 'toolName',
      width: 150,
      render: (v: string) => <Tag color="geekblue">{v}</Tag>,
    },
    {
      title: '风险',
      dataIndex: 'riskLevel',
      key: 'riskLevel',
      width: 90,
      render: (v: string, r: ToolApproval) => (
        <TooltipWrapper
          text={r.minApprovals ? `权限码策略要求 ${r.minApprovals} 名互不相同的人类审批人` : undefined}
        >
          <Tag color={riskColor(v)}>{RISK_LABELS[v] ?? v ?? '未知'}</Tag>
        </TooltipWrapper>
      ),
    },
    {
      title: '参数',
      dataIndex: 'arguments',
      key: 'arguments',
      ellipsis: true,
      render: (v: string) => (
        <Text type="secondary" style={{ fontSize: 12 }}>
          {prettyArgs(v).slice(0, 120)}
          {prettyArgs(v).length > 120 ? '…' : ''}
        </Text>
      ),
    },
    {
      title: '放行依据',
      dataIndex: 'policyVersion',
      key: 'policyVersion',
      width: 110,
      render: (v: string) => {
        const view = policyVersionView(v);
        return (
          <TooltipWrapper text={view.hint}>
            <Tag color={view.color}>{view.label}</Tag>
          </TooltipWrapper>
        );
      },
    },
    {
      title: '权限校验',
      dataIndex: 'permissionCheck',
      key: 'permissionCheck',
      width: 120,
      render: (v: string, r: ToolApproval) => (
        <TooltipWrapper text={r.permissionReason}>
          <Tag color={permissionColor(v)}>{v || '-'}</Tag>
        </TooltipWrapper>
      ),
    },
    {
      title: '状态',
      dataIndex: 'approvalState',
      key: 'approvalState',
      width: 150,
      render: (v: string, r: ToolApproval) => {
        const isAgentFlow = v.startsWith('pending_agent') || v === 'awaiting_second';
        const progress = approvalProgress(v, r.minApprovals);
        return (
          <Space size={4} wrap>
            {isAgentFlow && (
              <TooltipWrapper text="该调用由 AI 发起，需人类审批后才会执行；发起人不可自批">
                <Tag color="purple">AI 发起</Tag>
              </TooltipWrapper>
            )}
            <Tag color={stateColor(v)}>{STATE_LABELS[v] ?? v}</Tag>
            {progress && (
              <Tag color={progress.startsWith('1/') ? 'orange' : 'default'}>审批进度 {progress}</Tag>
            )}
          </Space>
        );
      },
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 170,
      render: (v: string) => formatDateTime(v),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_: unknown, r: ToolApproval) => {
        // 可操作状态：人发起的单人审批 + AI 发起的三种在途状态。
        // （此前只认 pending，AI 发起的调用全部显示「已处理」且无按钮，
        //  人工审批闭环在 UI 上是断的。）
        const actionable =
          r.approvalState === 'pending' ||
          r.approvalState === 'pending_agent' ||
          r.approvalState === 'pending_agent_two' ||
          r.approvalState === 'awaiting_second';
        // 审批动作挂 ai:approve（与后端 POST /agent/tools/:id/approve 的
        // RequirePermission("ai","approve") 同码）；无权限只隐藏按钮，授权以后端为准。
        if (!actionable) return <Text type="secondary">已处理</Text>;
        if (!canApprove) {
          return <Text type="secondary">无审批权限（需 ai:approve）</Text>;
        }
        return (
          <Space>
            <Button
              type="primary"
              size="small"
              icon={<CheckCircle2 size={14} />}
              onClick={() => handleApprove(r.id)}
            >
              通过
            </Button>
            <Button danger size="small" icon={<XCircle size={14} />} onClick={() => setRejectId(r.id)}>
              驳回
            </Button>
          </Space>
        );
      },
    },
  ];

  return (
    <div className="space-y-6">
      <Space align="center" style={{ justifyContent: 'space-between', width: '100%' }}>
        <Space align="center">
          <ShieldAlert size={22} color="#1677ff" />
          <Title level={4} style={{ margin: 0 }}>
            AI 工具审批队列
          </Title>
        </Space>
        <Space>
          <Select
            style={{ width: 140 }}
            value={state}
            onChange={(v) => setState(v)}
            options={Object.entries(STATE_LABELS).map(([k, label]) => ({ value: k, label }))}
          />
          <a onClick={refreshAll}>
            <RefreshCw size={16} /> 刷新
          </a>
        </Space>
      </Space>

      <ControlPlanePanel state={controlPlane} onRefresh={fetchControlPlane} />

      <Card>
        <Table
          rowKey="id"
          size="middle"
          loading={listLoading}
          dataSource={items}
          columns={columns}
          pagination={{ pageSize: 20, showTotal: (t) => `共 ${t} 条` }}
          expandable={{
            onExpand: handleExpand,
            expandedRowRender: (r) => (
              <ExpandedInvocation
                record={r}
                chain={runChains[(r.runId ?? '').trim()]}
                preflight={preflights[r.id]}
              />
            ),
          }}
          locale={{ emptyText: <Empty description="当前筛选条件下没有待处理项" /> }}
        />
      </Card>

      <Modal
        title="驳回工具调用"
        open={rejectId != null}
        onOk={handleRejectOk}
        onCancel={() => {
          setRejectId(null);
          setRejectReason('');
        }}
        okText="确认驳回"
        okButtonProps={{ danger: true }}
      >
        <Input.TextArea
          rows={3}
          placeholder="驳回原因（可选）"
          value={rejectReason}
          onChange={(e) => setRejectReason(e.target.value)}
        />
      </Modal>
    </div>
  );
};

// 轻量 Tooltip 包装：避免 antd Tooltip 在 SSR/严格模式下对 children 的告警
const TooltipWrapper: React.FC<{ text?: string; children: React.ReactElement }> = ({ text, children }) => {
  if (!text) return children;
  return (
    <span title={text} style={{ cursor: 'help' }}>
      {children}
    </span>
  );
};

/**
 * 展开行：本次调用参数 + 同一运行轮次的完整证据链。
 *
 * 为什么审批人需要整轮而不是单条：审批一个写操作前要判断「这个动作合不合理」，
 * 答案在上下文里——这一轮 Agent 前面查了什么、看到了什么。只看孤立的
 * create_ticket 无法区分「按调查结论建单」与「凭空建单」。
 *
 * chain 由父组件按 runId 成对持有，这里只按 runId 匹配当前行：
 * 展开 A 后立刻展开 B、而 A 的响应后才返回时，不会把 B 的链画到 A 上。
 */
const ExpandedInvocation: React.FC<{
  record: ToolApproval;
  chain: ToolRunChain | null;
  preflight: ToolPreflight | null;
}> = ({ record, chain, preflight }) => {
  const ownRunId = (record.runId ?? '').trim();
  // 只有 chain 属于本行时才展示，否则视为尚未加载（不是「空轮次」）。
  const chainForThisRow = chain && chain.runId === ownRunId ? chain : null;
  // 复核结论按记录 id 分桶，父组件已保证 key 正确；这里再校验工具名，
  // 防止「同一 id 的旧结论落到已变更的记录上」这种理论串台。
  const preflightForThisRow =
    preflight && preflight.toolName === record.toolName ? preflight : null;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div>
        <Text type="secondary" style={{ fontSize: 12 }}>
          本次调用参数
        </Text>
        <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all', fontSize: 12 }}>
          {prettyArgs(record.arguments)}
        </pre>
      </div>

      <div>
        <Text type="secondary" style={{ fontSize: 12 }}>
          决策复核（按发起时的主体与当前策略重新判定）
        </Text>
        {!preflightForThisRow && (
          <div style={{ marginTop: 4 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              正在核对判定结论…
            </Text>
          </div>
        )}
        {preflightForThisRow?.loading && (
          <div style={{ marginTop: 4 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              校验中…
            </Text>
          </div>
        )}
        {preflightForThisRow?.error && (
          <div style={{ marginTop: 4 }}>
            <Text type="danger" style={{ fontSize: 12 }}>
              复核失败：{preflightForThisRow.error}（这不等于该操作会被拒绝，请重试）
            </Text>
          </div>
        )}
        {preflightForThisRow?.result && (
          <div style={{ marginTop: 4, display: 'flex', flexDirection: 'column', gap: 6 }}>
            <Space size={6} wrap>
              {preflightForThisRow.result.allowed ? (
                <Tag color="green">判定通过</Tag>
              ) : (
                <Tag color="red">判定拒绝</Tag>
              )}
              {/* 主体必须显示：同一工具在 human / agent 下结论可能不同，
                  不说明「按谁判的」，判定拒绝会被读成「工具本身不能用」。 */}
              {preflightForThisRow.result.evaluatedPrincipal && (
                <TooltipWrapper
                  text={
                    preflightForThisRow.result.evaluatedPrincipal === 'agent'
                      ? '按 AI 主体的策略判定（Gate 2.5 零信任）；不是按您自己的权限判定'
                      : '按人类主体的角色 RBAC 判定'
                  }
                >
                  <Tag color="purple">
                    按 {PRINCIPAL_LABELS[preflightForThisRow.result.evaluatedPrincipal] ??
                      preflightForThisRow.result.evaluatedPrincipal} 主体判定
                  </Tag>
                </TooltipWrapper>
              )}
              {preflightForThisRow.result.needsApproval ? (
                <Tag color="orange">
                  需 {preflightForThisRow.result.minApprovals} 人审批
                </Tag>
              ) : (
                <Tag color="default">无需审批，批准即刻执行</Tag>
              )}
              {preflightForThisRow.result.riskLevel && (
                <Tag color={preflightForThisRow.result.riskLevel === 'unknown' ? 'red' : 'blue'}>
                  风险 {preflightForThisRow.result.riskLevel}
                </Tag>
              )}
              {preflightForThisRow.result.approvalState && (
                <Tag color={stateColor(preflightForThisRow.result.approvalState)}>
                  {STATE_LABELS[preflightForThisRow.result.approvalState] ??
                    preflightForThisRow.result.approvalState}
                </Tag>
              )}
            </Space>
            {(() => {
              // 三态展示口径与本行的 policyVersion 完全一致，不在这里另立一套。
              const pv = policyVersionView(preflightForThisRow.result.policyVersion);
              return (
                <TooltipWrapper text={pv.hint}>
                  <Space size={6}>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      当前放行依据
                    </Text>
                    <Tag color={pv.color}>{pv.label}</Tag>
                  </Space>
                </TooltipWrapper>
              );
            })()}
            {preflightForThisRow.result.policyComparison && (
              <PolicyComparisonNote
                comparison={preflightForThisRow.result.policyComparison}
                baseline={preflightForThisRow.result.baselinePolicyVersion}
                current={preflightForThisRow.result.policyVersion}
              />
            )}
            {preflightForThisRow.result.reason && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                {preflightForThisRow.result.reason}
              </Text>
            )}
          </div>
        )}
      </div>

      <div>
        <Text type="secondary" style={{ fontSize: 12 }}>
          {ownRunId ? '同一轮次的全部动作（按发生顺序）' : '运行轮次上下文'}
        </Text>
        {!ownRunId && (
          <div style={{ marginTop: 4 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              该记录未归属任何运行轮次（早于轮次标识上线），无法追溯前序动作。
            </Text>
          </div>
        )}
        {ownRunId && chainForThisRow?.loading && (
          <div style={{ marginTop: 4 }}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              加载中…
            </Text>
          </div>
        )}
        {ownRunId && chainForThisRow?.error && (
          <div style={{ marginTop: 4 }}>
            <Text type="danger" style={{ fontSize: 12 }}>
              加载轮次记录失败：{chainForThisRow.error}（这不等于该轮没有动作，请重试）
            </Text>
          </div>
        )}
        {ownRunId && chainForThisRow && !chainForThisRow.loading && !chainForThisRow.error && (
          chainForThisRow.items.length === 0 ? (
            <div style={{ marginTop: 4 }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                该轮次没有可读的工具调用记录。
              </Text>
            </div>
          ) : (
            <div style={{ marginTop: 4, display: 'flex', flexDirection: 'column', gap: 4 }}>
              {chainForThisRow.items.map((it, idx) => {
                const isCurrent = it.id === record.id;
                const pv = policyVersionView(it.policyVersion);
                return (
                  <Space key={it.id} size={6} wrap>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {idx + 1}.
                    </Text>
                    <Tag color={isCurrent ? 'blue' : 'default'}>{it.toolName}</Tag>
                    <Tag color={stateColor(it.approvalState)}>{STATE_LABELS[it.approvalState] ?? it.approvalState}</Tag>
                    <TooltipWrapper text={pv.hint}>
                      <Tag color={pv.color}>{pv.label}</Tag>
                    </TooltipWrapper>
                    {isCurrent && (
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        ← 当前条目
                      </Text>
                    )}
                  </Space>
                );
              })}
            </div>
          )
        )}
      </div>
    </div>
  );
};

export default AIApprovalQueue;
