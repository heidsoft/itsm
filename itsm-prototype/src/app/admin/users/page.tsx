"use client";

import { Plus, Clock, Search, Settings, Trash2, Edit, Calendar, User, Shield, Download, Upload, UserCheck, UserX, ChevronDown, MapPin, Mail, Phone, ChevronDown } from 'lucide-react';

import React, { useState } from "react";
// 用户数据类型定义
interface User {
  id: number;
  username: string;
  email: string;
  fullName: string;
  role: string;
  department: string;
  status: "active" | "inactive" | "locked";
  lastLogin: string;
  phone: string;
  createdAt: string;
}

// 模拟用户数据
const mockUsers: User[] = [
  {
    id: 1,
    username: "admin",
    email: "admin@company.com",
    fullName: "系统管理员",
    role: "系统管理员",
    department: "IT部门",
    status: "active",
    lastLogin: "2024-01-15 10:30",
    phone: "13800138000",
    createdAt: "2023-01-01",
  },
  {
    id: 2,
    username: "john.doe",
    email: "john.doe@company.com",
    fullName: "约翰·多伊",
    role: "IT支持工程师",
    department: "IT部门",
    status: "active",
    lastLogin: "2024-01-15 09:15",
    phone: "13800138001",
    createdAt: "2023-03-15",
  },
  {
    id: 3,
    username: "jane.smith",
    email: "jane.smith@company.com",
    fullName: "简·史密斯",
    role: "业务分析师",
    department: "业务部门",
    status: "inactive",
    lastLogin: "2024-01-10 16:45",
    phone: "13800138002",
    createdAt: "2023-06-20",
  },
  {
    id: 4,
    username: "mike.wilson",
    email: "mike.wilson@company.com",
    fullName: "迈克·威尔逊",
    role: "服务台专员",
    department: "IT部门",
    status: "locked",
    lastLogin: "2024-01-12 14:20",
    phone: "13800138003",
    createdAt: "2023-09-10",
  },
];

