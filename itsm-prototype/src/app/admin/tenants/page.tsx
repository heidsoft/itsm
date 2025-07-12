"use client";

import React, { useState } from "react";
import {
  Plus,
  Search,
  Filter,
  MoreHorizontal,
  Building2,
  Users,
  Calendar,
  Settings,
  AlertCircle,
  CheckCircle,
  XCircle,
  Clock,
} from "lucide-react";

// 租户状态配置
const TENANT_STATUS = {
  active: {
    label: "活跃",
    color: "bg-green-100 text-green-800",
    icon: CheckCircle,
  },
  suspended: {
    label: "暂停",
    color: "bg-yellow-100 text-yellow-800",
    icon: AlertCircle,
  },
  expired: { label: "过期", color: "bg-red-100 text-red-800", icon: XCircle },
  trial: { label: "试用", color: "bg-blue-100 text-blue-800", icon: Clock },
};

// 租户类型配置
const TENANT_TYPES = {
  trial: { label: "试用版", color: "bg-gray-100 text-gray-800" },
  standard: { label: "标准版", color: "bg-blue-100 text-blue-800" },
  professional: { label: "专业版", color: "bg-purple-100 text-purple-800" },
  enterprise: { label: "企业版", color: "bg-gold-100 text-gold-800" },
};

// 模拟租户数据
const mockTenants = [
  {
    id: 1,
    name: "阿里巴巴集团",
    code: "alibaba",
    domain: "alibaba.itsm.com",
    type: "enterprise",
    status: "active",
    userCount: 1250,
    ticketCount: 8934,
    createdAt: "2024-01-15",
    expiresAt: "2025-01-15",
    quota: {
      maxUsers: 2000,
      maxTickets: 50000,
      storage: "100GB",
    },
  },
  {
    id: 2,
    name: "腾讯科技",
    code: "tencent",
    domain: "tencent.itsm.com",
    type: "enterprise",
    status: "active",
    userCount: 980,
    ticketCount: 6721,
    createdAt: "2024-02-01",
    expiresAt: "2025-02-01",
    quota: {
      maxUsers: 1500,
      maxTickets: 30000,
      storage: "80GB",
    },
  },
  {
    id: 3,
    name: "字节跳动",
    code: "bytedance",
    domain: "bytedance.itsm.com",
    type: "professional",
    status: "trial",
    userCount: 156,
    ticketCount: 892,
    createdAt: "2024-03-10",
    expiresAt: "2024-04-10",
    quota: {
      maxUsers: 500,
      maxTickets: 10000,
      storage: "20GB",
    },
  },
];

