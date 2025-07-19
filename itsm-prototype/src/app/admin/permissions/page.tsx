"use client";

import { CheckCircle, Users, Search, Settings, XCircle, Key, Shield, RefreshCw, Save } from 'lucide-react';

import React, { useState } from "react";
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

// 权限模块配置
const MODULE_CONFIG = {
  [PERMISSION_MODULES.DASHBOARD]: {
    label: "仪表盘",
    icon: "📊",
    description: "系统仪表盘和概览信息",
    category: "核心功能",
  },
  [PERMISSION_MODULES.TICKETS]: {
    label: "工单管理",
    icon: "🎫",
    description: "工单的创建、处理和管理",
    category: "核心功能",
  },
  [PERMISSION_MODULES.INCIDENTS]: {
    label: "事件管理",
    icon: "🚨",
    description: "IT事件的记录和处理",
    category: "核心功能",
  },
  [PERMISSION_MODULES.PROBLEMS]: {
    label: "问题管理",
    icon: "🔧",
    description: "根本原因分析和问题解决",
    category: "核心功能",
  },
  [PERMISSION_MODULES.CHANGES]: {
    label: "变更管理",
    icon: "🔄",
    description: "IT变更的规划和实施",
    category: "核心功能",
  },
  [PERMISSION_MODULES.SERVICE_CATALOG]: {
    label: "服务目录",
    icon: "📋",
    description: "IT服务目录管理",
    category: "服务管理",
  },
  [PERMISSION_MODULES.KNOWLEDGE_BASE]: {
    label: "知识库",
    icon: "📚",
    description: "知识文档和解决方案",
    category: "服务管理",
  },
  [PERMISSION_MODULES.REPORTS]: {
    label: "报告分析",
    icon: "📈",
    description: "数据报告和分析功能",
    category: "分析工具",
  },
  [PERMISSION_MODULES.ADMIN]: {
    label: "系统管理",
    icon: "⚙️",
    description: "系统管理和配置",
    category: "系统管理",
  },
  [PERMISSION_MODULES.USERS]: {
    label: "用户管理",
    icon: "👥",
    description: "用户账户管理",
    category: "系统管理",
  },
  [PERMISSION_MODULES.ROLES]: {
    label: "角色管理",
    icon: "🛡️",
    description: "角色和权限管理",
    category: "系统管理",
  },
  [PERMISSION_MODULES.WORKFLOWS]: {
    label: "工作流",
    icon: "🔀",
    description: "业务流程配置",
    category: "系统管理",
  },
  [PERMISSION_MODULES.SYSTEM_CONFIG]: {
    label: "系统配置",
    icon: "🔧",
    description: "系统参数和设置",
    category: "系统管理",
  },
};

// 权限操作配置
const ACTION_CONFIG = {
  [PERMISSION_ACTIONS.VIEW]: {
    label: "查看",
    color: "text-blue-600",
    bgColor: "bg-blue-50",
    description: "查看和浏览权限",
  },
  [PERMISSION_ACTIONS.CREATE]: {
    label: "创建",
    color: "text-green-600",
    bgColor: "bg-green-50",
    description: "创建新记录权限",
  },
  [PERMISSION_ACTIONS.EDIT]: {
    label: "编辑",
    color: "text-yellow-600",
    bgColor: "bg-yellow-50",
    description: "修改现有记录权限",
  },
  [PERMISSION_ACTIONS.DELETE]: {
    label: "删除",
    color: "text-red-600",
    bgColor: "bg-red-50",
    description: "删除记录权限",
  },
  [PERMISSION_ACTIONS.APPROVE]: {
    label: "审批",
    color: "text-purple-600",
    bgColor: "bg-purple-50",
    description: "审批和批准权限",
  },
  [PERMISSION_ACTIONS.ASSIGN]: {
    label: "分配",
    color: "text-indigo-600",
    bgColor: "bg-indigo-50",
    description: "分配和指派权限",
  },
  [PERMISSION_ACTIONS.EXPORT]: {
    label: "导出",
    color: "text-gray-600",
    bgColor: "bg-gray-50",
    description: "数据导出权限",
  },
};

