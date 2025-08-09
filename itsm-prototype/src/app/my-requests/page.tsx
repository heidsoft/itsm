"use client";

import {
  Search,
  Filter,
  Calendar,
  FileText,
  RefreshCw,
  ChevronRight,
  Clock,
  Hourglass,
  CheckCircle,
  XCircle,
} from "lucide-react";

import React, { useState, useEffect } from "react";
import Link from "next/link";
// API 接口类型定义
interface ServiceRequest {
  id: number;
  catalog_id: number;
  requester_id: number;
  status: "pending" | "in_progress" | "completed" | "rejected";
  reason: string;
  created_at: string;
  catalog?: {
    id: number;
    name: string;
    category: string;
    description: string;
  };
  requester?: {
    id: number;
    name: string;
    email: string;
  };
}

interface ApiResponse {
  requests: ServiceRequest[];
  total: number;
  page: number;
  size: number;
}

import { serviceRequestAPI, ServiceRequest } from "../lib/service-request-api";
import { mockRequestsData } from "../lib/mock-data";

const RequestStatusBadge = ({ status }: { status: string }) => {
  const statusConfig = {
    pending: {
      label: "待审批",
      color: "bg-yellow-100 text-yellow-800 border-yellow-200",
      icon: Clock,
      pulse: true,
    },
    in_progress: {
      label: "处理中",
      color: "bg-blue-100 text-blue-800 border-blue-200",
      icon: Hourglass,
      pulse: true,
    },
    completed: {
      label: "已完成",
      color: "bg-green-100 text-green-800 border-green-200",
      icon: CheckCircle,
      pulse: false,
    },
    rejected: {
      label: "已拒绝",
      color: "bg-red-100 text-red-800 border-red-200",
      icon: XCircle,
      pulse: false,
    },
  };

  const config =
    statusConfig[status as keyof typeof statusConfig] || statusConfig.pending;
  const Icon = config.icon;

  return (
    <span
      className={`inline-flex items-center px-3 py-1 text-xs font-medium rounded-full border ${
        config.color
      } ${config.pulse ? "animate-pulse" : ""}`}
    >
      <Icon className="w-3 h-3 mr-1.5" />
      {config.label}
    </span>
  );
};

const RequestCard = ({ request }: { request: ServiceRequest }) => {
  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  return (
    <div className="bg-white rounded-xl shadow-sm border border-gray-100 hover:shadow-md transition-all duration-200 hover:border-gray-200">
      <div className="p-6">
        <div className="flex items-start justify-between mb-4">
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-2">
              <span className="text-sm font-mono text-gray-500 bg-gray-50 px-2 py-1 rounded">
                REQ-{String(request.id).padStart(5, "0")}
              </span>
              <RequestStatusBadge status={request.status} />
            </div>
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              {request.catalog?.name || "未知服务"}
            </h3>
            <p className="text-sm text-gray-600 mb-3">
              {request.catalog?.description || request.reason}
            </p>
          </div>
        </div>

        <div className="flex items-center justify-between text-sm text-gray-500">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-1">
              <Calendar className="w-4 h-4" />
              <span>{formatDate(request.created_at)}</span>
            </div>
            <div className="flex items-center gap-1">
              <FileText className="w-4 h-4" />
              <span>{request.catalog?.category || "其他"}</span>
            </div>
          </div>
          <Link
            href={`/service-catalog/request/${request.id}`}
            className="inline-flex items-center gap-1 text-blue-600 hover:text-blue-700 font-medium transition-colors"
          >
            查看详情
            <ChevronRight className="w-4 h-4" />
          </Link>
        </div>
      </div>
    </div>
  );
};

