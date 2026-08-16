'use client';

/**
 * 变更详情组件
 */

import React, { useState, useEffect } from 'react';
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
  message,
} from 'antd';
import { ArrowLeft, CheckCircle, XCircle } from 'lucide-react';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';
import { Modal, Input } from 'antd';

import { ChangeApi } from '@/lib/api/';
import { useI18n } from '@/lib/i18n/useI18n';
import {
  ChangeStatus,
  ChangeStatusLabels,
  ChangeTypeLabels,
  ChangePriorityLabels,
  ChangeImpactLabels,
  ChangeRiskLabels,
} from '@/constants/change';
import type { Change, ApprovalRecord } from '@/types/biz/change';
import ChangeRiskAssessment from './ChangeRiskAssessment';
import ChangeCMDBImpactPanel from './ChangeCMDBImpactPanel';
import ChangeImpactAnalysis from './ChangeImpactAnalysis';
import ChangeRollbackPlan from './ChangeRollbackPlan';
import { SafeTextBlock } from '@/components/common/SafeContent';
import { formatDateTime } from '@/lib/formatters';

const { Title, Text, Paragraph } = Typography;

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
  const [approvals, setApprovals] = useState<ApprovalRecord[]>([]);
  const [riskAssessment, setRiskAssessment] = useState<any>(null);
  const [impactAnalysis, setImpactAnalysis] = useState<any>(null);
  const [rollbackPlan, setRollbackPlan] = useState<any>(null);
  const [assessmentLoading, setAssessmentLoading] = useState(false);
  const [approvalModalVisible, setApprovalModalVisible] = useState(false);
  const [rejectModalVisible, setRejectModalVisible] = useState(false);
  const [approvalComment, setApprovalComment] = useState('');
  const [processing, setProcessing] = useState(false);

  useEffect(() => {
    if (id) {
      loadDetail();
    }
     
  }, [id]);

  // 生命周期 action bar 可见性(修复 P0-1: 详情页无生命周期推进按钮)
  // draft → 提交审批;pending → 批准/拒绝;approved → 开始实施;
  // in_progress → 完成实施/回滚;非终态 → 取消
  const isTerminal =
    change?.status === ChangeStatus.COMPLETED ||
    change?.status === ChangeStatus.CANCELLED ||
    change?.status === ChangeStatus.ROLLED_BACK ||
    change?.status === ChangeStatus.REJECTED ||
    change?.status === ChangeStatus.FAILED ||
    change?.status === ChangeStatus.CLOSED;
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
    change?.status === ChangeStatus.IN_PROGRESS;

  // 提交审批
  const handleSubmit = async () => {
    if (!change) return;
    setProcessing(true);
    try {
      await ChangeApi.submitForApproval(change.id);
      message.success(t('changeDetail.submitSuccess'));
      loadDetail();
    } catch (error) {
      message.error(t('changeDetail.submitFailed'));
    } finally {
      setProcessing(false);
    }
  };

  // 排期
  const handleSchedule = async () => {
    if (!change) return;
    setProcessing(true);
    try {
      await ChangeApi.scheduleChange(change.id);
      message.success(t('changeDetail.scheduleSuccess'));
      loadDetail();
    } catch (error) {
      message.error(t('changeDetail.scheduleFailed'));
    } finally {
      setProcessing(false);
    }
  };

  // 开始实施
  const handleStart = async () => {
    if (!change) return;
    setProcessing(true);
    try {
      await ChangeApi.startImplementation(change.id);
      message.success(t('changeDetail.startSuccess'));
      loadDetail();
    } catch (error) {
      message.error(t('changeDetail.startFailed'));
    } finally {
      setProcessing(false);
    }
  };

  // 完成实施
  const handleComplete = async () => {
    if (!change) return;
    setProcessing(true);
    try {
      await ChangeApi.completeImplementation(change.id);
      message.success(t('changeDetail.completeSuccess'));
      loadDetail();
    } catch (error) {
      message.error(t('changeDetail.completeFailed'));
    } finally {
      setProcessing(false);
    }
  };

  // 关闭变更（已完成/已回滚 → 已关闭）
  const handleClose = async () => {
    if (!change) return;
    setProcessing(true);
    try {
      await ChangeApi.closeChange(change.id);
      message.success(t('changeDetail.closeSuccess'));
      loadDetail();
    } catch (error) {
      message.error(t('changeDetail.closeFailed'));
    } finally {
      setProcessing(false);
    }
  };

  // 回滚
  const handleRollback = async () => {
    if (!change) return;
    setProcessing(true);
    try {
      await ChangeApi.rollbackChange(change.id, approvalComment);
      message.success(t('changeDetail.rollbackSuccess'));
      setApprovalComment('');
      loadDetail();
    } catch (error) {
      message.error(t('changeDetail.rollbackFailed'));
    } finally {
      setProcessing(false);
    }
  };

  // 取消
  const handleCancel = async () => {
    if (!change) return;
    setProcessing(true);
    try {
      await ChangeApi.cancelChange(change.id, approvalComment);
      message.success(t('changeDetail.cancelSuccess'));
      setApprovalComment('');
      loadDetail();
    } catch (error) {
      message.error(t('changeDetail.cancelFailed'));
    } finally {
      setProcessing(false);
    }
  };

  // 批准变更
  const handleApprove = async () => {
    if (!change) return;
    setProcessing(true);
    try {
      await ChangeApi.approveChange(change.id, { comment: approvalComment });
      message.success(t('changeDetail.approveSuccess'));
      setApprovalModalVisible(false);
      setApprovalComment('');
      loadDetail();
    } catch (error) {
      message.error(t('changeDetail.approveFailed'));
    } finally {
      setProcessing(false);
    }
  };

  // 拒绝变更
  const handleReject = async () => {
    if (!change) return;
    setProcessing(true);
    try {
      await ChangeApi.rejectChange(change.id, { comment: approvalComment });
      message.success(t('changeDetail.rejectSuccess'));
      setRejectModalVisible(false);
      setApprovalComment('');
      loadDetail();
    } catch (error) {
      message.error(t('changeDetail.rejectFailed'));
    } finally {
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
        setApprovals(summary as unknown as ApprovalRecord[]);
      } catch (e) {
        // console.warn('Failed to load approval summary', e);
      }
    } catch (error) {
      // console.error(error);
      message.error(t('changeDetail.loadDetailFailed'));
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
      message.error(t('changeDetail.saveFailed'));
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
      message.error(t('changeDetail.saveFailed'));
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
      message.error(t('changeDetail.saveFailed'));
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
                      onClick={() => setApprovalModalVisible(true)}
                    >
                      {t('changeDetail.approve')}
                    </Button>
                    <Button
                      danger
                      icon={<XCircle />}
                      loading={processing}
                      onClick={() => setRejectModalVisible(true)}
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
                    <Button danger loading={processing} onClick={handleRollback}>
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
                </>
              ),
            },
            {
              key: '2',
              label: t('changeDetail.approvalRecords'),
              children:
                approvals.length > 0 ? (
                  <List
                    itemLayout="horizontal"
                    dataSource={approvals}
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
                            <Space>
                              <Text strong>{record.approverName}</Text>
                              <Tag color={statusColors[record.status]}>
                                {ChangeStatusLabels[record.status]}
                              </Tag>
                              <Text type="secondary">
                                {record.createdAt ? dayjs(record.createdAt).format('YYYY-MM-DD HH:mm') : '-'}
                              </Text>
                            </Space>
                          }
                          description={record.comment || t('changeDetail.noComment')}
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

      {/* 批准弹窗 */}
      <Modal
        title={t('changeDetail.approveTitle')}
        open={approvalModalVisible}
        onCancel={() => setApprovalModalVisible(false)}
        footer={[
          <Button key="cancel" onClick={() => setApprovalModalVisible(false)}>
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
        onCancel={() => setRejectModalVisible(false)}
        footer={[
          <Button key="cancel" onClick={() => setRejectModalVisible(false)}>
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
