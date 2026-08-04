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
import { ArrowUp, Plus, Save, Pencil, FileText, Clock, AlertCircle, CheckCircle, Plug, AreaChart, UserCheck, Siren } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';
import dayjs from 'dayjs';

import { IncidentAPI } from '@/lib/api/';
import { UserApi } from '@/lib/api/user-api';
import type { User } from '@/lib/api/user-api';
import {
  IncidentStatus,
  IncidentStatusLabels,
  IncidentPriorityLabels,
  IncidentSeverityLabels,
} from '@/constants/incident';
import type { Incident } from '@/types/biz/incident';
import { useErrorHandler } from '@/lib/hooks/useErrorHandler';
import { SafeContent, SafeTextBlock } from '@/components/common/SafeContent';
import { isValidIncidentTransition } from '@/lib/utils/workflow-state-machine';

// 根因分析类型
interface RootCauseData {
  id?: number;
  analysisMethod?: string;
  rootCause?: string;
  contributingFactors?: string[];
  evidence?: string[];
  preventiveActions?: string[];
  status?: string;
  createdAt?: string;
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
  const [form] = Form.useForm();
  const [resolveForm] = Form.useForm();

  // ===== 指派：用户列表 + 指派弹窗状态 =====
  const [users, setUsers] = useState<User[]>([]);
  const [loadingUsers, setLoadingUsers] = useState(false);
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

