"use client";

import {
  Search,
  Edit,
  Eye,
  AlertTriangle,
  Tag as TagIcon,
  PlusCircle,
  RefreshCw,
} from "lucide-react";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import {
  Card,
  Table,
  Button,
  Space,
  Row,
  Col,
  Statistic,
  Select,
  Input,
  Tooltip,
  message,
  Tag,
} from "antd";
import {
  problemService,
  Problem,
  ProblemStatus,
  ProblemPriority,
  ListProblemsParams,
} from "../lib/services/problem-service";
import { LoadingEmptyError } from "../components/ui/LoadingEmptyError";
// AppLayout is handled by layout.tsx
import { useRouter } from "next/navigation";

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

  const router = useRouter();

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
      setPagination((prev) => ({
        ...prev,
        total: response.total,
        current: response.page,
        pageSize: response.page_size,
      }));
    } catch (error) {
      console.error("加载问题列表失败:", error);
      message.error("加载问题列表失败");
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
      console.error("加载统计数据失败:", error);
      message.error("加载统计数据失败");
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
    setPagination((prev) => ({
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
            PRB-{String(id).padStart(5, "0")}
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
      render: (assignee: any) => assignee?.name || "-",
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 150,
      render: (date: string) => new Date(date).toLocaleDateString("zh-CN"),
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
    <>
      {/* 页面头部操作 */}
      <div className="mb-6 flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">问题管理</h1>
          <p className="text-gray-600 mt-1">管理和跟踪系统问题，防止重复事件发生</p>
        </div>
        <Space>
          <Button
            icon={<RefreshCw size={16} />}
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
              placeholder="状态筛选"
              style={{ width: "100%" }}
              allowClear
              value={filter}
              onChange={setFilter}
            >
              <Option value="">全部状态</Option>
              <Option value={ProblemStatus.OPEN}>待处理</Option>
              <Option value={ProblemStatus.IN_PROGRESS}>处理中</Option>
              <Option value={ProblemStatus.RESOLVED}>已解决</Option>
              <Option value={ProblemStatus.CLOSED}>已关闭</Option>
            </Select>
          </Col>
          <Col span={4}>
            <Button type="primary" onClick={() => loadProblems()}>
              搜索
            </Button>
          </Col>
        </Row>
      </Card>

      {/* 问题列表 */}
      <Card>
        <LoadingEmptyError
          state={
            loading ? "loading" : problems.length === 0 ? "empty" : "success"
          }
          loadingText="正在加载问题列表..."
          empty={{
            title: "暂无问题",
            description: "当前没有问题记录，您可以创建新的问题",
            actions: [
              {
                text: "新建问题",
                icon: <PlusCircle size={16} />,
                onClick: () => router.push("/problems/new"),
                type: "primary",
              },
            ],
          }}
          error={{
            title: "加载失败",
            description: "无法获取问题列表，请稍后重试",
            onRetry: () => loadProblems(),
          }}
        >
          <Table
            columns={columns}
            dataSource={problems}
            rowKey="id"
            loading={false}
            pagination={{
              current: pagination.current,
              pageSize: pagination.pageSize,
              total: pagination.total,
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total, range) =>
                `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
              onChange: (page, size) => {
                setPagination((prev) => ({
                  ...prev,
                  current: page,
                  pageSize: size,
                }));
              },
            }}
          />
        </LoadingEmptyError>
      </Card>
    </>
  );
};

export default ProblemListPage;
