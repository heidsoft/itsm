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
} from 'antd';
import { ArrowUp, Plus, Save, Pencil, FileText, Clock, AlertCircle, CheckCircle, Plug, AreaChart } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';
import dayjs from 'dayjs';

import { IncidentAPI } from '@/lib/api/';
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
  const [data, setData] = useState<Incident | null>(null);
  const [escalateModalVisible, setEscalateModalVisible] = useState(false);
  const [resolveModalVisible, setResolveModalVisible] = useState(false);
  const [escalating, setEscalating] = useState(false);
  const [resolving, setResolving] = useState(false);
  const [form] = Form.useForm();
  const [resolveForm] = Form.useForm();

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
    try {
      const resp = await IncidentAPI.getIncident(Number(id));
      setData(resp as unknown as Incident);
    } catch (error) {
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
      auto_assign: true,
    });
    setEscalateModalVisible(true);
  };

  const handleEscalateSubmit = async (values: any) => {
    if (!data) return;
    setEscalating(true);
    try {
      await IncidentAPI.escalateIncident(data.id, {
        escalationLevel: values.escalationLevel || values.escalation_level,
        reason: values.reason,
        autoAssign: values.autoAssign || values.auto_assign,
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
        resolution_code: values.resolutionCode,
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
    return <Card>未找到事件</Card>;
  }

  return (
    <>
      <Space orientation="vertical" style={{ width: '100%' }} size="middle">
        {/* 头部操作栏 */}
        <Card styles={{ body: { padding: '16px 24px' } }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div>
              <span style={{ fontSize: 20, fontWeight: 500, marginRight: 16 }}>
                {data.incidentNumber} {data.title}
              </span>
              <Tag color={data.status === IncidentStatus.RESOLVED ? 'success' : 'blue'}>
                {IncidentStatusLabels[data.status]}
              </Tag>
            </div>
            <Space>
              <Button
                icon={<Pencil />}
                onClick={() => router.push(`/incidents/${data.id}/edit`)}
              >
                编辑
              </Button>
              <Button icon={<ArrowUp />} onClick={handleEscalate} loading={escalating}>
                升级
              </Button>
              {data.status !== IncidentStatus.RESOLVED && (
                <Button
                  type="primary"
                  icon={<CheckCircle />}
                  onClick={handleResolveClick}
                  loading={resolving}
                >
                  解决
                </Button>
              )}
            </Space>
          </div>
        </Card>

        {/* 基本信息 */}
        <Card title="基本信息" extra={<Button type="link" icon={<Pencil />} onClick={handleEditCategory}>编辑分类</Button>}>
          <Descriptions column={2}>
            <Descriptions.Item label="报告人ID">{data.reporterId}</Descriptions.Item>
            <Descriptions.Item label="负责人ID">{data.assigneeId || '-'}</Descriptions.Item>
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
              name="escalation_level"
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
            <Form.Item name="auto_assign" label="自动分配">
              <Select placeholder="是否自动分配给上级">
                <Select.Option value={true}>是</Select.Option>
                <Select.Option value={false}>否</Select.Option>
              </Select>
            </Form.Item>
          </Form>
        </Modal>
      )}

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
