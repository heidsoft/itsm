"use client";

import {
  Plus,
  CheckCircle,
  Search,
  XCircle,
  Tag as TagIcon,
  AlertCircle,
  BarChart3,
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
} from "antd";
const { Title, Text } = Typography;

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
    name: "云资源申请交付时间",
    service: "云资源服务",
    target: "90% 2工作日内交付",
    actual: "88%",
    status: "违约",
    lastReview: "2025-06-20",
  },
  {
    id: "SLA-004",
    name: "邮件服务可用性",
    service: "邮件服务",
    target: "99.99% 可用性",
    actual: "99.99%",
    status: "达标",
    lastReview: "2025-06-05",
  },
];

const SLAListPage = () => {
  const { token } = theme.useToken();
  const [filter, setFilter] = useState("全部");
  const [searchText, setSearchText] = useState("");
  const [stats, setStats] = useState({
    total: 0,
    compliant: 0,
    warning: 0,
    violation: 0,
  });

  useEffect(() => {
    // 计算统计数据
    const total = mockSLAs.length;
    const compliant = mockSLAs.filter((sla) => sla.status === "达标").length;
    const warning = mockSLAs.filter((sla) => sla.status === "轻微违约").length;
    const violation = mockSLAs.filter((sla) => sla.status === "违约").length;

    setStats({ total, compliant, warning, violation });
  }, []);

  const getStatusColor = (status: string) => {
    const colors = {
      达标: "success",
      轻微违约: "warning",
      违约: "error",
    };
    return colors[status as keyof typeof colors] || "default";
  };

  const filteredSLAs = mockSLAs.filter((sla) => {
    if (filter !== "全部" && sla.status !== filter) return false;
    if (
      searchText &&
      !sla.name.toLowerCase().includes(searchText.toLowerCase()) &&
      !sla.service.toLowerCase().includes(searchText.toLowerCase())
    )
      return false;
    return true;
  });

  const columns = [
    {
      title: "SLA ID",
      dataIndex: "id",
      key: "id",
      width: 120,
      render: (id: string) => (
        <Link
          href={`/sla/${id}`}
          style={{ color: token.colorPrimary, fontWeight: 600 }}
        >
          {id}
        </Link>
      ),
    },
    {
      title: "SLA 名称",
      dataIndex: "name",
      key: "name",
      ellipsis: true,
      render: (name: string) => (
        <Text strong style={{ color: token.colorText }}>
          {name}
        </Text>
      ),
    },
    {
      title: "服务对象",
      dataIndex: "service",
      key: "service",
      width: 150,
      render: (service: string) => <Tag color="blue">{service}</Tag>,
    },
    {
      title: "目标",
      dataIndex: "target",
      key: "target",
      width: 180,
    },
    {
      title: "实际达成",
      dataIndex: "actual",
      key: "actual",
      width: 120,
      render: (actual: string) => <Text strong>{actual}</Text>,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 120,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{status}</Tag>
      ),
    },
    {
      title: "最后评审",
      dataIndex: "lastReview",
      key: "lastReview",
      width: 120,
      render: (date: string) => <Text type="secondary">{date}</Text>,
    },
  ];

  return (
    <div style={{ padding: token.paddingLG }}>
      {/* 页面头部 */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: token.marginLG,
        }}
      >
        <div>
          <Title level={2} style={{ margin: 0, color: token.colorText }}>
            服务级别管理
          </Title>
          <Text type="secondary" style={{ marginTop: token.marginXS }}>
            定义、监控和管理IT服务的性能和质量
          </Text>
        </div>
        <Link href="/sla/new">
          <Button
            type="primary"
            icon={<Plus className="w-4 h-4" />}
            size="large"
          >
            新建SLA
          </Button>
        </Link>
      </div>

      {/* 统计卡片 */}
      <Row gutter={[16, 16]} style={{ marginBottom: token.marginLG }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="总SLA数量"
              value={stats.total}
              prefix={<BarChart3 className="w-5 h-5" />}
              valueStyle={{ color: token.colorPrimary }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="达标"
              value={stats.compliant}
              prefix={<CheckCircle className="w-5 h-5" />}
              valueStyle={{ color: token.colorSuccess }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="轻微违约"
              value={stats.warning}
              prefix={<AlertCircle className="w-5 h-5" />}
              valueStyle={{ color: token.colorWarning }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="违约"
              value={stats.violation}
              prefix={<XCircle className="w-5 h-5" />}
              valueStyle={{ color: token.colorError }}
            />
          </Card>
        </Col>
      </Row>

      {/* 筛选器 */}
      <Card style={{ marginBottom: token.marginLG }}>
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={12} md={8}>
            <Input
              placeholder="搜索SLA名称或服务..."
              prefix={<Search className="w-4 h-4" />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              allowClear
            />
          </Col>
          <Col xs={24} sm={12} md={6}>
            <Select
              placeholder="状态筛选"
              value={filter}
              onChange={setFilter}
              style={{ width: "100%" }}
            >
              <Select.Option value="全部">全部</Select.Option>
              <Select.Option value="达标">达标</Select.Option>
              <Select.Option value="轻微违约">轻微违约</Select.Option>
              <Select.Option value="违约">违约</Select.Option>
            </Select>
          </Col>
        </Row>
      </Card>

      {/* SLA列表表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={filteredSLAs}
          rowKey="id"
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          scroll={{ x: 800 }}
        />
      </Card>
    </div>
  );
};

export default SLAListPage;
