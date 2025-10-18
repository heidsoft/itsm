"use client";

import React, { useState, useEffect, useCallback } from "react";
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  DatePicker,
  Space,
  Tag,
  Modal,
  Form,
  message,
  Tabs,
  Statistic,
  Row,
  Col,
  Timeline,
  Alert,
  Tooltip,
  Badge,
  Drawer,
  List,
  Avatar,
  Typography,
  Divider,
  Progress,
  Empty,
} from "antd";
import {
  PlusOutlined,
  SearchOutlined,
  FilterOutlined,
  ReloadOutlined,
  EyeOutlined,
  EditOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  InfoCircleOutlined,
  BellOutlined,
  BarChartOutlined,
  SettingOutlined,
  LinkOutlined,
  UserOutlined,
  CalendarOutlined,
  EnvironmentOutlined,
} from "@ant-design/icons";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import "dayjs/locale/zh-cn";

dayjs.extend(relativeTime);
dayjs.locale("zh-cn");

const { RangePicker } = DatePicker;
const { Option } = Select;
const { TextArea } = Input;
const { Title, Text } = Typography;
const { TabPane } = Tabs;

// 事件接口定义
interface Incident {
  id: number;
  title: string;
  description: string;
  status: string;
  priority: string;
  severity: string;
  incident_number: string;
  reporter_id: number;
  assignee_id?: number;
  category?: string;
  subcategory?: string;
  impact_analysis?: any;
  root_cause?: any;
  resolution_steps?: any[];
  detected_at: string;
  resolved_at?: string;
  closed_at?: string;
  escalated_at?: string;
  escalation_level: number;
  is_automated: boolean;
  source: string;
  metadata?: any;
  tenant_id: number;
  created_at: string;
  updated_at: string;
}

interface IncidentEvent {
  id: number;
  incident_id: number;
  event_type: string;
  event_name: string;
  description: string;
  status: string;
  severity: string;
  data?: any;
  occurred_at: string;
  user_id?: number;
  source: string;
  metadata?: any;
}

interface IncidentAlert {
  id: number;
  incident_id: number;
  alert_type: string;
  alert_name: string;
  message: string;
  severity: string;
  status: string;
  channels: string[];
  recipients: string[];
  triggered_at: string;
  acknowledged_at?: string;
  resolved_at?: string;
  acknowledged_by?: number;
}

interface IncidentMetric {
  id: number;
  incident_id: number;
  metric_type: string;
  metric_name: string;
  metric_value: number;
  unit?: string;
  measured_at: string;
  tags?: Record<string, string>;
  metadata?: any;
}

