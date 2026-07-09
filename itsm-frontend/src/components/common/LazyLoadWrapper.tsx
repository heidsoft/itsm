import React from 'react';
import { Skeleton, Card } from 'antd';

interface LazyLoadWrapperProps {
  /** 加载中显示的高度 */
  height?: number | string;
  /** 是否显示卡片样式 */
  card?: boolean;
  /** 加载提示文本 */
  tip?: string;
  children: React.ReactNode;
}

export default function LazyLoadWrapper({
  height = 300,
  card = false,
  tip = '加载中...',
  children,
}: LazyLoadWrapperProps) {
  const content = (
    <div className="w-full" style={{ height }}>
      <Skeleton active paragraph={{ rows: 8 }} title={{ width: '40%' }} />
      {tip && <p className="text-center text-gray-400 text-sm mt-4">{tip}</p>}
    </div>
  );

  if (card) {
    return (
      <Card className="w-full" bodyStyle={{ padding: '24px' }}>
        <React.Suspense fallback={content}>{children}</React.Suspense>
      </Card>
    );
  }

  return <React.Suspense fallback={content}>{children}</React.Suspense>;
}
