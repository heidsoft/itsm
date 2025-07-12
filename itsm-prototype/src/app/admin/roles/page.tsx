"use client";

import React, { useState } from "react";
import {
  Plus,
  Search,
  Edit,
  Trash2,
  Users,
  Shield,
  Key,
  CheckCircle,
  XCircle,
  Settings,
  Eye,
  Lock,
  Unlock,
} from "lucide-react";

// 权限模块定义
const PERMISSION_MODULES = {
  DASHBOARD: "dashboard",
  TICKETS: "tickets",
  INCIDENTS: "incidents",
  PROBLEMS: "problems",
  CHANGES: "changes",
  SERVICE_CATALOG: "service_catalog",
  KNOWLEDGE_BASE: "knowledge_base",
  REPORTS: "reports",
  ADMIN: "admin",
  USERS: "users",
  ROLES: "roles",
  WORKFLOWS: "workflows",
  SYSTEM_CONFIG: "system_config",
} as const;

// 权限操作类型
const PERMISSION_ACTIONS = {
  VIEW: "view",
  CREATE: "create",
  EDIT: "edit",
  DELETE: "delete",
  APPROVE: "approve",
  ASSIGN: "assign",
  EXPORT: "export",
} as const;

// 模拟角色数据
const mockRoles = [
  {
    id: 1,
    name: "系统管理员",
    description: "拥有系统所有权限的超级管理员",
    userCount: 2,
    isSystem: true,
    isActive: true,
    createdAt: "2024-01-01 00:00",
    permissions: {
      [PERMISSION_MODULES.DASHBOARD]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.TICKETS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.INCIDENTS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.PROBLEMS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.CHANGES]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.SERVICE_CATALOG]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.KNOWLEDGE_BASE]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.REPORTS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.ADMIN]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.USERS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.ROLES]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.WORKFLOWS]: Object.values(PERMISSION_ACTIONS),
      [PERMISSION_MODULES.SYSTEM_CONFIG]: Object.values(PERMISSION_ACTIONS),
    },
  },
  {
    id: 2,
    name: "IT支持工程师",
    description: "处理日常IT支持请求和事件",
    userCount: 8,
    isSystem: false,
    isActive: true,
    createdAt: "2024-01-05 10:30",
    permissions: {
      [PERMISSION_MODULES.DASHBOARD]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.TICKETS]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.CREATE,
        PERMISSION_ACTIONS.EDIT,
        PERMISSION_ACTIONS.ASSIGN,
      ],
      [PERMISSION_MODULES.INCIDENTS]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.CREATE,
        PERMISSION_ACTIONS.EDIT,
      ],
      [PERMISSION_MODULES.SERVICE_CATALOG]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.KNOWLEDGE_BASE]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.CREATE,
        PERMISSION_ACTIONS.EDIT,
      ],
      [PERMISSION_MODULES.REPORTS]: [PERMISSION_ACTIONS.VIEW],
    },
  },
  {
    id: 3,
    name: "业务分析师",
    description: "分析业务需求和生成报告",
    userCount: 5,
    isSystem: false,
    isActive: true,
    createdAt: "2024-01-08 14:15",
    permissions: {
      [PERMISSION_MODULES.DASHBOARD]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.TICKETS]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.SERVICE_CATALOG]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.CREATE,
        PERMISSION_ACTIONS.EDIT,
      ],
      [PERMISSION_MODULES.KNOWLEDGE_BASE]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.REPORTS]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.CREATE,
        PERMISSION_ACTIONS.EXPORT,
      ],
    },
  },
  {
    id: 4,
    name: "普通用户",
    description: "只能提交和查看自己的请求",
    userCount: 150,
    isSystem: true,
    isActive: true,
    createdAt: "2024-01-01 00:00",
    permissions: {
      [PERMISSION_MODULES.DASHBOARD]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.TICKETS]: [
        PERMISSION_ACTIONS.VIEW,
        PERMISSION_ACTIONS.CREATE,
      ],
      [PERMISSION_MODULES.SERVICE_CATALOG]: [PERMISSION_ACTIONS.VIEW],
      [PERMISSION_MODULES.KNOWLEDGE_BASE]: [PERMISSION_ACTIONS.VIEW],
    },
  },
];

