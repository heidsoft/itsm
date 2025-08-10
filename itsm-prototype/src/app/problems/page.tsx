"use client";

import React, { useState, useEffect } from "react";
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
  message,
  Spin,
} from "antd";
import { PlusCircle, Search, Eye, Edit, AlertTriangle, Reload } from "lucide-react";
import { problemService, Problem, ProblemStatus, ProblemPriority, ListProblemsParams } from "../lib/services/problem-service";
import LoadingEmptyError from "../components/ui/LoadingEmptyError";

const { Search: SearchInput } = Input;
const { Option } = Select;

const ProblemListPage = () => {
  const [problems, setProblems] = useState<Problem[]>([]);
  const [loading, setLoading] = useState(false);
  const [statsLoading, setStatsLoading] = useState(false);
  const [filter, setFilter] = useState<string>("");
  const [searchText, setSearchText] = useState("");
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });
  const [stats, setStats] = useState({
    total: 0,
    open: 0,
    in_progress: 0,
    resolved: 0,
    closed: 0,
    high_priority: 0,
  });

  // 加载问题列表
  const loadProblems = async (params: ListProblemsParams = {}) => {
    setLoading(true);
    try {
      const response = await problemService.listProblems({
        page: pagination.current,
        page_size: pagination.pageSize,
        status: filter || undefined,
        keyword: searchText || undefined,
        ...params,
      });
      
      setProblems(response.problems);
      setPagination(prev => ({
        ...prev,
        total: response.total,
        current: response.page,
        pageSize: response.page_size,
      }));
    } catch (error) {
      console.error('加载问题列表失败:', error);
      message.error('加载问题列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 加载统计数据
  const loadStats = async () => {
    setStatsLoading(true);
    try {
      const response = await problemService.getProblemStats();
      setStats(response);
    } catch (error) {
      console.error('加载统计数据失败:', error);
      message.error('加载统计数据失败');
    } finally {
      setStatsLoading(false);
    }
  };

  // 初始加载
  useEffect(() => {
    loadProblems();
    loadStats();
  }, []);

  // 搜索和筛选变化时重新加载
  useEffect(() => {
    const timer = setTimeout(() => {
      loadProblems({ page: 1 });
    }, 500);
    return () => clearTimeout(timer);
  }, [searchText, filter]);

  // 处理分页变化
  const handleTableChange = (pagination: any) => {
    setPagination(prev => ({
      ...prev,
      current: pagination.current,
      pageSize: pagination.pageSize,
    }));
    loadProblems({
      page: pagination.current,
      page_size: pagination.pageSize,
    });
  };

  // 刷新数据
  const handleRefresh = () => {
    loadProblems();
    loadStats();
  };

  const columns = [
    {
      title: "问题ID",
      dataIndex: "id",
      key: "id",
      width: 120,
      render: (id: number) => (
        <Link href={`/problems/${id}`}>
          <span className="text-blue-600 hover:text-blue-800 font-medium cursor-pointer">
            PRB-{String(id).padStart(5, '0')}
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
      render: (priority: ProblemPriority) => (
        <Tag color={problemService.getPriorityColor(priority)}>
          {problemService.getPriorityLabel(priority)}
        </Tag>
      ),
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 100,
      render: (status: ProblemStatus) => (
        <Tag color={problemService.getStatusColor(status)}>
          {problemService.getStatusLabel(status)}
        </Tag>
      ),
    },
    {
      title: "分类",
      dataIndex: "category",
      key: "category",
      width: 120,
    },
    {
      title: "负责人",
      dataIndex: "assignee",
      key: "assignee",
      width: 100,
      render: (assignee: any) => assignee?.name || '-',
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 150,
      render: (date: string) => new Date(date).toLocaleDateString('zh-CN'),
    },
    {
      title: "操作",
      key: "actions",
      width: 120,
      render: (_, record: Problem) => (
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
        <Space>
          <Button 
            icon={<Reload size={16} />} 
            onClick={handleRefresh}
            loading={loading || statsLoading}
          >
            刷新
          </Button>
          <Link href="/problems/new">
            <Button type="primary" icon={<PlusCircle size={16} />}>
              新建问题
            </Button>
          </Link>
        </Space>
      </div>

      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总问题数"
              value={stats.total}
              prefix={<AlertTriangle size={16} style={{ color: "#3b82f6" }} />}
              loading={statsLoading}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="待处理"
              value={stats.open}
              valueStyle={{ color: "#faad14" }}
              loading={statsLoading}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="处理中"
              value={stats.in_progress}
              valueStyle={{ color: "#1890ff" }}
              loading={statsLoading}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已解决"
              value={stats.resolved}
              valueStyle={{ color: "#52c41a" }}
              loading={statsLoading}
            />
          </Card>
        </Col>
      </Row>

      {/* 筛选和搜索 */}
      <Card style={{ marginBottom: 24 }}>
        <Row gutter={16} align="middle">
          <Col span={8}>
            <SearchInput
              placeholder="搜索问题标题或描述"
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
              allowClear
            >
              <Option value="">全部状态</Option>
              <Option value="open">待处理</Option>
              <Option value="in_progress">处理中</Option>
              <Option value="resolved">已解决</Option>
              <Option value="closed">已关闭</Option>
            </Select>
          </Col>
        </Row>
      </Card>

      {/* 问题列表 */}
      <Card>
        <Table
          columns={columns}
          dataSource={problems}
          rowKey="id"
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) =>
              `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
          onChange={handleTableChange}
          scroll={{ x: 800 }}
        />
      </Card>
    </div>
  );
};

export default ProblemListPage;
