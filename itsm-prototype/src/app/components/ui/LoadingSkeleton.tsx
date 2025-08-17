"use client";

import React from "react";
import { Skeleton } from "antd";

interface LoadingSkeletonProps {
  type?: "table" | "card" | "form" | "list" | "chart";
  rows?: number;
  columns?: number;
  className?: string;
}

export const LoadingSkeleton: React.FC<LoadingSkeletonProps> = ({
  type = "card",
  rows = 3,
  columns = 1,
  className = "",
}) => {
  const renderTableSkeleton = () => (
    <div className={className}>
      {/* 表头 */}
      <div className="mb-4">
        <Skeleton.Input active size="large" style={{ width: "100%", height: 40 }} />
      </div>
      
      {/* 表格行 */}
      {Array.from({ length: rows }).map((_, index) => (
        <div key={index} className="mb-3">
          <div className="flex gap-3">
            {Array.from({ length: columns }).map((_, colIndex) => (
              <Skeleton.Input
                key={colIndex}
                active
                size="small"
                style={{ 
                  width: colIndex === 0 ? "20%" : colIndex === columns - 1 ? "15%" : "100%",
                  height: 32 
                }}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  );

  const renderCardSkeleton = () => (
    <div className={className}>
      <div className="space-y-4">
        {/* 标题 */}
        <Skeleton.Input active size="large" style={{ width: "60%", height: 24 }} />
        
        {/* 内容 */}
        {Array.from({ length: rows }).map((_, index) => (
          <Skeleton.Input
            key={index}
            active
            size="small"
            style={{ 
              width: index === 0 ? "100%" : index === 1 ? "80%" : "60%",
              height: 16 
            }}
          />
        ))}
        
        {/* 操作按钮 */}
        <div className="flex gap-2 pt-2">
          <Skeleton.Button active size="small" style={{ width: 80, height: 32 }} />
          <Skeleton.Button active size="small" style={{ width: 60, height: 32 }} />
        </div>
      </div>
    </div>
  );

  const renderFormSkeleton = () => (
    <div className={className}>
      <div className="space-y-6">
        {/* 表单标题 */}
        <Skeleton.Input active size="large" style={{ width: "40%", height: 28 }} />
        
        {/* 表单项 */}
        {Array.from({ length: rows }).map((_, index) => (
          <div key={index} className="space-y-2">
            <Skeleton.Input active size="small" style={{ width: "30%", height: 16 }} />
            <Skeleton.Input active size="large" style={{ width: "100%", height: 40 }} />
          </div>
        ))}
        
        {/* 提交按钮 */}
        <div className="pt-4">
          <Skeleton.Button active size="large" style={{ width: 120, height: 40 }} />
        </div>
      </div>
    </div>
  );

  const renderListSkeleton = () => (
    <div className={className}>
      <div className="space-y-3">
        {Array.from({ length: rows }).map((_, index) => (
          <div key={index} className="flex items-center space-x-3 p-3 border rounded-lg">
            <Skeleton.Avatar active size="large" />
            <div className="flex-1 space-y-2">
              <Skeleton.Input active size="small" style={{ width: "40%", height: 16 }} />
              <Skeleton.Input active size="small" style={{ width: "80%", height: 14 }} />
            </div>
            <Skeleton.Button active size="small" style={{ width: 60, height: 28 }} />
          </div>
        ))}
      </div>
    </div>
  );

  const renderChartSkeleton = () => (
    <div className={className}>
      <div className="space-y-4">
        {/* 图表标题 */}
        <Skeleton.Input active size="large" style={{ width: "50%", height: 24 }} />
        
        {/* 图表区域 */}
        <div className="bg-gray-50 rounded-lg p-6">
          <div className="flex items-end justify-between h-48">
            {Array.from({ length: 7 }).map((_, index) => (
              <div key={index} className="flex flex-col items-center space-y-2">
                <Skeleton.Input
                  active
                  size="small"
                  style={{ 
                    width: 20,
                    height: Math.random() * 100 + 50
                  }}
                />
                <Skeleton.Input active size="small" style={{ width: 40, height: 12 }} />
              </div>
            ))}
          </div>
        </div>
        
        {/* 图例 */}
        <div className="flex justify-center space-x-6">
          {Array.from({ length: 3 }).map((_, index) => (
            <div key={index} className="flex items-center space-x-2">
              <Skeleton.Avatar active size="small" shape="square" />
              <Skeleton.Input active size="small" style={{ width: 60, height: 14 }} />
            </div>
          ))}
        </div>
      </div>
    </div>
  );

  switch (type) {
    case "table":
      return renderTableSkeleton();
    case "card":
      return renderCardSkeleton();
    case "form":
      return renderFormSkeleton();
    case "list":
      return renderListSkeleton();
    case "chart":
      return renderChartSkeleton();
    default:
      return renderCardSkeleton();
  }
};

// 预定义的骨架屏组件
export const TableSkeleton = (props: Omit<LoadingSkeletonProps, "type">) => (
  <LoadingSkeleton type="table" {...props} />
);

export const CardSkeleton = (props: Omit<LoadingSkeletonProps, "type">) => (
  <LoadingSkeleton type="card" {...props} />
);

export const FormSkeleton = (props: Omit<LoadingSkeletonProps, "type">) => (
  <LoadingSkeleton type="form" {...props} />
);

export const ListSkeleton = (props: Omit<LoadingSkeletonProps, "type">) => (
  <LoadingSkeleton type="list" {...props} />
);

export const ChartSkeleton = (props: Omit<LoadingSkeletonProps, "type">) => (
  <LoadingSkeleton type="chart" {...props} />
);
