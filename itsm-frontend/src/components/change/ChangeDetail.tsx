'use client';

/**
 * 变更详情组件
 */

import React, { useState, useEffect, useRef } from 'react';
import {
  Card,
  Descriptions,
  Tag,
  Button,
  Skeleton,
  Result,
  Divider,
  List,
  Typography,
  Steps,
  Spin,
  Empty,
  Tabs,
  Space,
  Progress,
  Alert,
  message,
} from 'antd';
import { ArrowLeft, CheckCircle, XCircle } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import { Modal, Input } from 'antd';

import { ChangeApi } from '@/lib/api/';
import { serverErrorMessage } from '@/lib/api/http-client';
import { useI18n } from '@/lib/i18n/useI18n';
import {
  ChangeStatus,
  ChangeStatusLabels,
  ChangeTypeLabels,
  ChangePriorityLabels,
  ChangeImpactLabels,
  ChangeRiskLabels,
} from '@/constants/change';
import type { Change, ApprovalChainItem, RollbackStep } from '@/types/biz/change';
import ReviewStepBadge from '@/components/change-review/ReviewStepBadge';
import ChangeRiskAssessment from './ChangeRiskAssessment';
import ChangeCMDBImpactPanel from './ChangeCMDBImpactPanel';
import ChangeImpactAnalysis from './ChangeImpactAnalysis';
import ChangeRollbackPlan from './ChangeRollbackPlan';
import { SafeTextBlock } from '@/components/common/SafeContent';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text, Paragraph } = Typography;

// 回滚步骤状态 → 标签色。pending 用默认灰、done 绿、skipped 橙、failed 红：
// 颜色只表达"是否已确认 + 是否出问题"，避免把失败和成功混在同一个色系里。
const rollbackStepColors: Record<string, string> = {
  pending: 'default',
  done: 'green',
  skipped: 'orange',
  failed: 'red',
};

const statusColors: Record<string, string> = {
  [ChangeStatus.DRAFT]: 'default',
  [ChangeStatus.PENDING]: 'orange',
  [ChangeStatus.APPROVED]: 'cyan',
  [ChangeStatus.IN_PROGRESS]: 'blue',
  [ChangeStatus.COMPLETED]: 'green',
  [ChangeStatus.REJECTED]: 'red',
  [ChangeStatus.ROLLED_BACK]: 'magenta',
};

