"use client";

import { TrendingUp, AlertTriangle, ArrowLeft } from 'lucide-react';

import React, { useState, useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

// 模拟SLA详情数据
const mockSLADetail = {
  "SLA-001": {
    name: "生产CRM系统可用性",
    service: "CRM系统",
    target: "99.9% 可用性",
    actual: "99.85%",
    status: "轻微违约",
    lastReview: "2025-06-01",
    description:
      "确保生产CRM系统在服务时间内保持高可用性，以支持销售和客服团队的日常运营。",
    serviceDescription:
      "本SLA涵盖生产CRM系统的Web服务、应用服务和数据库服务。服务提供方为IT运维部，客户为销售部和客服部。目标是提供稳定、高效的客户关系管理服务。",
    serviceQualityTarget: "99.9% 可用性 (每月)",
    responseTimeTarget: "P1事件15分钟内响应，P2事件30分钟内响应",
    resolutionTimeTarget: "P1事件4小时内解决，P2事件8小时内解决",
    performanceMeasures:
      "可用性通过Prometheus监控系统自动采集，响应和解决时间通过ServiceNow工单系统记录。每月生成SLA报告。",
    penalties: "若月度可用性低于99.5%，将对销售部进行服务费减免5%。",
    cancellationConditions:
      "若客户连续三个月未支付服务费用，或服务需求发生重大变化且无法通过协商达成一致，SLA自动失效。",
    metrics: [
      { date: "2025-01", value: 99.95 },
      { date: "2025-02", value: 99.92 },
      { date: "2025-03", value: 99.98 },
      { date: "2025-04", value: 99.9 },
      { date: "2025-05", value: 99.88 },
      { date: "2025-06", value: 99.85 },
    ],
    violations: [
      {
        date: "2025-06-10",
        description: "CRM系统因数据库连接问题停机15分钟。",
      },
    ],
  },
  "SLA-002": {
    name: "内部IT服务台响应时间",
    service: "IT服务台",
    target: "80% 15分钟内响应",
    actual: "85%",
    status: "达标",
    lastReview: "2025-06-15",
    description: "确保用户提交的服务请求在规定时间内得到服务台的首次响应。",
    serviceDescription:
      "本SLA涵盖所有通过服务台提交的服务请求和事件。服务提供方为IT服务台团队，客户为公司全体员工。目标是提供及时、高效的IT支持服务。",
    serviceQualityTarget: "80% 的服务请求在15分钟内首次响应",
    responseTimeTarget: "所有服务请求在15分钟内首次响应",
    resolutionTimeTarget: "根据请求类型，解决时间目标不同，详见服务目录。",
    performanceMeasures:
      "通过工单系统自动记录首次响应时间。每月生成服务台绩效报告。",
    penalties: "若连续两个月响应时间达标率低于70%，将启动持续改进计划。",
    cancellationConditions:
      "若服务台团队结构发生重大调整，或服务范围发生重大变化，SLA需重新协商。",
    metrics: [
      { date: "2025-01", value: 75 },
      { date: "2025-02", value: 80 },
      { date: "2025-03", value: 82 },
      { date: "2025-04", value: 85 },
      { date: "2025-05", value: 83 },
      { date: "2025-06", value: 85 },
    ],
    violations: [],
  },
};

const SLAStatusBadge = ({ status }) => {
  const colors = {
    达标: "bg-green-100 text-green-800",
    轻微违约: "bg-yellow-100 text-yellow-800",
    违约: "bg-red-100 text-red-800",
  };
  return (
    <span
      className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status]}`}
    >
      {status}
    </span>
  );
};

const SLADetailPage = () => {
  const params = useParams();
  const router = useRouter();
  const slaId = params.slaId as string;
  const sla = mockSLADetail[slaId];

  if (!sla) {
    return <div className="p-10">SLA不存在或加载失败。</div>;
  }

  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8">
        <button
          onClick={() => router.back()}
          className="flex items-center text-blue-600 hover:underline mb-4"
        >
          <ArrowLeft className="w-5 h-5 mr-2" />
          返回SLA列表
        </button>
        <h2 className="text-4xl font-bold text-gray-800">
          SLA详情：{sla.name}
        </h2>
        <p className="text-gray-500 mt-1">服务对象: {sla.service}</p>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* 左侧：SLA描述和性能趋势 */}
        <div className="lg:col-span-2 bg-white p-8 rounded-lg shadow-md">
          <h3 className="text-xl font-semibold text-gray-700 mb-4">服务描述</h3>
          <p className="text-gray-600 mb-8">{sla.serviceDescription}</p>

          <h3 className="text-xl font-semibold text-gray-700 mb-4">
            服务质量与响应性目标
          </h3>
          <ul className="list-disc list-inside text-gray-600 mb-8 space-y-2">
            <li>
              <span className="font-semibold">服务质量目标:</span>{" "}
              {sla.serviceQualityTarget}
            </li>
            <li>
              <span className="font-semibold">响应时间目标:</span>{" "}
              {sla.responseTimeTarget}
            </li>
            <li>
              <span className="font-semibold">解决时间目标:</span>{" "}
              {sla.resolutionTimeTarget}
            </li>
          </ul>

          <h3 className="text-xl font-semibold text-gray-700 mb-4">绩效衡量</h3>
          <p className="text-gray-600 mb-8">{sla.performanceMeasures}</p>

          <h3 className="text-xl font-semibold text-gray-700 mb-4">附加条款</h3>
          <p className="text-gray-600 mb-2">
            <span className="font-semibold">惩罚:</span> {sla.penalties}
          </p>
          <p className="text-gray-600 mb-8">
            <span className="font-semibold">取消条件:</span>{" "}
            {sla.cancellationConditions}
          </p>

          <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
            <TrendingUp className="w-5 h-5 mr-2" /> 性能趋势
          </h3>
          <div className="h-80 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart
                data={sla.metrics}
                margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
              >
                <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
                <XAxis dataKey="date" stroke="#6b7280" />
                <YAxis stroke="#6b7280" domain={["auto", "auto"]} />
                <Tooltip />
                <Line
                  type="monotone"
                  dataKey="value"
                  stroke="#3b82f6"
                  activeDot={{ r: 8 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* 右侧：SLA信息和违约记录 */}
        <div className="space-y-8">
          <div className="bg-white p-6 rounded-lg shadow-md">
            <h3 className="text-xl font-semibold text-gray-700 mb-4">
              SLA 概览
            </h3>
            <div className="space-y-3 text-sm">
              <div className="flex justify-between">
                <span>目标:</span>
                <span className="font-semibold">{sla.target}</span>
              </div>
              <div className="flex justify-between">
                <span>实际达成:</span>
                <span className="font-semibold">{sla.actual}</span>
              </div>
              <div className="flex justify-between">
                <span>状态:</span>
                <SLAStatusBadge status={sla.status} />
              </div>
              <div className="flex justify-between">
                <span>最后评审:</span>
                <span className="font-semibold">{sla.lastReview}</span>
              </div>
            </div>
          </div>

          <div className="bg-white p-6 rounded-lg shadow-md">
            <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
              <AlertTriangle className="w-5 h-5 mr-2" /> 违约记录
            </h3>
            <div className="space-y-3">
              {sla.violations.length > 0 ? (
                sla.violations.map((violation, i) => (
                  <div key={i} className="p-3 bg-red-50 rounded-lg">
                    <p className="font-semibold text-red-700">
                      {violation.date}
                    </p>
                    <p className="text-sm text-gray-600">
                      {violation.description}
                    </p>
                  </div>
                ))
              ) : (
                <p className="text-sm text-gray-500">无违约记录。</p>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default SLADetailPage;
