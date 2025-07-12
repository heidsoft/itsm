"use client";

import React, { useState } from "react";
import {
  Plus,
  Search,
  Edit,
  Trash2,
  Play,
  Pause,
  Copy,
  Settings,
  Users,
  CheckCircle,
  XCircle,
  Clock,
  ArrowRight,
  ArrowDown,
  UserCheck,
  AlertTriangle,
  FileText,
  GitBranch,
} from "lucide-react";

// 审批链状态枚举
const APPROVAL_CHAIN_STATUS = {
  ACTIVE: "active",
  INACTIVE: "inactive",
  DRAFT: "draft",
} as const;

// 审批类型枚举
const APPROVAL_TYPES = {
  SEQUENTIAL: "sequential", // 顺序审批
  PARALLEL: "parallel", // 并行审批
  CONDITIONAL: "conditional", // 条件审批
  ESCALATION: "escalation", // 升级审批
} as const;

// 审批节点类型
const NODE_TYPES = {
  USER: "user", // 用户审批
  ROLE: "role", // 角色审批
  GROUP: "group", // 用户组审批
  AUTO: "auto", // 自动审批
} as const;

// 模拟审批链数据
const mockApprovalChains = [
  {
    id: 1,
    name: "服务请求审批链",
    description: "标准服务请求的多级审批流程",
    type: APPROVAL_TYPES.SEQUENTIAL,
    status: APPROVAL_CHAIN_STATUS.ACTIVE,
    category: "服务管理",
    createdBy: "系统管理员",
    createdAt: "2024-01-10 14:30",
    lastModified: "2024-01-15 09:15",
    nodesCount: 4,
    usageCount: 156,
    avgApprovalTime: "2.5小时",
    nodes: [
      {
        id: 1,
        name: "直属主管审批",
        type: NODE_TYPES.ROLE,
        approver: "直属主管",
        condition: "金额 < 5000",
        timeout: 24,
        isRequired: true,
      },
      {
        id: 2,
        name: "部门经理审批",
        type: NODE_TYPES.ROLE,
        approver: "部门经理",
        condition: "金额 >= 5000",
        timeout: 48,
        isRequired: true,
      },
      {
        id: 3,
        name: "IT部门审批",
        type: NODE_TYPES.GROUP,
        approver: "IT部门",
        condition: "涉及IT资源",
        timeout: 72,
        isRequired: false,
      },
      {
        id: 4,
        name: "财务审批",
        type: NODE_TYPES.ROLE,
        approver: "财务经理",
        condition: "金额 >= 10000",
        timeout: 48,
        isRequired: true,
      },
    ],
  },
  {
    id: 2,
    name: "变更管理审批链",
    description: "IT变更的风险评估和审批流程",
    type: APPROVAL_TYPES.CONDITIONAL,
    status: APPROVAL_CHAIN_STATUS.ACTIVE,
    category: "变更管理",
    createdBy: "变更管理员",
    createdAt: "2024-01-08 16:45",
    lastModified: "2024-01-14 11:20",
    nodesCount: 5,
    usageCount: 89,
    avgApprovalTime: "4.2小时",
    nodes: [
      {
        id: 1,
        name: "变更发起人确认",
        type: NODE_TYPES.USER,
        approver: "变更发起人",
        condition: "自动",
        timeout: 12,
        isRequired: true,
      },
      {
        id: 2,
        name: "技术负责人审批",
        type: NODE_TYPES.ROLE,
        approver: "技术负责人",
        condition: "技术变更",
        timeout: 24,
        isRequired: true,
      },
      {
        id: 3,
        name: "CAB委员会审批",
        type: NODE_TYPES.GROUP,
        approver: "CAB委员会",
        condition: "高风险变更",
        timeout: 72,
        isRequired: true,
      },
    ],
  },
  {
    id: 3,
    name: "采购申请审批链",
    description: "设备和软件采购的审批流程",
    type: APPROVAL_TYPES.PARALLEL,
    status: APPROVAL_CHAIN_STATUS.DRAFT,
    category: "采购管理",
    createdBy: "采购管理员",
    createdAt: "2024-01-12 10:00",
    lastModified: "2024-01-12 10:00",
    nodesCount: 3,
    usageCount: 0,
    avgApprovalTime: "0小时",
    nodes: [
      {
        id: 1,
        name: "预算审批",
        type: NODE_TYPES.ROLE,
        approver: "财务经理",
        condition: "并行",
        timeout: 48,
        isRequired: true,
      },
      {
        id: 2,
        name: "技术审批",
        type: NODE_TYPES.ROLE,
        approver: "技术经理",
        condition: "并行",
        timeout: 48,
        isRequired: true,
      },
    ],
  },
];

