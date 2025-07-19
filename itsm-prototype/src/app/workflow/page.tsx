"use client";

import React, { useState, useEffect } from "react";
import {
  GitBranch,
  Play,
  Pause,
  CheckCircle,
  XCircle,
  Clock,
  Users,
  Settings,
  Plus,
  Search,
  Filter,
  MoreVertical,
} from "lucide-react";

interface WorkflowInstance {
  id: number;
  workflow_name: string;
  status: "running" | "completed" | "failed" | "paused";
  current_step: string;
  progress: number;
  created_at: string;
  updated_at: string;
  assignee?: string;
}

const WorkflowStatusBadge = ({ status }: { status: string }) => {
  const statusConfig = {
    running: {
      label: "运行中",
      color: "bg-blue-100 text-blue-800 border-blue-200",
      icon: Play,
    },
    completed: {
      label: "已完成",
      color: "bg-green-100 text-green-800 border-green-200",
      icon: CheckCircle,
    },
    failed: {
      label: "失败",
      color: "bg-red-100 text-red-800 border-red-200",
      icon: XCircle,
    },
    paused: {
      label: "暂停",
      color: "bg-yellow-100 text-yellow-800 border-yellow-200",
      icon: Pause,
    },
  };

  const config =
    statusConfig[status as keyof typeof statusConfig] || statusConfig.running;
  const Icon = config.icon;

  return (
    <span
      className={`inline-flex items-center px-3 py-1 text-xs font-medium rounded-full border ${config.color}`}
    >
      <Icon className="w-3 h-3 mr-1.5" />
      {config.label}
    </span>
  );
};

const WorkflowCard = ({ workflow }: { workflow: WorkflowInstance }) => {
  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-100 hover:shadow-md transition-all duration-200">
      <div className="p-6">
        <div className="flex items-start justify-between mb-4">
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-2">
              <GitBranch className="w-5 h-5 text-gray-500" />
              <h3 className="text-lg font-semibold text-gray-900">
                {workflow.workflow_name}
              </h3>
              <WorkflowStatusBadge status={workflow.status} />
            </div>
            <p className="text-sm text-gray-600 mb-3">
              当前步骤: {workflow.current_step}
            </p>
          </div>
          <button className="p-2 text-gray-400 hover:text-gray-600 rounded-lg hover:bg-gray-50">
            <MoreVertical className="w-4 h-4" />
          </button>
        </div>

        {/* 进度条 */}
        <div className="mb-4">
          <div className="flex items-center justify-between text-sm text-gray-600 mb-2">
            <span>进度</span>
            <span>{workflow.progress}%</span>
          </div>
          <div className="w-full bg-gray-200 rounded-full h-2">
            <div
              className="bg-blue-600 h-2 rounded-full transition-all duration-300"
              style={{ width: `${workflow.progress}%` }}
            ></div>
          </div>
        </div>

        <div className="flex items-center justify-between text-sm text-gray-500">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-1">
              <Clock className="w-4 h-4" />
              <span>
                {new Date(workflow.updated_at).toLocaleString("zh-CN")}
              </span>
            </div>
            {workflow.assignee && (
              <div className="flex items-center gap-1">
                <Users className="w-4 h-4" />
                <span>{workflow.assignee}</span>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

const WorkflowPage = () => {
  const [workflows, setWorkflows] = useState<WorkflowInstance[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("all");
  const [searchTerm, setSearchTerm] = useState("");

  // Mock数据 - 实际使用时替换为API调用
  useEffect(() => {
    const mockWorkflows: WorkflowInstance[] = [
      {
        id: 1,
        workflow_name: "服务请求审批流程",
        status: "running",
        current_step: "部门经理审批",
        progress: 60,
        created_at: "2025-01-28T10:00:00Z",
        updated_at: "2025-01-28T14:30:00Z",
        assignee: "张经理",
      },
      {
        id: 2,
        workflow_name: "变更管理流程",
        status: "completed",
        current_step: "已完成",
        progress: 100,
        created_at: "2025-01-27T09:00:00Z",
        updated_at: "2025-01-28T16:00:00Z",
        assignee: "李工程师",
      },
      {
        id: 3,
        workflow_name: "事件处理流程",
        status: "paused",
        current_step: "等待用户反馈",
        progress: 40,
        created_at: "2025-01-28T08:00:00Z",
        updated_at: "2025-01-28T12:00:00Z",
        assignee: "王技术员",
      },
    ];

    setTimeout(() => {
      setWorkflows(mockWorkflows);
      setLoading(false);
    }, 1000);
  }, []);

  const filteredWorkflows = workflows.filter((workflow) => {
    const matchesFilter = filter === "all" || workflow.status === filter;
    const matchesSearch =
      !searchTerm ||
      workflow.workflow_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      workflow.current_step.toLowerCase().includes(searchTerm.toLowerCase());
    return matchesFilter && matchesSearch;
  });

  const filterOptions = [
    { value: "all", label: "全部" },
    { value: "running", label: "运行中" },
    { value: "completed", label: "已完成" },
    { value: "paused", label: "暂停" },
    { value: "failed", label: "失败" },
  ];

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* 页面头部 */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 mb-2">
                流程管理
              </h1>
              <p className="text-gray-600">管理和监控所有工作流程实例</p>
            </div>
            <div className="flex gap-3">
              <button className="inline-flex items-center gap-2 px-4 py-2 bg-white border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition-colors">
                <Settings className="w-4 h-4" />
                流程设置
              </button>
              <button className="inline-flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
                <Plus className="w-4 h-4" />
                新建流程
              </button>
            </div>
          </div>
        </div>

        {/* 搜索和筛选 */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6 mb-6">
          <div className="flex flex-col lg:flex-row gap-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
                <input
                  type="text"
                  placeholder="搜索流程名称或步骤..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                />
              </div>
            </div>

            <div className="flex items-center gap-2">
              <Filter className="w-5 h-5 text-gray-500" />
              <div className="flex gap-2">
                {filterOptions.map((option) => (
                  <button
                    key={option.value}
                    onClick={() => setFilter(option.value)}
                    className={`px-4 py-2 text-sm font-medium rounded-lg transition-all ${
                      filter === option.value
                        ? "bg-blue-600 text-white shadow-sm"
                        : "bg-gray-100 text-gray-700 hover:bg-gray-200"
                    }`}
                  >
                    {option.label}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* 流程列表 */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            <span className="ml-3 text-gray-600">加载中...</span>
          </div>
        ) : filteredWorkflows.length > 0 ? (
          <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
            {filteredWorkflows.map((workflow) => (
              <WorkflowCard key={workflow.id} workflow={workflow} />
            ))}
          </div>
        ) : (
          <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-12 text-center">
            <GitBranch className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">暂无流程</h3>
            <p className="text-gray-600 mb-6">还没有运行中的工作流程</p>
            <button className="inline-flex items-center gap-2 px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
              <Plus className="w-4 h-4" />
              创建新流程
            </button>
          </div>
        )}
      </div>
    </div>
  );
};

export default WorkflowPage;
