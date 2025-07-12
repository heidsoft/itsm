"use client";

import React, { useState } from "react";
import {
  Plus,
  Search,
  Filter,
  Edit,
  Trash2,
  Play,
  Pause,
  Copy,
  Settings,
  GitBranch,
  Clock,
  Users,
  CheckCircle,
  XCircle,
  AlertCircle,
} from "lucide-react";

// 工作流状态枚举
const WORKFLOW_STATUS = {
  ACTIVE: "active",
  INACTIVE: "inactive",
  DRAFT: "draft",
} as const;

// 工作流类型枚举
const WORKFLOW_TYPES = {
  INCIDENT: "incident",
  SERVICE_REQUEST: "service_request",
  CHANGE: "change",
  PROBLEM: "problem",
  APPROVAL: "approval",
} as const;

// 模拟工作流数据
const mockWorkflows = [
  {
    id: 1,
    name: "事件处理流程",
    description: "标准事件处理和解决流程",
    type: WORKFLOW_TYPES.INCIDENT,
    status: WORKFLOW_STATUS.ACTIVE,
    version: "v1.2",
    createdBy: "系统管理员",
    createdAt: "2024-01-10 14:30",
    lastModified: "2024-01-15 09:15",
    stepsCount: 5,
    activeInstances: 23,
    completedInstances: 156,
  },
  {
    id: 2,
    name: "服务请求审批流程",
    description: "服务请求的多级审批流程",
    type: WORKFLOW_TYPES.SERVICE_REQUEST,
    status: WORKFLOW_STATUS.ACTIVE,
    version: "v2.0",
    createdBy: "流程管理员",
    createdAt: "2024-01-08 16:45",
    lastModified: "2024-01-14 11:20",
    stepsCount: 7,
    activeInstances: 12,
    completedInstances: 89,
  },
  {
    id: 3,
    name: "变更管理流程",
    description: "IT变更的评估、审批和实施流程",
    type: WORKFLOW_TYPES.CHANGE,
    status: WORKFLOW_STATUS.DRAFT,
    version: "v1.0",
    createdBy: "变更管理员",
    createdAt: "2024-01-12 10:00",
    lastModified: "2024-01-12 10:00",
    stepsCount: 8,
    activeInstances: 0,
    completedInstances: 0,
  },
];

// 工作流类型配置
const WORKFLOW_TYPE_CONFIG = {
  [WORKFLOW_TYPES.INCIDENT]: {
    label: "事件管理",
    color: "bg-red-100 text-red-800",
    icon: AlertCircle,
  },
  [WORKFLOW_TYPES.SERVICE_REQUEST]: {
    label: "服务请求",
    color: "bg-blue-100 text-blue-800",
    icon: Users,
  },
  [WORKFLOW_TYPES.CHANGE]: {
    label: "变更管理",
    color: "bg-orange-100 text-orange-800",
    icon: GitBranch,
  },
  [WORKFLOW_TYPES.PROBLEM]: {
    label: "问题管理",
    color: "bg-purple-100 text-purple-800",
    icon: Settings,
  },
  [WORKFLOW_TYPES.APPROVAL]: {
    label: "审批流程",
    color: "bg-green-100 text-green-800",
    icon: CheckCircle,
  },
};

// 工作流状态配置
const STATUS_CONFIG = {
  [WORKFLOW_STATUS.ACTIVE]: {
    label: "已启用",
    color: "bg-green-100 text-green-800",
    icon: CheckCircle,
  },
  [WORKFLOW_STATUS.INACTIVE]: {
    label: "已停用",
    color: "bg-gray-100 text-gray-800",
    icon: XCircle,
  },
  [WORKFLOW_STATUS.DRAFT]: {
    label: "草稿",
    color: "bg-yellow-100 text-yellow-800",
    icon: Clock,
  },
};

