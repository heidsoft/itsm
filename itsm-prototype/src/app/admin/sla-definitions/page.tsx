"use client";

import { Plus, CheckCircle, Clock, Search, Settings, Trash2, Edit, Filter, Eye, AlertTriangle, Target } from 'lucide-react';

import React, { useState } from "react";
import Link from "next/link";
import  from 'lucide-react';

// SLA定义的数据类型
interface SLADefinition {
  id: string;
  name: string;
  description: string;
  serviceType: string;
  priority: "P1" | "P2" | "P3" | "P4";
  responseTime: string;
  resolutionTime: string;
  availability: string;
  businessHours: string;
  escalationRules: string[];
  applicableServices: string[];
  status: "active" | "inactive" | "draft";
  createdAt: string;
  updatedAt: string;
  createdBy: string;
}

// 模拟SLA定义数据
const mockSLADefinitions: SLADefinition[] = [
  {
    id: "SLA-DEF-001",
    name: "关键业务系统SLA",
    description: "针对关键业务系统的服务级别协议，包括ERP、CRM等核心业务系统",
    serviceType: "关键业务系统",
    priority: "P1",
    responseTime: "15分钟",
    resolutionTime: "4小时",
    availability: "99.9%",
    businessHours: "7x24",
    escalationRules: [
      "15分钟升级至高级工程师",
      "1小时升级至技术经理",
      "4小时升级至CTO",
    ],
    applicableServices: ["ERP系统", "CRM系统", "财务系统"],
    status: "active",
    createdAt: "2024-01-15",
    updatedAt: "2024-06-01",
    createdBy: "张三",
  },
  {
    id: "SLA-DEF-002",
    name: "一般业务系统SLA",
    description: "针对一般业务系统的服务级别协议，包括OA、HR等支撑系统",
    serviceType: "一般业务系统",
    priority: "P2",
    responseTime: "30分钟",
    resolutionTime: "8小时",
    availability: "99.5%",
    businessHours: "工作时间",
    escalationRules: ["30分钟升级至高级工程师", "2小时升级至技术经理"],
    applicableServices: ["OA系统", "HR系统", "采购系统"],
    status: "active",
    createdAt: "2024-01-20",
    updatedAt: "2024-05-15",
    createdBy: "李四",
  },
  {
    id: "SLA-DEF-003",
    name: "基础设施SLA",
    description: "针对基础设施服务的服务级别协议，包括网络、服务器等",
    serviceType: "基础设施",
    priority: "P1",
    responseTime: "10分钟",
    resolutionTime: "2小时",
    availability: "99.95%",
    businessHours: "7x24",
    escalationRules: ["10分钟升级至网络工程师", "30分钟升级至基础设施经理"],
    applicableServices: ["网络服务", "服务器", "存储系统"],
    status: "active",
    createdAt: "2024-02-01",
    updatedAt: "2024-06-10",
    createdBy: "王五",
  },
  {
    id: "SLA-DEF-004",
    name: "开发测试环境SLA",
    description: "针对开发测试环境的服务级别协议",
    serviceType: "开发测试",
    priority: "P3",
    responseTime: "2小时",
    resolutionTime: "1工作日",
    availability: "95%",
    businessHours: "工作时间",
    escalationRules: ["4小时升级至开发经理"],
    applicableServices: ["开发环境", "测试环境", "CI/CD平台"],
    status: "draft",
    createdAt: "2024-06-01",
    updatedAt: "2024-06-15",
    createdBy: "赵六",
  },
];

// 优先级颜色映射
const priorityColors = {
  P1: "bg-red-100 text-red-800",
  P2: "bg-orange-100 text-orange-800",
  P3: "bg-yellow-100 text-yellow-800",
  P4: "bg-green-100 text-green-800",
};

// 状态颜色映射
const statusColors = {
  active: "bg-green-100 text-green-800",
  inactive: "bg-gray-100 text-gray-800",
  draft: "bg-blue-100 text-blue-800",
};

// 状态标签映射
const statusLabels = {
  active: "启用",
  inactive: "停用",
  draft: "草稿",
};