const TenantManagement = () => {
  const [tenants, setTenants] = useState(mockTenants);
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [typeFilter, setTypeFilter] = useState("all");
  const [showModal, setShowModal] = useState(false);
  const [selectedTenant, setSelectedTenant] = useState(null);

  // 过滤租户
  const filteredTenants = tenants.filter((tenant) => {
    const matchesSearch =
      tenant.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      tenant.code.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesStatus =
      statusFilter === "all" || tenant.status === statusFilter;
    const matchesType = typeFilter === "all" || tenant.type === typeFilter;
    return matchesSearch && matchesStatus && matchesType;
  });

  // 统计数据
  const stats = {
    total: tenants.length,
    active: tenants.filter((t) => t.status === "active").length,
    trial: tenants.filter((t) => t.status === "trial").length,
    suspended: tenants.filter((t) => t.status === "suspended").length,
  };

  return (
    <div className="p-6 space-y-6">
      {/* 页面头部 */}
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">租户管理</h1>
          <p className="text-gray-600 mt-1">管理系统中的所有租户和订阅</p>
        </div>
        <button
          onClick={() => setShowModal(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700 flex items-center gap-2"
        >
          <Plus className="w-4 h-4" />
          新建租户
        </button>
      </div>

      {/* 统计卡片 */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">总租户数</p>
              <p className="text-2xl font-bold text-gray-900">{stats.total}</p>
            </div>
            <Building2 className="w-8 h-8 text-blue-600" />
          </div>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">活跃租户</p>
              <p className="text-2xl font-bold text-green-600">
                {stats.active}
              </p>
            </div>
            <CheckCircle className="w-8 h-8 text-green-600" />
          </div>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">试用租户</p>
              <p className="text-2xl font-bold text-blue-600">{stats.trial}</p>
            </div>
            <Clock className="w-8 h-8 text-blue-600" />
          </div>
        </div>
        <div className="bg-white p-6 rounded-lg shadow-sm border">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-gray-600">暂停租户</p>
              <p className="text-2xl font-bold text-yellow-600">
                {stats.suspended}
              </p>
            </div>
            <AlertCircle className="w-8 h-8 text-yellow-600" />
          </div>
        </div>
      </div>

      {/* 搜索和过滤 */}
      <div className="bg-white p-4 rounded-lg shadow-sm border">
        <div className="flex flex-col md:flex-row gap-4">
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-4 h-4" />
            <input
              type="text"
              placeholder="搜索租户名称或代码..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">所有状态</option>
            <option value="active">活跃</option>
            <option value="trial">试用</option>
            <option value="suspended">暂停</option>
            <option value="expired">过期</option>
          </select>
          <select
            value={typeFilter}
            onChange={(e) => setTypeFilter(e.target.value)}
            className="px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">所有类型</option>
            <option value="trial">试用版</option>
            <option value="standard">标准版</option>
            <option value="professional">专业版</option>
            <option value="enterprise">企业版</option>
          </select>
        </div>
      </div>

      {/* 租户列表 */}
      <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  租户信息
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  类型/状态
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  使用情况
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  到期时间
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  操作
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {filteredTenants.map((tenant) => {
                const StatusIcon =
                  TENANT_STATUS[tenant.status]?.icon || AlertCircle;
                return (
                  <tr key={tenant.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <div className="flex-shrink-0 h-10 w-10">
                          <div className="h-10 w-10 rounded-lg bg-blue-100 flex items-center justify-center">
                            <Building2 className="w-5 h-5 text-blue-600" />
                          </div>
                        </div>
                        <div className="ml-4">
                          <div className="text-sm font-medium text-gray-900">
                            {tenant.name}
                          </div>
                          <div className="text-sm text-gray-500">
                            {tenant.code} • {tenant.domain}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="space-y-1">
                        <span
                          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                            TENANT_TYPES[tenant.type]?.color ||
                            "bg-gray-100 text-gray-800"
                          }`}
                        >
                          {TENANT_TYPES[tenant.type]?.label || tenant.type}
                        </span>
                        <div className="flex items-center">
                          <StatusIcon className="w-3 h-3 mr-1" />
                          <span
                            className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                              TENANT_STATUS[tenant.status]?.color ||
                              "bg-gray-100 text-gray-800"
                            }`}
                          >
                            {TENANT_STATUS[tenant.status]?.label ||
                              tenant.status}
                          </span>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      <div className="space-y-1">
                        <div className="flex items-center">
                          <Users className="w-4 h-4 mr-1 text-gray-400" />
                          <span>
                            {tenant.userCount}/{tenant.quota.maxUsers} 用户
                          </span>
                        </div>
                        <div className="text-xs text-gray-500">
                          {tenant.ticketCount} 工单 • {tenant.quota.storage}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      <div className="flex items-center">
                        <Calendar className="w-4 h-4 mr-1 text-gray-400" />
                        {tenant.expiresAt}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <div className="flex items-center space-x-2">
                        <button
                          onClick={() => {
                            setSelectedTenant(tenant);
                            setShowModal(true);
                          }}
                          className="text-blue-600 hover:text-blue-900"
                        >
                          <Settings className="w-4 h-4" />
                        </button>
                        <button className="text-gray-400 hover:text-gray-600">
                          <MoreHorizontal className="w-4 h-4" />
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
    </div>
  );
};

export default TenantManagement;
