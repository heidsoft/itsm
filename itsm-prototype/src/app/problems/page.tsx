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
} from "antd";
import { PlusCircle, Search, Eye, Edit, AlertTriangle } from "lucide-react";
import { mockProblemsData } from "../lib/mock-data";

const { Search: SearchInput } = Input;
const { Option } = Select;

const getPriorityColor = (priority: string) => {
  switch (priority) {
    case "高":
      return "red";
    case "中":
      return "orange";
    case "低":
      return "green";
    default:
      return "default";
  }
};

const getStatusColor = (status: string) => {
  switch (status) {
    case "调查中":
      return "processing";
    case "已解决":
      return "success";
    case "已知错误":
      return "warning";
    case "已关闭":
      return "default";
    default:
      return "default";
  }
};

const ProblemListPage = () => {
  const [filter, setFilter] = useState("全部");
  const [searchText, setSearchText] = useState("");
  const [problems] = useState(mockProblemsData);

  const filteredProblems = problems.filter((problem) => {
    const matchesFilter = filter === "全部" || problem.status === filter;
    const matchesSearch =
      problem.title.toLowerCase().includes(searchText.toLowerCase()) ||
      problem.id.toLowerCase().includes(searchText.toLowerCase());
    return matchesFilter && matchesSearch;
  });

  const statusCounts = {
    total: problems.length,
    investigating: problems.filter((p) => p.status === "调查中").length,
    resolved: problems.filter((p) => p.status === "已解决").length,
    knownError: problems.filter((p) => p.status === "已知错误").length,
    closed: problems.filter((p) => p.status === "已关闭").length,
  };

  const columns = [
    {
      title: "问题ID",
      dataIndex: "id",
      key: "id",
      width: 120,
      render: (text: string) => (
        <Link href={`/problems/${text}`}>
          <span className="text-blue-600 hover:text-blue-800 font-medium cursor-pointer">
            {text}
          </span>
        </Link>
      ),
    },
    {
      title: "标题",
      dataIndex: "title",
      key: "title",
      ellipsis: true,
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
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{status}</Tag>
      ),
    },
    {
      title: "负责人",
      dataIndex: "assignee",
      key: "assignee",
      width: 100,
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 150,
    },
    {
      title: "操作",
      key: "actions",
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Link href={`/problems/${record.id}`}>
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
            问题管理
          </h1>
          <p style={{ color: "#64748b", marginTop: 4, marginBottom: 0 }}>
            管理和跟踪系统问题，防止重复事件发生
          </p>
        </div>
        <Link href="/problems/new">
          <Button type="primary" icon={<PlusCircle size={16} />}>
            新建问题
          </Button>
        </Link>
      </div>

      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总问题数"
              value={statusCounts.total}
              prefix={<AlertTriangle size={16} style={{ color: "#3b82f6" }} />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="调查中"
              value={statusCounts.investigating}
              valueStyle={{ color: "#faad14" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已解决"
              value={statusCounts.resolved}
              valueStyle={{ color: "#52c41a" }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已知错误"
              value={statusCounts.knownError}
              valueStyle={{ color: "#f5222d" }}
            />
          </Card>
        </Col>
      </Row>

      {/* 筛选和搜索 */}
      <Card style={{ marginBottom: 24 }}>
        <Row gutter={16} align="middle">
          <Col span={8}>
            <SearchInput
              placeholder="搜索问题ID或标题"
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
              <Option value="调查中">调查中</Option>
              <Option value="已解决">已解决</Option>
              <Option value="已知错误">已知错误</Option>
              <Option value="已关闭">已关闭</Option>
            </Select>
          </Col>
        </Row>
      </Card>

      {/* 问题列表 */}
      <Card>
        <Table
          columns={columns}
          dataSource={filteredProblems}
          rowKey="id"
          pagination={{
            total: filteredProblems.length,
            pageSize: 10,
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

export default ProblemListPage;
