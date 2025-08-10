"use client";

import React, { useState } from "react";
import { useRouter } from "next/navigation";
import { TicketApi } from "../../lib/ticket-api";
import { CreateTicketRequest } from "../../lib/api-config";
import {
  ArrowLeft,
  Save,
  Brain,
  Sparkles,
  UserCircle,
  BookOpen,
} from "lucide-react";
import Link from "next/link";
import { aiSearchKB, aiSummarize, aiTriage, RagAnswer } from "../../lib/ai-api";
import { AIFeedback } from "../../components/AIFeedback";

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

  const [aiLoading, setAiLoading] = useState(false);
  const [aiError, setAiError] = useState<string | null>(null);
  const [kbAnswers, setKbAnswers] = useState<RagAnswer[]>([]);
  const [triage, setTriage] = useState<{
    category: string;
    priority: string;
    assignee_id: number;
    confidence: number;
  } | null>(null);
  const [summary, setSummary] = useState<string>("");

  // 添加优先级映射函数
  const priorityMap: { [key: string]: string } = {
    低: "low",
    中: "medium",
    高: "high",
    紧急: "critical",
  };

  // 反向映射显示如需可启用

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

      if (response && response.id) {
        router.push(`/tickets/${response.id}`);
      } else {
        setError("创建工单失败");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "网络错误");
    } finally {
      setLoading(false);
    }
  };

  const handleInputChange = (
    field: keyof CreateTicketRequest,
    value: unknown
  ) => {
    setFormData((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleAISuggest = async () => {
    if (!formData.title.trim() && !formData.description.trim()) return;
    setAiLoading(true);
    setAiError(null);
    try {
      const [t, kb, sum] = await Promise.all([
        aiTriage(formData.title, formData.description),
        aiSearchKB(`${formData.title}\n${formData.description}`, 5),
        aiSummarize(`${formData.title}\n${formData.description}`, 160),
      ]);
      setTriage(t);
      setKbAnswers(kb.answers || []);
      setSummary(sum.summary);
      // 回填优先级
      if (t?.priority) {
        // triage 返回的可能是 low|medium|high|urgent，简单映射
        const p = t.priority.toLowerCase();
        const normalized =
          p === "urgent"
            ? "critical"
            : ["low", "medium", "high", "critical"].includes(p)
            ? p
            : "medium";
        setFormData((prev) => ({
          ...prev,
          priority: normalized as CreateTicketRequest["priority"],
          assignee_id: t.assignee_id || prev.assignee_id,
          form_fields: {
            ...(prev.form_fields || {}),
            category: t.category || (prev.form_fields as any)?.category || "",
          },
        }));
      }
      // TODO: 回填受理人（assignee_id）与分类（需要前端有对应字段/下拉）
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : "智能建议失败";
      setAiError(msg);
    } finally {
      setAiLoading(false);
    }
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

          {/* 受理人ID（可选） */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              受理人ID（可选）
            </label>
            <input
              type="number"
              value={formData.assignee_id ?? ""}
              onChange={(e) =>
                handleInputChange(
                  "assignee_id",
                  e.target.value ? Number(e.target.value) : undefined
                )
              }
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="填写受理人用户ID"
            />
          </div>

          {/* 分类（可选，写入表单字段） */}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              分类（可选）
            </label>
            <input
              type="text"
              value={(formData.form_fields as any)?.category || ""}
              onChange={(e) =>
                setFormData((prev) => ({
                  ...prev,
                  form_fields: {
                    ...(prev.form_fields || {}),
                    category: e.target.value,
                  },
                }))
              }
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="如 database / network / general"
            />
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

          {/* 智能建议 */}
          <div className="p-4 border rounded-md bg-gray-50">
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center">
                <Brain className="w-4 h-4 mr-2" />
                <span className="font-semibold">智能建议</span>
              </div>
              <button
                type="button"
                onClick={handleAISuggest}
                disabled={aiLoading}
                className="text-sm flex items-center px-2 py-1 border rounded hover:bg-gray-100 disabled:opacity-50"
              >
                <Sparkles className="w-4 h-4 mr-1" /> 一键生成
              </button>
            </div>
            {aiError && (
              <div className="text-sm text-red-600 mb-2">{aiError}</div>
            )}
            {triage && (
              <div className="text-sm text-gray-700 mb-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center">
                    <UserCircle className="w-4 h-4 mr-1" />
                    建议分类：{triage.category}，优先级：{triage.priority}
                    ，建议受理人ID：{triage.assignee_id}，置信度：
                    {Math.round((triage.confidence || 0) * 100)}%
                  </div>
                  <AIFeedback
                    kind="triage"
                    query={`${formData.title}\n${formData.description}`}
                    className="ml-2"
                  />
                </div>
              </div>
            )}
            {summary && (
              <div className="text-sm text-gray-600 mb-2">
                <div className="flex items-center justify-between">
                  <span>摘要：{summary}</span>
                  <AIFeedback
                    kind="summarize"
                    query={`${formData.title}\n${formData.description}`}
                    className="ml-2"
                  />
                </div>
              </div>
            )}
            {kbAnswers.length > 0 && (
              <div>
                <div className="text-sm font-semibold mb-2 flex items-center">
                  <BookOpen className="w-4 h-4 mr-1" /> 知识推荐
                </div>
                <ul className="space-y-2">
                  {kbAnswers.map((a, idx) => (
                    <li
                      key={idx}
                      className="text-sm p-2 bg-white border rounded"
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="font-medium">{a.title || "相关内容"}</div>
                          <div className="text-gray-600">{a.snippet}</div>
                        </div>
                        <AIFeedback
                          kind="search"
                          query={`${formData.title}\n${formData.description}`}
                          itemType="knowledge"
                          itemId={a.id}
                          className="ml-2 flex-shrink-0"
                        />
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            )}
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
