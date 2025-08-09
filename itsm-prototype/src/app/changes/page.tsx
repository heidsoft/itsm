"use client";

import React, { useState } from "react";
import Link from "next/link";
import {
  Card,
  Table,
  Button,
  Tag,
  Space,
  Row,
  Col,
  Statistic,
  Select,
  Input,
  Tooltip,
  Progress,
} from "antd";
import {
  PlusCircle,
  Search,
  Eye,
  Edit,
  GitBranch,
  Calendar,
} from "lucide-react";
import { mockChangesData } from "../lib/mock-data";

const { Search: SearchInput } = Input;
const { Option } = Select;

const getChangeTypeColor = (type: string) => {
  switch (type) {
    case "普通变更":
      return "blue";
    case "标准变更":
      return "green";
    case "紧急变更":
      return "red";
    default:
      return "default";
  }
};

const getStatusColor = (status: string) => {
  switch (status) {
    case "待审批":
      return "warning";
    case "已批准":
      return "success";
    case "实施中":
      return "processing";
    case "已完成":
      return "success";
    case "已回滚":
      return "error";
    case "已拒绝":
      return "error";
    default:
      return "default";
  }
};

const getRiskLevel = (risk: string) => {
  switch (risk) {
    case "高":
      return { color: "red", percent: 80 };
    case "中":
      return { color: "orange", percent: 50 };
    case "低":
      return { color: "green", percent: 20 };
    default:
      return { color: "gray", percent: 0 };
  }
};

const ChangeListPage = () => {
  const [filter, setFilter] = useState("全部");
  const [searchText, setSearchText] = useState("");
  const [changes] = useState(mockChangesData);

  const filteredChanges = changes.filter((change) => {
    const matchesFilter = filter === "全部" || change.status === filter;
    const matchesSearch =
      change.title.toLowerCase().includes(searchText.toLowerCase()) ||
      change.id.toLowerCase().includes(searchText.toLowerCase());
    return matchesFilter && matchesSearch;
  });

  const statusCounts = {
    total: changes.length,
    pending: changes.filter((c) => c.status === "待审批").length,
    approved: changes.filter((c) => c.status === "已批准").length,
    implementing: changes.filter((c) => c.status === "实施中").length,
    completed: changes.filter((c) => c.status === "已完成").length,
  };

  const columns = [
    {
      title: "变更ID",
      dataIndex: "id",
      key: "id",
      width: 120,
      render: (text: string) => (
        <Link href={`/changes/${text}`}>
          <span className="text-blue-600 hover:text-blue-800 font-medium cursor-pointer">
            {text}
          </span>
        </Link>
      ),
    },
    {
      title: "变更标题",
      dataIndex: "title",
      key: "title",
      ellipsis: true,
    },
    {
      title: "变更类型",
      dataIndex: "type",
      key: "type",
      width: 120,
      render: (type: string) => (
        <Tag color={getChangeTypeColor(type)}>{type}</Tag>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{status}</Tag>
      ),
    },
    {
      title: "风险等级",
      dataIndex: "risk",
      key: "risk",
      width: 120,
      render: (risk: string) => {
        const riskInfo = getRiskLevel(risk);
        return (
          <div style={{ width: 80 }}>
            <Progress
              percent={riskInfo.percent}
              size="small"
              status={
                riskInfo.color as "active" | "exception" | "normal" | "success"
              }
              format={() => risk}
            />
          </div>
        );
      },
    },
    {
      title: "申请人",
      dataIndex: "requester",
      key: "requester",
      width: 100,
    },
    {
      title: "计划实施时间",
      dataIndex: "scheduledDate",
      key: "scheduledDate",
      width: 150,
    },
    {
      title: "操作",
      key: "actions",
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Link href={`/changes/${record.id}`}>
              <Button type="text" size="small" icon={<Eye size={14} />} />
            </Link>
          </Tooltip>
          <Tooltip title="编辑">
            <Button type="text" size="small" icon={<Edit size={14} />} />
          </Tooltip>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 24 }}>
      {/* 页面头部 */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 24,
        }}
      >
        <div>
          <h1
            style={{
              fontSize: 24,
              fontWeight: 600,
              color: "#1e293b",
              margin: 0,
            }}
          >
            变更管理
          </h1>
          <p style={{ color: "#64748b", marginTop: 4, marginBottom: 0 }}>
            管理系统变更请求，确保变更的安全性和可控性
          </p>
        </div>
        <Link href="/changes/new">
          <Button type="primary" icon={<PlusCircle size={16} />}>
            新建变更
          </Button>
        </Link>
      </div>

      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总变更数"
              value={statusCounts.total}
              prefix={<GitBranch size={16} style={{ color: "#1890ff" }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="待审批"
              value={statusCounts.pending}
              valueStyle={{ color: "#faad14" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="实施中"
              value={statusCounts.implementing}
              valueStyle={{ color: "#1890ff" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已完成"
              value={statusCounts.completed}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 筛选和搜索 */}
      <Card className="mb-6">
        <Row gutter={16} align="middle">
          <Col span={8}>
            <SearchInput
              placeholder="搜索变更ID或标题"
              allowClear
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              prefix={<Search size={16} />}
            />
          </Col>
          <Col span={4}>
            <Select
              value={filter}
              onChange={setFilter}
              style={{ width: "100%" }}
              placeholder="状态筛选"
            >
              <Option value="全部">全部状态</Option>
              <Option value="待审批">待审批</Option>
              <Option value="已批准">已批准</Option>
              <Option value="实施中">实施中</Option>
              <Option value="已完成">已完成</Option>
              <Option value="已回滚">已回滚</Option>
              <Option value="已拒绝">已拒绝</Option>
            </Select>
          </Col>
          <Col span={4}>
            <Button icon={<Calendar size={16} />}>时间筛选</Button>
          </Col>
        </Row>
      </Card>

      {/* 变更列表 */}
      <Card>
        <Table
          columns={columns}
          dataSource={filteredChanges}
          rowKey="id"
          pagination={{
            total: filteredChanges.length,
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          scroll={{ x: 1000 }}
        />
      </Card>
    </div>
  );
};

export default ChangeListPage;
