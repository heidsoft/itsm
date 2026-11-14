'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert, App, Button, Card, Space, Spin, Tag, Timeline, Tooltip, Typography } from 'antd';

import {
  aiInvestigateTicket,
  aiUpdateAuditAccepted,
  type InvestigationStep,
  type TicketInvestigationResponse,
} from '@/lib/api/ai-api';
import { ApiRequestError, isPermissionDeniedError } from '@/lib/api/http-client';

const { Text, Paragraph } = Typography;

// 步骤名 → 人读标签（后端契约枚举：triage | knowledge | cmdb | synthesis）
const STEP_LABELS: Record<string, string> = {
  triage: '智能分诊',
  knowledge: '知识库检索',
  cmdb: 'CMDB 影响分析',
  synthesis: 'AI 综合',
};

interface AIInvestigationPanelProps {
  ticketId: number;
  /** 工单标题：create_ticket 建议跳建单页时做预填 */
  ticketTitle?: string;
}

/**
 * 工单调查编排面板（POST /api/v1/ai/tickets/:id/investigate 的前端消费方）。
 *
 * 只读决策支持：结论里的 suggestedActions 仅是建议；create_ticket 建议跳建单页
 * 预填标题/描述（不自动提交），其余建议只展示描述、提示走对应人工入口。
 * 每次运行全量重跑（CMDB/知识库新鲜度敏感），结果不缓存、不按工单幂等。
 */
