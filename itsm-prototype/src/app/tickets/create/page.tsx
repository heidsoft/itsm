"use client";

import React, { useState } from "react";
import { useRouter } from "next/navigation";
import { TicketApi } from "../../lib/ticket-api";
import { CreateTicketRequest } from "../../lib/api-config";
import { ArrowLeft, Save } from "lucide-react";
import Link from "next/link";

const CreateTicketPage: React.FC = () => {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [formData, setFormData] = useState<CreateTicketRequest>({
    title: "",
    description: "",
    priority: "medium", // 改为英文默认值
    form_fields: {},
  });

  // 添加优先级映射函数
  const priorityMap: { [key: string]: string } = {
    低: "low",
    中: "medium",
    高: "high",
    紧急: "critical",
  };

  // 反向映射，用于显示
  const priorityDisplayMap: { [key: string]: string } = {
    low: "低",
    medium: "中",
    high: "高",
    critical: "紧急",
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!formData.title.trim() || !formData.description.trim()) {
      setError("请填写标题和描述");
      return;
    }

    try {
      setLoading(true);
      setError(null);

      // 创建提交数据，确保priority是英文值
      const submitData = {
        ...formData,
        priority: priorityMap[formData.priority] || formData.priority,
      };

      const response = await TicketApi.createTicket(submitData);

      if (response.code === 0) {
        router.push(`/tickets/${response.data.id}`);
      } else {
        setError(response.message || "创建工单失败");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "网络错误");
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (field: keyof CreateTicketRequest, value: any) => {
    setFormData((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  return (
    <div className="p-6">
      {/* 头部 */}
      <div className="flex items-center mb-6">
        <Link
          href="/tickets"
          className="flex items-center text-gray-600 hover:text-gray-900 mr-4"
        >
          <ArrowLeft className="w-4 h-4 mr-1" />
          返回
        </Link>
        <h1 className="text-2xl font-bold text-gray-900">创建工单</h1>
      </div>

      {/* 错误提示 */}
      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}

      {/* 表单 */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* 标题 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              工单标题 *
            </label>
            <input
              type="text"
              value={formData.title}
              onChange={(e) => handleInputChange("title", e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="请输入工单标题"
              required
            />
          </div>

          {/* 优先级 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              优先级
            </label>
            <select
              value={formData.priority}
              onChange={(e) => handleInputChange("priority", e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="low">低</option>
              <option value="medium">中</option>
              <option value="high">高</option>
              <option value="critical">紧急</option>
            </select>
          </div>

          {/* 描述 */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              问题描述 *
            </label>
            <textarea
              value={formData.description}
              onChange={(e) => handleInputChange("description", e.target.value)}
              rows={6}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="请详细描述遇到的问题..."
              required
            />
          </div>

          {/* 提交按钮 */}
          <div className="flex justify-end space-x-3">
            <Link
              href="/tickets"
              className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 transition-colors"
            >
              取消
            </Link>
            <button
              type="submit"
              disabled={loading}
              className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors disabled:opacity-50"
            >
              {loading ? (
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
              ) : (
                <Save className="w-4 h-4 mr-2" />
              )}
              {loading ? "创建中..." : "创建工单"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateTicketPage;
