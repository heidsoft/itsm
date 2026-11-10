'use client';

/**
 * 事件详情组件
 * 包含：基本信息、根因分析、影响评估、事件分类的编辑入口
 */

import React, { useState, useEffect, useCallback } from 'react';
import {
  Card,
  Descriptions,
  Tag,
  Button,
  Space,
  Timeline,
  Skeleton,
  message,
  Divider,
  Modal,
  Form,
  Select,
  Input,
  Badge,
  Empty,
  Spin,
  Alert,
} from 'antd';
import { ArrowUp, Plus, Save, Pencil, FileText, Clock, AlertCircle, CheckCircle, Plug, AreaChart, UserCheck, Siren, Play } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';
import dayjs from 'dayjs';

import { IncidentAPI } from '@/lib/api/';
import type { CreateIncidentClassificationRequest } from '@/lib/api/incident-api';
import { useUserPickerQuery } from '@/lib/hooks/useUserListQuery';
import {
  IncidentStatus,
  IncidentStatusLabels,
  IncidentSeverityLabels,
  incidentPriorityLabel,
} from '@/constants/incident';
import type { Incident } from '@/types/biz/incident';
import { useErrorHandler } from '@/lib/hooks/useErrorHandler';
import { SafeContent, SafeTextBlock } from '@/components/common/SafeContent';
import { isValidIncidentTransition } from '@/lib/utils/workflow-state-machine';

// 事件分类弹窗表单值：incidentId 由当前事件决定，置信度与自动分类标记由前端固定填充。
type IncidentClassificationFormValues = Omit<
  CreateIncidentClassificationRequest,
  'incidentId' | 'classificationConfidence' | 'autoClassified'
>;

// 根因分析类型
interface RootCauseData {
  analysisMethod?: string;
  rootCause?: string;
  contributingFactors?: string[];
  evidence?: string[];
  preventiveActions?: string[];
  status?: string;
}

// 影响评估类型
interface ImpactAssessmentData {
  id?: number;
  businessImpact?: string;
  technicalImpact?: string;
  affectedServices?: string[];
  affectedUsersCount?: number;
  financialImpact?: number;
  reputationImpact?: string;
  complianceImpact?: boolean;
  assessmentNotes?: string;
  createdAt?: string;
}

// 事件分类类型
interface IncidentClassificationData {
  id?: number;
  category?: string;
  subcategory?: string;
  serviceType?: string;
  failureType?: string;
  urgency?: string;
  impact?: string;
  classificationConfidence?: number;
  createdAt?: string;
}

