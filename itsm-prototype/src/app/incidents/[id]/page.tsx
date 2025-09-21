"use client";

import React, { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import {
  Card,
  Descriptions,
  Tag,
  Button,
  Space,
  Tabs,
  Timeline,
  Form,
  Input,
  Select,
  Modal,
  Alert,
  Row,
  Col,
  Divider,
  Progress,
  Badge,
} from "antd";
import {
  ArrowLeftOutlined,
  EditOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  BugOutlined,
  SafetyOutlined,
  FileTextOutlined,
} from "@ant-design/icons";
import {
  AlertTriangle,
  MessageSquare,
} from "lucide-react";
import { IncidentAPI, Incident } from "../../lib/incident-api";

const { TextArea } = Input;
const { Option } = Select;
const { TabPane } = Tabs;

// 根因分析接口
interface RootCauseAnalysis {
  id?: number;
  incident_id: number;
  analysis_method: string; // "5-whys" | "fishbone" | "timeline" | "fault-tree"
  root_cause: string;
  contributing_factors: string[];
  evidence: string[];
  preventive_actions: string[];
  status: "draft" | "in-progress" | "completed" | "approved";
  analyst_id?: number;
  analyst_name?: string;
  created_at?: string;
  updated_at?: string;
}

// 影响评估接口
interface ImpactAssessment {
  id?: number;
  incident_id: number;
  business_impact: "low" | "medium" | "high" | "critical";
  technical_impact: "low" | "medium" | "high" | "critical";
  affected_services: string[];
  affected_users_count: number;
  financial_impact: number;
  reputation_impact: "low" | "medium" | "high" | "critical";
  compliance_impact: boolean;
  assessment_notes: string;
  assessor_id?: number;
  assessor_name?: string;
  created_at?: string;
  updated_at?: string;
}

// 事件分类接口
interface IncidentClassification {
  id?: number;
  incident_id: number;
  category: string;
  subcategory: string;
  service_type: string;
  failure_type: string;
  urgency: "low" | "medium" | "high" | "critical";
  impact: "low" | "medium" | "high" | "critical";
  classification_confidence: number; // 0-100
  auto_classified: boolean;
  classifier_id?: number;
  classifier_name?: string;
  created_at?: string;
  updated_at?: string;
}

export default function IncidentDetailPage() {
  const params = useParams();
  const incidentId = parseInt(params.id as string);
  
  const [incident, setIncident] = useState<Incident | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState("overview");
  
  // 根因分析相关状态
  const [rootCauseAnalysis, setRootCauseAnalysis] = useState<RootCauseAnalysis | null>(null);
  const [showRootCauseModal, setShowRootCauseModal] = useState(false);
  const [rootCauseForm] = Form.useForm();
  
  // 影响评估相关状态
  const [impactAssessment, setImpactAssessment] = useState<ImpactAssessment | null>(null);
  const [showImpactModal, setShowImpactModal] = useState(false);
  const [impactForm] = Form.useForm();
  
  // 事件分类相关状态
  const [classification, setClassification] = useState<IncidentClassification | null>(null);
  const [showClassificationModal, setShowClassificationModal] = useState(false);
  const [classificationForm] = Form.useForm();

  useEffect(() => {
    if (incidentId) {
      loadIncidentDetail();
      loadRootCauseAnalysis();
      loadImpactAssessment();
      loadClassification();
    }
  }, [incidentId]);

  const loadIncidentDetail = async () => {
    try {
      setLoading(true);
      const response = await IncidentAPI.getIncident(incidentId);
      setIncident(response);
    } catch (error) {
      console.error("加载事件详情失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const loadRootCauseAnalysis = async () => {
    try {
      // 模拟API调用
      const mockData: RootCauseAnalysis = {
        id: 1,
        incident_id: incidentId,
        analysis_method: "5-whys",
        root_cause: "数据库连接池配置不当导致连接耗尽",
        contributing_factors: [
          "高并发访问量超出预期",
          "连接池最大连接数设置过低",
          "连接泄漏未及时发现",
          "监控告警阈值设置不合理"
        ],
        evidence: [
          "数据库连接池监控显示连接数达到上限",
          "应用日志显示大量连接超时错误",
          "系统负载监控显示CPU和内存使用正常"
        ],
        preventive_actions: [
          "调整数据库连接池配置参数",
          "实施连接池监控和告警",
          "定期进行性能压测",
          "建立容量规划流程"
        ],
        status: "completed",
        analyst_name: "张工程师",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };
      setRootCauseAnalysis(mockData);
    } catch (error) {
      console.error("加载根因分析失败:", error);
    }
  };

  const loadImpactAssessment = async () => {
    try {
      // 模拟API调用
      const mockData: ImpactAssessment = {
        id: 1,
        incident_id: incidentId,
        business_impact: "high",
        technical_impact: "medium",
        affected_services: ["用户登录服务", "订单处理系统", "支付网关"],
        affected_users_count: 15000,
        financial_impact: 50000,
        reputation_impact: "medium",
        compliance_impact: false,
        assessment_notes: "主要影响用户登录和新订单创建，现有订单处理不受影响",
        assessor_name: "李经理",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };
      setImpactAssessment(mockData);
    } catch (error) {
      console.error("加载影响评估失败:", error);
    }
  };

  const loadClassification = async () => {
    try {
      // 模拟API调用
      const mockData: IncidentClassification = {
        id: 1,
        incident_id: incidentId,
        category: "技术故障",
        subcategory: "数据库问题",
        service_type: "核心业务系统",
        failure_type: "性能问题",
        urgency: "high",
        impact: "high",
        classification_confidence: 95,
        auto_classified: true,
        classifier_name: "AI分类器",
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      };
      setClassification(mockData);
    } catch (error) {
      console.error("加载事件分类失败:", error);
    }
  };

  const handleSaveRootCause = async (values: Partial<RootCauseAnalysis>) => {
    try {
      console.log("保存根因分析:", values);
      // 这里应该调用API保存数据
      setShowRootCauseModal(false);
      loadRootCauseAnalysis();
    } catch (error) {
      console.error("保存根因分析失败:", error);
    }
  };

  const handleSaveImpact = async (values: Partial<ImpactAssessment>) => {
    try {
      console.log("保存影响评估:", values);
      // 这里应该调用API保存数据
      setShowImpactModal(false);
      loadImpactAssessment();
    } catch (error) {
      console.error("保存影响评估失败:", error);
    }
  };

  const handleSaveClassification = async (values: Partial<IncidentClassification>) => {
    try {
      console.log("保存事件分类:", values);
      // 这里应该调用API保存数据
      setShowClassificationModal(false);
      loadClassification();
    } catch (error) {
      console.error("保存事件分类失败:", error);
    }
  };

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      open: "orange",
      "in-progress": "blue",
      resolved: "green",
      closed: "default"
    };
    return colors[status] || "default";
  };

  const getPriorityColor = (priority: string) => {
    const colors: Record<string, string> = {
      low: "green",
      medium: "blue",
      high: "orange",
      critical: "red"
    };
    return colors[priority] || "default";
  };

  const getImpactColor = (impact: string) => {
    const colors: Record<string, string> = {
      low: "green",
      medium: "blue",
      high: "orange",
      critical: "red"
    };
    return colors[impact] || "default";
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <p className="text-gray-600">加载事件详情中...</p>
        </div>
      </div>
    );
  }

  if (!incident) {
    return (
      <div className="text-center py-12">
        <AlertTriangle className="w-16 h-16 text-gray-400 mx-auto mb-4" />
        <h3 className="text-lg font-medium text-gray-900 mb-2">事件不存在</h3>
        <p className="text-gray-600">请检查事件ID是否正确</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="bg-white rounded-lg shadow-sm border p-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center space-x-4">
            <Button
              icon={<ArrowLeftOutlined />}
              onClick={() => window.history.back()}
              className="flex items-center"
            >
              返回列表
            </Button>
            <div>
              <h1 className="text-2xl font-bold text-gray-900">{incident.title}</h1>
              <p className="text-gray-600">事件编号: {incident.incident_number}</p>
            </div>
          </div>
          <Space>
            <Tag color={getStatusColor(incident.status)} className="px-3 py-1">
              {incident.status === "open" && "待处理"}
              {incident.status === "in-progress" && "处理中"}
              {incident.status === "resolved" && "已解决"}
              {incident.status === "closed" && "已关闭"}
            </Tag>
            <Tag color={getPriorityColor(incident.priority)} className="px-3 py-1">
              {incident.priority === "low" && "低优先级"}
              {incident.priority === "medium" && "中优先级"}
              {incident.priority === "high" && "高优先级"}
              {incident.priority === "critical" && "紧急"}
            </Tag>
            <Button type="primary" icon={<EditOutlined />}>
              编辑事件
            </Button>
          </Space>
        </div>
      </div>

      {/* 主要内容区域 */}
      <Tabs activeKey={activeTab} onChange={setActiveTab} className="bg-white rounded-lg shadow-sm">
        <TabPane tab={<span><FileTextOutlined />概览</span>} key="overview">
          <div className="p-6 space-y-6">
            {/* 基本信息 */}
            <Card title="基本信息" className="shadow-sm">
              <Descriptions column={2} bordered>
                <Descriptions.Item label="事件标题">{incident.title}</Descriptions.Item>
                <Descriptions.Item label="事件编号">{incident.incident_number}</Descriptions.Item>
                <Descriptions.Item label="状态">
                  <Tag color={getStatusColor(incident.status)}>
                    {incident.status === "open" && "待处理"}
                    {incident.status === "in-progress" && "处理中"}
                    {incident.status === "resolved" && "已解决"}
                    {incident.status === "closed" && "已关闭"}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="优先级">
                  <Tag color={getPriorityColor(incident.priority)}>
                    {incident.priority === "low" && "低"}
                    {incident.priority === "medium" && "中"}
                    {incident.priority === "high" && "高"}
                    {incident.priority === "critical" && "紧急"}
                  </Tag>
                </Descriptions.Item>
                <Descriptions.Item label="报告人">
                  {incident.reporter?.name || "未知"}
                </Descriptions.Item>
                <Descriptions.Item label="负责人">
                  {incident.assignee?.name || "未分配"}
                </Descriptions.Item>
                <Descriptions.Item label="创建时间">
                  {new Date(incident.created_at).toLocaleString("zh-CN")}
                </Descriptions.Item>
                <Descriptions.Item label="更新时间">
                  {new Date(incident.updated_at).toLocaleString("zh-CN")}
                </Descriptions.Item>
                <Descriptions.Item label="描述" span={2}>
                  {incident.description}
                </Descriptions.Item>
              </Descriptions>
            </Card>

            {/* 快速操作面板 */}
            <Row gutter={[16, 16]}>
              <Col xs={24} sm={8}>
                <Card className="text-center hover:shadow-md transition-shadow cursor-pointer" onClick={() => setShowClassificationModal(true)}>
                  <div className="w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center mx-auto mb-3">
                    <BugOutlined className="text-blue-600 text-xl" />
                  </div>
                  <h4 className="font-medium mb-2">事件分类</h4>
                  <p className="text-gray-600 text-sm">管理事件分类信息</p>
                </Card>
              </Col>
              <Col xs={24} sm={8}>
                <Card className="text-center hover:shadow-md transition-shadow cursor-pointer" onClick={() => setShowRootCauseModal(true)}>
                  <div className="w-12 h-12 bg-green-100 rounded-full flex items-center justify-center mx-auto mb-3">
                    <SafetyOutlined className="text-green-600 text-xl" />
                  </div>
                  <h4 className="font-medium mb-2">根因分析</h4>
                  <p className="text-gray-600 text-sm">分析事件根本原因</p>
                </Card>
              </Col>
              <Col xs={24} sm={8}>
                <Card className="text-center hover:shadow-md transition-shadow cursor-pointer" onClick={() => setShowImpactModal(true)}>
                  <div className="w-12 h-12 bg-orange-100 rounded-full flex items-center justify-center mx-auto mb-3">
                    <ExclamationCircleOutlined className="text-orange-600 text-xl" />
                  </div>
                  <h4 className="font-medium mb-2">影响评估</h4>
                  <p className="text-gray-600 text-sm">评估事件业务影响</p>
                </Card>
              </Col>
            </Row>
          </div>
        </TabPane>

        <TabPane tab={<span><BugOutlined />分类信息</span>} key="classification">
          <div className="p-6">
            {classification ? (
              <Card title="事件分类信息" extra={
                <Button icon={<EditOutlined />} onClick={() => setShowClassificationModal(true)}>
                  编辑分类
                </Button>
              }>
                <Descriptions column={2} bordered>
                  <Descriptions.Item label="主分类">{classification.category}</Descriptions.Item>
                  <Descriptions.Item label="子分类">{classification.subcategory}</Descriptions.Item>
                  <Descriptions.Item label="服务类型">{classification.service_type}</Descriptions.Item>
                  <Descriptions.Item label="故障类型">{classification.failure_type}</Descriptions.Item>
                  <Descriptions.Item label="紧急程度">
                    <Tag color={getImpactColor(classification.urgency)}>
                      {classification.urgency === "low" && "低"}
                      {classification.urgency === "medium" && "中"}
                      {classification.urgency === "high" && "高"}
                      {classification.urgency === "critical" && "紧急"}
                    </Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="影响范围">
                    <Tag color={getImpactColor(classification.impact)}>
                      {classification.impact === "low" && "低"}
                      {classification.impact === "medium" && "中"}
                      {classification.impact === "high" && "高"}
                      {classification.impact === "critical" && "严重"}
                    </Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label="分类置信度">
                    <Progress percent={classification.classification_confidence} size="small" />
                  </Descriptions.Item>
                  <Descriptions.Item label="自动分类">
                    <Badge status={classification.auto_classified ? "success" : "default"} text={classification.auto_classified ? "是" : "否"} />
                  </Descriptions.Item>
                  <Descriptions.Item label="分类人员" span={2}>
                    {classification.classifier_name}
                  </Descriptions.Item>
                </Descriptions>
              </Card>
            ) : (
              <Card>
                <div className="text-center py-12">
                  <BugOutlined className="text-4xl text-gray-400 mb-4" />
                  <h3 className="text-lg font-medium text-gray-900 mb-2">暂无分类信息</h3>
                  <p className="text-gray-600 mb-4">为此事件添加分类信息以便更好地管理</p>
                  <Button type="primary" icon={<BugOutlined />} onClick={() => setShowClassificationModal(true)}>
                    添加分类
                  </Button>
                </div>
              </Card>
            )}
          </div>
        </TabPane>

        <TabPane tab={<span><SafetyOutlined />根因分析</span>} key="rootcause">
          <div className="p-6">
            {rootCauseAnalysis ? (
              <Card title="根因分析报告" extra={
                <Button icon={<EditOutlined />} onClick={() => setShowRootCauseModal(true)}>
                  编辑分析
                </Button>
              }>
                <div className="space-y-6">
                  <div>
                    <h4 className="font-medium mb-2">分析方法</h4>
                    <Tag color="blue">{rootCauseAnalysis.analysis_method}</Tag>
                  </div>
                  
                  <div>
                    <h4 className="font-medium mb-2">根本原因</h4>
                    <Alert message={rootCauseAnalysis.root_cause} type="error" showIcon />
                  </div>
                  
                  <div>
                    <h4 className="font-medium mb-2">促成因素</h4>
                    <ul className="list-disc list-inside space-y-1">
                      {rootCauseAnalysis.contributing_factors.map((factor, index) => (
                        <li key={index} className="text-gray-700">{factor}</li>
                      ))}
                    </ul>
                  </div>
                  
                  <div>
                    <h4 className="font-medium mb-2">证据</h4>
                    <ul className="list-disc list-inside space-y-1">
                      {rootCauseAnalysis.evidence.map((evidence, index) => (
                        <li key={index} className="text-gray-700">{evidence}</li>
                      ))}
                    </ul>
                  </div>
                  
                  <div>
                    <h4 className="font-medium mb-2">预防措施</h4>
                    <ul className="list-disc list-inside space-y-1">
                      {rootCauseAnalysis.preventive_actions.map((action, index) => (
                        <li key={index} className="text-gray-700">{action}</li>
                      ))}
                    </ul>
                  </div>
                  
                  <Divider />
                  
                  <div className="flex justify-between text-sm text-gray-600">
                    <span>分析人员: {rootCauseAnalysis.analyst_name}</span>
                    <span>状态: <Tag color={rootCauseAnalysis.status === "completed" ? "green" : "orange"}>{rootCauseAnalysis.status}</Tag></span>
                  </div>
                </div>
              </Card>
            ) : (
              <Card>
                <div className="text-center py-12">
                  <SafetyOutlined className="text-4xl text-gray-400 mb-4" />
                  <h3 className="text-lg font-medium text-gray-900 mb-2">暂无根因分析</h3>
                  <p className="text-gray-600 mb-4">进行根因分析以找出事件的根本原因</p>
                  <Button type="primary" icon={<SafetyOutlined />} onClick={() => setShowRootCauseModal(true)}>
                    开始分析
                  </Button>
                </div>
              </Card>
            )}
          </div>
        </TabPane>

        <TabPane tab={<span><ExclamationCircleOutlined />影响评估</span>} key="impact">
          <div className="p-6">
            {impactAssessment ? (
              <Card title="影响评估报告" extra={
                <Button icon={<EditOutlined />} onClick={() => setShowImpactModal(true)}>
                  编辑评估
                </Button>
              }>
                <Row gutter={[24, 24]}>
                  <Col xs={24} md={12}>
                    <Card size="small" title="业务影响">
                      <div className="space-y-3">
                        <div className="flex justify-between items-center">
                          <span>业务影响等级:</span>
                          <Tag color={getImpactColor(impactAssessment.business_impact)}>
                            {impactAssessment.business_impact === "low" && "低"}
                            {impactAssessment.business_impact === "medium" && "中"}
                            {impactAssessment.business_impact === "high" && "高"}
                            {impactAssessment.business_impact === "critical" && "严重"}
                          </Tag>
                        </div>
                        <div className="flex justify-between items-center">
                          <span>技术影响等级:</span>
                          <Tag color={getImpactColor(impactAssessment.technical_impact)}>
                            {impactAssessment.technical_impact === "low" && "低"}
                            {impactAssessment.technical_impact === "medium" && "中"}
                            {impactAssessment.technical_impact === "high" && "高"}
                            {impactAssessment.technical_impact === "critical" && "严重"}
                          </Tag>
                        </div>
                        <div className="flex justify-between items-center">
                          <span>声誉影响:</span>
                          <Tag color={getImpactColor(impactAssessment.reputation_impact)}>
                            {impactAssessment.reputation_impact === "low" && "低"}
                            {impactAssessment.reputation_impact === "medium" && "中"}
                            {impactAssessment.reputation_impact === "high" && "高"}
                            {impactAssessment.reputation_impact === "critical" && "严重"}
                          </Tag>
                        </div>
                        <div className="flex justify-between items-center">
                          <span>合规影响:</span>
                          <Badge status={impactAssessment.compliance_impact ? "error" : "success"} text={impactAssessment.compliance_impact ? "有" : "无"} />
                        </div>
                      </div>
                    </Card>
                  </Col>
                  <Col xs={24} md={12}>
                    <Card size="small" title="量化指标">
                      <div className="space-y-3">
                        <div className="flex justify-between items-center">
                          <span>受影响用户:</span>
                          <span className="font-medium">{impactAssessment.affected_users_count.toLocaleString()} 人</span>
                        </div>
                        <div className="flex justify-between items-center">
                          <span>财务损失:</span>
                          <span className="font-medium text-red-600">¥{impactAssessment.financial_impact.toLocaleString()}</span>
                        </div>
                        <div>
                          <span className="block mb-2">受影响服务:</span>
                          <div className="space-y-1">
                            {impactAssessment.affected_services.map((service, index) => (
                              <Tag key={index} color="blue">{service}</Tag>
                            ))}
                          </div>
                        </div>
                      </div>
                    </Card>
                  </Col>
                  <Col xs={24}>
                    <Card size="small" title="评估说明">
                      <p className="text-gray-700">{impactAssessment.assessment_notes}</p>
                      <Divider />
                      <div className="flex justify-between text-sm text-gray-600">
                        <span>评估人员: {impactAssessment.assessor_name}</span>
                        <span>评估时间: {new Date(impactAssessment.created_at!).toLocaleString("zh-CN")}</span>
                      </div>
                    </Card>
                  </Col>
                </Row>
              </Card>
            ) : (
              <Card>
                <div className="text-center py-12">
                  <ExclamationCircleOutlined className="text-4xl text-gray-400 mb-4" />
                  <h3 className="text-lg font-medium text-gray-900 mb-2">暂无影响评估</h3>
                  <p className="text-gray-600 mb-4">评估此事件对业务和技术的影响</p>
                  <Button type="primary" icon={<ExclamationCircleOutlined />} onClick={() => setShowImpactModal(true)}>
                    开始评估
                  </Button>
                </div>
              </Card>
            )}
          </div>
        </TabPane>

        <TabPane tab={<span><MessageSquare />活动记录</span>} key="timeline">
          <div className="p-6">
            <Timeline>
              <Timeline.Item color="blue" dot={<ClockCircleOutlined />}>
                <div className="mb-2">
                  <span className="font-medium">事件创建</span>
                  <span className="text-gray-500 ml-2">{new Date(incident.created_at).toLocaleString("zh-CN")}</span>
                </div>
                <p className="text-gray-600">事件由 {incident.reporter?.name} 创建</p>
              </Timeline.Item>
              
              {classification && (
                <Timeline.Item color="green" dot={<BugOutlined />}>
                  <div className="mb-2">
                    <span className="font-medium">事件分类</span>
                    <span className="text-gray-500 ml-2">{new Date(classification.created_at!).toLocaleString("zh-CN")}</span>
                  </div>
                  <p className="text-gray-600">事件被分类为: {classification.category} - {classification.subcategory}</p>
                </Timeline.Item>
              )}
              
              {impactAssessment && (
                <Timeline.Item color="orange" dot={<ExclamationCircleOutlined />}>
                  <div className="mb-2">
                    <span className="font-medium">影响评估</span>
                    <span className="text-gray-500 ml-2">{new Date(impactAssessment.created_at!).toLocaleString("zh-CN")}</span>
                  </div>
                  <p className="text-gray-600">完成影响评估，业务影响等级: {impactAssessment.business_impact}</p>
                </Timeline.Item>
              )}
              
              {rootCauseAnalysis && (
                <Timeline.Item color="red" dot={<SafetyOutlined />}>
                  <div className="mb-2">
                    <span className="font-medium">根因分析</span>
                    <span className="text-gray-500 ml-2">{new Date(rootCauseAnalysis.created_at!).toLocaleString("zh-CN")}</span>
                  </div>
                  <p className="text-gray-600">完成根因分析，状态: {rootCauseAnalysis.status}</p>
                </Timeline.Item>
              )}
              
              <Timeline.Item color="gray" dot={<CheckCircleOutlined />}>
                <div className="mb-2">
                  <span className="font-medium">最后更新</span>
                  <span className="text-gray-500 ml-2">{new Date(incident.updated_at).toLocaleString("zh-CN")}</span>
                </div>
                <p className="text-gray-600">事件信息已更新</p>
              </Timeline.Item>
            </Timeline>
          </div>
        </TabPane>
      </Tabs>

      {/* 事件分类模态框 */}
      <Modal
        title="事件分类"
        open={showClassificationModal}
        onCancel={() => setShowClassificationModal(false)}
        footer={null}
        width={800}
      >
        <Form
          form={classificationForm}
          layout="vertical"
          onFinish={handleSaveClassification}
          initialValues={classification || {}}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="主分类" name="category" rules={[{ required: true, message: "请选择主分类" }]}>
                <Select placeholder="选择主分类">
                  <Option value="技术故障">技术故障</Option>
                  <Option value="服务请求">服务请求</Option>
                  <Option value="安全事件">安全事件</Option>
                  <Option value="变更相关">变更相关</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="子分类" name="subcategory" rules={[{ required: true, message: "请选择子分类" }]}>
                <Select placeholder="选择子分类">
                  <Option value="数据库问题">数据库问题</Option>
                  <Option value="网络问题">网络问题</Option>
                  <Option value="应用程序问题">应用程序问题</Option>
                  <Option value="硬件问题">硬件问题</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="服务类型" name="service_type" rules={[{ required: true, message: "请选择服务类型" }]}>
                <Select placeholder="选择服务类型">
                  <Option value="核心业务系统">核心业务系统</Option>
                  <Option value="支撑系统">支撑系统</Option>
                  <Option value="基础设施">基础设施</Option>
                  <Option value="第三方服务">第三方服务</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="故障类型" name="failure_type" rules={[{ required: true, message: "请选择故障类型" }]}>
                <Select placeholder="选择故障类型">
                  <Option value="性能问题">性能问题</Option>
                  <Option value="功能异常">功能异常</Option>
                  <Option value="服务中断">服务中断</Option>
                  <Option value="数据问题">数据问题</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="紧急程度" name="urgency" rules={[{ required: true, message: "请选择紧急程度" }]}>
                <Select placeholder="选择紧急程度">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">紧急</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="影响范围" name="impact" rules={[{ required: true, message: "请选择影响范围" }]}>
                <Select placeholder="选择影响范围">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">严重</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                保存分类
              </Button>
              <Button onClick={() => setShowClassificationModal(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 根因分析模态框 */}
      <Modal
        title="根因分析"
        open={showRootCauseModal}
        onCancel={() => setShowRootCauseModal(false)}
        footer={null}
        width={1000}
      >
        <Form
          form={rootCauseForm}
          layout="vertical"
          onFinish={handleSaveRootCause}
          initialValues={rootCauseAnalysis || {}}
        >
          <Form.Item label="分析方法" name="analysis_method" rules={[{ required: true, message: "请选择分析方法" }]}>
            <Select placeholder="选择分析方法">
              <Option value="5-whys">5个为什么</Option>
              <Option value="fishbone">鱼骨图分析</Option>
              <Option value="timeline">时间线分析</Option>
              <Option value="fault-tree">故障树分析</Option>
            </Select>
          </Form.Item>
          
          <Form.Item label="根本原因" name="root_cause" rules={[{ required: true, message: "请输入根本原因" }]}>
            <TextArea rows={3} placeholder="描述事件的根本原因" />
          </Form.Item>
          
          <Form.Item label="促成因素" name="contributing_factors">
            <Select mode="tags" placeholder="添加促成因素（按回车添加）" />
          </Form.Item>
          
          <Form.Item label="证据" name="evidence">
            <Select mode="tags" placeholder="添加支持证据（按回车添加）" />
          </Form.Item>
          
          <Form.Item label="预防措施" name="preventive_actions">
            <Select mode="tags" placeholder="添加预防措施（按回车添加）" />
          </Form.Item>
          
          <Form.Item label="分析状态" name="status" rules={[{ required: true, message: "请选择状态" }]}>
            <Select placeholder="选择分析状态">
              <Option value="draft">草稿</Option>
              <Option value="in-progress">进行中</Option>
              <Option value="completed">已完成</Option>
              <Option value="approved">已批准</Option>
            </Select>
          </Form.Item>
          
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                保存分析
              </Button>
              <Button onClick={() => setShowRootCauseModal(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 影响评估模态框 */}
      <Modal
        title="影响评估"
        open={showImpactModal}
        onCancel={() => setShowImpactModal(false)}
        footer={null}
        width={1000}
      >
        <Form
          form={impactForm}
          layout="vertical"
          onFinish={handleSaveImpact}
          initialValues={impactAssessment || {}}
        >
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="业务影响等级" name="business_impact" rules={[{ required: true, message: "请选择业务影响等级" }]}>
                <Select placeholder="选择业务影响等级">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">严重</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="技术影响等级" name="technical_impact" rules={[{ required: true, message: "请选择技术影响等级" }]}>
                <Select placeholder="选择技术影响等级">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">严重</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="声誉影响" name="reputation_impact" rules={[{ required: true, message: "请选择声誉影响" }]}>
                <Select placeholder="选择声誉影响">
                  <Option value="low">低</Option>
                  <Option value="medium">中</Option>
                  <Option value="high">高</Option>
                  <Option value="critical">严重</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="受影响用户数" name="affected_users_count" rules={[{ required: true, message: "请输入受影响用户数" }]}>
                <Input type="number" placeholder="输入受影响用户数" />
              </Form.Item>
            </Col>
          </Row>
          
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item label="财务损失 (元)" name="financial_impact" rules={[{ required: true, message: "请输入财务损失" }]}>
                <Input type="number" placeholder="输入财务损失金额" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item label="合规影响" name="compliance_impact" valuePropName="checked">
                <Select placeholder="是否有合规影响">
                  <Option value={true}>有</Option>
                  <Option value={false}>无</Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>
          
          <Form.Item label="受影响服务" name="affected_services">
            <Select mode="tags" placeholder="添加受影响的服务（按回车添加）" />
          </Form.Item>
          
          <Form.Item label="评估说明" name="assessment_notes">
            <TextArea rows={4} placeholder="详细描述影响评估结果" />
          </Form.Item>
          
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                保存评估
              </Button>
              <Button onClick={() => setShowImpactModal(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}