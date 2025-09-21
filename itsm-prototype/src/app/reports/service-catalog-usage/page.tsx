"use client";

import React from "react";
import { ChartPlaceholder } from "../../components/ChartPlaceholder";
import { mockRequestsData } from "../../lib/mock-data";

const ServiceCatalogUsagePage = () => {
  // 数据聚合逻辑已简化，使用占位符组件

  // 颜色配置已移除，使用占位符组件

  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8">
        <h2 className="text-4xl font-bold text-gray-800">
          Service Catalog Usage Report
        </h2>
        <p className="text-gray-500 mt-1">
          This report shows the usage of the service catalog.
        </p>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-xl font-semibold text-gray-800 mb-4">
            Requests by Service
          </h3>
          <ChartPlaceholder
            type="pie"
            title="按服务类型统计请求"
            description="显示不同服务类型的请求数量分布"
            height={300}
          />
        </div>

        <div className="bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-xl font-semibold text-gray-800 mb-4">
            Requests by Status
          </h3>
          <ChartPlaceholder
            type="bar"
            title="按状态统计请求"
            description="显示不同状态的请求数量分布"
            height={300}
          />
        </div>
      </div>
    </div>
  );
};

export default ServiceCatalogUsagePage;