const IncidentDetail: React.FC<{ id?: string }> = ({ id: propId }) => {
  const params = useParams();
  const router = useRouter();
  // 支持通过props传入id，或通过useParams获取
  const id = propId || (params?.id as string);
  const { handleError } = useErrorHandler();
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState(false);
  const [data, setData] = useState<Incident | null>(null);
  const [escalateModalVisible, setEscalateModalVisible] = useState(false);
  const [resolveModalVisible, setResolveModalVisible] = useState(false);
  const [escalating, setEscalating] = useState(false);
  const [resolving, setResolving] = useState(false);
  const [reopening, setReopening] = useState(false);
  const [starting, setStarting] = useState(false);
  const [form] = Form.useForm();
  const [resolveForm] = Form.useForm();

  // ===== 指派：用户列表 + 指派弹窗状态 =====
  // 与其余选人下拉共用同一份 React Query 缓存（5 分钟内不重复请求），
  // 不再各页面各拉一次。
  const { users, isLoading: loadingUsers } = useUserPickerQuery();
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [assigning, setAssigning] = useState(false);
  const [assignForm] = Form.useForm<{ assigneeId: number }>();

  // ===== 升级为重大事件：弹窗状态 =====
  const [majorModalVisible, setMajorModalVisible] = useState(false);
  const [escalatingMajor, setEscalatingMajor] = useState(false);
  const [majorForm] = Form.useForm<{
    impactScope: 'low' | 'medium' | 'high' | 'critical';
    businessImpact: string;
    communicationPlan?: string;
  }>();

  // ===== 根因分析、影响评估、事件分类状态 =====
  const [rootCauseData, setRootCauseData] = useState<RootCauseData | null>(null);
  const [impactData, setImpactData] = useState<ImpactAssessmentData | null>(null);
  const [classificationData, setClassificationData] = useState<IncidentClassificationData | null>(null);
  const [analysisLoading, setAnalysisLoading] = useState(false);

  // 编辑弹窗状态
  const [rootCauseModalVisible, setRootCauseModalVisible] = useState(false);
  const [impactModalVisible, setImpactModalVisible] = useState(false);
  const [categoryModalVisible, setCategoryModalVisible] = useState(false);
  const [savingAnalysis, setSavingAnalysis] = useState(false);

  // 表单
  const [rootCauseForm] = Form.useForm();
  const [impactForm] = Form.useForm();
  const [categoryForm] = Form.useForm();

  const loadData = async () => {
    if (!id) return;
    setLoading(true);
    setLoadError(false);
    try {
      const resp = await IncidentAPI.getIncident(Number(id));
      setData(resp as unknown as Incident);
    } catch (error) {
      setLoadError(true);
      handleError(error, 'loadIncident', 'Failed to load event details');
    } finally {
      setLoading(false);
    }
  };

  // 加载分析数据（根因分析、影响评估、事件分类）
  const loadAnalysisData = useCallback(async () => {
    if (!id) return;
    setAnalysisLoading(true);
    try {
      const incidentId = Number(id);

      // 并行加载三个数据
      const [rootCause, impact, classification] = await Promise.allSettled([
        IncidentAPI.getRootCauseAnalysis(incidentId).catch(() => null),
        IncidentAPI.getImpactAssessment(incidentId).catch(() => null),
        IncidentAPI.getIncidentClassification(incidentId).catch(() => null),
      ]);

      if (rootCause.status === 'fulfilled') {
        setRootCauseData(rootCause.value || null);
      }
      if (impact.status === 'fulfilled') {
        setImpactData(impact.value || null);
      }
      if (classification.status === 'fulfilled') {
        setClassificationData(classification.value || null);
      }
    } catch (error) {
      console.error('Failed to load analysis data:', error);
    } finally {
      setAnalysisLoading(false);
    }
  }, [id]);

  useEffect(() => {
    loadData();
  }, [id]);

  // 用户 ID → 姓名（找不到时回退为 #ID）
  const getUserName = (userId?: number | null) => {
    if (!userId) return '-';
    const user = users.find(u => u.id === userId);
    return user ? user.name || user.username : `#${userId}`;
  };

  // 当数据加载完成后，异步加载分析数据
  useEffect(() => {
    if (data?.id) {
      loadAnalysisData();
    }
  }, [data?.id, loadAnalysisData]);

  const handleEscalate = () => {
    form.setFieldsValue({
      escalationLevel: (data?.escalationLevel || 0) + 1,
      reason: '',
      autoAssign: true,
    });
    setEscalateModalVisible(true);
  };

  const handleEscalateSubmit = async (values: any) => {
    if (!data) return;
    setEscalating(true);
    try {
      await IncidentAPI.escalateIncident(data.id, {
        escalationLevel: values.escalationLevel,
        reason: values.reason,
        autoAssign: values.autoAssign,
      });
      message.success('Incident escalated successfully');
      setEscalateModalVisible(false);
      loadData(); // 刷新数据
    } catch (error) {
      // 不传硬编码文案，直接展示服务端返回的真实错误消息（如：已处于终态、级别不高于当前等）
      handleError(error, 'escalateIncident');
    } finally {
      setEscalating(false);
    }
  };

  // 打开解决确认弹窗
  const handleResolveClick = () => {
    resolveForm.resetFields();
    setResolveModalVisible(true);
  };

  // 开始处理：将事件推进到 in_progress（解决前的必要步骤）
  const handleStartProcessing = async () => {
    if (!data) return;

    if (!isValidIncidentTransition(data.status, IncidentStatus.IN_PROGRESS)) {
      message.error('Current status does not allow processing to start');
      return;
    }

    setStarting(true);
    try {
      await IncidentAPI.updateIncidentStatus(data.id, { status: IncidentStatus.IN_PROGRESS });
      message.success('Processing started');
      loadData();
    } catch (error) {
      handleError(error, 'startProcessing', 'Failed to start processing');
    } finally {
      setStarting(false);
    }
  };

  // 提交解决方案（ITIL 合规：要求填写解决方案）
  const handleResolveSubmit = async (values: { resolution: string; resolutionCode?: string }) => {
    if (!data) return;

    // 状态转换验证
    if (!isValidIncidentTransition(data.status, 'resolved')) {
      message.error('The current status does not allow direct resolution');
      return;
    }

    setResolving(true);
    try {
      // 使用专门的 resolve 端点，而非直接更新状态
      await IncidentAPI.resolveIncident(data.id, {
        resolution: values.resolution,
        resolutionCode: values.resolutionCode,
      });
      message.success('Incident resolved');
      setResolveModalVisible(false);
      loadData();
    } catch (error) {
      handleError(error, 'resolveIncident', 'Resolution failed');
    } finally {
      setResolving(false);
    }
  };

  const handleReopen = async () => {
    if (!data) return;

    setReopening(true);
    try {
      await IncidentAPI.reopenIncident(data.id);
      message.success('Incident reopened');
      loadData();
    } catch (error) {
      handleError(error, 'reopenIncident', 'Reopen failed');
    } finally {
      setReopening(false);
    }
  };

  // 打开指派弹窗
  const handleAssignClick = () => {
    assignForm.setFieldsValue({ assigneeId: data?.assigneeId ?? undefined });
    setAssignModalVisible(true);
  };

  // 提交指派（使用专用 assign 端点）
  const handleAssignSubmit = async (values: { assigneeId: number }) => {
    if (!data) return;
    setAssigning(true);
    try {
      await IncidentAPI.assignIncident(data.id, values.assigneeId);
      message.success('Incident assigned successfully');
      setAssignModalVisible(false);
      assignForm.resetFields();
      loadData();
    } catch (error) {
      handleError(error, 'assignIncident', 'Assignment failed');
    } finally {
      setAssigning(false);
    }
  };

  // 提交升级为重大事件（使用专用 major-incident 端点）
  const handleMajorSubmit = async (values: {
    impactScope: 'low' | 'medium' | 'high' | 'critical';
    businessImpact: string;
    communicationPlan?: string;
  }) => {
    if (!data) return;
    setEscalatingMajor(true);
    try {
      await IncidentAPI.escalateMajorIncident(data.id, values);
      message.success('Escalated to major incident');
      setMajorModalVisible(false);
      majorForm.resetFields();
      loadData();
    } catch (error) {
      handleError(error, 'escalateMajorIncident', 'Failed to escalate to major incident');
    } finally {
      setEscalatingMajor(false);
    }
  };

  // ===== 根因分析、影响评估、事件分类处理函数 =====

  // 打开根因分析编辑弹窗
  const handleEditRootCause = () => {
    rootCauseForm.setFieldsValue({
      analysisMethod: rootCauseData?.analysisMethod || '5-whys',
      rootCause: rootCauseData?.rootCause || '',
      contributingFactors: rootCauseData?.contributingFactors?.join('\n') || '',
      evidence: rootCauseData?.evidence?.join('\n') || '',
      preventiveActions: rootCauseData?.preventiveActions?.join('\n') || '',
      status: rootCauseData?.status || 'draft',
    });
    setRootCauseModalVisible(true);
  };

  // 保存根因分析
  const handleSaveRootCause = async (values: any) => {
    if (!data) return;
    setSavingAnalysis(true);
    try {
      const request = {
        incidentId: data.id,
        analysisMethod: values.analysisMethod,
        rootCause: values.rootCause,
        contributingFactors: values.contributingFactors?.split('\n').filter(Boolean) || [],
        evidence: values.evidence?.split('\n').filter(Boolean) || [],
        preventiveActions: values.preventiveActions?.split('\n').filter(Boolean) || [],
        status: values.status,
      };

      await IncidentAPI.saveRootCauseAnalysis(data.id, request);
      message.success("Root cause analysis saved");
      setRootCauseModalVisible(false);
      loadAnalysisData();
    } catch (error) {
      handleError(error, 'saveRootCause', 'Failed to save root cause analysis');
    } finally {
      setSavingAnalysis(false);
    }
  };

  // 打开影响评估编辑弹窗
  const handleEditImpact = () => {
    impactForm.setFieldsValue({
      businessImpact: impactData?.businessImpact || 'medium',
      technicalImpact: impactData?.technicalImpact || 'medium',
      affectedServices: impactData?.affectedServices?.join(', ') || '',
      affectedUsersCount: impactData?.affectedUsersCount || 0,
      financialImpact: impactData?.financialImpact || 0,
      reputationImpact: impactData?.reputationImpact || 'low',
      complianceImpact: impactData?.complianceImpact || false,
      assessmentNotes: impactData?.assessmentNotes || '',
    });
    setImpactModalVisible(true);
  };

  // 保存影响评估
  const handleSaveImpact = async (values: any) => {
    if (!data) return;
    setSavingAnalysis(true);
    try {
      const request = {
        incidentId: data.id,
        businessImpact: values.businessImpact,
        technicalImpact: values.technicalImpact,
        affectedServices: values.affectedServices?.split(',').map((s: string) => s.trim()).filter(Boolean) || [],
        affectedUsersCount: values.affectedUsersCount || 0,
        financialImpact: values.financialImpact || 0,
        reputationImpact: values.reputationImpact,
        complianceImpact: values.complianceImpact || false,
        assessmentNotes: values.assessmentNotes || '',
      };

      if (impactData?.id) {
        await IncidentAPI.updateImpactAssessment(impactData.id, request);
        message.success('Impact assessment updated');
      } else {
        await IncidentAPI.createImpactAssessment(request);
        message.success('Impact assessment created');
      }
      setImpactModalVisible(false);
      loadAnalysisData();
    } catch (error) {
      handleError(error, 'saveImpact', 'Failed to save impact assessment');
    } finally {
      setSavingAnalysis(false);
    }
  };

  // 打开事件分类编辑弹窗
  const handleEditCategory = () => {
    categoryForm.setFieldsValue({
      category: classificationData?.category || data?.category || '',
      subcategory: classificationData?.subcategory || data?.subcategory || '',
      serviceType: classificationData?.serviceType || '',
      failureType: classificationData?.failureType || '',
      urgency: classificationData?.urgency || 'medium',
      impact: classificationData?.impact || 'medium',
    });
    setCategoryModalVisible(true);
  };

  // 保存事件分类
  const handleSaveCategory = async (values: IncidentClassificationFormValues) => {
    if (!data) return;
    setSavingAnalysis(true);
    try {
      const request: CreateIncidentClassificationRequest = {
        incidentId: data.id,
        category: values.category,
        subcategory: values.subcategory,
        serviceType: values.serviceType,
        failureType: values.failureType,
        urgency: values.urgency,
        impact: values.impact,
        classificationConfidence: 100,
        autoClassified: false,
      };

      // 分类接口在后端就是「更新事件」的同一用例（UpdateClassification -> service.Update），
      // 它已经把 category/subcategory 落到事件行并推进了乐观锁 version。此处若再补一次
      // PUT /incidents/:id，携带的是 POST 之前读到的 version，必然被判为并发冲突并弹出
      // 「事件已被修改」——数据其实已保存成功。一次用户操作只发一次写请求。
      // 后端 GetClassification 只返回 {incidentId, category, subcategory}，永远没有 id，
      // 故不存在「更新已有分类」分支；POST 即唯一且幂等的写入口。
      await IncidentAPI.createIncidentClassification(request);
      message.success('Event classification saved');

      setCategoryModalVisible(false);
      loadAnalysisData();
      loadData(); // 刷新事件基本信息
    } catch (error) {
      // handleError 只接受 (error, context)，第三个参数会被静默丢弃，用户看不到任何
      // 提示；这里显式展示后端的语义化文案（冲突 4090 / 越权 2003）。
      handleError(error, 'saveCategory');
      const detail = error instanceof Error ? error.message : '';
      message.error(detail || 'Failed to save incident classification');
    } finally {
      setSavingAnalysis(false);
    }
  };

  // 渲染影响等级标签
  const renderImpactTag = (level?: string) => {
    const colorMap: Record<string, string> = {
      critical: 'red',
      high: 'orange',
      medium: 'blue',
      low: 'green',
    };
    return <Tag color={colorMap[level || ''] || 'default'}>{level || '-'}</Tag>;
  };

  if (loading) {
    return (
      <Card>
        <Skeleton active />
      </Card>
    );
  }

  if (!data) {
    return (
      <Card>
        <Empty description={loadError ? 'Failed to load event details' : 'Incident not found'}>
          {loadError && <Button type="primary" onClick={loadData}>Reload</Button>}
        </Empty>
      </Card>
    );
  }

  return (
    <>
      <Space orientation="vertical" style={{ width: '100%' }} size="middle">
        {/* 头部操作栏 */}
        <Card styles={{ body: { padding: '16px 24px' } }}>
          <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div className="min-w-0">
              <span style={{ fontSize: 20, fontWeight: 500, marginRight: 16 }}>
                {data.incidentNumber} {data.title}
              </span>
              {data.isMajorIncident && (
                <Tag color="red" icon={<AlertCircle size={12} style={{ marginRight: 4, verticalAlign: -1 }} />}>

                  Major Incident
                </Tag>
              )}
              <Tag color={data.status === IncidentStatus.RESOLVED ? 'success' : 'blue'}>
                {IncidentStatusLabels[data.status]}
              </Tag>
            </div>
            <Space wrap>
              <Button
                icon={<Pencil />}
                onClick={() => router.push(`/incidents/${data.id}/edit`)}
              >

                Edit
              </Button>
              <Button icon={<ArrowUp />} onClick={handleEscalate} loading={escalating}>

                Escalate
              </Button>
              {data.status !== IncidentStatus.RESOLVED && data.status !== IncidentStatus.CLOSED && (
                <Button icon={<UserCheck />} onClick={handleAssignClick} loading={loadingUsers}>

                  Assign
                </Button>
              )}
              {!data.isMajorIncident &&
                data.status !== IncidentStatus.RESOLVED &&
                data.status !== IncidentStatus.CLOSED && (
                  <Button danger icon={<Siren />} onClick={() => setMajorModalVisible(true)}>

                    Escalate to Major Incident
                  </Button>
                )}
              {/* 开始处理：new/acknowledged/assigned/on_hold → in_progress（解决前置步骤） */}
              {isValidIncidentTransition(data.status, IncidentStatus.IN_PROGRESS) && (
                <Button icon={<Play />} onClick={handleStartProcessing} loading={starting}>

                  Start Processing
                </Button>
              )}
              {data.status !== IncidentStatus.RESOLVED && data.status !== IncidentStatus.CLOSED && (
                <Button
                  type="primary"
                  icon={<CheckCircle />}
                  onClick={handleResolveClick}
                  loading={resolving}
                >

                  Resolve
                </Button>
              )}
              {(data.status === IncidentStatus.RESOLVED || data.status === IncidentStatus.CLOSED) && (
                <Button onClick={handleReopen} loading={reopening}>

                  Reopen
                </Button>
              )}
            </Space>
          </div>
          {(data.escalationLevel ?? 0) > 0 && (
            <Alert className="mt-4" type="warning" showIcon
              title={`This incident has been escalated to ${data.escalationLevel} level, please prioritize and keep communication records.`} />
          )}
        </Card>

        {/* 基本信息 */}
        <Card title="Basic Info" extra={<Button type="link" icon={<Pencil />} onClick={handleEditCategory}>Edit Category</Button>}>
          <Descriptions column={{ xs: 1, sm: 2 }}>
            <Descriptions.Item label="Reporter">{getUserName(data.reporterId)}</Descriptions.Item>
            <Descriptions.Item label="Assignee">{getUserName(data.assigneeId)}</Descriptions.Item>
            <Descriptions.Item label="Priority">
              {incidentPriorityLabel(data.priority)}
            </Descriptions.Item>
            <Descriptions.Item label="Severity">
              {IncidentSeverityLabels[data.severity]}
            </Descriptions.Item>
            <Descriptions.Item label="Category">{data.category || '-'}</Descriptions.Item>
            <Descriptions.Item label="Subcategory">{data.subcategory || '-'}</Descriptions.Item>
            <Descriptions.Item label="Detected At">
              {data.detectedAt ? dayjs(data.detectedAt).format('YYYY-MM-DD HH:mm:ss') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="Source">{data.source}</Descriptions.Item>
            {data.problemId && (
              <Descriptions.Item label="Association problem">
                <Button type="link" className="h-auto p-0" onClick={() => router.push(`/problems/${data.problemId}`)}>

                  View issue #{data.problemId}
                </Button>
              </Descriptions.Item>
            )}
          </Descriptions>
          <Divider />
          <Descriptions title="Detailed description" column={1}>
            <Descriptions.Item label="Description">
              <SafeTextBlock content={data.description} fallback="No description" />
            </Descriptions.Item>
          </Descriptions>

          {/* 影响分析 (如果有) */}
          {data.impactAnalysis && (
            <>
              <Divider />
              <Descriptions title="Impact Analysis" column={1}>
                <Descriptions.Item>
                  <pre>{JSON.stringify(data.impactAnalysis, null, 2)}</pre>
                </Descriptions.Item>
              </Descriptions>
            </>
          )}
        </Card>

        {/* 分析卡片区域 */}
        <Spin spinning={analysisLoading}>
          <Space orientation="vertical" style={{ width: '100%' }} size="middle">
            {/* 根因分析 */}
            <Card
              title={
                <Space>
                  <FileText />

                  Root Cause Analysis
                  {rootCauseData?.status && (
                    <Tag color={rootCauseData.status === 'completed' ? 'success' : 'processing'}>
                      {rootCauseData.status === 'completed' ? 'Completed' : 'In Progress'}
                    </Tag>
                  )}
                </Space>
              }
              extra={
                <Button type="link" icon={<Pencil />} onClick={handleEditRootCause}>
                  {rootCauseData ? "Edit" : "Add"}
                </Button>
              }
            >
              {rootCauseData ? (
                <Descriptions column={2} size="small">
                  <Descriptions.Item label="Analysis Method">{rootCauseData.analysisMethod || '-'}</Descriptions.Item>
                  <Descriptions.Item label="Status">{rootCauseData.status || '-'}</Descriptions.Item>
                  <Descriptions.Item label="Root Cause" span={2}>
                    <SafeTextBlock content={rootCauseData.rootCause} fallback="Not filled in" />
                  </Descriptions.Item>
                  <Descriptions.Item label="Contributing Factors" span={2}>
                    {rootCauseData.contributingFactors?.length ? (
                      <Space wrap>
                        {rootCauseData.contributingFactors.map((factor, i) => (
                          <Tag key={i}>{factor}</Tag>
                        ))}
                      </Space>
                    ) : '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label="Prevention Measure" span={2}>
                    {rootCauseData.preventiveActions?.length ? (
                      <Space wrap>
                        {rootCauseData.preventiveActions.map((action, i) => (
                          <Tag key={i} color="blue">{action}</Tag>
                        ))}
                      </Space>
                    ) : '-'}
                  </Descriptions.Item>
                </Descriptions>
              ) : (
                <Empty description="No root cause analysis yet" image={Empty.PRESENTED_IMAGE_SIMPLE}>
                  <Button type="primary" icon={<Plus />} onClick={handleEditRootCause}>

                    Add Root Cause Analysis
                  </Button>
                </Empty>
              )}
            </Card>

            {/* 影响评估 */}
            <Card
              title={
                <Space>
                  <AreaChart />

                  Impact Assessment
                  {(impactData?.businessImpact || impactData?.technicalImpact) && (
                    <Badge
                      status={impactData?.businessImpact === 'critical' ? 'error' : 'processing'}
                      text={
                        <Space>
                          {renderImpactTag(impactData?.businessImpact)}
                          {renderImpactTag(impactData?.technicalImpact)}
                        </Space>
                      }
                    />
                  )}
                </Space>
              }
              extra={
                <Button type="link" icon={<Pencil />} onClick={handleEditImpact}>
                  {impactData?.id ? 'Edit' : 'Add'}
                </Button>
              }
            >
              {impactData ? (
                <Descriptions column={3} size="small">
                  <Descriptions.Item label="Business Impact">{renderImpactTag(impactData.businessImpact)}</Descriptions.Item>
                  <Descriptions.Item label="Technical Impact">{renderImpactTag(impactData.technicalImpact)}</Descriptions.Item>
                  <Descriptions.Item label="Reputation Impact">{renderImpactTag(impactData.reputationImpact)}</Descriptions.Item>
                  <Descriptions.Item label="Affected Users">{impactData.affectedUsersCount || 0}</Descriptions.Item>
                  <Descriptions.Item label="Financial impact">¥{impactData.financialImpact || 0}</Descriptions.Item>
                  <Descriptions.Item label="Compliance Impact">
                    <Tag color={impactData.complianceImpact ? 'red' : 'default'}>
                      {impactData.complianceImpact ? 'Yes' : 'No'}
                    </Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="Affected services" span={3}>
                    {impactData.affectedServices?.length ? (
                      <Space wrap>
                        {impactData.affectedServices.map((service, i) => (
                          <Tag key={i}>{service}</Tag>
                        ))}
                      </Space>
                    ) : '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label="Assessment Notes" span={3}>
                    <SafeTextBlock content={impactData.assessmentNotes} fallback="-" />
                  </Descriptions.Item>
                  <Descriptions.Item label="Assessment Time">
                    {impactData.createdAt ? dayjs(impactData.createdAt).format('YYYY-MM-DD HH:mm') : '-'}
                  </Descriptions.Item>
                </Descriptions>
              ) : (
                <Empty description="No impact assessment" image={Empty.PRESENTED_IMAGE_SIMPLE}>
                  <Button type="primary" icon={<Plus />} onClick={handleEditImpact}>

                    Add Impact Assessment
                  </Button>
                </Empty>
              )}
            </Card>

            {/* 事件分类 */}
            <Card
              title={
                <Space>
                  <Plug />

                  Incident Category
                  {classificationData?.classificationConfidence !== undefined && (
                    <Tag color={classificationData.classificationConfidence >= 80 ? 'green' : 'orange'}>

                      Confidence {classificationData.classificationConfidence}%
                    </Tag>
                  )}
                </Space>
              }
              extra={
                <Button type="link" icon={<Pencil />} onClick={handleEditCategory}>
                  {classificationData?.id ? 'Edit' : 'Add'}
                </Button>
              }
            >
              {classificationData ? (
                <Descriptions column={3} size="small">
                  <Descriptions.Item label="Category">{classificationData.category || '-'}</Descriptions.Item>
                  <Descriptions.Item label="Subcategory">{classificationData.subcategory || '-'}</Descriptions.Item>
                  <Descriptions.Item label="Service Type">{classificationData.serviceType || '-'}</Descriptions.Item>
                  <Descriptions.Item label="Failure Type">{classificationData.failureType || '-'}</Descriptions.Item>
                  <Descriptions.Item label="Urgency">{renderImpactTag(classificationData.urgency)}</Descriptions.Item>
                  <Descriptions.Item label="Impact Level">{renderImpactTag(classificationData.impact)}</Descriptions.Item>
                  <Descriptions.Item label="Created" span={3}>
                    {classificationData.createdAt ? dayjs(classificationData.createdAt).format('YYYY-MM-DD HH:mm') : '-'}
                  </Descriptions.Item>
                </Descriptions>
              ) : (
                <Empty description="No incident classification" image={Empty.PRESENTED_IMAGE_SIMPLE}>
                  <Button type="primary" icon={<Plus />} onClick={handleEditCategory}>

                    Add Incident Classification
                  </Button>
                </Empty>
              )}
            </Card>
          </Space>
        </Spin>

        {/* 解决记录 (如果有) */}
        {data.resolutionSteps && data.resolutionSteps.length > 0 && (
          <Card title="Processing Steps">
            <Timeline>
              {data.resolutionSteps.map((step, index) => (
                <Timeline.Item key={index}>
                  <p>{(step as unknown as { description?: string }).description || 'Processing Step'}</p>
                  <span style={{ fontSize: '12px', color: '#999' }}>{(step as unknown as { timestamp?: string }).timestamp}</span>
                </Timeline.Item>
              ))}
            </Timeline>
          </Card>
        )}
      </Space>

      {escalateModalVisible && (
        <Modal
          title="Escalate Incident"
          open={escalateModalVisible}
          onCancel={() => setEscalateModalVisible(false)}
          confirmLoading={escalating}
          onOk={() => form.submit()}
        >
          <Form form={form} layout="vertical" onFinish={handleEscalateSubmit}>
            <Form.Item
              name="escalationLevel"
              label="Escalation Level"
              rules={[{ required: true, message: 'Please select an escalation level' }]}
            >
              <Select placeholder="Please select an escalation level">
                <Select.Option value={1}>Level 1 - Supervisor</Select.Option>
                <Select.Option value={2}>Level 2 - Manager</Select.Option>
                <Select.Option value={3}>Level 3 - Director</Select.Option>
                <Select.Option value={4}>Level 4 - VP</Select.Option>
              </Select>
            </Form.Item>
            <Form.Item
              name="reason"
              label="Escalation Reason"
              rules={[{ required: true, message: 'Please enter escalation reason' }]}
            >
              <Input.TextArea rows={3} placeholder="Please enter escalation reason" />
            </Form.Item>
            <Form.Item name="autoAssign" label="Auto Assign">
              <Select placeholder="Auto assign to superior">
                <Select.Option value={true}>Yes</Select.Option>
                <Select.Option value={false}>No</Select.Option>
              </Select>
            </Form.Item>
          </Form>
        </Modal>
      )}

      {/* 指派弹窗 */}
      <Modal
        title={
          <Space>
            <UserCheck style={{ color: '#1677ff' }} />

            Assign Incident
          </Space>
        }
        open={assignModalVisible}
        onCancel={() => {
          setAssignModalVisible(false);
          assignForm.resetFields();
        }}
        confirmLoading={assigning}
        onOk={() => assignForm.submit()}
        okText="Confirm Assignment"
        cancelText="Cancel"
        width={480}
      >
        <Form form={assignForm} layout="vertical" onFinish={handleAssignSubmit}>
          <Form.Item
            name="assigneeId"
            label="Assign To"
            rules={[{ required: true, message: 'Please select an assignee' }]}
          >
            <Select
              placeholder="Please select an assignee"
              loading={loadingUsers}
              showSearch
              optionFilterProp="label"
              options={users.map(user => ({
                value: user.id,
                label: `${user.name || user.username}${user.department ? ` (${user.department})` : ''}`,
              }))}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 升级为重大事件弹窗（影响评估 + 危机沟通） */}
      <Modal
        title={
          <Space>
            <Siren style={{ color: '#ff4d4f' }} />

            Escalate to Major Incident
          </Space>
        }
        open={majorModalVisible}
        onCancel={() => {
          setMajorModalVisible(false);
          majorForm.resetFields();
        }}
        confirmLoading={escalatingMajor}
        onOk={() => majorForm.submit()}
        okText="Confirm Escalation"
        okButtonProps={{ danger: true }}
        cancelText="Cancel"
        width={520}
      >
        <div style={{ marginBottom: 16, color: '#8c8c8c', fontSize: 13 }}>

          After the upgrade, the event severity will be increased to "Critical" and impact assessment and audit logs will be recorded. This operation cannot be undone.
        </div>
        <Form form={majorForm} layout="vertical" onFinish={handleMajorSubmit}>
          <Form.Item
            name="impactScope"
            label="Impact Range"
            rules={[{ required: true, message: 'Please select impact scope' }]}
          >
            <Select placeholder="Please select impact scope">
              <Select.Option value="low">Low - Few users affected</Select.Option>
              <Select.Option value="medium">Medium - Some departments/services affected</Select.Option>
              <Select.Option value="high">High - Multiple core services affected</Select.Option>
              <Select.Option value="critical">Critical - Global business interruption</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="businessImpact"
            label="Business Impact Assessment"
            rules={[
              { required: true, message: 'Please fill in business impact assessment' },
              { min: 10, message: 'Business impact assessment requires at least 10 characters' },
            ]}
          >
            <Input.TextArea
              rows={4}
              placeholder="Please describe the affected business/system scope, number of users, estimated losses, etc..."
              showCount
              maxLength={2000}
            />
          </Form.Item>
          <Form.Item name="communicationPlan" label="Crisis Communication Plan">
            <Input.TextArea
              rows={3}
              placeholder="Optional: describe notification targets, frequency, war room/emergency group arrangements..."
              showCount
              maxLength={1000}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 解决确认弹窗 (ITIL 合规：要求填写解决方案) */}
      <Modal
        title={
          <Space>
            <CheckCircle style={{ color: '#52c41a' }} />

            Resolve Incident
          </Space>
        }
        open={resolveModalVisible}
        onCancel={() => setResolveModalVisible(false)}
        confirmLoading={resolving}
        onOk={() => resolveForm.submit()}
        okText="Confirm Resolution"
        cancelText="Cancel"
        width={500}
      >
        <Form form={resolveForm} layout="vertical" onFinish={handleResolveSubmit}>
          <Form.Item
            name="resolution"
            label="Solutions"
            rules={[
              { required: true, message: 'Please fill in the resolution' },
              { min: 10, message: 'Resolution requires at least 10 characters' },
            ]}
          >
            <Input.TextArea
              rows={4}
              placeholder="Please describe the solution and processing steps of the problem in detail..."
              showCount
              maxLength={2000}
            />
          </Form.Item>
          <Form.Item name="resolutionCode" label="Resolution Code">
            <Select placeholder="Select resolution code (optional)">
              <Select.Option value="fixed">Fixed</Select.Option>
              <Select.Option value="workaround">Workaround</Select.Option>
              <Select.Option value="no_action">No Action Required</Select.Option>
              <Select.Option value="third_party">Third Party Resolved</Select.Option>
              <Select.Option value="user_error">User Error</Select.Option>
            </Select>
          </Form.Item>
          {data?.problemId && (
            <div style={{ padding: '8px 12px', background: '#f5f5f5', borderRadius: 4 }}>
              <AlertCircle style={{ marginRight: 8, color: '#faad14' }} />
              <span>This event has an associated problem record #{data.problemId}</span>
            </div>
          )}
        </Form>
      </Modal>

      {/* 根因分析编辑弹窗 */}
      <Modal
        title={
          <Space>
            <FileText />

            Root Cause Analysis
          </Space>
        }
        open={rootCauseModalVisible}
        onCancel={() => setRootCauseModalVisible(false)}
        confirmLoading={savingAnalysis}
        onOk={() => rootCauseForm.submit()}
        okText="Save"
        cancelText="Cancel"
        width={600}
      >
        <Form form={rootCauseForm} layout="vertical" onFinish={handleSaveRootCause}>
          <Form.Item name="analysisMethod" label="Analysis Method" rules={[{ required: true }]}>
            <Select placeholder="Select analysis method">
              <Select.Option value="5-whys">5 Whys</Select.Option>
              <Select.Option value="fishbone">Fishbone (Ishikawa)</Select.Option>
              <Select.Option value="timeline">Timeline Analysis</Select.Option>
              <Select.Option value="fault-tree">Fault Tree Analysis (FTA)</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="rootCause"
            label="Root Cause"
            rules={[{ required: true, message: 'Please fill in root cause' }]}
          >
            <Input.TextArea rows={3} placeholder="Analyze and describe the root cause..." showCount maxLength={500} />
          </Form.Item>
          <Form.Item name="contributingFactors" label="Contributing Factors (one per line)">
            <Input.TextArea rows={3} placeholder="List contributing factors, one per line..." showCount maxLength={500} />
          </Form.Item>
          <Form.Item name="evidence" label="Evidence (one per line)">
            <Input.TextArea rows={3} placeholder="List supporting evidence, one per line..." showCount maxLength={500} />
          </Form.Item>
          <Form.Item name="preventiveActions" label="Preventive Actions (one per line)">
            <Input.TextArea rows={3} placeholder="List preventive actions, one per line..." showCount maxLength={500} />
          </Form.Item>
          <Form.Item name="status" label="Status">
            <Select placeholder="Select status">
              <Select.Option value="draft">Draft</Select.Option>
              <Select.Option value="in-progress">In Progress</Select.Option>
              <Select.Option value="completed">Completed</Select.Option>
              <Select.Option value="approved">Approved</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 影响评估编辑弹窗 */}
      <Modal
        title={
          <Space>
            <AreaChart />

            Impact Assessment
          </Space>
        }
        open={impactModalVisible}
        onCancel={() => setImpactModalVisible(false)}
        confirmLoading={savingAnalysis}
        onOk={() => impactForm.submit()}
        okText="Save"
        cancelText="Cancel"
        width={600}
      >
        <Form form={impactForm} layout="vertical" onFinish={handleSaveImpact}>
          <Form.Item name="businessImpact" label="Business Impact" rules={[{ required: true }]}>
            <Select placeholder="Select business impact level">
              <Select.Option value="low">Low - Minimal business impact</Select.Option>
              <Select.Option value="medium">Medium - Some services are affected</Select.Option>
              <Select.Option value="high">High - Significant business impact</Select.Option>
              <Select.Option value="critical">Critical - Business interruption</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="technicalImpact" label="Technical Impact" rules={[{ required: true }]}>
            <Select placeholder="Select technology impact level">
              <Select.Option value="low">Low - minimal technical impact</Select.Option>
              <Select.Option value="medium">Medium - Some systems affected</Select.Option>
              <Select.Option value="high">High - Core system affected</Select.Option>
              <Select.Option value="critical">Critical - System unavailable</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="affectedServices" label="Affected Services (comma separated)">
            <Input.TextArea rows={2} placeholder="List affected services, comma separated..." />
          </Form.Item>
          <Form.Item name="affectedUsersCount" label="Affected User Count">
            <Input type="number" placeholder="Estimated number of affected users" min={0} />
          </Form.Item>
          <Form.Item name="financialImpact" label="Financial Impact">
            <Input type="number" placeholder="Estimated financial loss" min={0} />
          </Form.Item>
          <Form.Item name="reputationImpact" label="Reputation Impact">
            <Select placeholder="Select reputation impact level">
              <Select.Option value="low">Low</Select.Option>
              <Select.Option value="medium">Medium</Select.Option>
              <Select.Option value="high">High</Select.Option>
              <Select.Option value="critical">Critical</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="complianceImpact" label="Compliance Impact" valuePropName="checked">
            <Input type="checkbox" style={{ width: 16 }} />
            <span style={{ marginLeft: 8 }}>This incident involves compliance issues</span>
          </Form.Item>
          <Form.Item name="assessmentNotes" label="Assessment Notes">
            <Input.TextArea rows={3} placeholder="Additional assessment notes..." showCount maxLength={500} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 事件分类编辑弹窗 */}
      <Modal
        title={
          <Space>
            <Plug />

            Incident Category
          </Space>
        }
        open={categoryModalVisible}
        onCancel={() => setCategoryModalVisible(false)}
        confirmLoading={savingAnalysis}
        onOk={() => categoryForm.submit()}
        okText="Save"
        cancelText="Cancel"
        width={600}
      >
        <Form form={categoryForm} layout="vertical" onFinish={handleSaveCategory}>
          <Form.Item name="category" label="Incident Category" rules={[{ required: true, message: 'Please select an incident category' }]}>
            <Select placeholder="Select incident category">
              <Select.Option value="Infrastructure">Infrastructure</Select.Option>
              <Select.Option value="Application Systems">Application Systems</Select.Option>
              <Select.Option value="Network">Network</Select.Option>
              <Select.Option value="Security Incident">Security Incident</Select.Option>
              <Select.Option value="Data Issue">Data Issue</Select.Option>
              <Select.Option value="User Experience">User Experience</Select.Option>
              <Select.Option value="Other">Other</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="subcategory" label="Subcategory">
            <Input placeholder="Enter subcategory" />
          </Form.Item>
          <Form.Item name="serviceType" label="Service Type">
            <Select placeholder="Select service type">
              <Select.Option value="Compute">Compute</Select.Option>
              <Select.Option value="Storage">Storage</Select.Option>
              <Select.Option value="Network">Network</Select.Option>
              <Select.Option value="Database">Database</Select.Option>
              <Select.Option value="Middleware">Middleware</Select.Option>
              <Select.Option value="Application Service">Application Service</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="failureType" label="Failure Type">
            <Select placeholder="Select failure type">
              <Select.Option value="Performance Degradation">Performance Degradation</Select.Option>
              <Select.Option value="Service Unavailable">Service Unavailable</Select.Option>
              <Select.Option value="Function Error">Function Error</Select.Option>
              <Select.Option value="Data Loss">Data Loss</Select.Option>
              <Select.Option value="Security Vulnerability">Security Vulnerability</Select.Option>
              <Select.Option value="Configuration Error">Configuration Error</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="urgency" label="Urgency" rules={[{ required: true }]}>
            <Select placeholder="Select urgency">
              <Select.Option value="low">Low - Normal response</Select.Option>
              <Select.Option value="medium">Medium - Response within 4 hours</Select.Option>
              <Select.Option value="high">High - Response within 1 hour</Select.Option>
              <Select.Option value="critical">Critical - Immediate response</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="impact" label="Impact Level" rules={[{ required: true }]}>
            <Select placeholder="Select impact level">
              <Select.Option value="low">Low - Single user</Select.Option>
              <Select.Option value="medium">Medium - Some users</Select.Option>
              <Select.Option value="high">High - Department/team</Select.Option>
              <Select.Option value="critical">Critical - Global/customer</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default IncidentDetail;