const MyRequestsPage = () => {
  const [requests, setRequests] = useState<ServiceRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState("all");
  const [searchTerm, setSearchTerm] = useState("");
  const [currentPage, setCurrentPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [total, setTotal] = useState(0);
  const pageSize = 10;

  // 添加 loadMockData 函数
  const loadMockData = async () => {
    try {
      // 将 mockRequestsData 转换为 ServiceRequest 格式
      const mockRequests: ServiceRequest[] = mockRequestsData.map(
        (item, index) => ({
          id: parseInt(item.id.replace("REQ-", "")),
          catalog_id: index + 1,
          requester_id: 1,
          status:
            item.status === "处理中"
              ? "in_progress"
              : item.status === "已完成"
              ? "completed"
              : item.status === "待审批"
              ? "pending"
              : item.status === "已拒绝"
              ? "rejected"
              : "pending",
          reason: `申请${item.serviceName}`,
          created_at: item.submittedAt,
          catalog: {
            id: index + 1,
            name: item.serviceName,
            category: "IT服务",
            description: `${item.serviceName}服务申请`,
          },
          requester: {
            id: 1,
            name: "当前用户",
            email: "user@example.com",
          },
        })
      );

      setRequests(mockRequests);
      setTotal(mockRequests.length);
      setTotalPages(Math.ceil(mockRequests.length / pageSize));

      console.log("已加载Mock数据", mockRequests.length, "条记录");
    } catch (error) {
      console.error("加载Mock数据失败:", error);
      setRequests([]);
      setTotal(0);
      setTotalPages(1);
    }
  };

  // 获取服务请求数据
  const fetchRequests = async (page = 1, status = filter) => {
    setLoading(true);
    try {
      // 首先检查后端服务是否可用
      const isHealthy = await serviceRequestAPI.healthCheck();
      if (!isHealthy) {
        console.warn("后端服务不可用，使用Mock数据");
        await loadMockData();
        return;
      }

      const data = await serviceRequestAPI.getUserServiceRequests({
        page,
        size: pageSize,
        status: status === "all" ? undefined : status,
      });

      setRequests(data.requests);
      setTotal(data.total);
      setTotalPages(Math.ceil(data.total / pageSize));
    } catch (error) {
      console.error("API调用失败，回退到Mock数据:", error);
      await loadMockData();
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRequests(currentPage, filter);
  }, [currentPage, filter]);

  // 筛选数据
  const filteredRequests = requests.filter((request) => {
    const matchesSearch =
      !searchTerm ||
      request.catalog?.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      request.reason.toLowerCase().includes(searchTerm.toLowerCase());
    return matchesSearch;
  });

  const filterOptions = [
    { value: "all", label: "全部", count: total },
    { value: "pending", label: "待审批", count: 0 },
    { value: "in_progress", label: "处理中", count: 0 },
    { value: "completed", label: "已完成", count: 0 },
    { value: "rejected", label: "已拒绝", count: 0 },
  ];

  return (
    <div className="min-h-screen bg-gradient-to-br from-gray-50 to-gray-100">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* 页面头部 */}
        <div className="mb-8">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900 mb-2">
                我的请求
              </h1>
              <p className="text-gray-600">查看和跟踪您提交的所有服务请求</p>
            </div>
            <button
              onClick={() => fetchRequests(currentPage, filter)}
              className="inline-flex items-center gap-2 px-4 py-2 bg-white border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition-colors"
            >
              <RefreshCw className="w-4 h-4" />
              刷新
            </button>
          </div>
        </div>

        {/* 搜索和筛选 */}
        <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-6 mb-6">
          <div className="flex flex-col lg:flex-row gap-4">
            {/* 搜索框 */}
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
                <input
                  type="text"
                  placeholder="搜索服务名称或描述..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-full pl-10 pr-4 py-2.5 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent transition-all"
                />
              </div>
            </div>

            {/* 状态筛选 */}
            <div className="flex items-center gap-2">
              <Filter className="w-5 h-5 text-gray-500" />
              <div className="flex gap-2">
                {filterOptions.map((option) => (
                  <button
                    key={option.value}
                    onClick={() => {
                      setFilter(option.value);
                      setCurrentPage(1);
                    }}
                    className={`px-4 py-2 text-sm font-medium rounded-lg transition-all ${
                      filter === option.value
                        ? "bg-blue-600 text-white shadow-sm"
                        : "bg-gray-100 text-gray-700 hover:bg-gray-200"
                    }`}
                  >
                    {option.label}
                    {option.count > 0 && (
                      <span className="ml-1.5 text-xs opacity-75">
                        ({option.count})
                      </span>
                    )}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>

        {/* 请求列表 */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
            <span className="ml-3 text-gray-600">加载中...</span>
          </div>
        ) : filteredRequests.length > 0 ? (
          <div className="space-y-4">
            {filteredRequests.map((request) => (
              <RequestCard key={request.id} request={request} />
            ))}
          </div>
        ) : (
          <div className="bg-white rounded-xl shadow-sm border border-gray-200 p-12 text-center">
            <FileText className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">暂无请求</h3>
            <p className="text-gray-600 mb-6">您还没有提交任何服务请求</p>
            <Link
              href="/service-catalog"
              className="inline-flex items-center gap-2 px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
            >
              <FileText className="w-4 h-4" />
              浏览服务目录
            </Link>
          </div>
        )}

        {/* 分页 */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between mt-8 bg-white rounded-xl shadow-sm border border-gray-200 px-6 py-4">
            <div className="text-sm text-gray-700">
              显示第 {(currentPage - 1) * pageSize + 1} -{" "}
              {Math.min(currentPage * pageSize, total)} 条，共 {total} 条记录
            </div>
            <div className="flex gap-2">
              <button
                onClick={() => setCurrentPage((prev) => Math.max(1, prev - 1))}
                disabled={currentPage === 1}
                className="px-3 py-2 text-sm border border-gray-300 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 transition-colors"
              >
                上一页
              </button>
              <span className="px-3 py-2 text-sm bg-blue-50 text-blue-600 border border-blue-200 rounded-lg">
                {currentPage} / {totalPages}
              </span>
              <button
                onClick={() =>
                  setCurrentPage((prev) => Math.min(totalPages, prev + 1))
                }
                disabled={currentPage === totalPages}
                className="px-3 py-2 text-sm border border-gray-300 rounded-lg disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-50 transition-colors"
              >
                下一页
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default MyRequestsPage;
