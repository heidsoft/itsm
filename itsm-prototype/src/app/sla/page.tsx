"use client";

import {
  Plus,
  CheckCircle,
  Search,
  XCircle,
  AlertCircle,
  BarChart3,
  MoreHorizontal
} from "lucide-react";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import {
  Card,
  Table,
  Button,
  Input,
  Select,
  Row,
  Col,
  Statistic,
  theme,
  Typography,
  Tag,
  Space,
  Badge,
  Progress,
} from "antd";
// AppLayout is handled by layout.tsx

const { Text } = Typography;

// 模拟SLA数据
const mockSLAs = [
  {
    id: "SLA-001",
    name: "生产CRM系统可用性",
    service: "CRM系统",
    target: "99.9% 可用性",
    actual: "99.85%",
    status: "轻微违约",
    lastReview: "2025-06-01",
  },
  {
    id: "SLA-002",
    name: "内部IT服务台响应时间",
    service: "IT服务台",
    target: "80% 15分钟内响应",
    actual: "85%",
    status: "达标",
    lastReview: "2025-06-15",
  },
  {
    id: "SLA-003",
    name: "核心数据库性能",
    service: "数据库服务",
    target: "99.95% 可用性",
    actual: "99.97%",
    status: "优秀",
    lastReview: "2025-06-10",
  },
  {
    id: "SLA-004",
    name: "网络连接稳定性",
    service: "网络服务",
    target: "99.99% 可用性",
    actual: "99.99%",
    status: "优秀",
    lastReview: "2025-06-05",
  },
];

const getStatusConfig = (status: string) => {
  switch (status) {
    case "优秀":
      return {
        color: "#52c41a",
        text: "优秀",
        backgroundColor: "#f6ffed",
      };
    case "达标":
      return {
        color: "#1890ff",
        text: "达标",
        backgroundColor: "#e6f7ff",
      };
    case "轻微违约":
      return {
        color: "#fa8c16",
        text: "轻微违约",
        backgroundColor: "#fff7e6",
      };
    case "严重违约":
      return {
        color: "#ff4d4f",
        text: "严重违约",
        backgroundColor: "#fff2f0",
      };
    default:
      return {
        color: "#00000073",
        text: status,
        backgroundColor: "#fafafa",
      };
  }
};

export default function SLAPage() {
  const { token } = theme.useToken();
  const [slas, setSlas] = useState(mockSLAs);
  const [loading, setLoading] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [searchText, setSearchText] = useState("");

  // 统计数据
  const [stats, setStats] = useState({
    total: 0,
    excellent: 0,
    compliant: 0,
    violated: 0,
  });

  useEffect(() => {
    loadSLAs();
    loadStats();
  }, []);

  const loadSLAs = async () => {
    setLoading(true);
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));
      setSlas(mockSLAs);
    } catch (error) {
      console.error("加载SLA数据失败:", error);
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      // 模拟统计数据
      setStats({
        total: 12,
        excellent: 7,
        compliant: 4,
        violated: 1,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  };

  const handleCreateSLA = () => {
    window.location.href = "/sla/new";
  };

  // 渲染统计卡片
  const renderStatsCards = () => (
    <div style={{ marginBottom: 24 }}>
      <Row gutter={16}>
        <Col span={6}>
          <Card>
            <Statistic
              title="SLA总数"
              value={stats.total}
              prefix={<BarChart3 size={16} style={{ color: "#1890ff" }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="优秀"
              value={stats.excellent}
              valueStyle={{ color: "#52c41a" }}
              prefix={<CheckCircle size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="达标"
              value={stats.compliant}
              valueStyle={{ color: "#1890ff" }}
              prefix={<CheckCircle size={16} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="违约"
              value={stats.violated}
              valueStyle={{ color: "#ff4d4f" }}
              prefix={<XCircle size={16} />}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );

  // 渲染筛选器
  const renderFilters = () => (
    <Card style={{ marginBottom: 24 }}>
      <Row gutter={20} align="middle">
        <Col xs={24} sm={12} md={8}>
          <Input.Search
            placeholder="搜索SLA名称或服务..."
            allowClear
            onSearch={(value) => setSearchText(value)}
            size="large"
            enterButton
          />
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Button
            icon={<Search size={20} />}
            onClick={loadSLAs}
            loading={loading}
            size="large"
            style={{ width: "100%" }}
          >
            刷新
          </Button>
        </Col>
        <Col xs={24} sm={12} md={4}>
          <Button
            type="primary"
            icon={<Plus size={20} />}
            size="large"
            style={{ width: "100%" }}
            onClick={handleCreateSLA}
          >
            新建SLA
          </Button>
        </Col>
      </Row>
    </Card>
  );

  // 渲染SLA列表
  const renderSLAList = () => (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 16 }}>
        <div>
          {selectedRowKeys.length > 0 && (
            <Badge count={selectedRowKeys.length} showZero style={{ backgroundColor: "#1890ff" }} />
          )}
        </div>
      </div>

      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
        columns={columns}
        dataSource={slas}
        rowKey="id"
        loading={loading}
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total, range) =>
            `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
        }}
        scroll={{ x: 1000 }}
      />
    </div>
  );

  // 表格列定义
  const columns = [
    {
      title: "SLA信息",
      key: "sla_info",
      width: 300,
      render: (_: unknown, record: any) => (
        <div style={{ display: "flex", alignItems: "center" }}>
          <div style={{ width: 40, height: 40, backgroundColor: "#e6f7ff", borderRadius: 8, display: "flex", alignItems: "center", justifyContent: "center", marginRight: 12 }}>
            <BarChart3 size={20} style={{ color: "#1890ff" }} />
          </div>
          <div>
            <div style={{ fontWeight: "medium", color: "#000", marginBottom: 4 }}>{record.name}</div>
            <div style={{ fontSize: "small", color: "#666" }}>
              服务: {record.service}
            </div>
          </div>
        </div>
      ),
    },
    {
      title: "目标值",
      dataIndex: "target",
      key: "target",
      width: 200,
      render: (target: string) => (
        <div style={{ fontSize: "small" }}>{target}</div>
      ),
    },
    {
      title: "实际值",
      dataIndex: "actual",
      key: "actual",
      width: 150,
      render: (actual: string) => (
        <div style={{ fontSize: "small" }}>{actual}</div>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 120,
      render: (status: string) => {
        const config = getStatusConfig(status);
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
      title: "达成率",
      key: "achievement",
      width: 150,
      render: (_: unknown, record: any) => {
        // 简单计算达成率，实际应根据具体数据计算
        const rate = record.status === "优秀" ? 95 : 
                    record.status === "达标" ? 85 : 
                    record.status === "轻微违约" ? 75 : 40;
        return (
          <div>
            <Progress percent={rate} size="small" />
          </div>
        );
      },
    },
    {
      title: "最后评审",
      dataIndex: "lastReview",
      key: "lastReview",
      width: 120,
      render: (lastReview: string) => (
        <div style={{ fontSize: "small" }}>{lastReview}</div>
      ),
    },
    {
      title: "操作",
      key: "actions",
      width: 150,
      render: (_: unknown, record: any) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<BarChart3 size={16} />}
            onClick={() => window.open(`/sla/${record.id}`)}
          />
          <Button
            type="text"
            size="small"
            icon={<MoreHorizontal size={16} />}
          />
        </Space>
      ),
    },
  ];

  return (
    <div>
      {renderStatsCards()}
      {renderFilters()}
      {renderSLAList()}
    </div>
  );
}