const WorkflowManagement = () => {
  const [workflows, setWorkflows] = useState(mockWorkflows);
  const [searchTerm, setSearchTerm] = useState("");
  const [typeFilter, setTypeFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedWorkflow, setSelectedWorkflow] = useState(null);

  // 过滤工作流
  const filteredWorkflows = workflows.filter((workflow) => {
    const matchesSearch = workflow.name
      .toLowerCase()
      .includes(searchTerm.toLowerCase());
    const matchesType = typeFilter === "all" || workflow.type === typeFilter;
    const matchesStatus =
      statusFilter === "all" || workflow.status === statusFilter;

    return matchesSearch && matchesType && matchesStatus;
  });

  // 处理工作流状态切换
  const handleStatusToggle = (workflowId: number) => {
    setWorkflows((prev) =>
      prev.map((workflow) => {
        if (workflow.id === workflowId) {
          const newStatus =
            workflow.status === WORKFLOW_STATUS.ACTIVE
              ? WORKFLOW_STATUS.INACTIVE
              : WORKFLOW_STATUS.ACTIVE;
          return { ...workflow, status: newStatus };
        }
        return workflow;
      })
    );
  };

  // 处理工作流复制
  const handleDuplicate = (workflow: any) => {
    const newWorkflow = {
      ...workflow,
      id: Math.max(...workflows.map((w) => w.id)) + 1,
      name: `${workflow.name} (副本)`,
      status: WORKFLOW_STATUS.DRAFT,
      version: "v1.0",
      createdAt: new Date().toLocaleString("zh-CN"),
      lastModified: new Date().toLocaleString("zh-CN"),
      activeInstances: 0,
      completedInstances: 0,
    };
    setWorkflows((prev) => [newWorkflow, ...prev]);
  };

  // 处理工作流删除
  const handleDelete = (workflowId: number) => {
    if (confirm("确定要删除这个工作流吗？此操作不可撤销。")) {
      setWorkflows((prev) => prev.filter((w) => w.id !== workflowId));
    }
  };

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">工作流管理</h1>
          <p className="text-gray-600 mt-1">
            设计和管理业务流程，配置审批节点和自动化规则
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 flex items-center gap-2"
        >
          <Plus className="w-4 h-4" />
          创建工作流
        </button>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">总工作流数</p>
              <p className="text-2xl font-bold text-gray-900">
                {workflows.length}
              </p>
            </div>
            <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center">
              <GitBranch className="w-6 h-6 text-blue-600" />
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">活跃工作流</p>
              <p className="text-2xl font-bold text-gray-900">
                {
                  workflows.filter((w) => w.status === WORKFLOW_STATUS.ACTIVE)
                    .length
                }
              </p>
            </div>
            <div className="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center">
              <CheckCircle className="w-6 h-6 text-green-600" />
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">运行中实例</p>
              <p className="text-2xl font-bold text-gray-900">
                {workflows.reduce((sum, w) => sum + w.activeInstances, 0)}
              </p>
            </div>
            <div className="w-12 h-12 bg-orange-100 rounded-lg flex items-center justify-center">
              <Play className="w-6 h-6 text-orange-600" />
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">已完成实例</p>
              <p className="text-2xl font-bold text-gray-900">
                {workflows.reduce((sum, w) => sum + w.completedInstances, 0)}
              </p>
            </div>
            <div className="w-12 h-12 bg-purple-100 rounded-lg flex items-center justify-center">
              <CheckCircle className="w-6 h-6 text-purple-600" />
            </div>
          </div>
        </div>
      </div>

      {/* 搜索和过滤 */}
      <div className="bg-white p-6 rounded-lg shadow-sm border">
        <div className="flex flex-col md:flex-row gap-4">
          {/* 搜索框 */}
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
            <input
              type="text"
              placeholder="搜索工作流名称..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>

          {/* 类型过滤 */}
          <div className="flex items-center gap-2">
            <Filter className="w-4 h-4 text-gray-400" />
            <select
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value)}
              className="border border-gray-300 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="all">所有类型</option>
              {Object.entries(WORKFLOW_TYPE_CONFIG).map(([key, config]) => (
                <option key={key} value={key}>
                  {config.label}
                </option>
              ))}
            </select>
          </div>

          {/* 状态过滤 */}
          <div className="flex items-center gap-2">
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="border border-gray-300 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            >
              <option value="all">所有状态</option>
              {Object.entries(STATUS_CONFIG).map(([key, config]) => (
                <option key={key} value={key}>
                  {config.label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* 工作流列表 */}
      <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  工作流信息
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  类型
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  步骤数
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  实例统计
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  最后修改
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {filteredWorkflows.map((workflow) => {
                const typeConfig = WORKFLOW_TYPE_CONFIG[workflow.type];
                const statusConfig = STATUS_CONFIG[workflow.status];
                const TypeIcon = typeConfig.icon;
                const StatusIcon = statusConfig.icon;

                return (
                  <tr key={workflow.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div>
                        <div className="text-sm font-medium text-gray-900">
                          {workflow.name}
                        </div>
                        <div className="text-sm text-gray-500">
                          {workflow.description}
                        </div>
                        <div className="text-xs text-gray-400 mt-1">
                          版本: {workflow.version} | 创建者:{" "}
                          {workflow.createdBy}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${typeConfig.color}`}
                      >
                        <TypeIcon className="w-3 h-3 mr-1" />
                        {typeConfig.label}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusConfig.color}`}
                      >
                        <StatusIcon className="w-3 h-3 mr-1" />
                        {statusConfig.label}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {workflow.stepsCount} 个步骤
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900">
                        运行中: {workflow.activeInstances}
                      </div>
                      <div className="text-sm text-gray-500">
                        已完成: {workflow.completedInstances}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {workflow.lastModified}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <div className="flex items-center justify-end gap-2">
                        {/* 启用/停用按钮 */}
                        <button
                          onClick={() => handleStatusToggle(workflow.id)}
                          className={`p-1 rounded hover:bg-gray-100 ${
                            workflow.status === WORKFLOW_STATUS.ACTIVE
                              ? "text-orange-600"
                              : "text-green-600"
                          }`}
                          title={
                            workflow.status === WORKFLOW_STATUS.ACTIVE
                              ? "停用工作流"
                              : "启用工作流"
                          }
                        >
                          {workflow.status === WORKFLOW_STATUS.ACTIVE ? (
                            <Pause className="w-4 h-4" />
                          ) : (
                            <Play className="w-4 h-4" />
                          )}
                        </button>

                        {/* 编辑按钮 */}
                        <button
                          onClick={() => setSelectedWorkflow(workflow)}
                          className="p-1 rounded hover:bg-gray-100 text-blue-600"
                          title="编辑工作流"
                        >
                          <Edit className="w-4 h-4" />
                        </button>

                        {/* 复制按钮 */}
                        <button
                          onClick={() => handleDuplicate(workflow)}
                          className="p-1 rounded hover:bg-gray-100 text-green-600"
                          title="复制工作流"
                        >
                          <Copy className="w-4 h-4" />
                        </button>

                        {/* 删除按钮 */}
                        <button
                          onClick={() => handleDelete(workflow.id)}
                          className="p-1 rounded hover:bg-gray-100 text-red-600"
                          title="删除工作流"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>

        {/* 空状态 */}
        {filteredWorkflows.length === 0 && (
          <div className="text-center py-12">
            <GitBranch className="mx-auto h-12 w-12 text-gray-400" />
            <h3 className="mt-2 text-sm font-medium text-gray-900">
              没有找到工作流
            </h3>
            <p className="mt-1 text-sm text-gray-500">
              {searchTerm || typeFilter !== "all" || statusFilter !== "all"
                ? "尝试调整搜索条件或过滤器"
                : "开始创建您的第一个工作流"}
            </p>
            {!searchTerm && typeFilter === "all" && statusFilter === "all" && (
              <div className="mt-6">
                <button
                  onClick={() => setShowCreateModal(true)}
                  className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 flex items-center gap-2 mx-auto"
                >
                  <Plus className="w-4 h-4" />
                  创建工作流
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      {/* 创建工作流模态框占位符 */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-medium text-gray-900 mb-4">
              创建新工作流
            </h3>
            <p className="text-gray-600 mb-4">
              工作流设计器功能正在开发中，敬请期待！
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setShowCreateModal(false)}
                className="px-4 py-2 text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200"
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 编辑工作流模态框占位符 */}
      {selectedWorkflow && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-medium text-gray-900 mb-4">
              编辑工作流: {selectedWorkflow.name}
            </h3>
            <p className="text-gray-600 mb-4">
              工作流编辑器功能正在开发中，敬请期待！
            </p>
            <div className="flex justify-end gap-3">
              <button
                onClick={() => setSelectedWorkflow(null)}
                className="px-4 py-2 text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200"
              >
                关闭
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default WorkflowManagement;
