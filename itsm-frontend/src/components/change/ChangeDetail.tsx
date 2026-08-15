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
      message.success('变更已提交审批');
      loadDetail();
    } catch (error) {
      message.error('提交审批失败');
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
      message.success('变更已排期');
      loadDetail();
    } catch (error) {
      message.error('排期失败');
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
      message.success('变更已开始实施');
      loadDetail();
    } catch (error) {
      message.error('开始实施失败');
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
      message.success('变更实施完成');
      loadDetail();
    } catch (error) {
      message.error('完成实施失败');
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
      message.success('变更已关闭归档');
      loadDetail();
    } catch (error) {
      message.error('关闭变更失败');
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
      message.success('变更已回滚');
      setApprovalComment('');
      loadDetail();
    } catch (error) {
      message.error('回滚失败');
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
      message.success('变更已取消');
      setApprovalComment('');
      loadDetail();
    } catch (error) {
      message.error('取消失败');
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
      message.success('变更已批准');
      setApprovalModalVisible(false);
      setApprovalComment('');
      loadDetail();
    } catch (error) {
      message.error('批准失败');
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
      message.success('变更已拒绝');
      setRejectModalVisible(false);
      setApprovalComment('');
      loadDetail();
    } catch (error) {
      message.error('拒绝失败');
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
      message.error('加载变更详情失败');
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
      message.success('风险评估保存成功');
      loadRiskAssessment();
    } catch (error) {
      message.error('保存失败');
    }
  };

  // 保存影响分析
  const handleSaveImpactAnalysis = async (data: any) => {
    if (!id) return;
    try {
      await ChangeApi.updateImpactAnalysis(Number(id), data);
      message.success('影响分析保存成功');
      loadImpactAnalysis();
    } catch (error) {
      message.error('保存失败');
    }
  };

  // 保存回滚计划
  const handleSaveRollbackPlan = async (data: any) => {
    if (!id) return;
    try {
      // 回滚计划暂时通过更新变更的rollback_plan字段实现
      await ChangeApi.updateChange(Number(id), { rollbackPlan: JSON.stringify(data) });
      message.success('回滚计划保存成功');
    } catch (error) {
      message.error('保存失败');
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
          subTitle="抱歉，您访问的变更不存在"
          extra={
            <Button type="primary" onClick={() => router.push('/changes')}>
              返回列表
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
            返回列表
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
                    提交审批
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
                      批准
                    </Button>
                    <Button
                      danger
                      icon={<XCircle />}
                      loading={processing}
                      onClick={() => setRejectModalVisible(true)}
                    >
                      拒绝
                    </Button>
                  </>
                )}
                {canSchedule && (
                  <Button type="primary" loading={processing} onClick={handleSchedule}>
                    排期
                  </Button>
                )}
                {canStart && (
                  <Button type="primary" loading={processing} onClick={handleStart}>
                    开始实施
                  </Button>
                )}
                {canComplete && (
                  <>
                    <Button type="primary" loading={processing} onClick={handleComplete}>
                      完成实施
                    </Button>
                    <Button danger loading={processing} onClick={handleRollback}>
                      回滚
                    </Button>
                  </>
                )}
                {canCancel && (
                  <Button danger loading={processing} onClick={handleCancel}>
                    取消变更
                  </Button>
                )}
              </Space>
            )}
            {canClose && (
              <Button type="primary" loading={processing} onClick={handleClose} style={{ marginLeft: 12 }}>
                关闭
              </Button>
            )}
          </div>
        </div>

        <Descriptions bordered column={2}>
          <Descriptions.Item label="变更编号">{change.id}</Descriptions.Item>
          <Descriptions.Item label="变更类型">{ChangeTypeLabels[change.type]}</Descriptions.Item>
          <Descriptions.Item label="优先级">
            {ChangePriorityLabels[change.priority]}
          </Descriptions.Item>
          <Descriptions.Item label="风险等级">
            {ChangeRiskLabels[change.riskLevel]}
          </Descriptions.Item>
          <Descriptions.Item label="影响范围">
            {ChangeImpactLabels[change.impactScope]}
          </Descriptions.Item>
          <Descriptions.Item label="负责人">{change.assigneeName || '未分配'}</Descriptions.Item>
          <Descriptions.Item label="计划起始">
            {change.plannedStartDate
              ? formatDateTime(change.plannedStartDate) || '-'
              : '-'}
          </Descriptions.Item>
          <Descriptions.Item label="计划截止">
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
              label: '基础信息',
              children: (
                <>
                  <Title level={5}>变更原因 / 理由</Title>
                  <SafeTextBlock content={change.justification} fallback="无" />

                  <Title level={5}>变更描述</Title>
                  <SafeTextBlock content={change.description} fallback="无" />

                  <Divider />

                  <Title level={5}>实施计划</Title>
                  <SafeTextBlock
                    content={change.implementationPlan}
                    fallback="未提供实施计划"
                    preserveNewlines
                  />

                  <Title level={5}>回滚计划</Title>
                  <SafeTextBlock
                    content={change.rollbackPlan}
                    fallback="未提供回滚计划"
                    preserveNewlines
                  />
                </>
              ),
            },
            {
              key: '2',
              label: '审批记录',
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
                          description={record.comment || '无意见'}
                        />
                      </List.Item>
                    )}
                  />
                ) : (
                  <Empty description="暂无审批记录" />
                ),
            },
            {
              key: '3',
              label: '风险评估',
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
              label: '影响分析',
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
              label: '回滚计划',
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
              label: 'CMDB 影响摘要',
              children: <ChangeCMDBImpactPanel changeId={Number(id)} />,
            },
            {
              key: '6',
              label: '实施后审查 (PIR)',
              children: (
                <div className="py-4">
                  <p className="text-gray-500 mb-4">评估变更实施结果，总结经验教训</p>
                  <Button type="primary" onClick={() => router.push(`/changes/${id}/pir`)}>
                    {change.status === 'completed' ? '填写PIR' : '查看PIR'}
                  </Button>
                </div>
              ),
            },
          ]}
        />
      </Card>

      {/* 批准弹窗 */}
      <Modal
        title="批准变更"
        open={approvalModalVisible}
        onCancel={() => setApprovalModalVisible(false)}
        footer={[
          <Button key="cancel" onClick={() => setApprovalModalVisible(false)}>
            取消
          </Button>,
          <Button key="approve" type="primary" loading={processing} onClick={handleApprove}>
            批准
          </Button>,
        ]}
      >
        <div className="py-4">
          <p className="mb-2">审批意见（可选）：</p>
          <Input.TextArea
            value={approvalComment}
            onChange={e => setApprovalComment(e.target.value)}
            placeholder="请输入审批意见..."
            rows={3}
          />
        </div>
      </Modal>

      {/* 拒绝弹窗 */}
      <Modal
        title="拒绝变更"
        open={rejectModalVisible}
        onCancel={() => setRejectModalVisible(false)}
        footer={[
          <Button key="cancel" onClick={() => setRejectModalVisible(false)}>
            取消
          </Button>,
          <Button key="reject" danger loading={processing} onClick={handleReject}>
            拒绝
          </Button>,
        ]}
      >
        <div className="py-4">
          <p className="mb-2">拒绝原因（可选）：</p>
          <Input.TextArea
            value={approvalComment}
            onChange={e => setApprovalComment(e.target.value)}
            placeholder="请输入拒绝原因..."
            rows={3}
          />
        </div>
      </Modal>
    </Space>
  );
};

export default ChangeDetail;
