"use client";

import React, { useEffect, useState } from "react";

import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Statistic,
  Row,
  Col,
  Input,
  Select,
  DatePicker,
} from "antd";
import {
  SearchOutlined,
  PlusOutlined,
  DownloadOutlined,
} from "@ant-design/icons";
import { AlertTriangle, Clock, CheckCircle, AlertCircle } from "lucide-react";
import { IncidentAPI, Incident } from "../lib/incident-api";

const { Search } = Input;
const { Option } = Select;

export default function IncidentsPage() {
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [filters, setFilters] = useState({
    status: "",
    priority: "",
    source: "",
    keyword: "",
  });

  // 统计数据
  const [metrics, setMetrics] = useState({
    total_incidents: 0,
    critical_incidents: 0,
    major_incidents: 0,
    avg_resolution_time: 0,
  });

  useEffect(() => {
    loadIncidents();
    loadMetrics();
  }, [currentPage, pageSize, filters]);

  const loadIncidents = async () => {
    setLoading(true);
    try {
      const response = await IncidentAPI.listIncidents({
        page: currentPage,
        page_size: pageSize,
        ...filters,
      });
      setIncidents(response.incidents);
      setTotal(response.total);
    } catch (error) {
      console.error("加载事件列表失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const loadMetrics = async () => {
    try {
      const metricsData = await IncidentAPI.getIncidentMetrics();
      setMetrics(metricsData);
    } catch (error) {
      console.error("加载事件统计失败:", error);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case "open":
        return "red";
      case "in_progress":
        return "blue";
      case "resolved":
        return "green";
      case "closed":
        return "default";
      default:
        return "default";
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority.toLowerCase()) {
      case "critical":
        return "red";
      case "high":
        return "orange";
      case "medium":
        return "yellow";
      case "low":
        return "green";
      default:
        return "default";
    }
  };

  const getStatusText = (status: string) => {
    switch (status.toLowerCase()) {
      case "open":
        return "待处理";
      case "in_progress":
        return "处理中";
      case "resolved":
        return "已解决";
      case "closed":
        return "已关闭";
      default:
        return status;
    }
  };

  const getPriorityText = (priority: string) => {
    switch (priority.toLowerCase()) {
      case "critical":
        return "严重";
      case "high":
        return "高";
      case "medium":
        return "中";
      case "low":
        return "低";
      default:
        return priority;
    }
  };

  const columns = [
    {
      title: "事件编号",
      dataIndex: "incident_number",
      key: "incident_number",
      width: 120,
    },
    {
      title: "标题",
      dataIndex: "title",
      key: "title",
      render: (text: string, record: Incident) => (
        <div>
          <div className="font-medium">{text}</div>
          <div className="text-sm text-gray-500">
            {record.type} · {record.reporter?.name || "未知"}
          </div>
        </div>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{getStatusText(status)}</Tag>
      ),
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 100,
      render: (priority: string) => (
        <Tag color={getPriorityColor(priority)}>
          {getPriorityText(priority)}
        </Tag>
      ),
    },
    {
      title: "来源",
      dataIndex: "source",
      key: "source",
      width: 120,
    },
    {
      title: "处理人",
      dataIndex: "assignee",
      key: "assignee",
      width: 100,
      render: (assignee: any) => assignee?.name || "未分配",
    },
    {
      title: "检测时间",
      dataIndex: "detected_at",
      key: "detected_at",
      width: 150,
      render: (date: string) => new Date(date).toLocaleString("zh-CN"),
    },
    {
      title: "操作",
      key: "action",
      width: 120,
      render: (record: Incident) => (
        <Space>
          <Button type="link" size="small">
            查看
          </Button>
          <Button type="link" size="small">
            编辑
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      {/* 统计卡片 */}
      <Row gutter={16} className="mb-6">
        <Col span={6}>
          <Card>
            <Statistic
              title="总事件数"
              value={metrics.total_incidents}
              prefix={<AlertTriangle style={{ color: "#1890ff" }} />}
              valueStyle={{ color: "#1890ff" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="待处理"
              value={incidents.filter((i) => i.status === "open").length}
              prefix={<Clock size={16} style={{ color: "#faad14" }} />}
              valueStyle={{ color: "#faad14" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已解决"
              value={incidents.filter((i) => i.status === "resolved").length}
              prefix={<CheckCircle size={16} style={{ color: "#52c41a" }} />}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="严重事件"
              value={metrics.critical_incidents}
              prefix={<AlertCircle size={16} style={{ color: "#ff4d4f" }} />}
              valueStyle={{ color: "#ff4d4f" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 操作栏 */}
      <Card className="mb-6">
        <div className="flex justify-between items-center">
          <div className="flex items-center space-x-4">
            <Search
              placeholder="搜索事件..."
              style={{ width: 300 }}
              onSearch={(value) => setFilters({ ...filters, keyword: value })}
            />
            <Select
              placeholder="状态筛选"
              style={{ width: 120 }}
              allowClear
              onChange={(value) => setFilters({ ...filters, status: value })}
            >
              <Option value="open">待处理</Option>
              <Option value="in_progress">处理中</Option>
              <Option value="resolved">已解决</Option>
              <Option value="closed">已关闭</Option>
            </Select>
            <Select
              placeholder="优先级"
              style={{ width: 120 }}
              allowClear
              onChange={(value) => setFilters({ ...filters, priority: value })}
            >
              <Option value="critical">严重</Option>
              <Option value="high">高</Option>
              <Option value="medium">中</Option>
              <Option value="low">低</Option>
            </Select>
            <Select
              placeholder="来源"
              style={{ width: 120 }}
              allowClear
              onChange={(value) => setFilters({ ...filters, source: value })}
            >
              <Option value="manual">手动创建</Option>
              <Option value="alibaba_cloud">阿里云</Option>
              <Option value="security_event">安全事件</Option>
              <Option value="cloud_product">云产品事件</Option>
            </Select>
          </div>
          <div className="flex items-center space-x-2">
            <Button type="primary" icon={<PlusOutlined />}>
              新建事件
            </Button>
            <Button icon={<DownloadOutlined />}>导出</Button>
          </div>
        </div>
      </Card>

      {/* 事件列表 */}
      <Card title="事件列表">
        <Table
          columns={columns}
          dataSource={incidents}
          rowKey="id"
          loading={loading}
          pagination={{
            current: currentPage,
            pageSize: pageSize,
            total: total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
            onChange: (page, size) => {
              setCurrentPage(page);
              setPageSize(size);
            },
          }}
        />
      </Card>
    </div>
  );
}
