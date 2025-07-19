"use client";

import React, { useState, useEffect } from "react";

export default function DebugAuthPage() {
  const [authInfo, setAuthInfo] = useState<any>({});
  const [apiTestResult, setApiTestResult] = useState<any>(null);

  useEffect(() => {
    const info = {
      access_token: localStorage.getItem("access_token"),
      refresh_token: localStorage.getItem("refresh_token"),
      current_tenant_id: localStorage.getItem("current_tenant_id"),
      user: localStorage.getItem("user"),
      tenant: localStorage.getItem("tenant"),
    };
    setAuthInfo(info);
  }, []);

  const testApiCall = async () => {
    try {
      const token = localStorage.getItem("access_token");
      const tenantId = localStorage.getItem("current_tenant_id");

      console.log("Testing API with token:", token);
      console.log("Testing API with tenant ID:", tenantId);

      const response = await fetch(
        "http://localhost:8080/api/incidents?page=1&page_size=10",
        {
          headers: {
            Authorization: `Bearer ${token}`,
            "X-Tenant-ID": tenantId || "1",
            "Content-Type": "application/json",
          },
        }
      );

      console.log("Response status:", response.status);
      console.log("Response headers:", response.headers);

      const data = await response.json();
      console.log("Response data:", data);

      setApiTestResult({
        status: response.status,
        data: data,
        headers: Object.fromEntries(response.headers.entries()),
      });

      alert(
        `API响应状态: ${response.status}\n数据: ${JSON.stringify(
          data,
          null,
          2
        )}`
      );
    } catch (error) {
      console.error("API调用失败:", error);
      alert(`API调用失败: ${error}`);
    }
  };

  const testHttpClient = async () => {
    try {
      const { httpClient } = await import("../lib/http-client");
      console.log("HTTP Client token:", httpClient.getTenantId());

      const response = await httpClient.get("/api/incidents", {
        page: 1,
        page_size: 10,
      });
      console.log("HTTP Client response:", response);

      alert(`HTTP Client响应: ${JSON.stringify(response, null, 2)}`);
    } catch (error) {
      console.error("HTTP Client测试失败:", error);
      alert(`HTTP Client测试失败: ${error}`);
    }
  };

  return (
    <div className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-2xl font-bold mb-6">认证调试页面</h1>

        <div className="bg-white p-6 rounded-lg shadow mb-6">
          <h2 className="text-lg font-semibold mb-4">当前认证信息</h2>
          <pre className="bg-gray-100 p-4 rounded text-sm overflow-auto">
            {JSON.stringify(authInfo, null, 2)}
          </pre>
        </div>

        <div className="bg-white p-6 rounded-lg shadow mb-6">
          <h2 className="text-lg font-semibold mb-4">操作</h2>
          <div className="space-y-4">
            <button
              onClick={testApiCall}
              className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
            >
              测试原生fetch API调用
            </button>

            <button
              onClick={testHttpClient}
              className="bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700 ml-4"
            >
              测试HTTP Client
            </button>
          </div>
        </div>

        {apiTestResult && (
          <div className="bg-white p-6 rounded-lg shadow mb-6">
            <h2 className="text-lg font-semibold mb-4">API测试结果</h2>
            <pre className="bg-gray-100 p-4 rounded text-sm overflow-auto">
              {JSON.stringify(apiTestResult, null, 2)}
            </pre>
          </div>
        )}

        <div className="bg-white p-6 rounded-lg shadow">
          <h2 className="text-lg font-semibold mb-4">手动设置认证信息</h2>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-2">
                Access Token
              </label>
              <input
                type="text"
                className="w-full p-2 border rounded"
                value={authInfo.access_token || ""}
                onChange={(e) => {
                  localStorage.setItem("access_token", e.target.value);
                  setAuthInfo({ ...authInfo, access_token: e.target.value });
                }}
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2">
                Tenant ID
              </label>
              <input
                type="text"
                className="w-full p-2 border rounded"
                value={authInfo.current_tenant_id || ""}
                onChange={(e) => {
                  localStorage.setItem("current_tenant_id", e.target.value);
                  setAuthInfo({
                    ...authInfo,
                    current_tenant_id: e.target.value,
                  });
                }}
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
