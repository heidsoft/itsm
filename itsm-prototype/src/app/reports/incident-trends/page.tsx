"use client";

import React from "react";
import { ChartPlaceholder } from "../../components/ChartPlaceholder";
import { mockIncidentsData } from "../../lib/mock-data";

const IncidentTrendsPage = () => {
  // 数据聚合逻辑已简化，使用占位符组件

  return (
    <div className="p-10 bg-gray-50 min-h-full">
      <header className="mb-8">
        <h2 className="text-4xl font-bold text-gray-800">
          Incident Trends Report
        </h2>
        <p className="text-gray-500 mt-1">
          This report shows the trend of incidents over time and by priority.
        </p>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
        <div className="bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-xl font-semibold text-gray-800 mb-4">
            Incidents per Day
          </h3>
          <ChartPlaceholder
            type="line"
            title="每日事件数量趋势"
            description="显示每日事件数量的变化趋势"
            height={300}
          />
        </div>

        <div className="bg-white p-6 rounded-lg shadow-md">
          <h3 className="text-xl font-semibold text-gray-800 mb-4">
            Incidents by Priority
          </h3>
          <ChartPlaceholder
            type="bar"
            title="按优先级统计事件"
            description="显示不同优先级的事件数量分布"
            height={300}
          />
        </div>
      </div>
    </div>
  );
};

export default IncidentTrendsPage;