// 审批类型配置
const APPROVAL_TYPE_CONFIG = {
  [APPROVAL_TYPES.SEQUENTIAL]: {
    label: "顺序审批",
    color: "bg-blue-100 text-blue-800",
    icon: ArrowRight,
    description: "按顺序逐级审批",
  },
  [APPROVAL_TYPES.PARALLEL]: {
    label: "并行审批",
    color: "bg-green-100 text-green-800",
    icon: GitBranch,
    description: "多个审批人同时审批",
  },
  [APPROVAL_TYPES.CONDITIONAL]: {
    label: "条件审批",
    color: "bg-purple-100 text-purple-800",
    icon: Settings,
    description: "根据条件动态审批",
  },
  [APPROVAL_TYPES.ESCALATION]: {
    label: "升级审批",
    color: "bg-orange-100 text-orange-800",
    icon: AlertTriangle,
    description: "超时自动升级审批",
  },
};

// 状态配置
const STATUS_CONFIG = {
  [APPROVAL_CHAIN_STATUS.ACTIVE]: {
    label: "已启用",
    color: "bg-green-100 text-green-800",
    icon: CheckCircle,
  },
  [APPROVAL_CHAIN_STATUS.INACTIVE]: {
    label: "已停用",
    color: "bg-gray-100 text-gray-800",
    icon: XCircle,
  },
  [APPROVAL_CHAIN_STATUS.DRAFT]: {
    label: "草稿",
    color: "bg-yellow-100 text-yellow-800",
    icon: Clock,
  },
};

// 节点类型配置
const NODE_TYPE_CONFIG = {
  [NODE_TYPES.USER]: {
    label: "用户审批",
    color: "bg-blue-100 text-blue-800",
    icon: UserCheck,
  },
  [NODE_TYPES.ROLE]: {
    label: "角色审批",
    color: "bg-purple-100 text-purple-800",
    icon: Users,
  },
  [NODE_TYPES.GROUP]: {
    label: "用户组审批",
    color: "bg-green-100 text-green-800",
    icon: Users,
  },
  [NODE_TYPES.AUTO]: {
    label: "自动审批",
    color: "bg-gray-100 text-gray-800",
    icon: Settings,
  },
};