const ChangeDetail: React.FC = () => {
  const { id } = useParams() as { id: string };
  const router = useRouter();
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [change, setChange] = useState<Change | null>(null);
  const [approvalChain, setApprovalChain] = useState<ApprovalChainItem[]>([]);
  const [riskAssessment, setRiskAssessment] = useState<any>(null);
  const [impactAnalysis, setImpactAnalysis] = useState<any>(null);
  const [rollbackPlan, setRollbackPlan] = useState<any>(null);
  const [assessmentLoading, setAssessmentLoading] = useState(false);
  const [approvalModalVisible, setApprovalModalVisible] = useState(false);
  const [rejectModalVisible, setRejectModalVisible] = useState(false);
  const [approvalComment, setApprovalComment] = useState('');
  // 实施结果弹窗：回滚与「标记失败」都要求必填原因（后端拒收无原因的 rolled_back/failed），
  // 因此不再让按钮直接发请求，统一走一个带必填校验的弹窗。
  const [outcomeModal, setOutcomeModal] = useState<'rollback' | 'fail' | null>(null);
  // 回滚清单：逐项确认的弹窗（跳过/失败必须写依据）
  const [resolvingStep, setResolvingStep] = useState<{
    step: RollbackStep;
    status: 'skipped' | 'failed';
  } | null>(null);
  const [stepNote, setStepNote] = useState('');
  // 清单编排弹窗（一行一步）
  const [planModalOpen, setPlanModalOpen] = useState(false);
  const [planText, setPlanText] = useState('');
  const [outcomeReason, setOutcomeReason] = useState('');
  const [processing, setProcessing] = useState(false);
  // 写操作 in-flight 标志：按钮 loading 状态要等 render 提交后才生效，
  // 快速连续点击存在重复提交窗口。
  // 用 ref 做同步重入拦截，同一时刻只允许一个写请求在飞。
  const inFlightRef = useRef(false);

  useEffect(() => {
    if (id) {
      loadDetail();
    }
     
  }, [id]);

  // 生命周期 action bar 可见性：详情页要能推进状态机，终态只剩查看
  // draft → 提交审批;pending → 批准/拒绝;approved → 开始实施;
  // in_progress → 完成实施/回滚;非终态 → 取消
  // failed 是"实施没成功"，不是终态：状态机允许 failed → scheduled（重新排期）、
  // failed → rolled_back（回滚）、failed → cancelled。把它当终态会让记录完失败原因后
  // 页面无路可走，只能去库里改状态。
  const isTerminal =
    change?.status === ChangeStatus.COMPLETED ||
    change?.status === ChangeStatus.CANCELLED ||
    change?.status === ChangeStatus.ROLLED_BACK ||
    change?.status === ChangeStatus.REJECTED ||
    change?.status === ChangeStatus.CLOSED;
  const canRecoverFromFailure = change?.status === ChangeStatus.FAILED;
  const canSubmit = change?.status === ChangeStatus.DRAFT;
  const canApprove = change?.status === ChangeStatus.PENDING;
  const canSchedule = change?.status === ChangeStatus.APPROVED;
  const canStart = change?.status === ChangeStatus.SCHEDULED;
  const canComplete = change?.status === ChangeStatus.IN_PROGRESS;
  // 已完成 → 可显式关闭以归档；已回滚也可选择关闭
  const canClose =
    change?.status === ChangeStatus.COMPLETED ||
    change?.status === ChangeStatus.ROLLED_BACK;
  const canCancel =
    change?.status === ChangeStatus.DRAFT ||
    change?.status === ChangeStatus.APPROVED ||
    change?.status === ChangeStatus.SCHEDULED ||
    change?.status === ChangeStatus.IN_PROGRESS ||
    change?.status === ChangeStatus.FAILED;

  // ---- 回滚清单（失败可回退） ----
  // 清单即计划：后端按 step_no 有序返回，resolved 由后端派生（status !== 'pending'）。
  const rollbackSteps: RollbackStep[] = change?.rollbackSteps ?? [];
  const pendingSteps = rollbackSteps.filter(step => !step.resolved);
  const resolvedCount = rollbackSteps.length - pendingSteps.length;
  // 编排只在实施开始前允许：实施开始后清单成为执行依据，后端会以 409 拒绝改写。
  const canEditChecklist =
    change?.status === ChangeStatus.DRAFT ||
    change?.status === ChangeStatus.PENDING ||
    change?.status === ChangeStatus.APPROVED ||
    change?.status === ChangeStatus.SCHEDULED;
  // 逐项确认只在实施中/实施失败时允许（与后端 rollbackStepActionable 一一对应）。
  const canResolveSteps =
    change?.status === ChangeStatus.IN_PROGRESS || change?.status === ChangeStatus.FAILED;
  // 清单还有未确认步骤时，禁止标记已回滚：否则「已回滚」可能只是回滚到一半，
  // 而 PIR 会据此判定「影响已复原」。后端同样会 422，这里只是把话说在前面。
  const rollbackBlocked = pendingSteps.length > 0;
  const rollbackStepLabels: Record<string, string> = {
    pending: t('changeDetail.stepStatusPending'),
    done: t('changeDetail.stepStatusDone'),
    skipped: t('changeDetail.stepStatusSkipped'),
    failed: t('changeDetail.stepStatusFailed'),
  };

  // 提交审批
  const handleSubmit = async () => {
    if (!change || inFlightRef.current) return;
    inFlightRef.current = true;
    setProcessing(true);
    try {
      await ChangeApi.submitForApproval(change.id);
      message.success(t('changeDetail.submitSuccess'));
      loadDetail();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.submitFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  // 排期
  const handleSchedule = async () => {
    if (!change || inFlightRef.current) return;
    inFlightRef.current = true;
    setProcessing(true);
    try {
      await ChangeApi.scheduleChange(change.id);
      message.success(t('changeDetail.scheduleSuccess'));
      loadDetail();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.scheduleFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  // 开始实施
  const handleStart = async () => {
    if (!change || inFlightRef.current) return;
    inFlightRef.current = true;
    setProcessing(true);
    try {
      await ChangeApi.startImplementation(change.id);
      message.success(t('changeDetail.startSuccess'));
      loadDetail();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.startFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  // 完成实施
  const handleComplete = async () => {
    if (!change || inFlightRef.current) return;
    inFlightRef.current = true;
    setProcessing(true);
    try {
      await ChangeApi.completeImplementation(change.id);
      message.success(t('changeDetail.completeSuccess'));
      loadDetail();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.completeFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  // 关闭变更（已完成/已回滚 → 已关闭）
  const handleClose = async () => {
    if (!change || inFlightRef.current) return;
    inFlightRef.current = true;
    setProcessing(true);
    try {
      await ChangeApi.closeChange(change.id);
      message.success(t('changeDetail.closeSuccess'));
      loadDetail();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.closeFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  // 回滚（原因由弹窗收集，必填）
  const handleRollback = async (reason: string) => {
    if (!change || inFlightRef.current) return;
    const trimmed = reason.trim();
    if (!trimmed) {
      message.warning(t('changeDetail.outcomeReasonRequired'));
      return;
    }
    inFlightRef.current = true;
    setProcessing(true);
    try {
      await ChangeApi.rollbackChange(change.id, trimmed);
      message.success(t('changeDetail.rollbackSuccess'));
      setOutcomeModal(null);
      setOutcomeReason('');
      loadDetail();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.rollbackFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  // 标记实施失败（原因写入 outcome_note，进 PIR 与审计）
  const handleFail = async (reason: string) => {
    if (!change || inFlightRef.current) return;
    const trimmed = reason.trim();
    if (!trimmed) {
      message.warning(t('changeDetail.outcomeReasonRequired'));
      return;
    }
    inFlightRef.current = true;
    setProcessing(true);
    try {
      await ChangeApi.failChange(change.id, trimmed);
      message.success(t('changeDetail.failSuccess'));
      setOutcomeModal(null);
      setOutcomeReason('');
      loadDetail();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.failFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  // 取消
  const handleCancel = async () => {
    if (!change || inFlightRef.current) return;
    inFlightRef.current = true;
    setProcessing(true);
    try {
      await ChangeApi.cancelChange(change.id, approvalComment);
      message.success(t('changeDetail.cancelSuccess'));
      setApprovalComment('');
      loadDetail();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.cancelFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  // 批准变更
  const handleApprove = async () => {
    if (!change || inFlightRef.current) return;
    inFlightRef.current = true;
    setProcessing(true);
    try {
      const updated = await ChangeApi.approveChange(change.id, { comment: approvalComment });
      // 会签/阈值/多级链未集齐时后端返回 200 但状态仍是 pending：这一票已记录，
      // 不能说"变更已批准"，否则审批人会以为整单已经放行。
      if (updated && updated.status && updated.status !== 'approved') {
        message.info(t('changeDetail.approveRecorded'));
      } else {
        message.success(t('changeDetail.approveSuccess'));
      }
      setApprovalModalVisible(false);
      setApprovalComment('');
      loadDetail();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.approveFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  // 拒绝变更
  const handleReject = async () => {
    if (!change || inFlightRef.current) return;
    inFlightRef.current = true;
    setProcessing(true);
    try {
      await ChangeApi.rejectChange(change.id, { comment: approvalComment });
      message.success(t('changeDetail.rejectSuccess'));
      setRejectModalVisible(false);
      setApprovalComment('');
      loadDetail();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.rejectFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  const loadDetail = async () => {
    setLoading(true);
    try {
      const data = await ChangeApi.getChange(Number(id!));
      setChange(data as Change);

      // Try to load approval summary
      try {
        const summary = await ChangeApi.getApprovalSummary(Number(id));
        const chain = (summary as { chain?: ApprovalChainItem[] })?.chain ?? [];
        setApprovalChain(chain);
      } catch (e) {
        // console.warn('Failed to load approval summary', e);
      }
    } catch (error) {
      // console.error(error);
      message.error(serverErrorMessage(error, t('changeDetail.loadDetailFailed')));
    } finally {
      setLoading(false);
    }
  };

  // 加载风险评估数据
  const loadRiskAssessment = async () => {
    if (!id) return;
    setAssessmentLoading(true);
    try {
      const data = await ChangeApi.getRiskAssessment(Number(id));
      setRiskAssessment(data);
    } catch (error) {
      console.error('Failed to load risk assessment:', error);
    } finally {
      setAssessmentLoading(false);
    }
  };

  // 加载影响分析数据
  const loadImpactAnalysis = async () => {
    if (!id) return;
    setAssessmentLoading(true);
    try {
      const data = await ChangeApi.getImpactAnalysis(Number(id));
      setImpactAnalysis(data);
    } catch (error) {
      console.error('Failed to load impact analysis:', error);
    } finally {
      setAssessmentLoading(false);
    }
  };

  // 保存风险评估
  const handleSaveRiskAssessment = async (data: any) => {
    if (!id) return;
    try {
      await ChangeApi.updateRisk(Number(id), data);
      message.success(t('changeDetail.riskSaveSuccess'));
      loadRiskAssessment();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.saveFailed')));
    }
  };

  // 保存影响分析
  const handleSaveImpactAnalysis = async (data: any) => {
    if (!id) return;
    try {
      await ChangeApi.updateImpactAnalysis(Number(id), data);
      message.success(t('changeDetail.impactSaveSuccess'));
      loadImpactAnalysis();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.saveFailed')));
    }
  };

  // 保存回滚计划
  const handleSaveRollbackPlan = async (data: any) => {
    if (!id) return;
    try {
      // 回滚计划暂时通过更新变更的rollback_plan字段实现
      await ChangeApi.updateChange(Number(id), { rollbackPlan: JSON.stringify(data) });
      message.success(t('changeDetail.rollbackPlanSaveSuccess'));
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.saveFailed')));
    }
  };

  // ---- 回滚清单交互 ----

  /**
   * 从既有「回滚计划」里捞出步骤，避免让人把同样的内容再敲一遍。
   *
   * rollback_plan 存在两种历史形态：计划页写入的 JSON（rollbackSteps[].title），
   * 以及纯文本。这里都做尽力而为的解析，解析不出来就返回空由用户自己写——
   * 绝不"猜"出一份看起来像清单的东西：清单会被当成执行依据逐项确认。
   */
  const deriveStepsFromPlan = (): string[] => {
    const raw = (change?.rollbackPlan ?? '').trim();
    if (!raw) return [];
    try {
      const parsed = JSON.parse(raw);
      const steps = Array.isArray(parsed?.rollbackSteps) ? parsed.rollbackSteps : [];
      return steps
        .map((step: { title?: string; description?: string }) =>
          (step?.title || step?.description || '').trim()
        )
        .filter(Boolean);
    } catch {
      return raw
        .split('\n')
        .map(line => line.replace(/^\s*\d+[.、)]\s*/, '').trim())
        .filter(Boolean);
    }
  };

  const openChecklistEditor = () => {
    const existing = rollbackSteps.map(step => step.description);
    const seed = existing.length > 0 ? existing : deriveStepsFromPlan();
    setPlanText(seed.join('\n'));
    setPlanModalOpen(true);
  };

  const handleSaveChecklist = async () => {
    if (!change || inFlightRef.current) return;
    const steps = planText
      .split('\n')
      .map(line => line.trim())
      .filter(Boolean);
    inFlightRef.current = true;
    setProcessing(true);
    try {
      await ChangeApi.replaceRollbackSteps(change.id, steps);
      message.success(t('changeDetail.checklistSaved'));
      setPlanModalOpen(false);
      await loadDetail();
    } catch (error) {
      message.error(serverErrorMessage(error, t('changeDetail.checklistSaveFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  const handleResolveStep = async (step: RollbackStep, status: RollbackStep['status'], note?: string) => {
    if (!change || inFlightRef.current) return;
    inFlightRef.current = true;
    setProcessing(true);
    try {
      await ChangeApi.resolveRollbackStep(change.id, step.id, status, note);
      message.success(
        status === 'pending' ? t('changeDetail.stepReset') : t('changeDetail.stepResolved')
      );
      setResolvingStep(null);
      setStepNote('');
      await loadDetail();
    } catch (error) {
      // 409（已被别人确认）与 422（缺依据）都带可行动文案，原样透出。
      message.error(serverErrorMessage(error, t('changeDetail.stepResolveFailed')));
    } finally {
      inFlightRef.current = false;
      setProcessing(false);
    }
  };

  if (loading)
    return (
      <Card>
        <Skeleton active />
      </Card>
    );

  if (!change) {
    return (
      <Card>
        <Result
          status="404"
          title="404"
          subTitle={t('changeDetail.notFoundDesc')}
          extra={
            <Button type="primary" onClick={() => router.push('/changes')}>
              {t('changeDetail.back')}
            </Button>
          }
        />
      </Card>
    );
  }

  return (
    <Space orientation="vertical" style={{ width: '100%' }} size="large">
      <Card>
        <div style={{ marginBottom: 24 }}>
          <Button
            icon={<ArrowLeft />}
            onClick={() => router.push('/changes')}
            style={{ marginBottom: 16 }}
          >
            {t('changeDetail.back')}
          </Button>
          <div
            style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}
          >
            <Title level={3}>{change.title}</Title>
            <Tag color={statusColors[change.status]} style={{ padding: '4px 12px', fontSize: 14 }}>
              {ChangeStatusLabels[change.status]}
            </Tag>
            {!isTerminal && (
              <Space>
                {canSubmit && (
                  <Button type="primary" loading={processing} onClick={handleSubmit}>
                    {t('changeDetail.submit')}
                  </Button>
                )}
                {canApprove && (
                  <>
                    <Button
                      type="primary"
                      icon={<CheckCircle />}
                      loading={processing}
                      onClick={() => {
                        setApprovalComment('');
                        setApprovalModalVisible(true);
                      }}
                    >
                      {t('changeDetail.approve')}
                    </Button>
                    <Button
                      danger
                      icon={<XCircle />}
                      loading={processing}
                      onClick={() => {
                        setApprovalComment('');
                        setRejectModalVisible(true);
                      }}
                    >
                      {t('changeDetail.reject')}
                    </Button>
                  </>
                )}
                {canSchedule && (
                  <Button type="primary" loading={processing} onClick={handleSchedule}>
                    {t('changeDetail.schedule')}
                  </Button>
                )}
                {canStart && (
                  <Button type="primary" loading={processing} onClick={handleStart}>
                    {t('changeDetail.startImplementation')}
                  </Button>
                )}
                {canComplete && (
                  <>
                    <Button type="primary" loading={processing} onClick={handleComplete}>
                      {t('changeDetail.complete')}
                    </Button>
                    {/* 清单还有未确认步骤时禁用：后端会 422，这里先让人看见原因，
                        而不是点完再收到错误。 */}
                    <Button
                      danger
                      loading={processing}
                      disabled={rollbackBlocked}
                      onClick={() => {
                        setOutcomeReason('');
                        setOutcomeModal('rollback');
                      }}
                    >
                      {t('changeDetail.rollback')}
                    </Button>
                    {/* 实施失败必须能如实记录：此前没有入口，失败的单只能报成"完成"
                        或直接回滚，统计与 PIR 结论都随之失真。 */}
                    <Button
                      danger
                      loading={processing}
                      onClick={() => {
                        setOutcomeReason('');
                        setOutcomeModal('fail');
                      }}
                    >
                      {t('changeDetail.markFailed')}
                    </Button>
                  </>
                )}
                {canRecoverFromFailure && (
                  <>
                    <Button type="primary" loading={processing} onClick={handleSchedule}>
                      {t('changeDetail.reschedule')}
                    </Button>
                    {/* 清单还有未确认步骤时禁用：后端会 422，这里先让人看见原因，
                        而不是点完再收到错误。 */}
                    <Button
                      danger
                      loading={processing}
                      disabled={rollbackBlocked}
                      onClick={() => {
                        setOutcomeReason('');
                        setOutcomeModal('rollback');
                      }}
                    >
                      {t('changeDetail.rollback')}
                    </Button>
                  </>
                )}
                {canCancel && (
                  <Button danger loading={processing} onClick={handleCancel}>
                    {t('changeDetail.cancelChange')}
                  </Button>
                )}
              </Space>
            )}
            {canClose && (
              <Button type="primary" loading={processing} onClick={handleClose} style={{ marginLeft: 12 }}>
                {t('changeDetail.close')}
              </Button>
            )}
          </div>
        </div>

        <Descriptions bordered column={2}>
          <Descriptions.Item label={t('changeDetail.labelChangeNumber')}>{change.id}</Descriptions.Item>
          <Descriptions.Item label={t('changeDetail.labelType')}>{ChangeTypeLabels[change.type]}</Descriptions.Item>
          <Descriptions.Item label={t('changeDetail.labelPriority')}>
            {ChangePriorityLabels[change.priority]}
          </Descriptions.Item>
          <Descriptions.Item label={t('changeDetail.labelRisk')}>
            {ChangeRiskLabels[change.riskLevel]}
          </Descriptions.Item>
          <Descriptions.Item label={t('changeDetail.labelImpactScope')}>
            {ChangeImpactLabels[change.impactScope]}
          </Descriptions.Item>
          <Descriptions.Item label={t('changeDetail.labelAssignee')}>{change.assigneeName || t('changeDetail.unassigned')}</Descriptions.Item>
          <Descriptions.Item label={t('changeDetail.labelPlannedStart')}>
            {change.plannedStartDate
              ? formatDateTime(change.plannedStartDate) || '-'
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('changeDetail.labelPlannedEnd')}>
            {formatDateTime(change.plannedEndDate) || '-'}
          </Descriptions.Item>
        </Descriptions>

        <Tabs
          defaultActiveKey="1"
          style={{ marginTop: 24 }}
          onChange={activeKey => {
            if (activeKey === '3' && !riskAssessment) loadRiskAssessment();
            if (activeKey === '4' && !impactAnalysis) loadImpactAnalysis();
          }}
          items={[
            {
              key: '1',
              label: t('changeDetail.basicInfo'),
              children: (
                <>
                  <Title level={5}>{t('changeDetail.justificationTitle')}</Title>
                  <SafeTextBlock content={change.justification} fallback={t('changeDetail.none')} />

                  <Title level={5}>{t('changeDetail.descriptionTitle')}</Title>
                  <SafeTextBlock content={change.description} fallback={t('changeDetail.none')} />

                  <Divider />

                  <Title level={5}>{t('changeDetail.implementationPlanTitle')}</Title>
                  <SafeTextBlock
                    content={change.implementationPlan}
                    fallback={t('changeDetail.noImplementationPlan')}
                    preserveNewlines
                  />

                  <Title level={5}>{t('changeDetail.rollbackPlan')}</Title>
                  <SafeTextBlock
                    content={change.rollbackPlan}
                    fallback={t('changeDetail.noRollbackPlan')}
                    preserveNewlines
                  />

                  {/* 实施结果说明只在存在时展示：没有结果就说明还没实施完，
                      给一个"暂无"反而让人以为漏填了。 */}
                  {change.outcomeNote && (
                    <>
                      <Title level={5}>{t('changeDetail.outcomeNote')}</Title>
                      <SafeTextBlock content={change.outcomeNote} preserveNewlines />
                    </>
                  )}
                </>
              ),
            },
            {
              key: '2',
              label: t('changeDetail.approvalRecords'),
              children:
                approvalChain.length > 0 ? (
                  <List
                    itemLayout="horizontal"
                    dataSource={approvalChain}
                    renderItem={record => (
                      <List.Item>
                        <List.Item.Meta
                          avatar={
                            record.status === 'approved' ? (
                              <CheckCircle style={{ color: '#52c41a', fontSize: 24 }} />
                            ) : (
                              <XCircle style={{ color: '#ff4d4f', fontSize: 24 }} />
                            )
                          }
                          title={
                            <Space wrap>
                              <Text strong>{record.approverName}</Text>
                              <ReviewStepBadge role={record.role} isRequired={record.isRequired} />
                              <Tag color={statusColors[record.status]}>
                                {ChangeStatusLabels[record.status as ChangeStatus]}
                              </Tag>
                              <Text type="secondary">
                                {record.createdAt ? dayjs(record.createdAt).format('YYYY-MM-DD HH:mm') : '-'}
                              </Text>
                            </Space>
                          }
                          description={
                            record.approvalType && record.approvalType !== 'serial' && record.threshold ? (
                              <Tag>{`${record.approvalType} (${record.threshold})`}</Tag>
                            ) : undefined
                          }
                        />
                      </List.Item>
                    )}
                  />
                ) : (
                  <Empty description={t('changeDetail.noApprovalRecords')} />
                ),
            },
            {
              key: '3',
              label: t('changeDetail.riskAssessment'),
              children: (
                <Spin spinning={assessmentLoading}>
                  <ChangeRiskAssessment
                    changeId={Number(id)}
                    initialData={riskAssessment}
                    onSave={handleSaveRiskAssessment}
                  />
                </Spin>
              ),
            },
            {
              key: '4',
              label: t('changeDetail.impactAnalysis'),
              children: (
                <Spin spinning={assessmentLoading}>
                  <ChangeImpactAnalysis
                    changeId={Number(id)}
                    initialData={impactAnalysis}
                    onSave={handleSaveImpactAnalysis}
                  />
                </Spin>
              ),
            },
            {
              key: '4b',
              label: t('changeDetail.rollbackChecklist'),
              children: (
                <div className="py-2" data-testid="rollback-checklist">
                  {rollbackSteps.length > 0 && (
                    <div className="mb-4">
                      <Progress
                        percent={Math.round((resolvedCount / rollbackSteps.length) * 100)}
                        status={pendingSteps.length === 0 ? 'success' : 'active'}
                        format={() =>
                          t('changeDetail.checklistProgress')
                            .replace('{resolved}', String(resolvedCount))
                            .replace('{total}', String(rollbackSteps.length))
                        }
                      />
                    </div>
                  )}

                  {/* 门禁前置说明：后端同样会 422，这里把"为什么按钮点不动"说清楚 */}
                  {rollbackBlocked && (
                    <Alert
                      type="warning"
                      showIcon
                      className="mb-4"
                      message={t('changeDetail.checklistBlockedHint').replace(
                        '{count}',
                        String(pendingSteps.length)
                      )}
                    />
                  )}

                  {rollbackSteps.length === 0 ? (
                    <Empty
                      description={t('changeDetail.checklistEmpty')}
                      image={Empty.PRESENTED_IMAGE_SIMPLE}
                    >
                      {canEditChecklist && (
                        <Button type="primary" onClick={openChecklistEditor}>
                          {t('changeDetail.checklistEdit')}
                        </Button>
                      )}
                    </Empty>
                  ) : (
                    <>
                      <List
                        size="small"
                        dataSource={rollbackSteps}
                        renderItem={step => (
                          <List.Item
                            actions={
                              canResolveSteps
                                ? step.resolved
                                  ? [
                                      <Button
                                        key="reset"
                                        type="link"
                                        size="small"
                                        loading={processing}
                                        onClick={() =>
                                          handleResolveStep(step, 'pending', step.note)
                                        }
                                      >
                                        {t('changeDetail.stepResetAction')}
                                      </Button>,
                                    ]
                                  : [
                                      <Button
                                        key="done"
                                        type="link"
                                        size="small"
                                        loading={processing}
                                        onClick={() => handleResolveStep(step, 'done')}
                                      >
                                        {t('changeDetail.stepDone')}
                                      </Button>,
                                      <Button
                                        key="skipped"
                                        type="link"
                                        size="small"
                                        onClick={() => {
                                          setStepNote('');
                                          setResolvingStep({ step, status: 'skipped' });
                                        }}
                                      >
                                        {t('changeDetail.stepSkipped')}
                                      </Button>,
                                      <Button
                                        key="failed"
                                        type="link"
                                        size="small"
                                        danger
                                        onClick={() => {
                                          setStepNote('');
                                          setResolvingStep({ step, status: 'failed' });
                                        }}
                                      >
                                        {t('changeDetail.stepFailed')}
                                      </Button>,
                                    ]
                                : undefined
                            }
                          >
                            <List.Item.Meta
                              avatar={
                                <Tag color={rollbackStepColors[step.status]}>
                                  {t('changeDetail.stepNo').replace('{no}', String(step.stepNo))}
                                </Tag>
                              }
                              title={<Text>{step.description}</Text>}
                              description={
                                <Space direction="vertical" size={0}>
                                  <Text type="secondary">
                                    {rollbackStepLabels[step.status]}
                                    {step.resolvedByName
                                      ? ` · ${step.resolvedByName}`
                                      : ''}
                                    {step.resolvedAt
                                      ? ` · ${dayjs(step.resolvedAt).format('YYYY-MM-DD HH:mm')}`
                                      : ''}
                                  </Text>
                                  {step.note && <Text type="secondary">{step.note}</Text>}
                                </Space>
                              }
                            />
                          </List.Item>
                        )}
                      />
                      {canEditChecklist && (
                        <Button className="mt-4" onClick={openChecklistEditor}>
                          {t('changeDetail.checklistEdit')}
                        </Button>
                      )}
                    </>
                  )}
                </div>
              ),
            },
            {
              key: '5',
              label: t('changeDetail.rollbackPlan'),
              children: (
                <Spin spinning={assessmentLoading}>
                  <ChangeRollbackPlan
                    changeId={Number(id)}
                    initialData={rollbackPlan}
                    onSave={handleSaveRollbackPlan}
                  />
                </Spin>
              ),
            },
            {
              key: '7',
              label: t('changeDetail.cmdbImpactSummary'),
              children: <ChangeCMDBImpactPanel changeId={Number(id)} />,
            },
            {
              key: '6',
              label: t('changeDetail.pir'),
              children: (
                <div className="py-4">
                  <p className="text-gray-500 mb-4">{t('changeDetail.pirDescription')}</p>
                  <Button type="primary" onClick={() => router.push(`/changes/${id}/pir`)}>
                    {change.status === 'completed' ? t('changeDetail.fillPir') : t('changeDetail.viewPir')}
                  </Button>
                </div>
              ),
            },
          ]}
        />
      </Card>

      {/* 实施结果弹窗（回滚 / 标记失败，原因必填） */}
      <Modal
        title={
          outcomeModal === 'fail' ? t('changeDetail.failTitle') : t('changeDetail.rollbackTitle')
        }
        open={outcomeModal !== null}
        onCancel={() => {
          setOutcomeModal(null);
          setOutcomeReason('');
        }}
        footer={[
          <Button
            key="cancel"
            onClick={() => {
              setOutcomeModal(null);
              setOutcomeReason('');
            }}
          >
            {t('changeDetail.cancel')}
          </Button>,
          <Button
            key="confirm"
            danger
            loading={processing}
            disabled={!outcomeReason.trim()}
            onClick={() => {
              if (outcomeModal === 'fail') {
                handleFail(outcomeReason);
              } else {
                handleRollback(outcomeReason);
              }
            }}
          >
            {outcomeModal === 'fail' ? t('changeDetail.markFailed') : t('changeDetail.rollback')}
          </Button>,
        ]}
      >
        <div className="py-4">
          <p className="mb-2">
            {outcomeModal === 'fail'
              ? t('changeDetail.failReasonHint')
              : t('changeDetail.rollbackReasonHint')}
          </p>
          <Input.TextArea
            value={outcomeReason}
            onChange={e => setOutcomeReason(e.target.value)}
            placeholder={t('changeDetail.outcomeReasonPlaceholder')}
            rows={4}
            aria-label={
              outcomeModal === 'fail' ? t('changeDetail.failReason') : t('changeDetail.rollbackReason')
            }
          />
        </div>
      </Modal>

      {/* 回滚步骤备注弹窗（跳过/失败必须写依据） */}
      <Modal
        title={
          resolvingStep?.status === 'failed'
            ? t('changeDetail.stepFailedTitle')
            : t('changeDetail.stepSkippedTitle')
        }
        open={resolvingStep !== null}
        onCancel={() => {
          setResolvingStep(null);
          setStepNote('');
        }}
        footer={[
          <Button
            key="cancel"
            onClick={() => {
              setResolvingStep(null);
              setStepNote('');
            }}
          >
            {t('changeDetail.cancel')}
          </Button>,
          <Button
            key="confirm"
            danger={resolvingStep?.status === 'failed'}
            loading={processing}
            disabled={!stepNote.trim()}
            onClick={() => {
              if (resolvingStep) {
                handleResolveStep(resolvingStep.step, resolvingStep.status, stepNote);
              }
            }}
          >
            {t('changeDetail.confirm')}
          </Button>,
        ]}
      >
        <div className="py-4">
          <p className="mb-2">
            {resolvingStep?.status === 'failed'
              ? t('changeDetail.stepFailedHint')
              : t('changeDetail.stepSkippedHint')}
          </p>
          <Input.TextArea
            value={stepNote}
            onChange={e => setStepNote(e.target.value)}
            placeholder={t('changeDetail.stepNotePlaceholder')}
            rows={3}
            aria-label={t('changeDetail.stepNote')}
          />
        </div>
      </Modal>

      {/* 回滚清单编排弹窗（一行一步，按执行顺序） */}
      <Modal
        title={t('changeDetail.checklistEdit')}
        open={planModalOpen}
        onCancel={() => setPlanModalOpen(false)}
        footer={[
          <Button key="cancel" onClick={() => setPlanModalOpen(false)}>
            {t('changeDetail.cancel')}
          </Button>,
          <Button key="confirm" type="primary" loading={processing} onClick={handleSaveChecklist}>
            {t('changeDetail.save')}
          </Button>,
        ]}
      >
        <div className="py-4">
          <p className="mb-2">{t('changeDetail.checklistEditHint')}</p>
          <Input.TextArea
            value={planText}
            onChange={e => setPlanText(e.target.value)}
            rows={6}
            aria-label={t('changeDetail.rollbackChecklist')}
          />
        </div>
      </Modal>

      {/* 批准弹窗 */}
      <Modal
        title={t('changeDetail.approveTitle')}
        open={approvalModalVisible}
        onCancel={() => {
          setApprovalModalVisible(false);
          setApprovalComment('');
        }}
        footer={[
          <Button
            key="cancel"
            onClick={() => {
              setApprovalModalVisible(false);
              setApprovalComment('');
            }}
          >
            {t('changeDetail.cancel')}
          </Button>,
          <Button key="approve" type="primary" loading={processing} onClick={handleApprove}>
            {t('changeDetail.approve')}
          </Button>,
        ]}
      >
        <div className="py-4">
          <p className="mb-2">{t('changeDetail.approvalCommentOptional')}</p>
          <Input.TextArea
            value={approvalComment}
            onChange={e => setApprovalComment(e.target.value)}
            placeholder={t('changeDetail.approvalCommentPlaceholder')}
            rows={3}
          />
        </div>
      </Modal>

      {/* 拒绝弹窗 */}
      <Modal
        title={t('changeDetail.rejectTitle')}
        open={rejectModalVisible}
        onCancel={() => {
          setRejectModalVisible(false);
          setApprovalComment('');
        }}
        footer={[
          <Button
            key="cancel"
            onClick={() => {
              setRejectModalVisible(false);
              setApprovalComment('');
            }}
          >
            {t('changeDetail.cancel')}
          </Button>,
          <Button key="reject" danger loading={processing} onClick={handleReject}>
            {t('changeDetail.reject')}
          </Button>,
        ]}
      >
        <div className="py-4">
          <p className="mb-2">{t('changeDetail.rejectReasonOptional')}</p>
          <Input.TextArea
            value={approvalComment}
            onChange={e => setApprovalComment(e.target.value)}
            placeholder={t('changeDetail.rejectReasonPlaceholder')}
            rows={3}
          />
        </div>
      </Modal>
    </Space>
  );
};

export default ChangeDetail;