// 模拟权限配置数据
const mockPermissionConfig = {
  modules: Object.values(PERMISSION_MODULES).map((moduleKey) => ({
    id: moduleKey,
    name: MODULE_CONFIG[moduleKey].label,
    description: MODULE_CONFIG[moduleKey].description,
    category: MODULE_CONFIG[moduleKey].category,
    icon: MODULE_CONFIG[moduleKey].icon,
    isEnabled: true,
    actions: Object.values(PERMISSION_ACTIONS).map((actionKey) => ({
      id: actionKey,
      name: ACTION_CONFIG[actionKey].label,
      description: ACTION_CONFIG[actionKey].description,
      isEnabled: true,
    })),
  })),
  categories: [
    { id: "core", name: "核心功能", description: "系统核心业务功能" },
    { id: "service", name: "服务管理", description: "IT服务相关功能" },
    { id: "analysis", name: "分析工具", description: "数据分析和报告" },
    { id: "system", name: "系统管理", description: "系统配置和管理" },
  ],
};

const PermissionConfiguration = () => {
  const [permissionConfig, setPermissionConfig] =
    useState(mockPermissionConfig);
  const [searchTerm, setSearchTerm] = useState("");
  const [categoryFilter, setCategoryFilter] = useState("all");
  const [showModuleModal, setShowModuleModal] = useState(false);
  const [selectedModule, setSelectedModule] = useState(null);
  const [showActionModal, setShowActionModal] = useState(false);
  const [selectedAction, setSelectedAction] = useState(null);

  // 过滤模块
  const filteredModules = permissionConfig.modules.filter((module) => {
    const matchesSearch =
      module.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      module.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory =
      categoryFilter === "all" || module.category === categoryFilter;
    return matchesSearch && matchesCategory;
  });

  // 按分类分组模块
  const modulesByCategory = filteredModules.reduce((acc, module) => {
    if (!acc[module.category]) {
      acc[module.category] = [];
    }
    acc[module.category].push(module);
    return acc;
  }, {});

  // 处理模块状态切换
  const handleToggleModule = (moduleId) => {
    setPermissionConfig((prev) => ({
      ...prev,
      modules: prev.modules.map((module) =>
        module.id === moduleId
          ? { ...module, isEnabled: !module.isEnabled }
          : module
      ),
    }));
  };

  // 处理操作状态切换
  const handleToggleAction = (moduleId, actionId) => {
    setPermissionConfig((prev) => ({
      ...prev,
      modules: prev.modules.map((module) =>
        module.id === moduleId
          ? {
              ...module,
              actions: module.actions.map((action) =>
                action.id === actionId
                  ? { ...action, isEnabled: !action.isEnabled }
                  : action
              ),
            }
          : module
      ),
    }));
  };

  // 统计信息
  const stats = {
    totalModules: permissionConfig.modules.length,
    enabledModules: permissionConfig.modules.filter((m) => m.isEnabled).length,
    totalActions: permissionConfig.modules.reduce(
      (sum, m) => sum + m.actions.length,
      0
    ),
    enabledActions: permissionConfig.modules.reduce(
      (sum, m) => sum + m.actions.filter((a) => a.isEnabled).length,
      0
    ),
  };

  return (
    <div className="space-y-6">
      {/* 页面头部 */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <div className="flex justify-between items-start">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 flex items-center">
              <Shield className="w-8 h-8 text-blue-600 mr-3" />
              权限配置
            </h1>
            <p className="text-gray-600 mt-2">
              管理系统权限模块和操作，配置细粒度的访问控制
            </p>
          </div>
          <div className="flex space-x-3">
            <button className="flex items-center px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-colors">
              <RefreshCw className="w-4 h-4 mr-2" />
              重置配置
            </button>
            <button className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors">
              <Save className="w-4 h-4 mr-2" />
              保存配置
            </button>
          </div>
        </div>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <div className="flex items-center">
            <div className="p-3 bg-blue-100 rounded-lg">
              <Shield className="w-6 h-6 text-blue-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">权限模块</p>
              <p className="text-2xl font-bold text-gray-900">
                {stats.enabledModules}/{stats.totalModules}
              </p>
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <div className="flex items-center">
            <div className="p-3 bg-green-100 rounded-lg">
              <Key className="w-6 h-6 text-green-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">权限操作</p>
              <p className="text-2xl font-bold text-gray-900">
                {stats.enabledActions}/{stats.totalActions}
              </p>
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <div className="flex items-center">
            <div className="p-3 bg-purple-100 rounded-lg">
              <Settings className="w-6 h-6 text-purple-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">功能分类</p>
              <p className="text-2xl font-bold text-gray-900">
                {permissionConfig.categories.length}
              </p>
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
          <div className="flex items-center">
            <div className="p-3 bg-yellow-100 rounded-lg">
              <Users className="w-6 h-6 text-yellow-600" />
            </div>
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-600">配置完整度</p>
              <p className="text-2xl font-bold text-gray-900">
                {Math.round((stats.enabledActions / stats.totalActions) * 100)}%
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* 搜索和过滤 */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-6">
        <div className="flex flex-col sm:flex-row gap-4">
          <div className="flex-1">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
              <input
                type="text"
                placeholder="搜索权限模块..."
                className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </div>
          </div>
          <div className="sm:w-48">
            <select
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              value={categoryFilter}
              onChange={(e) => setCategoryFilter(e.target.value)}
            >
              <option value="all">所有分类</option>
              {permissionConfig.categories.map((category) => (
                <option key={category.id} value={category.name}>
                  {category.name}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>

      {/* 权限模块列表 */}
      <div className="space-y-6">
        {Object.entries(modulesByCategory).map(([category, modules]) => (
          <div
            key={category}
            className="bg-white rounded-lg shadow-sm border border-gray-200"
          >
            <div className="px-6 py-4 border-b border-gray-200">
              <h3 className="text-lg font-semibold text-gray-900">
                {category}
              </h3>
              <p className="text-sm text-gray-600 mt-1">
                {
                  permissionConfig.categories.find((c) => c.name === category)
                    ?.description
                }
              </p>
            </div>
            <div className="p-6">
              <div className="space-y-4">
                {modules.map((module) => (
                  <div
                    key={module.id}
                    className="border border-gray-200 rounded-lg p-4"
                  >
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center">
                        <span className="text-2xl mr-3">{module.icon}</span>
                        <div>
                          <h4 className="text-lg font-medium text-gray-900">
                            {module.name}
                          </h4>
                          <p className="text-sm text-gray-600">
                            {module.description}
                          </p>
                        </div>
                      </div>
                      <div className="flex items-center space-x-3">
                        <button
                          onClick={() => handleToggleModule(module.id)}
                          className={`flex items-center px-3 py-1 rounded-full text-sm font-medium ${
                            module.isEnabled
                              ? "bg-green-100 text-green-800"
                              : "bg-red-100 text-red-800"
                          }`}
                        >
                          {module.isEnabled ? (
                            <>
                              <CheckCircle className="w-4 h-4 mr-1" />
                              已启用
                            </>
                          ) : (
                            <>
                              <XCircle className="w-4 h-4 mr-1" />
                              已禁用
                            </>
                          )}
                        </button>
                      </div>
                    </div>

                    {/* 权限操作 */}
                    <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-3">
                      {module.actions.map((action) => (
                        <div
                          key={action.id}
                          className={`p-3 rounded-lg border-2 transition-all ${
                            action.isEnabled
                              ? "border-blue-200 bg-blue-50"
                              : "border-gray-200 bg-gray-50"
                          }`}
                        >
                          <div className="flex items-center justify-between mb-2">
                            <span
                              className={`text-sm font-medium ${
                                action.isEnabled
                                  ? "text-blue-700"
                                  : "text-gray-500"
                              }`}
                            >
                              {action.name}
                            </span>
                            <button
                              onClick={() =>
                                handleToggleAction(module.id, action.id)
                              }
                              className={`w-5 h-5 rounded-full border-2 flex items-center justify-center ${
                                action.isEnabled
                                  ? "border-blue-500 bg-blue-500"
                                  : "border-gray-300 bg-white"
                              }`}
                            >
                              {action.isEnabled && (
                                <CheckCircle className="w-3 h-3 text-white" />
                              )}
                            </button>
                          </div>
                          <p
                            className={`text-xs ${
                              action.isEnabled
                                ? "text-blue-600"
                                : "text-gray-400"
                            }`}
                          >
                            {action.description}
                          </p>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default PermissionConfiguration;