// 权限模块配置
const MODULE_CONFIG = {
  [PERMISSION_MODULES.DASHBOARD]: { label: "仪表盘", icon: "📊" },
  [PERMISSION_MODULES.TICKETS]: { label: "工单管理", icon: "🎫" },
  [PERMISSION_MODULES.INCIDENTS]: { label: "事件管理", icon: "🚨" },
  [PERMISSION_MODULES.PROBLEMS]: { label: "问题管理", icon: "🔧" },
  [PERMISSION_MODULES.CHANGES]: { label: "变更管理", icon: "🔄" },
  [PERMISSION_MODULES.SERVICE_CATALOG]: { label: "服务目录", icon: "📋" },
  [PERMISSION_MODULES.KNOWLEDGE_BASE]: { label: "知识库", icon: "📚" },
  [PERMISSION_MODULES.REPORTS]: { label: "报告分析", icon: "📈" },
  [PERMISSION_MODULES.ADMIN]: { label: "系统管理", icon: "⚙️" },
  [PERMISSION_MODULES.USERS]: { label: "用户管理", icon: "👥" },
  [PERMISSION_MODULES.ROLES]: { label: "角色管理", icon: "🛡️" },
  [PERMISSION_MODULES.WORKFLOWS]: { label: "工作流", icon: "🔀" },
  [PERMISSION_MODULES.SYSTEM_CONFIG]: { label: "系统配置", icon: "🔧" },
};

// 权限操作配置
const ACTION_CONFIG = {
  [PERMISSION_ACTIONS.VIEW]: { label: "查看", color: "text-blue-600" },
  [PERMISSION_ACTIONS.CREATE]: { label: "创建", color: "text-green-600" },
  [PERMISSION_ACTIONS.EDIT]: { label: "编辑", color: "text-yellow-600" },
  [PERMISSION_ACTIONS.DELETE]: { label: "删除", color: "text-red-600" },
  [PERMISSION_ACTIONS.APPROVE]: { label: "审批", color: "text-purple-600" },
  [PERMISSION_ACTIONS.ASSIGN]: { label: "分配", color: "text-indigo-600" },
  [PERMISSION_ACTIONS.EXPORT]: { label: "导出", color: "text-gray-600" },
};