// 事件管理主组件
export const IncidentManagement: React.FC = () => {
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState<any>({});
  const [selectedIncident, setSelectedIncident] = useState<Incident | null>(null);
  const [detailVisible, setDetailVisible] = useState(false);
  const [createVisible, setCreateVisible] = useState(false);
  const [monitoringVisible, setMonitoringVisible] = useState(false);

  // 获取事件列表
  const fetchIncidents = useCallback(async () => {
    setLoading(true);
    try {
      const params = {
        page: currentPage,
        size: pageSize,
        ...filters,
      };

      const response = await fetch("/api/v1/incidents", {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${localStorage.getItem("access_token")}`,
          "X-Tenant-ID": localStorage.getItem("tenant_id") || "",
        },
      });

      if (!response.ok) {
        throw new Error("获取事件列表失败");
      }

      const data = await response.json();
      setIncidents(data.data || []);
      setTotal(data.total || 0);
    } catch (error) {
      message.error("获取事件列表失败");
      console.error("获取事件列表失败:", error);
    } finally {
      setLoading(false);
    }
  }, [currentPage, pageSize, filters]);

  useEffect(() => {
    fetchIncidents();
  }, [fetchIncidents]);

  // 处理筛选
  const handleFilterChange = (key: string, value: any) => {
    setFilters((prev: any) => ({
      ...prev,
      [key]: value,
    }));
    setCurrentPage(1);
  };

  // 处理分页
  const handleTableChange = (pagination: any) => {
    setCurrentPage(pagination.current);
    setPageSize(pagination.pageSize);
  };

  // 获取状态颜色
  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      new: "blue",
      in_progress: "orange",
      resolved: "green",
      closed: "gray",
    };
    return colors[status] || "default";
  };

  // 获取优先级颜色
  const getPriorityColor = (priority: string) => {
    const colors: Record<string, string> = {
      low: "green",
      medium: "orange",
      high: "red",
      urgent: "purple",
    };
    return colors[priority] || "default";
  };

  // 获取严重程度颜色
  const getSeverityColor = (severity: string) => {
    const colors: Record<string, string> = {
      low: "green",
      medium: "orange",
      high: "red",
      critical: "purple",
    };
    return colors[severity] || "default";
  };

  // 获取状态图标
  const getStatusIcon = (status: string) => {
    const icons: Record<string, React.ReactNode> = {
      new: <InfoCircleOutlined />,
      in_progress: <ClockCircleOutlined />,
      resolved: <CheckCircleOutlined />,
      closed: <CloseCircleOutlined />,
    };
    return icons[status] || <InfoCircleOutlined />;
  };

  // 表格列定义
  const columns = [
    {
      title: "事件编号",
      dataIndex: "incident_number",
      key: "incident_number",
      width: 120,
      render: (text: string) => (
        <Text code style={{ fontSize: "12px" }}>
          {text}
        </Text>
      ),
    },
    {
      title: "标题",
      dataIndex: "title",
      key: "title",
      ellipsis: true,
      render: (text: string, record: Incident) => (
        <Button
          type="link"
          onClick={() => {
            setSelectedIncident(record);
            setDetailVisible(true);
          }}
          style={{ padding: 0, height: "auto" }}
        >
          {text}
        </Button>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: string) => (
        <Tag color={getStatusColor(status)} icon={getStatusIcon(status)}>
          {status}
        </Tag>
      ),
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 100,
      render: (priority: string) => (
        <Tag color={getPriorityColor(priority)}>{priority}</Tag>
      ),
    },
    {
      title: "严重程度",
      dataIndex: "severity",
      key: "severity",
      width: 100,
      render: (severity: string) => (
        <Tag color={getSeverityColor(severity)}>{severity}</Tag>
      ),
    },
    {
      title: "分类",
      dataIndex: "category",
      key: "category",
      width: 120,
      render: (category: string) => category || "-",
    },
    {
      title: "来源",
      dataIndex: "source",
      key: "source",
      width: 100,
      render: (source: string) => (
        <Tag color={source === "monitoring" ? "blue" : "green"}>
          {source}
        </Tag>
      ),
    },
    {
      title: "检测时间",
      dataIndex: "detected_at",
      key: "detected_at",
      width: 150,
      render: (time: string) => dayjs(time).format("YYYY-MM-DD HH:mm"),
    },
    {
      title: "升级级别",
      dataIndex: "escalation_level",
      key: "escalation_level",
      width: 100,
      render: (level: number) => (
        level > 0 ? (
          <Badge count={level} style={{ backgroundColor: "#f50" }} />
        ) : "-"
      ),
    },
    {
      title: "操作",
      key: "action",
      width: 120,
      render: (_, record: Incident) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              type="text"
              icon={<EyeOutlined />}
              onClick={() => {
                setSelectedIncident(record);
                setDetailVisible(true);
              }}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined />}
              onClick={() => {
                setSelectedIncident(record);
                setCreateVisible(true);
              }}
            />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div className="incident-management">
      {/* 页面标题和操作 */}
      <div className="mb-6">
        <div className="flex justify-between items-center">
          <div>
            <Title level={2} className="mb-2">
              事件管理
            </Title>
            <Text type="secondary">
              管理和监控IT事件，确保系统稳定运行
            </Text>
          </div>
          <Space>
            <Button
              icon={<BarChartOutlined />}
              onClick={() => setMonitoringVisible(true)}
            >
              监控面板
            </Button>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setCreateVisible(true)}
            >
              创建事件
            </Button>
          </Space>
        </div>
      </div>

      {/* 统计卡片 */}
      <Row gutter={16} className="mb-6">
        <Col span={6}>
          <Card>
            <Statistic
              title="总事件数"
              value={total}
              prefix={<InfoCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="进行中"
              value={incidents.filter(i => i.status === "in_progress").length}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: "#1890ff" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已解决"
              value={incidents.filter(i => i.status === "resolved").length}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="严重事件"
              value={incidents.filter(i => i.severity === "critical").length}
              prefix={<ExclamationCircleOutlined />}
              valueStyle={{ color: "#f5222d" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 筛选器 */}
      <Card className="mb-6">
        <Row gutter={16} align="middle">
          <Col span={6}>
            <Input
              placeholder="搜索事件标题或编号"
              prefix={<SearchOutlined />}
              onChange={(e) => handleFilterChange("search", e.target.value)}
            />
          </Col>
          <Col span={4}>
            <Select
              placeholder="状态"
              allowClear
              onChange={(value) => handleFilterChange("status", value)}
            >
              <Option value="new">新建</Option>
              <Option value="in_progress">进行中</Option>
              <Option value="resolved">已解决</Option>
              <Option value="closed">已关闭</Option>
            </Select>
          </Col>
          <Col span={4}>
            <Select
              placeholder="优先级"
              allowClear
              onChange={(value) => handleFilterChange("priority", value)}
            >
              <Option value="low">低</Option>
              <Option value="medium">中</Option>
              <Option value="high">高</Option>
              <Option value="urgent">紧急</Option>
            </Select>
          </Col>
          <Col span={4}>
            <Select
              placeholder="严重程度"
              allowClear
              onChange={(value) => handleFilterChange("severity", value)}
            >
              <Option value="low">低</Option>
              <Option value="medium">中</Option>
              <Option value="high">高</Option>
              <Option value="critical">严重</Option>
            </Select>
          </Col>
          <Col span={4}>
            <RangePicker
              placeholder={["开始时间", "结束时间"]}
              onChange={(dates) => {
                if (dates) {
                  handleFilterChange("date_range", {
                    start: dates[0]?.format("YYYY-MM-DD"),
                    end: dates[1]?.format("YYYY-MM-DD"),
                  });
                } else {
                  handleFilterChange("date_range", null);
                }
              }}
            />
          </Col>
          <Col span={2}>
            <Button
              icon={<ReloadOutlined />}
              onClick={fetchIncidents}
              loading={loading}
            />
          </Col>
        </Row>
      </Card>

      {/* 事件列表 */}
      <Card>
        <Table
          columns={columns}
          dataSource={incidents}
          loading={loading}
          rowKey="id"
          pagination={{
            current: currentPage,
            pageSize: pageSize,
            total: total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          onChange={handleTableChange}
          scroll={{ x: 1200 }}
        />
      </Card>

      {/* 事件详情抽屉 */}
      <IncidentDetailDrawer
        visible={detailVisible}
        incident={selectedIncident}
        onClose={() => {
          setDetailVisible(false);
          setSelectedIncident(null);
        }}
        onRefresh={fetchIncidents}
      />

      {/* 创建/编辑事件模态框 */}
      <IncidentFormModal
        visible={createVisible}
        incident={selectedIncident}
        onClose={() => {
          setCreateVisible(false);
          setSelectedIncident(null);
        }}
        onSuccess={() => {
          setCreateVisible(false);
          setSelectedIncident(null);
          fetchIncidents();
        }}
      />

      {/* 监控面板 */}
      <IncidentMonitoringPanel
        visible={monitoringVisible}
        onClose={() => setMonitoringVisible(false)}
      />
    </div>
  );
};

// 事件详情抽屉组件
const IncidentDetailDrawer: React.FC<{
  visible: boolean;
  incident: Incident | null;
  onClose: () => void;
  onRefresh: () => void;
}> = ({ visible, incident, onClose, onRefresh }) => {
  const [events, setEvents] = useState<IncidentEvent[]>([]);
  const [alerts, setAlerts] = useState<IncidentAlert[]>([]);
  const [metrics, setMetrics] = useState<IncidentMetric[]>([]);
  const [loading, setLoading] = useState(false);

  // 获取事件详情数据
  const fetchIncidentDetails = useCallback(async () => {
    if (!incident) return;

    setLoading(true);
    try {
      const [eventsRes, alertsRes, metricsRes] = await Promise.all([
        fetch(`/api/v1/incidents/${incident.id}/events`),
        fetch(`/api/v1/incidents/${incident.id}/alerts`),
        fetch(`/api/v1/incidents/${incident.id}/metrics`),
      ]);

      if (eventsRes.ok) {
        const eventsData = await eventsRes.json();
        setEvents(eventsData.data || []);
      }

      if (alertsRes.ok) {
        const alertsData = await alertsRes.json();
        setAlerts(alertsData.data || []);
      }

      if (metricsRes.ok) {
        const metricsData = await metricsRes.json();
        setMetrics(metricsData.data || []);
      }
    } catch (error) {
      console.error("获取事件详情失败:", error);
    } finally {
      setLoading(false);
    }
  }, [incident]);

  useEffect(() => {
    if (visible && incident) {
      fetchIncidentDetails();
    }
  }, [visible, incident, fetchIncidentDetails]);

  if (!incident) return null;

  return (
    <Drawer
      title={`事件详情 - ${incident.incident_number}`}
      width={800}
      open={visible}
      onClose={onClose}
      extra={
        <Space>
          <Button icon={<EditOutlined />}>编辑</Button>
          <Button icon={<ReloadOutlined />} onClick={fetchIncidentDetails}>
            刷新
          </Button>
        </Space>
      }
    >
      <Tabs defaultActiveKey="overview">
        <TabPane tab="概览" key="overview">
          <IncidentOverview incident={incident} />
        </TabPane>
        <TabPane tab="活动记录" key="events">
          <IncidentEvents events={events} loading={loading} />
        </TabPane>
        <TabPane tab="告警" key="alerts">
          <IncidentAlerts alerts={alerts} loading={loading} />
        </TabPane>
        <TabPane tab="指标" key="metrics">
          <IncidentMetrics metrics={metrics} loading={loading} />
        </TabPane>
        <TabPane tab="影响分析" key="impact">
          <IncidentImpactAnalysis incident={incident} />
        </TabPane>
      </Tabs>
    </Drawer>
  );
};

// 事件概览组件
const IncidentOverview: React.FC<{ incident: Incident }> = ({ incident }) => {
  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      new: "blue",
      in_progress: "orange",
      resolved: "green",
      closed: "gray",
    };
    return colors[status] || "default";
  };

  const getPriorityColor = (priority: string) => {
    const colors: Record<string, string> = {
      low: "green",
      medium: "orange",
      high: "red",
      urgent: "purple",
    };
    return colors[priority] || "default";
  };

  const getSeverityColor = (severity: string) => {
    const colors: Record<string, string> = {
      low: "green",
      medium: "orange",
      high: "red",
      critical: "purple",
    };
    return colors[severity] || "default";
  };

  return (
    <div className="space-y-6">
      {/* 基本信息 */}
      <Card title="基本信息">
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <div className="mb-4">
              <Text strong>事件标题</Text>
              <div className="mt-1">
                <Text>{incident.title}</Text>
              </div>
            </div>
          </Col>
          <Col span={12}>
            <div className="mb-4">
              <Text strong>事件编号</Text>
              <div className="mt-1">
                <Text code>{incident.incident_number}</Text>
              </div>
            </div>
          </Col>
          <Col span={12}>
            <div className="mb-4">
              <Text strong>状态</Text>
              <div className="mt-1">
                <Tag color={getStatusColor(incident.status)}>
                  {incident.status}
                </Tag>
              </div>
            </div>
          </Col>
          <Col span={12}>
            <div className="mb-4">
              <Text strong>优先级</Text>
              <div className="mt-1">
                <Tag color={getPriorityColor(incident.priority)}>
                  {incident.priority}
                </Tag>
              </div>
            </div>
          </Col>
          <Col span={12}>
            <div className="mb-4">
              <Text strong>严重程度</Text>
              <div className="mt-1">
                <Tag color={getSeverityColor(incident.severity)}>
                  {incident.severity}
                </Tag>
              </div>
            </div>
          </Col>
          <Col span={12}>
            <div className="mb-4">
              <Text strong>来源</Text>
              <div className="mt-1">
                <Tag color={incident.source === "monitoring" ? "blue" : "green"}>
                  {incident.source}
                </Tag>
              </div>
            </div>
          </Col>
          <Col span={24}>
            <div className="mb-4">
              <Text strong>描述</Text>
              <div className="mt-1">
                <Text>{incident.description || "无描述"}</Text>
              </div>
            </div>
          </Col>
        </Row>
      </Card>

      {/* 时间信息 */}
      <Card title="时间信息">
        <Row gutter={[16, 16]}>
          <Col span={12}>
            <div className="mb-4">
              <Text strong>检测时间</Text>
              <div className="mt-1">
                <Text>{dayjs(incident.detected_at).format("YYYY-MM-DD HH:mm:ss")}</Text>
              </div>
            </div>
          </Col>
          <Col span={12}>
            <div className="mb-4">
              <Text strong>创建时间</Text>
              <div className="mt-1">
                <Text>{dayjs(incident.created_at).format("YYYY-MM-DD HH:mm:ss")}</Text>
              </div>
            </div>
          </Col>
          {incident.resolved_at && (
            <Col span={12}>
              <div className="mb-4">
                <Text strong>解决时间</Text>
                <div className="mt-1">
                  <Text>{dayjs(incident.resolved_at).format("YYYY-MM-DD HH:mm:ss")}</Text>
                </div>
              </div>
            </Col>
          )}
          {incident.closed_at && (
            <Col span={12}>
              <div className="mb-4">
                <Text strong>关闭时间</Text>
                <div className="mt-1">
                  <Text>{dayjs(incident.closed_at).format("YYYY-MM-DD HH:mm:ss")}</Text>
                </div>
              </div>
            </Col>
          )}
        </Row>
      </Card>

      {/* 升级信息 */}
      {incident.escalation_level > 0 && (
        <Card title="升级信息">
          <Row gutter={[16, 16]}>
            <Col span={12}>
              <div className="mb-4">
                <Text strong>升级级别</Text>
                <div className="mt-1">
                  <Badge count={incident.escalation_level} style={{ backgroundColor: "#f50" }} />
                </div>
              </div>
            </Col>
            {incident.escalated_at && (
              <Col span={12}>
                <div className="mb-4">
                  <Text strong>升级时间</Text>
                  <div className="mt-1">
                    <Text>{dayjs(incident.escalated_at).format("YYYY-MM-DD HH:mm:ss")}</Text>
                  </div>
                </div>
              </Col>
            )}
          </Row>
        </Card>
      )}
    </div>
  );
};

// 事件活动记录组件
const IncidentEvents: React.FC<{ events: IncidentEvent[]; loading: boolean }> = ({
  events,
  loading,
}) => {
  const getEventIcon = (eventType: string) => {
    const icons: Record<string, React.ReactNode> = {
      creation: <PlusOutlined />,
      update: <EditOutlined />,
      escalation: <ExclamationCircleOutlined />,
      assignment: <UserOutlined />,
      status_change: <ClockCircleOutlined />,
    };
    return icons[eventType] || <InfoCircleOutlined />;
  };

  const getSeverityColor = (severity: string) => {
    const colors: Record<string, string> = {
      low: "green",
      medium: "orange",
      high: "red",
      critical: "purple",
    };
    return colors[severity] || "default";
  };

  return (
    <div>
      {loading ? (
        <div className="text-center py-8">
          <Text>加载中...</Text>
        </div>
      ) : events.length > 0 ? (
        <Timeline>
          {events.map((event) => (
            <Timeline.Item
              key={event.id}
              dot={getEventIcon(event.event_type)}
              color={getSeverityColor(event.severity)}
            >
              <div className="mb-2">
                <Text strong>{event.event_name}</Text>
                <Tag color={getSeverityColor(event.severity)} className="ml-2">
                  {event.severity}
                </Tag>
              </div>
              <div className="mb-1">
                <Text>{event.description}</Text>
              </div>
              <div className="text-gray-500 text-sm">
                <Text type="secondary">
                  {dayjs(event.occurred_at).format("YYYY-MM-DD HH:mm:ss")}
                </Text>
                {event.source && (
                  <Text type="secondary" className="ml-2">
                    来源: {event.source}
                  </Text>
                )}
              </div>
            </Timeline.Item>
          ))}
        </Timeline>
      ) : (
        <Empty description="暂无活动记录" />
      )}
    </div>
  );
};

// 事件告警组件
const IncidentAlerts: React.FC<{ alerts: IncidentAlert[]; loading: boolean }> = ({
  alerts,
  loading,
}) => {
  const getAlertIcon = (alertType: string) => {
    const icons: Record<string, React.ReactNode> = {
      escalation: <ExclamationCircleOutlined />,
      notification: <BellOutlined />,
      warning: <WarningOutlined />,
    };
    return icons[alertType] || <BellOutlined />;
  };

  const getSeverityColor = (severity: string) => {
    const colors: Record<string, string> = {
      low: "green",
      medium: "orange",
      high: "red",
      critical: "purple",
    };
    return colors[severity] || "default";
  };

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      active: "red",
      acknowledged: "orange",
      resolved: "green",
    };
    return colors[status] || "default";
  };

  return (
    <div>
      {loading ? (
        <div className="text-center py-8">
          <Text>加载中...</Text>
        </div>
      ) : alerts.length > 0 ? (
        <List
          dataSource={alerts}
          renderItem={(alert) => (
            <List.Item>
              <List.Item.Meta
                avatar={
                  <Avatar
                    icon={getAlertIcon(alert.alert_type)}
                    style={{ backgroundColor: getSeverityColor(alert.severity) }}
                  />
                }
                title={
                  <div className="flex items-center justify-between">
                    <Text strong>{alert.alert_name}</Text>
                    <Space>
                      <Tag color={getSeverityColor(alert.severity)}>
                        {alert.severity}
                      </Tag>
                      <Tag color={getStatusColor(alert.status)}>
                        {alert.status}
                      </Tag>
                    </Space>
                  </div>
                }
                description={
                  <div>
                    <div className="mb-2">
                      <Text>{alert.message}</Text>
                    </div>
                    <div className="text-gray-500 text-sm">
                      <Text type="secondary">
                        触发时间: {dayjs(alert.triggered_at).format("YYYY-MM-DD HH:mm:ss")}
                      </Text>
                      {alert.channels && alert.channels.length > 0 && (
                        <Text type="secondary" className="ml-4">
                          渠道: {alert.channels.join(", ")}
                        </Text>
                      )}
                    </div>
                  </div>
                }
              />
            </List.Item>
          )}
        />
      ) : (
        <Empty description="暂无告警记录" />
      )}
    </div>
  );
};

// 事件指标组件
const IncidentMetrics: React.FC<{ metrics: IncidentMetric[]; loading: boolean }> = ({
  metrics,
  loading,
}) => {
  const getMetricIcon = (metricType: string) => {
    const icons: Record<string, React.ReactNode> = {
      response_time: <ClockCircleOutlined />,
      resolution_time: <CheckCircleOutlined />,
      cpu_usage: <BarChartOutlined />,
      memory_usage: <BarChartOutlined />,
    };
    return icons[metricType] || <BarChartOutlined />;
  };

  return (
    <div>
      {loading ? (
        <div className="text-center py-8">
          <Text>加载中...</Text>
        </div>
      ) : metrics.length > 0 ? (
        <Row gutter={[16, 16]}>
          {metrics.map((metric) => (
            <Col span={8} key={metric.id}>
              <Card size="small">
                <div className="text-center">
                  <div className="mb-2">
                    {getMetricIcon(metric.metric_type)}
                  </div>
                  <div className="mb-1">
                    <Text strong>{metric.metric_name}</Text>
                  </div>
                  <div className="text-2xl font-bold mb-1">
                    {metric.metric_value}
                    {metric.unit && (
                      <Text type="secondary" className="text-sm ml-1">
                        {metric.unit}
                      </Text>
                    )}
                  </div>
                  <div className="text-gray-500 text-sm">
                    <Text type="secondary">
                      {dayjs(metric.measured_at).format("YYYY-MM-DD HH:mm")}
                    </Text>
                  </div>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      ) : (
        <Empty description="暂无指标数据" />
      )}
    </div>
  );
};

// 事件影响分析组件
const IncidentImpactAnalysis: React.FC<{ incident: Incident }> = ({ incident }) => {
  const impactAnalysis = incident.impact_analysis;

  if (!impactAnalysis) {
    return (
      <Empty description="暂无影响分析数据" />
    );
  }

  return (
    <div className="space-y-6">
      {/* 时间影响 */}
      {impactAnalysis.time_impact && (
        <Card title="时间影响">
          <Row gutter={[16, 16]}>
            <Col span={12}>
              <Statistic
                title="创建后时长"
                value={impactAnalysis.time_impact.hours_since_creation}
                suffix="小时"
                precision={1}
              />
            </Col>
            <Col span={12}>
              <Statistic
                title="是否超时"
                value={impactAnalysis.time_impact.is_overdue ? "是" : "否"}
                valueStyle={{
                  color: impactAnalysis.time_impact.is_overdue ? "#f5222d" : "#52c41a",
                }}
              />
            </Col>
          </Row>
        </Card>
      )}

      {/* 业务影响 */}
      {impactAnalysis.business_impact && (
        <Card title="业务影响">
          <Row gutter={[16, 16]}>
            <Col span={8}>
              <Statistic
                title="受影响用户"
                value={impactAnalysis.business_impact.affected_users}
                suffix="人"
              />
            </Col>
            <Col span={8}>
              <Statistic
                title="收入影响"
                value={impactAnalysis.business_impact.revenue_impact}
                prefix="¥"
              />
            </Col>
            <Col span={8}>
              <Statistic
                title="服务可用性"
                value={impactAnalysis.business_impact.service_availability}
                suffix="%"
                precision={1}
              />
            </Col>
          </Row>
        </Card>
      )}

      {/* 指标影响 */}
      {impactAnalysis.metrics && (
        <Card title="指标影响">
          <Row gutter={[16, 16]}>
            <Col span={6}>
              <Statistic
                title="指标总数"
                value={impactAnalysis.metrics.total_count}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="平均值"
                value={impactAnalysis.metrics.average_value}
                precision={2}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="最大值"
                value={impactAnalysis.metrics.max_value}
                precision={2}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="最小值"
                value={impactAnalysis.metrics.min_value}
                precision={2}
              />
            </Col>
          </Row>
        </Card>
      )}
    </div>
  );
};

// 事件表单模态框组件
const IncidentFormModal: React.FC<{
  visible: boolean;
  incident?: Incident | null;
  onClose: () => void;
  onSuccess: () => void;
}> = ({ visible, incident, onClose, onSuccess }) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (visible && incident) {
      form.setFieldsValue({
        title: incident.title,
        description: incident.description,
        priority: incident.priority,
        severity: incident.severity,
        category: incident.category,
        subcategory: incident.subcategory,
      });
    } else if (visible) {
      form.resetFields();
    }
  }, [visible, incident, form]);

  const handleSubmit = async (values: any) => {
    setLoading(true);
    try {
      const url = incident
        ? `/api/v1/incidents/${incident.id}`
        : "/api/v1/incidents";
      const method = incident ? "PUT" : "POST";

      const response = await fetch(url, {
        method,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${localStorage.getItem("access_token")}`,
          "X-Tenant-ID": localStorage.getItem("tenant_id") || "",
        },
        body: JSON.stringify(values),
      });

      if (!response.ok) {
        throw new Error(incident ? "更新事件失败" : "创建事件失败");
      }

      message.success(incident ? "更新事件成功" : "创建事件成功");
      onSuccess();
    } catch (error) {
      message.error(error instanceof Error ? error.message : "操作失败");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title={incident ? "编辑事件" : "创建事件"}
      open={visible}
      onCancel={onClose}
      onOk={() => form.submit()}
      confirmLoading={loading}
      width={600}
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
      >
        <Form.Item
          name="title"
          label="事件标题"
          rules={[{ required: true, message: "请输入事件标题" }]}
        >
          <Input placeholder="请输入事件标题" />
        </Form.Item>

        <Form.Item
          name="description"
          label="事件描述"
        >
          <TextArea
            rows={4}
            placeholder="请输入事件描述"
          />
        </Form.Item>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item
              name="priority"
              label="优先级"
              rules={[{ required: true, message: "请选择优先级" }]}
            >
              <Select placeholder="请选择优先级">
                <Option value="low">低</Option>
                <Option value="medium">中</Option>
                <Option value="high">高</Option>
                <Option value="urgent">紧急</Option>
              </Select>
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="severity"
              label="严重程度"
              rules={[{ required: true, message: "请选择严重程度" }]}
            >
              <Select placeholder="请选择严重程度">
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
            <Form.Item
              name="category"
              label="分类"
            >
              <Select placeholder="请选择分类">
                <Option value="performance">性能</Option>
                <Option value="connectivity">连接</Option>
                <Option value="security">安全</Option>
                <Option value="storage">存储</Option>
                <Option value="network">网络</Option>
              </Select>
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="subcategory"
              label="子分类"
            >
              <Input placeholder="请输入子分类" />
            </Form.Item>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
};

// 事件监控面板组件
const IncidentMonitoringPanel: React.FC<{
  visible: boolean;
  onClose: () => void;
}> = ({ visible, onClose }) => {
  const [monitoringData, setMonitoringData] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const fetchMonitoringData = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/v1/incidents/monitoring", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${localStorage.getItem("access_token")}`,
          "X-Tenant-ID": localStorage.getItem("tenant_id") || "",
        },
        body: JSON.stringify({
          start_time: dayjs().subtract(7, "day").toISOString(),
          end_time: dayjs().toISOString(),
        }),
      });

      if (response.ok) {
        const data = await response.json();
        setMonitoringData(data);
      }
    } catch (error) {
      console.error("获取监控数据失败:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (visible) {
      fetchMonitoringData();
    }
  }, [visible]);

  return (
    <Drawer
      title="事件监控面板"
      width={1000}
      open={visible}
      onClose={onClose}
      extra={
        <Button icon={<ReloadOutlined />} onClick={fetchMonitoringData} loading={loading}>
          刷新
        </Button>
      }
    >
      {loading ? (
        <div className="text-center py-8">
          <Text>加载中...</Text>
        </div>
      ) : monitoringData ? (
        <div className="space-y-6">
          {/* 监控概览 */}
          <Row gutter={16}>
            <Col span={6}>
              <Card>
                <Statistic
                  title="总事件数"
                  value={monitoringData.total_incidents}
                  prefix={<InfoCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="进行中"
                  value={monitoringData.open_incidents}
                  prefix={<ClockCircleOutlined />}
                  valueStyle={{ color: "#1890ff" }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="已解决"
                  value={monitoringData.resolved_incidents}
                  prefix={<CheckCircleOutlined />}
                  valueStyle={{ color: "#52c41a" }}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="解决率"
                  value={monitoringData.resolution_rate}
                  suffix="%"
                  precision={1}
                  prefix={<BarChartOutlined />}
                  valueStyle={{ color: "#52c41a" }}
                />
              </Card>
            </Col>
          </Row>

          {/* 趋势图表 */}
          <Card title="事件趋势">
            <div className="h-64 flex items-center justify-center bg-gray-50 rounded">
              <Text type="secondary">图表组件待实现</Text>
            </div>
          </Card>

          {/* 严重事件列表 */}
          <Card title="严重事件">
            {monitoringData.critical_incidents > 0 ? (
              <Alert
                message={`发现 ${monitoringData.critical_incidents} 个严重事件`}
                type="error"
                showIcon
                action={
                  <Button size="small" danger>
                    查看详情
                  </Button>
                }
              />
            ) : (
              <Alert
                message="暂无严重事件"
                type="success"
                showIcon
              />
            )}
          </Card>
        </div>
      ) : (
        <Empty description="暂无监控数据" />
      )}
    </Drawer>
  );
};

export default IncidentManagement;
