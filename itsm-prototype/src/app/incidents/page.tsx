"use client";

import React, { useState, useEffect } from "react";
import Link from "next/link";
import {
  AlertTriangle,
  Shield,
  User,
  Cpu,
  Tag,
  Filter,
  Clock,
  PlusCircle,
  Zap,
} from "lucide-react";

import { IncidentAPI, Incident } from "../lib/incident-api";

const PriorityBadge = ({ priority }) => {
  const colors = {
    高: "bg-red-100 text-red-800 border-red-300",
    中: "bg-yellow-100 text-yellow-800 border-yellow-300",
    低: "bg-blue-100 text-blue-800 border-blue-300",
  };
  return (
    <span
      className={`px-2 py-1 text-xs font-semibold rounded-full border ${colors[priority]}`}
    >
      {priority}
    </span>
  );
};

const StatusBadge = ({ status }) => {
  const colors = {
    处理中: "bg-blue-100 text-blue-800",
    已分配: "bg-purple-100 text-purple-800",
    已解决: "bg-green-100 text-green-800",
    已关闭: "bg-gray-200 text-gray-800",
    挂起: "bg-yellow-100 text-yellow-800",
  };
  return (
    <span
      className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status]}`}
    >
      {status}
    </span>
  );
};

const IncidentListPage = () => {
  const [filter, setFilter] = useState("全部");
  const [showMajorIncidents, setShowMajorIncidents] = useState(false);

  const filteredIncidents = mockIncidentsData.filter((inc) => {
    const matchesStatus = filter === "全部" || inc.status === filter;
    const matchesMajor = !showMajorIncidents || inc.isMajorIncident;
    return matchesStatus && matchesMajor;
  });

  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8 flex justify-between items-center">
        <div>
          <h2 className="text-4xl font-bold text-gray-800">事件管理中心</h2>
          <p className="text-gray-500 mt-1">统一响应、处理和分析所有IT事件</p>
        </div>
        <Link href="/incidents/new" passHref>
          <button className="flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors">
            <PlusCircle className="w-5 h-5 mr-2" />
            新建事件
          </button>
        </Link>
      </header>

      {/* 筛选器 */}
      <div className="flex flex-wrap items-center mb-6 bg-white p-3 rounded-lg shadow-sm gap-4">
        <Filter className="w-5 h-5 text-gray-500 mr-1" />
        <span className="text-sm font-semibold">筛选:</span>
        <div className="flex space-x-2">
          {["全部", "处理中", "已分配", "已解决", "已关闭", "挂起"].map((f) => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-3 py-1 text-sm rounded-md ${
                filter === f
                  ? "bg-blue-600 text-white"
                  : "bg-gray-200 text-gray-700"
              }`}
            >
              {f}
            </button>
          ))}
        </div>
        <div className="flex items-center ml-4">
          <input
            type="checkbox"
            id="showMajorIncidents"
            checked={showMajorIncidents}
            onChange={(e) => setShowMajorIncidents(e.target.checked)}
            className="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
          />
          <label
            htmlFor="showMajorIncidents"
            className="ml-2 text-sm font-medium text-gray-700"
          >
            只显示重大事件
          </label>
        </div>
      </div>

      {/* 事件列表表格 */}
      <div className="bg-white rounded-lg shadow-md overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                事件ID
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                优先级
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                标题
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                状态
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                来源
              </th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                最后更新
              </th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {filteredIncidents.length > 0 ? (
              filteredIncidents.map((inc) => (
                <tr
                  key={inc.id}
                  className={`hover:bg-gray-50 ${
                    inc.isMajorIncident
                      ? "bg-red-50/50 border-l-4 border-red-500"
                      : ""
                  }`}
                >
                  <td className="px-6 py-4 whitespace-nowrap">
                    <Link
                      href={`/incidents/${inc.id}`}
                      className="text-blue-600 font-semibold hover:underline flex items-center"
                    >
                      {inc.id}
                      {inc.isMajorIncident && (
                        <Zap
                          className="w-4 h-4 ml-2 text-red-600"
                          title="重大事件"
                        />
                      )}
                    </Link>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <PriorityBadge priority={inc.priority} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap max-w-sm truncate">
                    {inc.title}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <StatusBadge status={inc.status} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center">
                      <inc.sourceIcon className="w-4 h-4 mr-2 text-gray-600" />
                      {inc.source}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {inc.lastUpdate}
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td
                  colSpan={6}
                  className="px-6 py-4 whitespace-nowrap text-center text-sm text-gray-500"
                >
                  没有找到匹配的事件。
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default IncidentListPage;