  // ===== 新增：根因分析、影响评估、事件分类状态 =====
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
      handleError(error, 'loadIncident', '加载事件详情失败');
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
      console.error('加载分析数据失败:', error);
    } finally {
      setAnalysisLoading(false);
    }
  }, [id]);

  useEffect(() => {
    loadData();
  }, [id]);

  // 加载用户列表，用于负责人/报告人姓名展示与指派选择
  useEffect(() => {
    const loadUsers = async () => {
      setLoadingUsers(true);
      try {
        const res = await UserApi.getUsers({ pageSize: 100 });
        setUsers(res.users || []);
      } catch (e) {
        // 用户获取失败时降级为展示 ID，不阻断详情页
        setUsers([]);
      } finally {
        setLoadingUsers(false);
      }
    };
    loadUsers();
  }, []);

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
      message.success('事件升级成功');
      setEscalateModalVisible(false);
      loadData(); // 刷新数据
    } catch (error) {
      handleError(error, 'escalateIncident', '升级失败');
    } finally {
      setEscalating(false);
    }
  };

  // 打开解决确认弹窗
  const handleResolveClick = () => {
    resolveForm.resetFields();
    setResolveModalVisible(true);
  };

  // 提交解决方案（ITIL 合规：要求填写解决方案）
  const handleResolveSubmit = async (values: { resolution: string; resolutionCode?: string }) => {
    if (!data) return;

    // 状态转换验证
    if (!isValidIncidentTransition(data.status, 'resolved')) {
      message.error('当前状态不允许直接解决');
      return;
    }

    setResolving(true);
    try {
      // 使用专门的 resolve 端点，而非直接更新状态
      await IncidentAPI.resolveIncident(data.id, {
        resolution: values.resolution,
        resolutionCode: values.resolutionCode,
      });
      message.success('事件已解决');
      setResolveModalVisible(false);
      loadData();
    } catch (error) {
      handleError(error, 'resolveIncident', '解决失败');
    } finally {
      setResolving(false);
    }
  };

  const handleReopen = async () => {
    if (!data) return;

    setReopening(true);
    try {
      await IncidentAPI.reopenIncident(data.id);
      message.success('事件已重新打开');
      loadData();
    } catch (error) {
      handleError(error, 'reopenIncident', '重新打开失败');
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
      message.success('事件指派成功');
      setAssignModalVisible(false);
      assignForm.resetFields();
      loadData();
    } catch (error) {
      handleError(error, 'assignIncident', '指派失败');
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
      message.success('已升级为重大事件');
      setMajorModalVisible(false);
      majorForm.resetFields();
      loadData();
    } catch (error) {
      handleError(error, 'escalateMajorIncident', '升级为重大事件失败');
    } finally {
      setEscalatingMajor(false);
    }
  };

  // ===== 新增：根因分析、影响评估、事件分类处理函数 =====

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

      if (rootCauseData?.id) {
        await IncidentAPI.updateRootCauseAnalysis(rootCauseData.id, request);
        message.success('根因分析已更新');
      } else {
        await IncidentAPI.createRootCauseAnalysis(request);
        message.success('根因分析已创建');
      }
      setRootCauseModalVisible(false);
      loadAnalysisData();
    } catch (error) {
      handleError(error, 'saveRootCause', '保存根因分析失败');
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
        message.success('影响评估已更新');
      } else {
        await IncidentAPI.createImpactAssessment(request);
        message.success('影响评估已创建');
      }
      setImpactModalVisible(false);
      loadAnalysisData();
    } catch (error) {
      handleError(error, 'saveImpact', '保存影响评估失败');
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
  const handleSaveCategory = async (values: any) => {
    if (!data) return;
    setSavingAnalysis(true);
    try {
      const request = {
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

      if (classificationData?.id) {
        await IncidentAPI.updateIncidentClassification(classificationData.id, request);
        message.success('事件分类已更新');
      } else {
        await IncidentAPI.createIncidentClassification(request);
        message.success('事件分类已创建');
      }

      // 同时更新事件的基本分类信息
      await IncidentAPI.updateIncident(data.id, {
        category: values.category,
        subcategory: values.subcategory,
      });

      setCategoryModalVisible(false);
      loadAnalysisData();
      loadData(); // 刷新事件基本信息
    } catch (error) {
      handleError(error, 'saveCategory', '保存事件分类失败');
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
        <Empty description={loadError ? '事件详情加载失败' : '未找到事件'}>
          {loadError && <Button type="primary" onClick={loadData}>重新加载</Button>}
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
                  重大事件
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
                编辑
              </Button>
              <Button icon={<ArrowUp />} onClick={handleEscalate} loading={escalating}>
                升级
              </Button>
              {data.status !== IncidentStatus.RESOLVED && data.status !== IncidentStatus.CLOSED && (
                <Button icon={<UserCheck />} onClick={handleAssignClick} loading={loadingUsers}>
                  指派
                </Button>
              )}
              {!data.isMajorIncident &&
                data.status !== IncidentStatus.RESOLVED &&
                data.status !== IncidentStatus.CLOSED && (
                  <Button danger icon={<Siren />} onClick={() => setMajorModalVisible(true)}>
                    升级为重大事件
                  </Button>
                )}
              {data.status !== IncidentStatus.RESOLVED && data.status !== IncidentStatus.CLOSED && (
                <Button
                  type="primary"
                  icon={<CheckCircle />}
                  onClick={handleResolveClick}
                  loading={resolving}
                >
                  解决
                </Button>
              )}
              {(data.status === IncidentStatus.RESOLVED || data.status === IncidentStatus.CLOSED) && (
                <Button onClick={handleReopen} loading={reopening}>
                  重新打开
                </Button>
              )}
            </Space>
          </div>
          {(data.escalationLevel ?? 0) > 0 && (
            <Alert className="mt-4" type="warning" showIcon
              message={`该事件已升级至 ${data.escalationLevel} 级，请优先处理并保持沟通记录。`} />
          )}
        </Card>

        {/* 基本信息 */}
        <Card title="基本信息" extra={<Button type="link" icon={<Pencil />} onClick={handleEditCategory}>编辑分类</Button>}>
          <Descriptions column={{ xs: 1, sm: 2 }}>
            <Descriptions.Item label="报告人">{getUserName(data.reporterId)}</Descriptions.Item>
            <Descriptions.Item label="负责人">{getUserName(data.assigneeId)}</Descriptions.Item>
            <Descriptions.Item label="优先级">
              {IncidentPriorityLabels[data.priority]}
            </Descriptions.Item>
            <Descriptions.Item label="严重程度">
              {IncidentSeverityLabels[data.severity]}
            </Descriptions.Item>
            <Descriptions.Item label="分类">{data.category || '-'}</Descriptions.Item>
            <Descriptions.Item label="子分类">{data.subcategory || '-'}</Descriptions.Item>
            <Descriptions.Item label="检测时间">
              {data.detectedAt ? dayjs(data.detectedAt).format('YYYY-MM-DD HH:mm:ss') : '-'}
            </Descriptions.Item>
            <Descriptions.Item label="来源">{data.source}</Descriptions.Item>
            {data.problemId && (
              <Descriptions.Item label="关联问题">
                <Button type="link" className="h-auto p-0" onClick={() => router.push(`/problems/${data.problemId}`)}>
                  查看问题 #{data.problemId}
                </Button>
              </Descriptions.Item>
            )}
          </Descriptions>
          <Divider />
          <Descriptions title="详细描述" column={1}>
            <Descriptions.Item label="描述">
              <SafeTextBlock content={data.description} fallback="暂无描述" />
            </Descriptions.Item>
          </Descriptions>

          {/* 影响分析 (如果有) */}
          {data.impactAnalysis && (
            <>
              <Divider />
              <Descriptions title="影响分析" column={1}>
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
                  根因分析
                  {rootCauseData?.status && (
                    <Tag color={rootCauseData.status === 'completed' ? 'success' : 'processing'}>
                      {rootCauseData.status === 'completed' ? '已完成' : '进行中'}
                    </Tag>
                  )}
                </Space>
              }
              extra={
                <Button type="link" icon={<Pencil />} onClick={handleEditRootCause}>
                  {rootCauseData?.id ? '编辑' : '添加'}
                </Button>
              }
            >
              {rootCauseData ? (
                <Descriptions column={2} size="small">
                  <Descriptions.Item label="分析方法">{rootCauseData.analysisMethod || '-'}</Descriptions.Item>
                  <Descriptions.Item label="状态">{rootCauseData.status || '-'}</Descriptions.Item>
                  <Descriptions.Item label="根本原因" span={2}>
                    <SafeTextBlock content={rootCauseData.rootCause} fallback="未填写" />
                  </Descriptions.Item>
                  <Descriptions.Item label="促成因素" span={2}>
                    {rootCauseData.contributingFactors?.length ? (
                      <Space wrap>
                        {rootCauseData.contributingFactors.map((factor, i) => (
                          <Tag key={i}>{factor}</Tag>
                        ))}
                      </Space>
                    ) : '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label="预防措施" span={2}>
                    {rootCauseData.preventiveActions?.length ? (
                      <Space wrap>
                        {rootCauseData.preventiveActions.map((action, i) => (
                          <Tag key={i} color="blue">{action}</Tag>
                        ))}
                      </Space>
                    ) : '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label="创建时间">
                    {rootCauseData.createdAt ? dayjs(rootCauseData.createdAt).format('YYYY-MM-DD HH:mm') : '-'}
                  </Descriptions.Item>
                </Descriptions>
              ) : (
                <Empty description="暂无根因分析" image={Empty.PRESENTED_IMAGE_SIMPLE}>
                  <Button type="primary" icon={<Plus />} onClick={handleEditRootCause}>
                    添加根因分析
                  </Button>
                </Empty>
              )}
            </Card>

            {/* 影响评估 */}
            <Card
              title={
                <Space>
                  <AreaChart />
                  影响评估
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
                  {impactData?.id ? '编辑' : '添加'}
                </Button>
              }
            >
              {impactData ? (
                <Descriptions column={3} size="small">
                  <Descriptions.Item label="业务影响">{renderImpactTag(impactData.businessImpact)}</Descriptions.Item>
                  <Descriptions.Item label="技术影响">{renderImpactTag(impactData.technicalImpact)}</Descriptions.Item>
                  <Descriptions.Item label="声誉影响">{renderImpactTag(impactData.reputationImpact)}</Descriptions.Item>
                  <Descriptions.Item label="受影响用户">{impactData.affectedUsersCount || 0}</Descriptions.Item>
                  <Descriptions.Item label="财务影响">¥{impactData.financialImpact || 0}</Descriptions.Item>
                  <Descriptions.Item label="合规影响">
                    <Tag color={impactData.complianceImpact ? 'red' : 'default'}>
                      {impactData.complianceImpact ? '是' : '否'}
                    </Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="受影响服务" span={3}>
                    {impactData.affectedServices?.length ? (
                      <Space wrap>
                        {impactData.affectedServices.map((service, i) => (
                          <Tag key={i}>{service}</Tag>
                        ))}
                      </Space>
                    ) : '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label="评估备注" span={3}>
                    <SafeTextBlock content={impactData.assessmentNotes} fallback="-" />
                  </Descriptions.Item>
                  <Descriptions.Item label="评估时间">
                    {impactData.createdAt ? dayjs(impactData.createdAt).format('YYYY-MM-DD HH:mm') : '-'}
                  </Descriptions.Item>
                </Descriptions>
              ) : (
                <Empty description="暂无影响评估" image={Empty.PRESENTED_IMAGE_SIMPLE}>
                  <Button type="primary" icon={<Plus />} onClick={handleEditImpact}>
                    添加影响评估
                  </Button>
                </Empty>
              )}
            </Card>

            {/* 事件分类 */}
            <Card
              title={
                <Space>
                  <Plug />
                  事件分类
                  {classificationData?.classificationConfidence !== undefined && (
                    <Tag color={classificationData.classificationConfidence >= 80 ? 'green' : 'orange'}>
                      置信度 {classificationData.classificationConfidence}%
                    </Tag>
                  )}
                </Space>
              }
              extra={
                <Button type="link" icon={<Pencil />} onClick={handleEditCategory}>
                  {classificationData?.id ? '编辑' : '添加'}
                </Button>
              }
            >
              {classificationData ? (
                <Descriptions column={3} size="small">
                  <Descriptions.Item label="分类">{classificationData.category || '-'}</Descriptions.Item>
                  <Descriptions.Item label="子分类">{classificationData.subcategory || '-'}</Descriptions.Item>
                  <Descriptions.Item label="服务类型">{classificationData.serviceType || '-'}</Descriptions.Item>
                  <Descriptions.Item label="故障类型">{classificationData.failureType || '-'}</Descriptions.Item>
                  <Descriptions.Item label="紧急程度">{renderImpactTag(classificationData.urgency)}</Descriptions.Item>
                  <Descriptions.Item label="影响程度">{renderImpactTag(classificationData.impact)}</Descriptions.Item>
                  <Descriptions.Item label="创建时间" span={3}>
                    {classificationData.createdAt ? dayjs(classificationData.createdAt).format('YYYY-MM-DD HH:mm') : '-'}
                  </Descriptions.Item>
                </Descriptions>
              ) : (
                <Empty description="暂无事件分类" image={Empty.PRESENTED_IMAGE_SIMPLE}>
                  <Button type="primary" icon={<Plus />} onClick={handleEditCategory}>
                    添加事件分类
                  </Button>
                </Empty>
              )}
            </Card>
          </Space>
        </Spin>

        {/* 解决记录 (如果有) */}
        {data.resolutionSteps && data.resolutionSteps.length > 0 && (
          <Card title="处理流程">
            <Timeline>
              {data.resolutionSteps.map((step, index) => (
                <Timeline.Item key={index}>
                  <p>{(step as unknown as { description?: string }).description || '处理步骤'}</p>
                  <span style={{ fontSize: '12px', color: '#999' }}>{(step as unknown as { timestamp?: string }).timestamp}</span>
                </Timeline.Item>
              ))}
            </Timeline>
          </Card>
        )}
      </Space>

      {escalateModalVisible && (
        <Modal
          title="升级事件"
          open={escalateModalVisible}
          onCancel={() => setEscalateModalVisible(false)}
          confirmLoading={escalating}
          onOk={() => form.submit()}
        >
          <Form form={form} layout="vertical" onFinish={handleEscalateSubmit}>
            <Form.Item
              name="escalationLevel"
              label="升级级别"
              rules={[{ required: true, message: '请选择升级级别' }]}
            >
              <Select placeholder="请选择升级级别">
                <Select.Option value={1}>级别 1 - 主管</Select.Option>
                <Select.Option value={2}>级别 2 - 经理</Select.Option>
                <Select.Option value={3}>级别 3 - 总监</Select.Option>
                <Select.Option value={4}>级别 4 - VP</Select.Option>
              </Select>
            </Form.Item>
            <Form.Item
              name="reason"
              label="升级原因"
              rules={[{ required: true, message: '请输入升级原因' }]}
            >
              <Input.TextArea rows={3} placeholder="请输入升级原因" />
            </Form.Item>
            <Form.Item name="autoAssign" label="自动分配">
              <Select placeholder="是否自动分配给上级">
                <Select.Option value={true}>是</Select.Option>
                <Select.Option value={false}>否</Select.Option>
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
            指派事件
          </Space>
        }
        open={assignModalVisible}
        onCancel={() => {
          setAssignModalVisible(false);
          assignForm.resetFields();
        }}
        confirmLoading={assigning}
        onOk={() => assignForm.submit()}
        okText="确认指派"
        cancelText="取消"
        width={480}
      >
        <Form form={assignForm} layout="vertical" onFinish={handleAssignSubmit}>
          <Form.Item
            name="assigneeId"
            label="指派给"
            rules={[{ required: true, message: '请选择处理人' }]}
          >
            <Select
              placeholder="请选择处理人"
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
            升级为重大事件
          </Space>
        }
        open={majorModalVisible}
        onCancel={() => {
          setMajorModalVisible(false);
          majorForm.resetFields();
        }}
        confirmLoading={escalatingMajor}
        onOk={() => majorForm.submit()}
        okText="确认升级"
        okButtonProps={{ danger: true }}
        cancelText="取消"
        width={520}
      >
        <div style={{ marginBottom: 16, color: '#8c8c8c', fontSize: 13 }}>
          升级后事件严重程度将提升为“严重”，并记录影响评估与审计日志，此操作不可撤销。
        </div>
        <Form form={majorForm} layout="vertical" onFinish={handleMajorSubmit}>
          <Form.Item
            name="impactScope"
            label="影响范围"
            rules={[{ required: true, message: '请选择影响范围' }]}
          >
            <Select placeholder="请选择影响范围">
              <Select.Option value="low">低 - 少量用户受影响</Select.Option>
              <Select.Option value="medium">中 - 部分部门/服务受影响</Select.Option>
              <Select.Option value="high">高 - 多个核心服务受影响</Select.Option>
              <Select.Option value="critical">严重 - 全局性业务中断</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="businessImpact"
            label="业务影响评估"
            rules={[
              { required: true, message: '请填写业务影响评估' },
              { min: 10, message: '业务影响评估至少需要10个字符' },
            ]}
          >
            <Input.TextArea
              rows={4}
              placeholder="请描述受影响的业务/系统范围、用户数量、预估损失等..."
              showCount
              maxLength={2000}
            />
          </Form.Item>
          <Form.Item name="communicationPlan" label="危机沟通计划">
            <Input.TextArea
              rows={3}
              placeholder="可选：说明通报对象、沟通频率、作战室/应急群等安排..."
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
            解决事件
          </Space>
        }
        open={resolveModalVisible}
        onCancel={() => setResolveModalVisible(false)}
        confirmLoading={resolving}
        onOk={() => resolveForm.submit()}
        okText="确认解决"
        cancelText="取消"
        width={500}
      >
        <Form form={resolveForm} layout="vertical" onFinish={handleResolveSubmit}>
          <Form.Item
            name="resolution"
            label="解决方案"
            rules={[
              { required: true, message: '请填写解决方案' },
              { min: 10, message: '解决方案至少需要10个字符' },
            ]}
          >
            <Input.TextArea
              rows={4}
              placeholder="请详细描述问题的解决方案和处理步骤..."
              showCount
              maxLength={2000}
            />
          </Form.Item>
          <Form.Item name="resolutionCode" label="解决分类">
            <Select placeholder="选择解决分类（可选）">
              <Select.Option value="fixed">已修复</Select.Option>
              <Select.Option value="workaround">临时解决方案</Select.Option>
              <Select.Option value="no_action">无需操作</Select.Option>
              <Select.Option value="third_party">第三方解决</Select.Option>
              <Select.Option value="user_error">用户错误</Select.Option>
            </Select>
          </Form.Item>
          {data?.problemId && (
            <div style={{ padding: '8px 12px', background: '#f5f5f5', borderRadius: 4 }}>
              <AlertCircle style={{ marginRight: 8, color: '#faad14' }} />
              <span>此事件已关联问题记录 #{data.problemId}</span>
            </div>
          )}
        </Form>
      </Modal>

      {/* ===== 新增：根因分析编辑弹窗 ===== */}
      <Modal
        title={
          <Space>
            <FileText />
            根因分析
          </Space>
        }
        open={rootCauseModalVisible}
        onCancel={() => setRootCauseModalVisible(false)}
        confirmLoading={savingAnalysis}
        onOk={() => rootCauseForm.submit()}
        okText="保存"
        cancelText="取消"
        width={600}
      >
        <Form form={rootCauseForm} layout="vertical" onFinish={handleSaveRootCause}>
          <Form.Item name="analysisMethod" label="分析方法" rules={[{ required: true }]}>
            <Select placeholder="选择分析方法">
              <Select.Option value="5-whys">5 Whys（五问法）</Select.Option>
              <Select.Option value="fishbone">鱼骨图（ Ishikawa）</Select.Option>
              <Select.Option value="timeline">时间线分析</Select.Option>
              <Select.Option value="fault-tree">故障树分析（FTA）</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item
            name="rootCause"
            label="根本原因"
            rules={[{ required: true, message: '请填写根本原因' }]}
          >
            <Input.TextArea rows={3} placeholder="分析并描述问题的根本原因..." showCount maxLength={500} />
          </Form.Item>
          <Form.Item name="contributingFactors" label="促成因素（每行一个）">
            <Input.TextArea rows={3} placeholder="列出促成因素，每行一个..." showCount maxLength={500} />
          </Form.Item>
          <Form.Item name="evidence" label="证据（每行一个）">
            <Input.TextArea rows={3} placeholder="列出支持分析的证据，每行一个..." showCount maxLength={500} />
          </Form.Item>
          <Form.Item name="preventiveActions" label="预防措施（每行一个）">
            <Input.TextArea rows={3} placeholder="列出预防措施，每行一个..." showCount maxLength={500} />
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select placeholder="选择状态">
              <Select.Option value="draft">草稿</Select.Option>
              <Select.Option value="in-progress">进行中</Select.Option>
              <Select.Option value="completed">已完成</Select.Option>
              <Select.Option value="approved">已批准</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* ===== 新增：影响评估编辑弹窗 ===== */}
      <Modal
        title={
          <Space>
            <AreaChart />
            影响评估
          </Space>
        }
        open={impactModalVisible}
        onCancel={() => setImpactModalVisible(false)}
        confirmLoading={savingAnalysis}
        onOk={() => impactForm.submit()}
        okText="保存"
        cancelText="取消"
        width={600}
      >
        <Form form={impactForm} layout="vertical" onFinish={handleSaveImpact}>
          <Form.Item name="businessImpact" label="业务影响" rules={[{ required: true }]}>
            <Select placeholder="选择业务影响等级">
              <Select.Option value="low">低 - 最小业务影响</Select.Option>
              <Select.Option value="medium">中 - 部分业务受影响</Select.Option>
              <Select.Option value="high">高 - 显著业务影响</Select.Option>
              <Select.Option value="critical">严重 - 业务中断</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="technicalImpact" label="技术影响" rules={[{ required: true }]}>
            <Select placeholder="选择技术影响等级">
              <Select.Option value="low">低 - 最小技术影响</Select.Option>
              <Select.Option value="medium">中 - 部分系统受影响</Select.Option>
              <Select.Option value="high">高 - 核心系统受影响</Select.Option>
              <Select.Option value="critical">严重 - 系统不可用</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="affectedServices" label="受影响服务（逗号分隔）">
            <Input.TextArea rows={2} placeholder="列出受影响的服务，用逗号分隔..." />
          </Form.Item>
          <Form.Item name="affectedUsersCount" label="受影响用户数">
            <Input type="number" placeholder="估计受影响的用户数量" min={0} />
          </Form.Item>
          <Form.Item name="financialImpact" label="财务影响（元）">
            <Input type="number" placeholder="估计的财务损失" min={0} />
          </Form.Item>
          <Form.Item name="reputationImpact" label="声誉影响">
            <Select placeholder="选择声誉影响等级">
              <Select.Option value="low">低</Select.Option>
              <Select.Option value="medium">中</Select.Option>
              <Select.Option value="high">高</Select.Option>
              <Select.Option value="critical">严重</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="complianceImpact" label="合规影响" valuePropName="checked">
            <Input type="checkbox" style={{ width: 16 }} />
            <span style={{ marginLeft: 8 }}>此事件涉及合规问题</span>
          </Form.Item>
          <Form.Item name="assessmentNotes" label="评估备注">
            <Input.TextArea rows={3} placeholder="补充评估说明..." showCount maxLength={500} />
          </Form.Item>
        </Form>
      </Modal>

      {/* ===== 新增：事件分类编辑弹窗 ===== */}
      <Modal
        title={
          <Space>
            <Plug />
            事件分类
          </Space>
        }
        open={categoryModalVisible}
        onCancel={() => setCategoryModalVisible(false)}
        confirmLoading={savingAnalysis}
        onOk={() => categoryForm.submit()}
        okText="保存"
        cancelText="取消"
        width={600}
      >
        <Form form={categoryForm} layout="vertical" onFinish={handleSaveCategory}>
          <Form.Item name="category" label="事件分类" rules={[{ required: true, message: '请选择事件分类' }]}>
            <Select placeholder="选择事件分类">
              <Select.Option value="基础设施">基础设施</Select.Option>
              <Select.Option value="应用系统">应用系统</Select.Option>
              <Select.Option value="网络连接">网络连接</Select.Option>
              <Select.Option value="安全事件">安全事件</Select.Option>
              <Select.Option value="数据问题">数据问题</Select.Option>
              <Select.Option value="用户体验">用户体验</Select.Option>
              <Select.Option value="其他">其他</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="subcategory" label="子分类">
            <Input placeholder="请输入子分类" />
          </Form.Item>
          <Form.Item name="serviceType" label="服务类型">
            <Select placeholder="选择服务类型">
              <Select.Option value="计算">计算</Select.Option>
              <Select.Option value="存储">存储</Select.Option>
              <Select.Option value="网络">网络</Select.Option>
              <Select.Option value="数据库">数据库</Select.Option>
              <Select.Option value="中间件">中间件</Select.Option>
              <Select.Option value="应用服务">应用服务</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="failureType" label="故障类型">
            <Select placeholder="选择故障类型">
              <Select.Option value="性能下降">性能下降</Select.Option>
              <Select.Option value="服务不可用">服务不可用</Select.Option>
              <Select.Option value="功能异常">功能异常</Select.Option>
              <Select.Option value="数据丢失">数据丢失</Select.Option>
              <Select.Option value="安全漏洞">安全漏洞</Select.Option>
              <Select.Option value="配置错误">配置错误</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="urgency" label="紧急程度" rules={[{ required: true }]}>
            <Select placeholder="选择紧急程度">
              <Select.Option value="low">低 - 普通响应</Select.Option>
              <Select.Option value="medium">中 - 4小时内响应</Select.Option>
              <Select.Option value="high">高 - 1小时内响应</Select.Option>
              <Select.Option value="critical">紧急 - 立即响应</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="impact" label="影响程度" rules={[{ required: true }]}>
            <Select placeholder="选择影响程度">
              <Select.Option value="low">低 - 单个用户</Select.Option>
              <Select.Option value="medium">中 - 部分用户</Select.Option>
              <Select.Option value="high">高 - 部门/团队</Select.Option>
              <Select.Option value="critical">严重 - 全局/客户</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};

export default IncidentDetail;