const SLADefinitionsPage = () => {
  const [searchTerm, setSearchTerm] = useState("");
  const [filterStatus, setFilterStatus] = useState("all");
  const [filterPriority, setFilterPriority] = useState("all");

  // 过滤SLA定义
  const filteredSLAs = mockSLADefinitions.filter((sla) => {
    const matchesSearch =
      sla.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      sla.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
      sla.serviceType.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus = filterStatus === "all" || sla.status === filterStatus;
    const matchesPriority =
      filterPriority === "all" || sla.priority === filterPriority;

    return matchesSearch && matchesStatus && matchesPriority;
  });

  // 统计数据
  const stats = {
    total: mockSLADefinitions.length,
    active: mockSLADefinitions.filter((sla) => sla.status === "active").length,
    draft: mockSLADefinitions.filter((sla) => sla.status === "draft").length,
    inactive: mockSLADefinitions.filter((sla) => sla.status === "inactive")
      .length,
  };

  return (
    <div className="p-8 bg-gray-50 min-h-full">
      {/* 页面头部 */}
      <div className="mb-8">
        <div className="flex justify-between items-start mb-6">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 mb-2">
              SLA定义管理
            </h1>
            <p className="text-gray-600">
              定义和管理服务级别协议，确保服务质量标准
            </p>
          </div>
          <button className="flex items-center bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 transition-colors">
            <Plus className="w-5 h-5 mr-2" />
            新建SLA定义
          </button>
        </div>

        {/* 统计卡片 */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">总计</p>
                <p className="text-2xl font-bold text-gray-900">
                  {stats.total}
                </p>
              </div>
              <div className="p-3 bg-blue-100 rounded-full">
                <Target className="w-6 h-6 text-blue-600" />
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">启用中</p>
                <p className="text-2xl font-bold text-green-600">
                  {stats.active}
                </p>
              </div>
              <div className="p-3 bg-green-100 rounded-full">
                <CheckCircle className="w-6 h-6 text-green-600" />
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">草稿</p>
                <p className="text-2xl font-bold text-blue-600">
                  {stats.draft}
                </p>
              </div>
              <div className="p-3 bg-blue-100 rounded-full">
                <Settings className="w-6 h-6 text-blue-600" />
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-600">停用</p>
                <p className="text-2xl font-bold text-gray-600">
                  {stats.inactive}
                </p>
              </div>
              <div className="p-3 bg-gray-100 rounded-full">
                <AlertTriangle className="w-6 h-6 text-gray-600" />
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 搜索和过滤 */}
      <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200 mb-6">
        <div className="flex flex-col md:flex-row gap-4">
          {/* 搜索框 */}
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
            <input
              type="text"
              placeholder="搜索SLA定义名称、描述或服务类型..."
              className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
            />
          </div>

          {/* 状态过滤 */}
          <div className="flex items-center gap-2">
            <Filter className="w-5 h-5 text-gray-400" />
            <select
              className="border border-gray-300 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
            >
              <option value="all">所有状态</option>
              <option value="active">启用</option>
              <option value="draft">草稿</option>
              <option value="inactive">停用</option>
            </select>
          </div>

          {/* 优先级过滤 */}
          <div>
            <select
              className="border border-gray-300 rounded-lg px-3 py-2 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              value={filterPriority}
              onChange={(e) => setFilterPriority(e.target.value)}
            >
              <option value="all">所有优先级</option>
              <option value="P1">P1 - 紧急</option>
              <option value="P2">P2 - 高</option>
              <option value="P3">P3 - 中</option>
              <option value="P4">P4 - 低</option>
            </select>
          </div>
        </div>
      </div>

      {/* SLA定义列表 */}
      <div className="bg-white rounded-lg shadow-sm border border-gray-200">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50 border-b border-gray-200">
              <tr>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  SLA定义
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  服务类型
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  优先级
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  响应时间
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  解决时间
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  可用性
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  状态
                </th>
                <th className="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {filteredSLAs.map((sla) => (
                <tr key={sla.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4">
                    <div>
                      <div className="text-sm font-medium text-gray-900">
                        {sla.name}
                      </div>
                      <div className="text-sm text-gray-500 max-w-xs truncate">
                        {sla.description}
                      </div>
                      <div className="text-xs text-gray-400 mt-1">
                        ID: {sla.id}
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <span className="text-sm text-gray-900">
                      {sla.serviceType}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                        priorityColors[sla.priority]
                      }`}
                    >
                      {sla.priority}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center text-sm text-gray-900">
                      <Clock className="w-4 h-4 mr-1 text-gray-400" />
                      {sla.responseTime}
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center text-sm text-gray-900">
                      <Target className="w-4 h-4 mr-1 text-gray-400" />
                      {sla.resolutionTime}
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <span className="text-sm text-gray-900">
                      {sla.availability}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                        statusColors[sla.status]
                      }`}
                    >
                      {statusLabels[sla.status]}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex items-center space-x-2">
                      <button className="text-blue-600 hover:text-blue-800 p-1">
                        <Eye className="w-4 h-4" />
                      </button>
                      <button className="text-green-600 hover:text-green-800 p-1">
                        <Edit className="w-4 h-4" />
                      </button>
                      <button className="text-red-600 hover:text-red-800 p-1">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {filteredSLAs.length === 0 && (
          <div className="text-center py-12">
            <Target className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">
              没有找到SLA定义
            </h3>
            <p className="text-gray-500">尝试调整搜索条件或创建新的SLA定义</p>
          </div>
        )}
      </div>
    </div>
  );
};

export default SLADefinitionsPage;