const ApprovalChainManagement = () => {
  const [approvalChains, setApprovalChains] = useState(mockApprovalChains);
  const [searchTerm, setSearchTerm] = useState("");
  const [typeFilter, setTypeFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [categoryFilter, setCategoryFilter] = useState("all");
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedChain, setSelectedChain] = useState(null);
  const [showDetailModal, setShowDetailModal] = useState(false);

  // 过滤审批链
  const filteredChains = approvalChains.filter((chain) => {
    const matchesSearch =
      chain.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      chain.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesType = typeFilter === "all" || chain.type === typeFilter;
    const matchesStatus =
      statusFilter === "all" || chain.status === statusFilter;
    const matchesCategory =
      categoryFilter === "all" || chain.category === categoryFilter;

    return matchesSearch && matchesType && matchesStatus && matchesCategory;
  });

  // 获取所有分类
  const categories = Array.from(
    new Set(approvalChains.map((chain) => chain.category))
  );

  // 处理状态切换
  const handleStatusToggle = (chainId) => {
    setApprovalChains((prev) =>
      prev.map((chain) => {
        if (chain.id === chainId) {
          const newStatus =
            chain.status === APPROVAL_CHAIN_STATUS.ACTIVE
              ? APPROVAL_CHAIN_STATUS.INACTIVE
              : APPROVAL_CHAIN_STATUS.ACTIVE;
          return { ...chain, status: newStatus };
        }
        return chain;
      })
    );
  };

  // 处理复制
  const handleDuplicate = (chain) => {
    const newChain = {
      ...chain,
      id: Math.max(...approvalChains.map((c) => c.id)) + 1,
      name: `${chain.name} (副本)`,
      status: APPROVAL_CHAIN_STATUS.DRAFT,
      createdAt: new Date().toLocaleString("zh-CN"),
      lastModified: new Date().toLocaleString("zh-CN"),
      usageCount: 0,
    };
    setApprovalChains((prev) => [newChain, ...prev]);
  };

  // 处理删除
  const handleDelete = (chainId) => {
    if (confirm("确定要删除这个审批链吗？此操作不可撤销。")) {
      setApprovalChains((prev) => prev.filter((c) => c.id !== chainId));
    }
  };

  // 查看详情
  const handleViewDetail = (chain) => {
    setSelectedChain(chain);
    setShowDetailModal(true);
  };

  // 统计信息
  const stats = {
    total: approvalChains.length,
    active: approvalChains.filter(
      (c) => c.status === APPROVAL_CHAIN_STATUS.ACTIVE
    ).length,
    draft: approvalChains.filter(
      (c) => c.status === APPROVAL_CHAIN_STATUS.DRAFT
    ).length,
    totalUsage: approvalChains.reduce((sum, c) => sum + c.usageCount, 0),
  };

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 flex items-center">
              <FileText className="w-8 h-8 text-blue-600 mr-3" />
              审批链配置
            </h1>
            <p className="text-gray-600 mt-2">
              设计和管理审批流程，配置多级审批节点和条件规则
            </p>
          </div>
          <button
            onClick={() => setShowCreateModal(true)}
            className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
          >
            <Plus className="w-4 h-4 mr-2" />
            新建审批链
          </button>
        </div>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <div className="flex items-center">
            <div className="p-3 bg-blue-100 rounded-lg">
              <FileText className="w-6 h-6 text-blue-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">审批链总数</p>
              <p className="text-2xl font-bold text-gray-900">{stats.total}</p>
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <div className="flex items-center">
            <div className="p-3 bg-green-100 rounded-lg">
              <CheckCircle className="w-6 h-6 text-green-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">已启用</p>
              <p className="text-2xl font-bold text-gray-900">{stats.active}</p>
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <div className="flex items-center">
            <div className="p-3 bg-yellow-100 rounded-lg">
              <Clock className="w-6 h-6 text-yellow-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">草稿状态</p>
              <p className="text-2xl font-bold text-gray-900">{stats.draft}</p>
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <div className="flex items-center">
            <div className="p-3 bg-purple-100 rounded-lg">
              <Users className="w-6 h-6 text-purple-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">总使用次数</p>
              <p className="text-2xl font-bold text-gray-900">
                {stats.totalUsage}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* 搜索和过滤 */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
            <input
              type="text"
              placeholder="搜索审批链..."
              className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>
          <select
            className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value)}
          >
            <option value="all">所有类型</option>
            {Object.entries(APPROVAL_TYPE_CONFIG).map(([key, config]) => (
              <option key={key} value={key}>
                {config.label}
              </option>
            ))}
          </select>
          <select
            className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="all">所有状态</option>
            {Object.entries(STATUS_CONFIG).map(([key, config]) => (
              <option key={key} value={key}>
                {config.label}
              </option>
            ))}
          </select>
          <select
            className="px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            value={categoryFilter}
            onChange={(e) => setCategoryFilter(e.target.value)}
          >
            <option value="all">所有分类</option>
            {categories.map((category) => (
              <option key={category} value={category}>
                {category}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* 审批链列表 */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="px-6 py-4 border-b border-gray-200">
          <h3 className="text-lg font-semibold text-gray-900">
            审批链列表 ({filteredChains.length})
          </h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  审批链信息
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  类型
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  节点数
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  使用次数
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  平均审批时间
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {filteredChains.map((chain) => {
                const typeConfig = APPROVAL_TYPE_CONFIG[chain.type];
                const statusConfig = STATUS_CONFIG[chain.status];
                const TypeIcon = typeConfig.icon;
                const StatusIcon = statusConfig.icon;

                return (
                  <tr key={chain.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4">
                      <div>
                        <div className="text-sm font-medium text-gray-900">
                          {chain.name}
                        </div>
                        <div className="text-sm text-gray-500">
                          {chain.description}
                        </div>
                        <div className="text-xs text-gray-400 mt-1">
                          分类: {chain.category} | 创建者: {chain.createdBy}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${typeConfig.color}`}
                      >
                        <TypeIcon className="w-3 h-3 mr-1" />
                        {typeConfig.label}
                      </span>
                    </td>
                    <td className="px-6 py-4">
                      <span
                        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusConfig.color}`}
                      >
                        <StatusIcon className="w-3 h-3 mr-1" />
                        {statusConfig.label}
                      </span>
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900">
                      {chain.nodesCount} 个节点
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900">
                      {chain.usageCount} 次
                    </td>
                    <td className="px-6 py-4 text-sm text-gray-900">
                      {chain.avgApprovalTime}
                    </td>
                    <td className="px-6 py-4">
                      <div className="flex items-center space-x-2">
                        <button
                          onClick={() => handleViewDetail(chain)}
                          className="text-blue-600 hover:text-blue-900 text-sm"
                          title="查看详情"
                        >
                          <Settings className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleStatusToggle(chain.id)}
                          className={`text-sm ${
                            chain.status === APPROVAL_CHAIN_STATUS.ACTIVE
                              ? "text-red-600 hover:text-red-900"
                              : "text-green-600 hover:text-green-900"
                          }`}
                          title={
                            chain.status === APPROVAL_CHAIN_STATUS.ACTIVE
                              ? "停用"
                              : "启用"
                          }
                        >
                          {chain.status === APPROVAL_CHAIN_STATUS.ACTIVE ? (
                            <Pause className="w-4 h-4" />
                          ) : (
                            <Play className="w-4 h-4" />
                          )}
                        </button>
                        <button
                          onClick={() => handleDuplicate(chain)}
                          className="text-gray-600 hover:text-gray-900 text-sm"
                          title="复制"
                        >
                          <Copy className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleDelete(chain.id)}
                          className="text-red-600 hover:text-red-900 text-sm"
                          title="删除"
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
      </div>

      {/* 审批链详情模态框 */}
      {showDetailModal && selectedChain && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-4xl w-full mx-4 max-h-[90vh] overflow-y-auto">
            <div className="px-6 py-4 border-b border-gray-200">
              <div className="flex justify-between items-center">
                <h3 className="text-lg font-semibold text-gray-900">
                  审批链详情: {selectedChain.name}
                </h3>
                <button
                  onClick={() => setShowDetailModal(false)}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <XCircle className="w-6 h-6" />
                </button>
              </div>
            </div>
            <div className="p-6">
              {/* 基本信息 */}
              <div className="mb-6">
                <h4 className="text-md font-medium text-gray-900 mb-3">
                  基本信息
                </h4>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <span className="text-gray-500">审批类型:</span>
                    <span className="ml-2 text-gray-900">
                      {APPROVAL_TYPE_CONFIG[selectedChain.type].label}
                    </span>
                  </div>
                  <div>
                    <span className="text-gray-500">状态:</span>
                    <span className="ml-2 text-gray-900">
                      {STATUS_CONFIG[selectedChain.status].label}
                    </span>
                  </div>
                  <div>
                    <span className="text-gray-500">分类:</span>
                    <span className="ml-2 text-gray-900">
                      {selectedChain.category}
                    </span>
                  </div>
                  <div>
                    <span className="text-gray-500">使用次数:</span>
                    <span className="ml-2 text-gray-900">
                      {selectedChain.usageCount} 次
                    </span>
                  </div>
                </div>
              </div>

              {/* 审批节点 */}
              <div>
                <h4 className="text-md font-medium text-gray-900 mb-3">
                  审批节点
                </h4>
                <div className="space-y-4">
                  {selectedChain.nodes.map((node, index) => {
                    const nodeConfig = NODE_TYPE_CONFIG[node.type];
                    const NodeIcon = nodeConfig.icon;

                    return (
                      <div key={node.id} className="flex items-center">
                        <div className="flex-shrink-0">
                          <div className="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center">
                            <span className="text-blue-600 text-sm font-medium">
                              {index + 1}
                            </span>
                          </div>
                        </div>
                        {index < selectedChain.nodes.length - 1 && (
                          <div className="flex-shrink-0 mx-4">
                            <ArrowDown className="w-4 h-4 text-gray-400" />
                          </div>
                        )}
                        <div className="flex-1 bg-gray-50 rounded-lg p-4">
                          <div className="flex items-center justify-between mb-2">
                            <h5 className="text-sm font-medium text-gray-900">
                              {node.name}
                            </h5>
                            <span
                              className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${nodeConfig.color}`}
                            >
                              <NodeIcon className="w-3 h-3 mr-1" />
                              {nodeConfig.label}
                            </span>
                          </div>
                          <div className="grid grid-cols-2 gap-4 text-xs text-gray-600">
                            <div>
                              <span className="font-medium">审批人:</span>{" "}
                              {node.approver}
                            </div>
                            <div>
                              <span className="font-medium">超时时间:</span>{" "}
                              {node.timeout}小时
                            </div>
                            <div>
                              <span className="font-medium">触发条件:</span>{" "}
                              {node.condition}
                            </div>
                            <div>
                              <span className="font-medium">是否必需:</span>
                              <span
                                className={
                                  node.isRequired
                                    ? "text-red-600"
                                    : "text-green-600"
                                }
                              >
                                {node.isRequired ? "必需" : "可选"}
                              </span>
                            </div>
                          </div>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ApprovalChainManagement;
