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
import { changeService, Change, ChangeStats } from "../lib/services/change-service";

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
  const [changes, setChanges] = useState<Change[]>([]);
  const [loading, setLoading] = useState(true);
  const [stats, setStats] = useState<ChangeStats>({
    total: 0,
    draft: 0,
    pending: 0,
    approved: 0,
    implementing: 0,
    completed: 0,
    cancelled: 0,
  });

  // 加载变更数据
  const loadChanges = async () => {
    setLoading(true);
    try {
      const isHealthy = await changeService.healthCheck();
      if (!isHealthy) {
        console.warn("后端服务不可用，使用Mock数据");
        // 这里可以添加fallback到mock数据的逻辑
        setChanges([]);
        return;
      }

      const data = await changeService.getChanges({
        status: filter === "全部" ? undefined : filter,
        search: searchText || undefined,
      });

      setChanges(data.changes);
      setStats({
        total: data.total,
        draft: data.changes.filter(c => c.status === 'draft').length,
        pending: data.changes.filter(c => c.status === 'pending').length,
        approved: data.changes.filter(c => c.status === 'approved').length,
        implementing: data.changes.filter(c => c.status === 'implementing').length,
        completed: data.changes.filter(c => c.status === 'completed').length,
        cancelled: data.changes.filter(c => c.status === 'cancelled').length,
      });
    } catch (error) {
      console.error("加载变更数据失败:", error);
      setChanges([]);
    } finally {
      setLoading(false);
    }
  };

  // 搜索和筛选
  const handleSearch = (value: string) => {
    setSearchText(value);
    loadChanges();
  };

  const handleFilterChange = (value: string) => {
    setFilter(value);
    loadChanges();
  };

  // 初始加载
  React.useEffect(() => {
    loadChanges();
  }, []);

  const statusCounts = {
    total: stats.total,
    pending: stats.pending,
    approved: stats.approved,
    implementing: stats.implementing,
    completed: stats.completed,
  };

  const columns = [
    {
      title: "变更ID",
      dataIndex: "id",
      key: "id",
      width: 120,
      render: (text: number) => (
        <Link href={`/changes/${text}`}>
          <span className="text-blue-600 hover:text-blue-800 font-medium cursor-pointer">
            {`CHG-${text.toString().padStart(5, '0')}`}
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
      render: (type: string) => {
        const typeMap = {
          'normal': '普通变更',
          'standard': '标准变更',
          'emergency': '紧急变更'
        };
        return (
          <Tag color={getChangeTypeColor(typeMap[type as keyof typeof typeMap] || type)}>
            {typeMap[type as keyof typeof typeMap] || type}
          </Tag>
        );
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: string) => {
        const statusMap = {
          'draft': '草稿',
          'pending': '待审批',
          'approved': '已批准',
          'rejected': '已拒绝',
          'implementing': '实施中',
          'completed': '已完成',
          'cancelled': '已取消'
        };
        return (
          <Tag color={getStatusColor(statusMap[status as keyof typeof statusMap] || status)}>
            {statusMap[status as keyof typeof statusMap] || status}
          </Tag>
        );
      },
    },
    {
      title: "风险等级",
      dataIndex: "riskLevel",
      key: "riskLevel",
      width: 120,
      render: (riskLevel: string) => {
        const riskInfo = getRiskLevel(riskLevel);
        return (
          <div style={{ width: 80 }}>
            <Progress
              percent={riskInfo.percent}
              size="small"
              status={
                riskInfo.color as "active" | "exception" | "normal" | "success"
              }
              format={() => riskLevel}
            />
          </div>
        );
      },
    },
    {
      title: "申请人",
      dataIndex: "createdByName",
      key: "createdByName",
      width: 100,
    },
    {
      title: "计划实施时间",
      dataIndex: "plannedStartDate",
      key: "plannedStartDate",
      width: 150,
      render: (date: string) => date ? new Date(date).toLocaleDateString() : '-',
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
              onSearch={handleSearch}
              onChange={(e) => setSearchText(e.target.value)}
              prefix={<Search size={16} />}
            />
          </Col>
          <Col span={4}>
            <Select
              value={filter}
              onChange={handleFilterChange}
              style={{ width: "100%" }}
              placeholder="状态筛选"
            >
              <Option value="全部">全部状态</Option>
              <Option value="pending">待审批</Option>
              <Option value="approved">已批准</Option>
              <Option value="implementing">实施中</Option>
              <Option value="completed">已完成</Option>
              <Option value="rejected">已拒绝</Option>
              <Option value="draft">草稿</Option>
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
          dataSource={changes}
          rowKey="id"
          loading={loading}
          pagination={{
            total: stats.total,
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