export function AIInvestigationPanel({ ticketId, ticketTitle }: AIInvestigationPanelProps) {
  const router = useRouter();
  const { message } = App.useApp();
  const [result, setResult] = useState<TicketInvestigationResponse | null>(null);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<unknown>(null);
  // 底账反馈本地态：翻转过一次就锁住，防重复改写底账（与 AISuggestionPanel 同口径）
  const [auditAccepted, setAuditAccepted] = useState<boolean | null>(null);
  const [auditReporting, setAuditReporting] = useState(false);

  // 显式触发的调查动作：与同页 AISuggestionPanel 一致用局部状态管理（无缓存语义、
  // 每次全量重跑），不用 React Query mutation，避免详情页测试需要额外 Provider。
  const start = async () => {
    if (running) {
      return;
    }
    setRunning(true);
    setError(null);
    try {
      const data = await aiInvestigateTicket(ticketId);
      setResult(data);
      setAuditAccepted(null);
    } catch (err) {
      setError(err);
    } finally {
      setRunning(false);
    }
  };

  const reportAudit = async (accepted: boolean) => {
    if (auditReporting || auditAccepted !== null) {
      return;
    }
    const auditId = result?.auditId;
    if (typeof auditId !== 'number' || auditId <= 0) {
      return;
    }
    setAuditReporting(true);
    try {
      // 翻转的是调查结论自己的底账行（auditId），不是工单 ID：
      // PATCH /ai/audit-logs/:id 的路径参数是 ai_audit 行 ID。
      await aiUpdateAuditAccepted(auditId, accepted);
      setAuditAccepted(accepted);
    } catch {
      // 反馈是 fire-and-forget：失败不阻塞结论展示，只提示（与 AISuggestionPanel 同口径）
      message.warning('反馈记录失败，不影响调查结论');
    } finally {
      setAuditReporting(false);
    }
  };


  const renderStepDetail = (step: InvestigationStep) => {
    const label = STEP_LABELS[step.step] ?? step.step;
    return (
      <div key={`${step.step}-${step.tool ?? ''}`}>
        <Space size={8} wrap>
          <Text strong>{label}</Text>
          {step.tool ? <Tag>{step.tool}</Tag> : null}
          {step.degraded ? (
            <Tooltip title={step.degradedReason ?? '该步骤依赖不可用，已降级'}>
              <Tag color="orange">降级</Tag>
            </Tooltip>
          ) : null}
          {typeof step.hitCount === 'number' && step.hitCount > 0 ? (
            <Tag>{step.hitCount} 条命中</Tag>
          ) : null}
        </Space>
      </div>
    );
  };

  const renderAction = (action: NonNullable<TicketInvestigationResponse['suggestedActions']>[number]) => {
    if (action.action === 'create_ticket') {
      const params = new URLSearchParams({
        prefillTitle: ticketTitle ?? '',
        prefillDescription: action.description,
      });
      return (
        <Button
          size="small"
          onClick={() => router.push(`/tickets/create?${params.toString()}`)}
        >
          去建单（预填）
        </Button>
      );
    }
    return (
      <Tooltip title="该建议没有直接入口，请在对应模块人工处理">
        <Text type="secondary">人工处理</Text>
      </Tooltip>
    );
  };

  const renderResult = () => {
    if (!result) {
      return null;
    }
    const degraded = result.degraded || result.status === 'degraded';
    return (
      <Space orientation="vertical" style={{ width: '100%' }} size={12}>
        <Space size={8} wrap>
          {degraded ? <Tag color="orange">降级结果</Tag> : <Tag color="green">完成</Tag>}
          <Text type="secondary">
            置信度 {(result.confidence * 100).toFixed(0)}%
            {result.model ? ` · 模型 ${result.model}` : ' · 确定性拼接'}
            {typeof result.latencyMs === 'number' && result.latencyMs > 0
              ? ` · 耗时 ${(result.latencyMs / 1000).toFixed(1)}s`
              : ''}
          </Text>
          <Tag>{result.promptVersion}</Tag>
        </Space>

        {degraded ? (
          <Alert
            type="warning"
            showIcon
            title="AI 综合暂不可用"
            description={
              result.degradedReason ?? '以下为基于分诊与检索结果的确定性摘要，不含 AI 推断。'
            }
          />
        ) : null}

        {result.suspectedRootCause ? (
          <Alert type="info" showIcon title="疑似根因" description={result.suspectedRootCause} />
        ) : null}

        {result.findings && result.findings.length > 0 ? (
          <div>
            <Text strong>调查发现</Text>
            <ul style={{ margin: '4px 0 0', paddingLeft: 20 }}>
              {result.findings.map(item => (
                <li key={item}>
                  <Paragraph style={{ margin: 0 }}>{item}</Paragraph>
                </li>
              ))}
            </ul>
          </div>
        ) : null}

        {result.impactedServices && result.impactedServices.length > 0 ? (
          <div>
            <Text strong>可能受影响服务</Text>
            <div style={{ marginTop: 4 }}>
              <Space size={4} wrap>
                {result.impactedServices.map(s => (
                  <Tag key={s} color="red">
                    {s}
                  </Tag>
                ))}
              </Space>
            </div>
          </div>
        ) : null}

        {result.steps.length > 0 ? (
          <div>
            <Text strong>调查步骤</Text>
            <div style={{ marginTop: 8 }}>
              <Timeline
                items={result.steps.map(step => ({
                  key: `${step.step}-${step.tool ?? ''}`,
                  children: renderStepDetail(step),
                  color: step.degraded ? 'orange' : 'green',
                }))}
              />
            </div>
          </div>
        ) : null}

        {result.suggestedActions && result.suggestedActions.length > 0 ? (
          <div>
            <Text strong>建议的后续动作</Text>
            <div style={{ marginTop: 4 }}>
              <Space orientation="vertical" size={6} style={{ width: '100%' }}>
                {result.suggestedActions.map(action => (
                  <Space key={`${action.action}-${action.description}`} size={8} wrap>
                    <Tag color="blue">{action.action}</Tag>
                    <Text>{action.description}</Text>
                    {action.requiresApproval ? <Tag>需人工执行</Tag> : null}
                    {renderAction(action)}
                  </Space>
                ))}
              </Space>
            </div>
            <Text type="secondary" style={{ fontSize: 12 }}>
              以上仅为 AI 建议：写操作由你在对应表单显式发起并走审批流，本面板不会自动执行。
            </Text>
          </div>
        ) : null}

        {typeof result.auditId === 'number' && result.auditId > 0 ? (
          <Space size={8}>
            <Text type="secondary" style={{ fontSize: 12 }}>
              本次调查已记录（底账 #{result.auditId}），这条结论对你有帮助吗？
            </Text>
            {auditAccepted === null ? (
              <>
                <Button
                  size="small"
                  loading={auditReporting}
                  onClick={() => void reportAudit(true)}
                >
                  有用
                </Button>
                <Button
                  size="small"
                  loading={auditReporting}
                  onClick={() => void reportAudit(false)}
                >
                  不准
                </Button>
              </>
            ) : (
              <Tag color={auditAccepted ? 'green' : 'default'}>
                {auditAccepted ? '已标记有用' : '已标记不准'}
              </Tag>
            )}
          </Space>
        ) : null}

        <Button onClick={start} loading={running} disabled={auditReporting}>
          重新调查
        </Button>
      </Space>
    );
  };

  const renderError = (err: unknown) => {
    if (isPermissionDeniedError(err)) {
      return (
        <Alert
          type="warning"
          showIcon
          title="无权执行 AI 调查"
          description="需要 ai:read 与 ticket:read 权限，请联系管理员开通。"
        />
      );
    }
    if (err instanceof ApiRequestError && err.status === 404) {
      return <Alert type="warning" showIcon title="工单不存在或已被删除" />;
    }
    return (
      <Space orientation="vertical" size={8}>
        <Alert
          type="error"
          showIcon
          title="AI 调查失败"
          description={err instanceof Error ? err.message : '请稍后重试'}
        />
        <Button onClick={start}>重试</Button>
      </Space>
    );
  };

  const body = () => {
    if (running) {
      return (
        <Space orientation="vertical" size={8}>
          <Spin />
          <Text type="secondary">调查进行中（检索知识库与 CMDB 并综合分析），通常需要 10–30 秒…</Text>
        </Space>
      );
    }
    if (error) {
      return renderError(error);
    }
    if (result) {
      return renderResult();
    }
    return (
      <Space orientation="vertical" size={8}>
        <Text type="secondary">
          运行多步调查：智能分诊 → 知识库检索 → CMDB 影响分析 → AI 综合结论。全程只读，
          不会执行任何写操作；每次运行都会重新检索（结果反映当前知识库与 CMDB 状态）。
        </Text>
        <Button type="primary" onClick={start}>
          运行 AI 调查
        </Button>
      </Space>
    );
  };

  return (
    <Card className="rounded-lg shadow-sm border border-gray-200" title="AI 调查">
      {body()}
    </Card>
  );
}