const RoleManagement = () => {
  // 状态管理
  const [roles, setRoles] = useState(mockRoles);
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [showModal, setShowModal] = useState(false);
  const [selectedRole, setSelectedRole] = useState(null);
  const [showPermissionModal, setShowPermissionModal] = useState(false);

  // 过滤角色
  const filteredRoles = roles.filter((role) => {
    const matchesSearch = role.name
      .toLowerCase()
      .includes(searchTerm.toLowerCase());
    const matchesStatus =
      statusFilter === "all" ||
      (statusFilter === "active" && role.isActive) ||
      (statusFilter === "inactive" && !role.isActive);
    return matchesSearch && matchesStatus;
  });

  // 处理角色状态切换
  const handleToggleStatus = (roleId) => {
    setRoles(
      roles.map((role) =>
        role.id === roleId ? { ...role, isActive: !role.isActive } : role
      )
    );
  };

  // 处理删除角色
  const handleDeleteRole = (roleId) => {
    if (window.confirm("确定要删除这个角色吗？")) {
      setRoles(roles.filter((role) => role.id !== roleId));
    }
  };

  // 处理编辑角色
  const handleEditRole = (role) => {
    setSelectedRole(role);
    setShowModal(true);
  };

  // 查看权限详情
  const handleViewPermissions = (role) => {
    setSelectedRole(role);
    setShowPermissionModal(true);
  };

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">角色权限管理</h1>
          <p className="text-gray-600 mt-1">
            管理系统角色和权限分配，控制用户访问范围
          </p>
        </div>
        <button
          onClick={() => {
            setSelectedRole(null);
            setShowModal(true);
          }}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors flex items-center gap-2"
        >
          <Plus className="w-4 h-4" />
          新建角色
        </button>
      </div>

      {/* 搜索和过滤 */}
      <div className="bg-white p-4 rounded-lg shadow-sm border">
        <div className="flex flex-col sm:flex-row gap-4">
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
            <input
              type="text"
              placeholder="搜索角色名称..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="all">全部状态</option>
            <option value="active">启用</option>
            <option value="inactive">禁用</option>
          </select>
        </div>
      </div>

      {/* 角色统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">总角色数</p>
              <p className="text-2xl font-bold text-gray-900">{roles.length}</p>
            </div>
            <Shield className="w-8 h-8 text-blue-600" />
          </div>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">启用角色</p>
              <p className="text-2xl font-bold text-green-600">
                {roles.filter((r) => r.isActive).length}
              </p>
            </div>
            <CheckCircle className="w-8 h-8 text-green-600" />
          </div>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">系统角色</p>
              <p className="text-2xl font-bold text-purple-600">
                {roles.filter((r) => r.isSystem).length}
              </p>
            </div>
            <Key className="w-8 h-8 text-purple-600" />
          </div>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">关联用户</p>
              <p className="text-2xl font-bold text-orange-600">
                {roles.reduce((sum, role) => sum + role.userCount, 0)}
              </p>
            </div>
            <Users className="w-8 h-8 text-orange-600" />
          </div>
        </div>
      </div>

      {/* 角色列表 */}
      <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  角色信息
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  用户数量
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  类型
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  创建时间
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {filteredRoles.map((role) => (
                <tr key={role.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div>
                      <div className="text-sm font-medium text-gray-900">
                        {role.name}
                      </div>
                      <div className="text-sm text-gray-500">
                        {role.description}
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <Users className="w-4 h-4 text-gray-400 mr-2" />
                      <span className="text-sm text-gray-900">
                        {role.userCount}
                      </span>
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        role.isActive
                          ? "bg-green-100 text-green-800"
                          : "bg-red-100 text-red-800"
                      }`}
                    >
                      {role.isActive ? (
                        <>
                          <CheckCircle className="w-3 h-3 mr-1" />
                          启用
                        </>
                      ) : (
                        <>
                          <XCircle className="w-3 h-3 mr-1" />
                          禁用
                        </>
                      )}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span
                      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        role.isSystem
                          ? "bg-purple-100 text-purple-800"
                          : "bg-blue-100 text-blue-800"
                      }`}
                    >
                      {role.isSystem ? (
                        <>
                          <Lock className="w-3 h-3 mr-1" />
                          系统角色
                        </>
                      ) : (
                        <>
                          <Unlock className="w-3 h-3 mr-1" />
                          自定义角色
                        </>
                      )}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {role.createdAt}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <div className="flex items-center justify-end gap-2">
                      <button
                        onClick={() => handleViewPermissions(role)}
                        className="text-blue-600 hover:text-blue-900 p-1 rounded"
                        title="查看权限"
                      >
                        <Eye className="w-4 h-4" />
                      </button>
                      {!role.isSystem && (
                        <>
                          <button
                            onClick={() => handleEditRole(role)}
                            className="text-indigo-600 hover:text-indigo-900 p-1 rounded"
                            title="编辑角色"
                          >
                            <Edit className="w-4 h-4" />
                          </button>
                          <button
                            onClick={() => handleToggleStatus(role.id)}
                            className={`p-1 rounded ${
                              role.isActive
                                ? "text-red-600 hover:text-red-900"
                                : "text-green-600 hover:text-green-900"
                            }`}
                            title={role.isActive ? "禁用角色" : "启用角色"}
                          >
                            {role.isActive ? (
                              <XCircle className="w-4 h-4" />
                            ) : (
                              <CheckCircle className="w-4 h-4" />
                            )}
                          </button>
                          <button
                            onClick={() => handleDeleteRole(role.id)}
                            className="text-red-600 hover:text-red-900 p-1 rounded"
                            title="删除角色"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* 权限详情模态框 */}
      {showPermissionModal && selectedRole && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-4xl max-h-[80vh] overflow-y-auto">
            <div className="flex justify-between items-center mb-6">
              <h3 className="text-lg font-semibold text-gray-900">
                {selectedRole.name} - 权限详情
              </h3>
              <button
                onClick={() => setShowPermissionModal(false)}
                className="text-gray-400 hover:text-gray-600"
              >
                <XCircle className="w-6 h-6" />
              </button>
            </div>

            <div className="space-y-6">
              {Object.entries(MODULE_CONFIG).map(([moduleKey, moduleInfo]) => {
                const permissions = selectedRole.permissions[moduleKey] || [];
                if (permissions.length === 0) return null;

                return (
                  <div key={moduleKey} className="border rounded-lg p-4">
                    <div className="flex items-center gap-2 mb-3">
                      <span className="text-lg">{moduleInfo.icon}</span>
                      <h4 className="font-medium text-gray-900">
                        {moduleInfo.label}
                      </h4>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {permissions.map((action) => (
                        <span
                          key={action}
                          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800`}
                        >
                          {ACTION_CONFIG[action]?.label || action}
                        </span>
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>

            <div className="mt-6 flex justify-end">
              <button
                onClick={() => setShowPermissionModal(false)}
                className="px-4 py-2 bg-gray-300 text-gray-700 rounded-lg hover:bg-gray-400 transition-colors"
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

export default RoleManagement;
