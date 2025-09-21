"use client";

import React, { useEffect, useState } from "react";

import {
  Card,
  Table,
  Tag,
  Button,
  Space,
  Row,
  Col,
  Input,
  Select,
  Badge,
} from "antd";
import {
  SearchOutlined,
  PlusOutlined,
  DownloadOutlined,
} from "@ant-design/icons";
import { 
  AlertTriangle, 
  Clock, 
  CheckCircle, 
  AlertCircle,
  Eye,
  Edit,
  MoreHorizontal
} from "lucide-react";
import { IncidentAPI, Incident } from "../lib/incident-api";
// AppLayout is handled by layout.tsx

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
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 统计数据
  const [metrics, setMetrics] = useState({
    total_incidents: 0,
    critical_incidents: 0,
    major_incidents: 0,
    avg_resolution_time: 0,
  });

  const statusConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
    open: {
      color: "#fa8c16",
      text: "待处理",
      backgroundColor: "#fff7e6",
    },
    "in-progress": {
      color: "#1890ff",
      text: "处理中",
      backgroundColor: "#e6f7ff",
    },
    resolved: {
      color: "#52c41a",
      text: "已解决",
      backgroundColor: "#f6ffed",
    },
    closed: {
      color: "#00000073",
      text: "已关闭",
      backgroundColor: "#fafafa",
    },
  };

  const priorityConfig: Record<string, { color: string; text: string; backgroundColor: string }> = {
    low: {
      color: "#52c41a",
      text: "低",
      backgroundColor: "#f6ffed",
    },
    medium: {
      color: "#1890ff",
      text: "中",
      backgroundColor: "#e6f7ff",
    },
    high: {
      color: "#fa8c16",
      text: "高",
      backgroundColor: "#fff7e6",
    },
    critical: {
      color: "#ff4d4f",
      text: "紧急",
      backgroundColor: "#fff2f0",
    },
  };

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
        status: filters.status,
        priority: filters.priority,
        source: filters.source,
        keyword: filters.keyword,
      });
      setIncidents(response.incidents);
      setTotal(response.total);
    } catch (error) {
      console.error("加载事件失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const loadMetrics = async () => {
    try {
      const response = await IncidentAPI.getIncidentMetrics();
      setMetrics(response);
    } catch (error) {
      console.error("加载统计信息失败:", error);
    }
  };

  const handleSearch = (value: string) => {
    setFilters({ ...filters, keyword: value });
    setCurrentPage(1);
  };

  const handleFilterChange = (key: string, value: string) => {
    setFilters({ ...filters, [key]: value });
    setCurrentPage(1);
  };

  const handleCreateIncident = () => {
    // 跳转到创建事件页面
    window.location.href = "/incidents/new";
  };

  // 渲染统计卡片
  const renderStatsCards = () => (
    <div className="mb-8">
      <Row gutter={[24, 24]}>
        <Col xs={24} sm={12} lg={6}>
          <Card className="text-center hover:shadow-2xl hover:scale-105 transition-all duration-300 border-0 bg-gradient-to-br from-blue-500 via-blue-600 to-indigo-700 text-white overflow-hidden relative h-full">
            <div className="absolute top-0 right-0 w-20 h-20 bg-white/10 rounded-full -mr-10 -mt-10"></div>
            <div className="absolute bottom-0 left-0 w-16 h-16 bg-white/5 rounded-full -ml-8 -mb-8"></div>
            <div className="relative z-10">
              <div className="w-12 h-12 bg-white/20 rounded-xl flex items-center justify-center mx-auto mb-4 backdrop-blur-sm">
                <AlertTriangle className="w-6 h-6 text-white" />
              </div>
              <div className="text-3xl font-bold mb-2">{metrics.total_incidents}</div>
              <div className="text-blue-100 font-medium text-sm">总事件数</div>
              <div className="mt-3 flex items-center justify-center space-x-2 bg-white/10 rounded-full px-3 py-1">
                <div className="w-2 h-2 bg-green-300 rounded-full animate-pulse"></div>
                <span className="text-sm font-medium text-blue-100">实时监控</span>
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="text-center hover:shadow-2xl hover:scale-105 transition-all duration-300 border-0 bg-gradient-to-br from-orange-500 via-orange-600 to-red-600 text-white overflow-hidden relative h-full">
            <div className="absolute top-0 right-0 w-20 h-20 bg-white/10 rounded-full -mr-10 -mt-10"></div>
            <div className="absolute bottom-0 left-0 w-16 h-16 bg-white/5 rounded-full -ml-8 -mb-8"></div>
            <div className="relative z-10">
              <div className="w-12 h-12 bg-white/20 rounded-xl flex items-center justify-center mx-auto mb-4 backdrop-blur-sm">
                <Clock className="w-6 h-6 text-white" />
              </div>
              <div className="text-3xl font-bold mb-2">{metrics.critical_incidents}</div>
              <div className="text-orange-100 font-medium text-sm">待处理事件</div>
              <div className="mt-3 flex items-center justify-center space-x-2 bg-white/10 rounded-full px-3 py-1">
                <div className="w-2 h-2 bg-yellow-300 rounded-full animate-pulse"></div>
                <span className="text-sm font-medium text-orange-100">需要关注</span>
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="text-center hover:shadow-2xl hover:scale-105 transition-all duration-300 border-0 bg-gradient-to-br from-green-500 via-green-600 to-emerald-700 text-white overflow-hidden relative h-full">
            <div className="absolute top-0 right-0 w-20 h-20 bg-white/10 rounded-full -mr-10 -mt-10"></div>
            <div className="absolute bottom-0 left-0 w-16 h-16 bg-white/5 rounded-full -ml-8 -mb-8"></div>
            <div className="relative z-10">
              <div className="w-12 h-12 bg-white/20 rounded-xl flex items-center justify-center mx-auto mb-4 backdrop-blur-sm">
                <CheckCircle className="w-6 h-6 text-white" />
              </div>
              <div className="text-3xl font-bold mb-2">{metrics.major_incidents}</div>
              <div className="text-green-100 font-medium text-sm">已解决事件</div>
              <div className="mt-3 flex items-center justify-center space-x-2 bg-white/10 rounded-full px-3 py-1">
                <div className="w-2 h-2 bg-green-300 rounded-full animate-pulse"></div>
                <span className="text-sm font-medium text-green-100">处理完成</span>
              </div>
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card className="text-center hover:shadow-2xl hover:scale-105 transition-all duration-300 border-0 bg-gradient-to-br from-purple-500 via-purple-600 to-indigo-700 text-white overflow-hidden relative h-full">
            <div className="absolute top-0 right-0 w-20 h-20 bg-white/10 rounded-full -mr-10 -mt-10"></div>
            <div className="absolute bottom-0 left-0 w-16 h-16 bg-white/5 rounded-full -ml-8 -mb-8"></div>
            <div className="relative z-10">
              <div className="w-12 h-12 bg-white/20 rounded-xl flex items-center justify-center mx-auto mb-4 backdrop-blur-sm">
                <AlertCircle className="w-6 h-6 text-white" />
              </div>
              <div className="text-3xl font-bold mb-2">{metrics.avg_resolution_time}</div>
              <div className="text-purple-100 font-medium text-sm">平均解决时间(小时)</div>
              <div className="mt-3 flex items-center justify-center space-x-2 bg-white/10 rounded-full px-3 py-1">
                <div className="w-2 h-2 bg-blue-300 rounded-full animate-pulse"></div>
                <span className="text-sm font-medium text-purple-100">效率指标</span>
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );

  // 渲染筛选器
  const renderFilters = () => (
    <Card className="mb-8 bg-white/60 backdrop-blur-sm border border-white/20 shadow-lg rounded-2xl overflow-hidden">
      <div className="bg-gradient-to-r from-gray-50 to-blue-50 p-6 border-b border-gray-100">
        <div className="flex items-center space-x-3 mb-4">
          <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-lg flex items-center justify-center">
            <SearchOutlined className="text-white text-sm" />
          </div>
          <h3 className="text-lg font-semibold text-gray-800">事件筛选与搜索</h3>
        </div>
        <Row gutter={[24, 16]} align="middle">
          <Col xs={24} sm={12} md={8}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">搜索事件</label>
              <Search
                placeholder="搜索事件标题、ID或描述..."
                allowClear
                onSearch={handleSearch}
                size="large"
                enterButton
                className="rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200"
              />
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">状态筛选</label>
              <Select
                placeholder="选择状态"
                size="large"
                allowClear
                value={filters.status}
                onChange={(value) => handleFilterChange("status", value)}
                className="w-full rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200"
              >
                <Option value="open">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-orange-500 rounded-full"></div>
                    <span>待处理</span>
                  </div>
                </Option>
                <Option value="in-progress">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                    <span>处理中</span>
                  </div>
                </Option>
                <Option value="resolved">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                    <span>已解决</span>
                  </div>
                </Option>
                <Option value="closed">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-gray-500 rounded-full"></div>
                    <span>已关闭</span>
                  </div>
                </Option>
              </Select>
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">优先级</label>
              <Select
                placeholder="选择优先级"
                size="large"
                allowClear
                value={filters.priority}
                onChange={(value) => handleFilterChange("priority", value)}
                className="w-full rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200"
              >
                <Option value="low">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                    <span>低</span>
                  </div>
                </Option>
                <Option value="medium">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                    <span>中</span>
                  </div>
                </Option>
                <Option value="high">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-orange-500 rounded-full"></div>
                    <span>高</span>
                  </div>
                </Option>
                <Option value="critical">
                  <div className="flex items-center space-x-2">
                    <div className="w-2 h-2 bg-red-500 rounded-full animate-pulse"></div>
                    <span>紧急</span>
                  </div>
                </Option>
              </Select>
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">来源</label>
              <Select
                placeholder="选择来源"
                size="large"
                allowClear
                value={filters.source}
                onChange={(value) => handleFilterChange("source", value)}
                className="w-full rounded-lg shadow-sm hover:shadow-md transition-shadow duration-200"
              >
                <Option value="email">📧 邮件</Option>
                <Option value="phone">📞 电话</Option>
                <Option value="web">🌐 网页</Option>
                <Option value="system">⚙️ 系统</Option>
              </Select>
            </div>
          </Col>
          <Col xs={24} sm={12} md={4}>
            <div className="space-y-2">
              <label className="text-sm font-medium text-gray-700">操作</label>
              <Button
                icon={<SearchOutlined />}
                onClick={loadIncidents}
                loading={loading}
                size="large"
                className="w-full bg-gradient-to-r from-blue-500 to-indigo-600 border-0 text-white hover:from-blue-600 hover:to-indigo-700 shadow-lg hover:shadow-xl transition-all duration-200 rounded-lg font-medium"
              >
                刷新数据
              </Button>
            </div>
          </Col>
        </Row>
      </div>
    </Card>
  );

  // 渲染事件列表
  const renderIncidentList = () => (
    <div>
      <div className="bg-white/60 backdrop-blur-sm rounded-2xl p-6 border border-white/20 shadow-lg mb-6">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-6">
          <div className="flex items-center space-x-4">
            <div className="w-10 h-10 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-xl flex items-center justify-center shadow-lg">
              <AlertTriangle className="w-5 h-5 text-white" />
            </div>
            <div>
              <h3 className="text-lg font-semibold text-gray-800">事件管理列表</h3>
              <p className="text-sm text-gray-600">管理和跟踪所有系统事件</p>
            </div>
            {selectedRowKeys.length > 0 && (
              <Badge 
                count={selectedRowKeys.length} 
                showZero 
                className="bg-gradient-to-r from-blue-500 to-indigo-600 text-white rounded-full px-3 py-1 text-sm font-medium shadow-lg"
              />
            )}
          </div>
          <Space size="middle" className="flex-wrap">
            <Button 
              icon={<DownloadOutlined />} 
              size="large"
              className="bg-white border-gray-200 text-gray-700 hover:bg-gray-50 hover:border-gray-300 shadow-sm hover:shadow-md transition-all duration-200 rounded-lg font-medium"
            >
              导出数据
            </Button>
            <Button 
              type="primary" 
              icon={<PlusOutlined />} 
              size="large" 
              onClick={handleCreateIncident}
              className="bg-gradient-to-r from-green-500 to-emerald-600 border-0 text-white hover:from-green-600 hover:to-emerald-700 shadow-lg hover:shadow-xl transition-all duration-200 rounded-lg font-medium"
            >
              创建事件
            </Button>
          </Space>
        </div>
      </div>

      <div className="bg-white rounded-2xl border border-gray-200 shadow-lg overflow-hidden">
        <Table
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
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
          scroll={{ x: 1200 }}
          className="[&_.ant-table-thead>tr>th]:bg-gray-50 [&_.ant-table-thead>tr>th]:border-b-2 [&_.ant-table-thead>tr>th]:border-gray-200 [&_.ant-table-thead>tr>th]:font-semibold [&_.ant-table-thead>tr>th]:text-gray-700 [&_.ant-table-tbody>tr:hover>td]:bg-blue-50 [&_.ant-table-tbody>tr>td]:border-b [&_.ant-table-tbody>tr>td]:border-gray-100 [&_.ant-table-tbody>tr>td]:py-4"
        />
      </div>
    </div>
  );

  // 表格列定义
  const columns = [
    {
      title: "事件信息",
      key: "incident_info",
      width: 300,
      render: (_: unknown, record: Incident) => (
        <div style={{ display: "flex", alignItems: "center" }}>
          <div style={{ width: 40, height: 40, backgroundColor: "#e6f7ff", borderRadius: 8, display: "flex", alignItems: "center", justifyContent: "center", marginRight: 12 }}>
            <AlertTriangle size={20} style={{ color: "#1890ff" }} />
          </div>
          <div>
            <div style={{ fontWeight: "medium", color: "#000", marginBottom: 4 }}>{record.title}</div>
            <div style={{ fontSize: "small", color: "#666" }}>
              #{record.incident_number} • {record.category}
            </div>
          </div>
        </div>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 120,
      render: (status: string) => {
        const config = statusConfig[status];
        return (
          <span
            style={{
              padding: "4px 12px",
              borderRadius: 16,
              fontSize: "small",
              fontWeight: 500,
              color: config.color,
              backgroundColor: config.backgroundColor,
            }}
          >
            {config.text}
          </span>
        );
      },
    },
    {
      title: "优先级",
      dataIndex: "priority",
      key: "priority",
      width: 100,
      render: (priority: string) => {
        const config = priorityConfig[priority];
        return (
          <span
            style={{
              padding: "4px 12px",
              borderRadius: 16,
              fontSize: "small",
              fontWeight: 500,
              color: config.color,
              backgroundColor: config.backgroundColor,
            }}
          >
            {config.text}
          </span>
        );
      },
    },
    {
      title: "影响范围",
      dataIndex: "impact",
      key: "impact",
      width: 120,
      render: (impact: string) => {
        const impactConfig: Record<string, { color: string; text: string }> = {
          low: { color: "green", text: "低" },
          medium: { color: "orange", text: "中" },
          high: { color: "red", text: "高" },
        };
        const config = impactConfig[impact] || { color: "default", text: impact };
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: "报告人",
      dataIndex: "reporter",
      key: "reporter",
      width: 150,
      render: (reporter: { name: string }) => (
        <div style={{ fontSize: "small" }}>{reporter?.name || "未知"}</div>
      ),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 150,
      render: (created_at: string) => (
        <div style={{ fontSize: "small", color: "#666" }}>
          {new Date(created_at).toLocaleDateString("zh-CN")}
        </div>
      ),
    },
    {
      title: "操作",
      key: "actions",
      width: 150,
      render: (_: unknown, record: Incident) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<Eye size={16} />}
            onClick={() => window.open(`/incidents/${record.id}`)}
            className="text-blue-600 hover:text-blue-700 hover:bg-blue-50 border-0 rounded-lg transition-all duration-200 p-2"
            title="查看详情"
          />
          <Button
            type="text"
            size="small"
            icon={<Edit size={16} />}
            onClick={() => window.open(`/incidents/${record.id}/edit`)}
            className="text-green-600 hover:text-green-700 hover:bg-green-50 border-0 rounded-lg transition-all duration-200 p-2"
            title="编辑事件"
          />
          <Button
            type="text"
            size="small"
            icon={<MoreHorizontal size={16} />}
            className="text-gray-600 hover:text-gray-700 hover:bg-gray-50 border-0 rounded-lg transition-all duration-200 p-2"
            title="更多操作"
          />
        </Space>
      ),
    },
  ];

  return (
    <div>
      {renderStatsCards()}
      {renderFilters()}
      {renderIncidentList()}
    </div>
  );
}