const UserManagement = () => {
  const [users, setUsers] = useState<User[]>(mockUsers);
  const [searchTerm, setSearchTerm] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [roleFilter, setRoleFilter] = useState("all");
  const [showModal, setShowModal] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [currentPage, setCurrentPage] = useState(1);
  const itemsPerPage = 10;

  // 获取所有角色列表
  const roles = Array.from(new Set(users.map((user) => user.role)));

  // 过滤用户
  const filteredUsers = users.filter((user) => {
    const matchesSearch =
      user.fullName.toLowerCase().includes(searchTerm.toLowerCase()) ||
      user.email.toLowerCase().includes(searchTerm.toLowerCase()) ||
      user.username.toLowerCase().includes(searchTerm.toLowerCase()) ||
      user.department.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesStatus =
      statusFilter === "all" || user.status === statusFilter;
    const matchesRole = roleFilter === "all" || user.role === roleFilter;

    return matchesSearch && matchesStatus && matchesRole;
  });

  // 分页计算
  const totalPages = Math.ceil(filteredUsers.length / itemsPerPage);
  const startIndex = (currentPage - 1) * itemsPerPage;
  const paginatedUsers = filteredUsers.slice(
    startIndex,
    startIndex + itemsPerPage
  );

  // 状态样式映射
  const getStatusStyle = (status: string) => {
    switch (status) {
      case "active":
        return "bg-green-100 text-green-800";
      case "inactive":
        return "bg-gray-100 text-gray-800";
      case "locked":
        return "bg-red-100 text-red-800";
      default:
        return "bg-gray-100 text-gray-800";
    }
  };

  // 状态文本映射
  const getStatusText = (status: string) => {
    switch (status) {
      case "active":
        return "活跃";
      case "inactive":
        return "非活跃";
      case "locked":
        return "锁定";
      default:
        return "未知";
    }
  };

  // 处理用户操作
  const handleEditUser = (user: User) => {
    setSelectedUser(user);
    setShowModal(true);
  };

  const handleDeleteUser = (userId: number) => {
    if (confirm("确定要删除这个用户吗？")) {
      setUsers(users.filter((user) => user.id !== userId));
    }
  };

  const handleToggleStatus = (userId: number) => {
    setUsers(
      users.map((user) => {
        if (user.id === userId) {
          return {
            ...user,
            status: user.status === "active" ? "inactive" : "active",
          };
        }
        return user;
      })
    );
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* 现代化页面头部 */}
        <div className="mb-8">
          <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
            <div className="space-y-2">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-gradient-to-r from-blue-600 to-indigo-600 rounded-xl flex items-center justify-center">
                  <Shield className="w-5 h-5 text-white" />
                </div>
                <div>
                  <h1 className="text-3xl font-bold bg-gradient-to-r from-gray-900 to-gray-600 bg-clip-text text-transparent">
                    用户管理
                  </h1>
                  <p className="text-gray-600 text-sm">
                    管理系统用户账户、角色和权限
                  </p>
                </div>
              </div>
            </div>

            <div className="flex items-center gap-3">
              <button className="inline-flex items-center gap-2 px-4 py-2.5 text-gray-700 bg-white border border-gray-200 rounded-xl hover:bg-gray-50 hover:border-gray-300 transition-all duration-200 shadow-sm">
                <Download className="w-4 h-4" />
                导出
              </button>
              <button className="inline-flex items-center gap-2 px-4 py-2.5 text-gray-700 bg-white border border-gray-200 rounded-xl hover:bg-gray-50 hover:border-gray-300 transition-all duration-200 shadow-sm">
                <Upload className="w-4 h-4" />
                导入
              </button>
              <button
                onClick={() => {
                  setSelectedUser(null);
                  setShowModal(true);
                }}
                className="inline-flex items-center gap-2 px-6 py-2.5 bg-gradient-to-r from-blue-600 to-indigo-600 text-white rounded-xl hover:from-blue-700 hover:to-indigo-700 transition-all duration-200 shadow-lg hover:shadow-xl transform hover:-translate-y-0.5"
              >
                <Plus className="w-4 h-4" />
                新建用户
              </button>
            </div>
          </div>
        </div>

        {/* 增强的统计卡片 */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <div className="group relative bg-white p-6 rounded-2xl shadow-sm border border-gray-100 hover:shadow-lg hover:border-blue-200 transition-all duration-300">
            <div className="absolute inset-0 bg-gradient-to-r from-blue-500/5 to-indigo-500/5 rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
            <div className="relative flex items-center justify-between">
              <div className="space-y-2">
                <p className="text-sm font-medium text-gray-600">总用户数</p>
                <p className="text-3xl font-bold text-gray-900">
                  {users.length}
                </p>
                <div className="flex items-center gap-1 text-xs text-green-600">
                  <span className="w-1 h-1 bg-green-500 rounded-full"></span>
                  <span>+12% 本月</span>
                </div>
              </div>
              <div className="w-12 h-12 bg-gradient-to-r from-blue-500 to-indigo-500 rounded-2xl flex items-center justify-center shadow-lg">
                <UserCheck className="w-6 h-6 text-white" />
              </div>
            </div>
          </div>

          <div className="group relative bg-white p-6 rounded-2xl shadow-sm border border-gray-100 hover:shadow-lg hover:border-green-200 transition-all duration-300">
            <div className="absolute inset-0 bg-gradient-to-r from-green-500/5 to-emerald-500/5 rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
            <div className="relative flex items-center justify-between">
              <div className="space-y-2">
                <p className="text-sm font-medium text-gray-600">活跃用户</p>
                <p className="text-3xl font-bold text-green-600">
                  {users.filter((u) => u.status === "active").length}
                </p>
                <div className="flex items-center gap-1 text-xs text-green-600">
                  <span className="w-1 h-1 bg-green-500 rounded-full animate-pulse"></span>
                  <span>在线状态</span>
                </div>
              </div>
              <div className="w-12 h-12 bg-gradient-to-r from-green-500 to-emerald-500 rounded-2xl flex items-center justify-center shadow-lg">
                <UserCheck className="w-6 h-6 text-white" />
              </div>
            </div>
          </div>

          <div className="group relative bg-white p-6 rounded-2xl shadow-sm border border-gray-100 hover:shadow-lg hover:border-amber-200 transition-all duration-300">
            <div className="absolute inset-0 bg-gradient-to-r from-amber-500/5 to-orange-500/5 rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
            <div className="relative flex items-center justify-between">
              <div className="space-y-2">
                <p className="text-sm font-medium text-gray-600">非活跃用户</p>
                <p className="text-3xl font-bold text-amber-600">
                  {users.filter((u) => u.status === "inactive").length}
                </p>
                <div className="flex items-center gap-1 text-xs text-amber-600">
                  <Clock className="w-3 h-3" />
                  <span>需要关注</span>
                </div>
              </div>
              <div className="w-12 h-12 bg-gradient-to-r from-amber-500 to-orange-500 rounded-2xl flex items-center justify-center shadow-lg">
                <UserX className="w-6 h-6 text-white" />
              </div>
            </div>
          </div>

          <div className="group relative bg-white p-6 rounded-2xl shadow-sm border border-gray-100 hover:shadow-lg hover:border-red-200 transition-all duration-300">
            <div className="absolute inset-0 bg-gradient-to-r from-red-500/5 to-rose-500/5 rounded-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-300"></div>
            <div className="relative flex items-center justify-between">
              <div className="space-y-2">
                <p className="text-sm font-medium text-gray-600">锁定用户</p>
                <p className="text-3xl font-bold text-red-600">
                  {users.filter((u) => u.status === "locked").length}
                </p>
                <div className="flex items-center gap-1 text-xs text-red-600">
                  <span className="w-1 h-1 bg-red-500 rounded-full"></span>
                  <span>安全风险</span>
                </div>
              </div>
              <div className="w-12 h-12 bg-gradient-to-r from-red-500 to-rose-500 rounded-2xl flex items-center justify-center shadow-lg">
                <UserX className="w-6 h-6 text-white" />
              </div>
            </div>
          </div>
        </div>

        {/* 现代化搜索和过滤区域 */}
        <div className="bg-white/80 backdrop-blur-sm p-6 rounded-2xl shadow-sm border border-gray-100 mb-8">
          <div className="flex flex-col lg:flex-row gap-4">
            {/* 增强的搜索框 */}
            <div className="flex-1 relative group">
              <div className="absolute inset-0 bg-gradient-to-r from-blue-500/10 to-indigo-500/10 rounded-xl opacity-0 group-focus-within:opacity-100 transition-opacity duration-300"></div>
              <Search className="absolute left-4 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5 group-focus-within:text-blue-500 transition-colors duration-200" />
              <input
                type="text"
                placeholder="搜索用户名、邮箱、姓名或部门..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="relative w-full pl-12 pr-4 py-3 border border-gray-200 rounded-xl focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200 bg-white/50 backdrop-blur-sm"
              />
            </div>

            {/* 现代化过滤器 */}
            <div className="flex items-center gap-3">
              <div className="relative">
                <select
                  value={statusFilter}
                  onChange={(e) => setStatusFilter(e.target.value)}
                  className="appearance-none bg-white/80 backdrop-blur-sm border border-gray-200 rounded-xl px-4 py-3 pr-10 focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200 cursor-pointer"
                >
                  <option value="all">所有状态</option>
                  <option value="active">活跃</option>
                  <option value="inactive">非活跃</option>
                  <option value="locked">锁定</option>
                </select>
                <ChevronDown className="absolute right-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
              </div>

              <div className="relative">
                <select
                  value={roleFilter}
                  onChange={(e) => setRoleFilter(e.target.value)}
                  className="appearance-none bg-white/80 backdrop-blur-sm border border-gray-200 rounded-xl px-4 py-3 pr-10 focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200 cursor-pointer"
                >
                  <option value="all">所有角色</option>
                  {roles.map((role) => (
                    <option key={role} value={role}>
                      {role}
                    </option>
                  ))}
                </select>
                <ChevronDown className="absolute right-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
              </div>

              <button className="p-3 bg-gray-100 hover:bg-gray-200 rounded-xl transition-colors duration-200">
                <Settings className="w-4 h-4 text-gray-600" />
              </button>
            </div>
          </div>
        </div>

        {/* 现代化用户列表 */}
        <div className="bg-white/80 backdrop-blur-sm rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gradient-to-r from-gray-50 to-gray-100/50">
                <tr>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                    用户信息
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                    角色部门
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                    状态
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                    最后登录
                  </th>
                  <th className="px-6 py-4 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                    联系方式
                  </th>
                  <th className="px-6 py-4 text-right text-xs font-semibold text-gray-600 uppercase tracking-wider">
                    操作
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white/50 backdrop-blur-sm divide-y divide-gray-100">
                {paginatedUsers.map((user, index) => (
                  <tr
                    key={user.id}
                    className="group hover:bg-blue-50/50 transition-all duration-200 animate-fade-in"
                    style={{ animationDelay: `${index * 50}ms` }}
                  >
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <div className="relative">
                          <div className="w-12 h-12 bg-gradient-to-r from-blue-500 to-indigo-500 rounded-2xl flex items-center justify-center shadow-lg">
                            <span className="text-white font-semibold text-sm">
                              {user.fullName.charAt(0)}
                            </span>
                          </div>
                          <div
                            className={`absolute -bottom-1 -right-1 w-4 h-4 rounded-full border-2 border-white ${
                              user.status === "active"
                                ? "bg-green-500"
                                : user.status === "inactive"
                                ? "bg-gray-400"
                                : "bg-red-500"
                            }`}
                          ></div>
                        </div>
                        <div className="ml-4">
                          <div className="text-sm font-semibold text-gray-900">
                            {user.fullName}
                          </div>
                          <div className="text-sm text-gray-500 font-mono">
                            @{user.username}
                          </div>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="space-y-1">
                        <div className="inline-flex items-center px-2.5 py-1 rounded-lg text-xs font-medium bg-blue-100 text-blue-800">
                          {user.role}
                        </div>
                        <div className="flex items-center gap-1 text-sm text-gray-500">
                          <MapPin className="w-3 h-3" />
                          {user.department}
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span
                        className={`inline-flex items-center px-3 py-1.5 rounded-full text-xs font-semibold ${
                          user.status === "active"
                            ? "bg-green-100 text-green-800 border border-green-200"
                            : user.status === "inactive"
                            ? "bg-gray-100 text-gray-800 border border-gray-200"
                            : "bg-red-100 text-red-800 border border-red-200"
                        }`}
                      >
                        <span
                          className={`w-1.5 h-1.5 rounded-full mr-2 ${
                            user.status === "active"
                              ? "bg-green-500 animate-pulse"
                              : user.status === "inactive"
                              ? "bg-gray-400"
                              : "bg-red-500"
                          }`}
                        ></span>
                        {getStatusText(user.status)}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center gap-2 text-sm text-gray-500">
                        <Calendar className="w-4 h-4" />
                        <span>{user.lastLogin}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="space-y-2">
                        <div className="flex items-center gap-2 text-sm text-gray-600">
                          <Mail className="w-4 h-4 text-gray-400" />
                          <span className="truncate max-w-[200px]">
                            {user.email}
                          </span>
                        </div>
                        <div className="flex items-center gap-2 text-sm text-gray-600">
                          <Phone className="w-4 h-4 text-gray-400" />
                          <span>{user.phone}</span>
                        </div>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right">
                      <div className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-200">
                        <button
                          onClick={() => handleEditUser(user)}
                          className="p-2 text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-lg transition-all duration-200"
                          title="编辑用户"
                        >
                          <Edit className="w-4 h-4" />
                        </button>
                        <button
                          onClick={() => handleToggleStatus(user.id)}
                          className={`p-2 rounded-lg transition-all duration-200 ${
                            user.status === "active"
                              ? "text-amber-600 hover:text-amber-800 hover:bg-amber-50"
                              : "text-green-600 hover:text-green-800 hover:bg-green-50"
                          }`}
                          title={
                            user.status === "active" ? "禁用用户" : "启用用户"
                          }
                        >
                          {user.status === "active" ? (
                            <UserX className="w-4 h-4" />
                          ) : (
                            <UserCheck className="w-4 h-4" />
                          )}
                        </button>
                        <button
                          onClick={() => handleDeleteUser(user.id)}
                          className="p-2 text-red-600 hover:text-red-800 hover:bg-red-50 rounded-lg transition-all duration-200"
                          title="删除用户"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {/* 现代化分页 */}
          {totalPages > 1 && (
            <div className="bg-white/80 backdrop-blur-sm px-6 py-4 flex items-center justify-between border-t border-gray-100">
              <div className="flex-1 flex justify-between sm:hidden">
                <button
                  onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                  disabled={currentPage === 1}
                  className="relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-xl text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 transition-all duration-200"
                >
                  上一页
                </button>
                <button
                  onClick={() =>
                    setCurrentPage(Math.min(totalPages, currentPage + 1))
                  }
                  disabled={currentPage === totalPages}
                  className="ml-3 relative inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-xl text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50 transition-all duration-200"
                >
                  下一页
                </button>
              </div>
              <div className="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
                <div>
                  <p className="text-sm text-gray-700">
                    显示第{" "}
                    <span className="font-semibold">{startIndex + 1}</span> 到{" "}
                    <span className="font-semibold">
                      {Math.min(
                        startIndex + itemsPerPage,
                        filteredUsers.length
                      )}
                    </span>{" "}
                    条，共{" "}
                    <span className="font-semibold">
                      {filteredUsers.length}
                    </span>{" "}
                    条记录
                  </p>
                </div>
                <div>
                  <nav className="relative z-0 inline-flex rounded-xl shadow-sm -space-x-px">
                    <button
                      onClick={() =>
                        setCurrentPage(Math.max(1, currentPage - 1))
                      }
                      disabled={currentPage === 1}
                      className="relative inline-flex items-center px-3 py-2 rounded-l-xl border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 transition-all duration-200"
                    >
                      上一页
                    </button>
                    {Array.from({ length: totalPages }, (_, i) => i + 1).map(
                      (page) => (
                        <button
                          key={page}
                          onClick={() => setCurrentPage(page)}
                          className={`relative inline-flex items-center px-4 py-2 border text-sm font-medium transition-all duration-200 ${
                            page === currentPage
                              ? "z-10 bg-gradient-to-r from-blue-500 to-indigo-500 border-blue-500 text-white shadow-lg"
                              : "bg-white border-gray-300 text-gray-500 hover:bg-gray-50"
                          }`}
                        >
                          {page}
                        </button>
                      )
                    )}
                    <button
                      onClick={() =>
                        setCurrentPage(Math.min(totalPages, currentPage + 1))
                      }
                      disabled={currentPage === totalPages}
                      className="relative inline-flex items-center px-3 py-2 rounded-r-xl border border-gray-300 bg-white text-sm font-medium text-gray-500 hover:bg-gray-50 disabled:opacity-50 transition-all duration-200"
                    >
                      下一页
                    </button>
                  </nav>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* 现代化模态框 */}
        {showModal && (
          <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
            <div className="bg-white rounded-2xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-y-auto">
              <div className="p-6 border-b border-gray-100">
                <h3 className="text-xl font-semibold text-gray-900">
                  {selectedUser ? "编辑用户" : "新建用户"}
                </h3>
                <p className="text-sm text-gray-600 mt-1">
                  {selectedUser
                    ? "修改用户信息和权限设置"
                    : "创建新的系统用户账户"}
                </p>
              </div>

              <div className="p-6">
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <label className="block text-sm font-semibold text-gray-700">
                      用户名 <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      className="w-full border border-gray-200 rounded-xl px-4 py-3 focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200"
                      defaultValue={selectedUser?.username || ""}
                      placeholder="请输入用户名"
                    />
                  </div>

                  <div className="space-y-2">
                    <label className="block text-sm font-semibold text-gray-700">
                      姓名 <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="text"
                      className="w-full border border-gray-200 rounded-xl px-4 py-3 focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200"
                      defaultValue={selectedUser?.fullName || ""}
                      placeholder="请输入真实姓名"
                    />
                  </div>

                  <div className="space-y-2">
                    <label className="block text-sm font-semibold text-gray-700">
                      邮箱 <span className="text-red-500">*</span>
                    </label>
                    <input
                      type="email"
                      className="w-full border border-gray-200 rounded-xl px-4 py-3 focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200"
                      defaultValue={selectedUser?.email || ""}
                      placeholder="请输入邮箱地址"
                    />
                  </div>

                  <div className="space-y-2">
                    <label className="block text-sm font-semibold text-gray-700">
                      手机号码
                    </label>
                    <input
                      type="tel"
                      className="w-full border border-gray-200 rounded-xl px-4 py-3 focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200"
                      defaultValue={selectedUser?.phone || ""}
                      placeholder="请输入手机号码"
                    />
                  </div>

                  <div className="space-y-2">
                    <label className="block text-sm font-semibold text-gray-700">
                      角色 <span className="text-red-500">*</span>
                    </label>
                    <div className="relative">
                      <select
                        className="w-full appearance-none border border-gray-200 rounded-xl px-4 py-3 pr-10 focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200"
                        defaultValue={selectedUser?.role || ""}
                      >
                        <option value="">选择角色</option>
                        <option value="系统管理员">系统管理员</option>
                        <option value="IT支持工程师">IT支持工程师</option>
                        <option value="业务分析师">业务分析师</option>
                        <option value="服务台专员">服务台专员</option>
                      </select>
                      <ChevronDown className="absolute right-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <label className="block text-sm font-semibold text-gray-700">
                      部门 <span className="text-red-500">*</span>
                    </label>
                    <div className="relative">
                      <select
                        className="w-full appearance-none border border-gray-200 rounded-xl px-4 py-3 pr-10 focus:ring-2 focus:ring-blue-500/20 focus:border-blue-500 transition-all duration-200"
                        defaultValue={selectedUser?.department || ""}
                      >
                        <option value="">选择部门</option>
                        <option value="IT部门">IT部门</option>
                        <option value="业务部门">业务部门</option>
                        <option value="财务部门">财务部门</option>
                        <option value="人事部门">人事部门</option>
                      </select>
                      <ChevronDown className="absolute right-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
                    </div>
                  </div>
                </div>

                {!selectedUser && (
                  <div className="mt-6 p-4 bg-blue-50 rounded-xl border border-blue-200">
                    <div className="flex items-start gap-3">
                      <Shield className="w-5 h-5 text-blue-600 mt-0.5" />
                      <div>
                        <h4 className="text-sm font-semibold text-blue-900">
                          安全提示
                        </h4>
                        <p className="text-sm text-blue-700 mt-1">
                          新用户将收到包含临时密码的邮件，首次登录时需要修改密码。
                        </p>
                      </div>
                    </div>
                  </div>
                )}
              </div>

              <div className="flex justify-end gap-3 p-6 border-t border-gray-100 bg-gray-50/50">
                <button
                  onClick={() => setShowModal(false)}
                  className="px-6 py-2.5 text-gray-700 border border-gray-300 rounded-xl hover:bg-gray-50 transition-all duration-200"
                >
                  取消
                </button>
                <button
                  onClick={() => {
                    // 这里应该处理保存逻辑
                    setShowModal(false);
                  }}
                  className="px-6 py-2.5 bg-gradient-to-r from-blue-600 to-indigo-600 text-white rounded-xl hover:from-blue-700 hover:to-indigo-700 transition-all duration-200 shadow-lg hover:shadow-xl"
                >
                  {selectedUser ? "更新用户" : "创建用户"}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default UserManagement;
