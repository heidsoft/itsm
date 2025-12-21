'use client';

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import { BarChart3, TrendingUp, Clock, Star, MessageSquare } from 'lucide-react';
import { aiGetMetrics, type AIMetrics as AIMetricsType } from '@/lib/api/ai-api';

interface AIMetricsProps {
  className?: string;
  days?: number;
}

const AIMetricsComponent: React.FC<AIMetricsProps> = ({ className = '', days = 7 }) => {
  const [metrics, setMetrics] = useState<AIMetricsType | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // 将所有hooks移到组件顶部
  const formatResponseTime = useCallback((seconds: number) => {
    if (seconds < 1) return `${Math.round(seconds * 1000)}ms`;
    if (seconds < 60) return `${seconds.toFixed(1)}s`;
    return `${Math.round(seconds / 60)}m ${Math.round(seconds % 60)}s`;
  }, []);

  const getKindLabel = useCallback((kind: string) => {
    const labels: Record<string, string> = {
      triage: '智能分类',
      search: '知识搜索',
      summarize: '智能摘要',
      'similar-incidents': '相似事件',
      chat: 'AI对话',
    };
    return labels[kind] || kind;
  }, []);

  // 缓存排序后的数据
  const sortedKindData = useMemo(() => {
    if (!metrics?.by_kind) return [];
    return Object.entries(metrics.by_kind).sort(([, a], [, b]) => b - a);
  }, [metrics?.by_kind]);

  const loadMetrics = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await aiGetMetrics(days);
      setMetrics(data);
    } catch (err) {
      console.error('AI Metrics加载失败:', err);

      // 根据错误类型提供不同的错误信息
      let errorMessage = '加载指标失败';
      if (err instanceof Error) {
        if (err.message.includes('401') || err.message.includes('Authentication failed')) {
          errorMessage = '需要登录才能查看AI指标';
        } else if (err.message.includes('500')) {
          errorMessage = '服务器暂时不可用，请稍后重试';
        } else if (err.message.includes('404')) {
          errorMessage = 'AI指标接口不可用';
        } else if (err.message.includes('Failed to fetch')) {
          errorMessage = '网络连接失败，请检查网络设置';
        } else {
          errorMessage = err.message;
        }
      }

      setError(errorMessage);
    } finally {
      setLoading(false);
    }
  }, [days]);

  useEffect(() => {
    loadMetrics();
  }, [loadMetrics]);

  if (loading) {
    return (
      <div className={`p-4 bg-white rounded-lg shadow ${className}`}>
        <div className='animate-pulse'>
          <div className='h-4 bg-gray-200 rounded w-1/3 mb-4'></div>
          <div className='space-y-3'>
            <div className='h-3 bg-gray-200 rounded'></div>
            <div className='h-3 bg-gray-200 rounded w-5/6'></div>
            <div className='h-3 bg-gray-200 rounded w-4/6'></div>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className={`p-4 bg-red-50 border border-red-200 rounded-lg ${className}`}>
        <div className='text-red-600 text-sm'>{error}</div>
        <button
          onClick={loadMetrics}
          className='mt-2 text-xs text-red-700 hover:text-red-800 underline'
        >
          重试
        </button>
      </div>
    );
  }

  if (!metrics) {
    return null;
  }

  return (
    <div className={`bg-white rounded-lg shadow ${className}`}>
      {/* Header */}
      <div className='p-4 border-b border-gray-200'>
        <div className='flex items-center justify-between'>
          <h3 className='text-lg font-semibold text-gray-800 flex items-center'>
            <BarChart3 className='w-5 h-5 mr-2 text-blue-600' />
            AI 使用指标
          </h3>
          <span className='text-sm text-gray-500'>最近 {days} 天</span>
        </div>
      </div>

      {/* Metrics Grid */}
      <div className='p-4'>
        <div className='grid grid-cols-2 md:grid-cols-4 gap-4 mb-6'>
          {/* Total Requests */}
          <div className='text-center p-3 bg-blue-50 rounded-lg'>
            <div className='text-2xl font-bold text-blue-600'>{metrics.total_requests}</div>
            <div className='text-sm text-gray-600 flex items-center justify-center'>
              <TrendingUp className='w-4 h-4 mr-1' />
              总请求数
            </div>
          </div>

          {/* Total Feedback */}
          <div className='text-center p-3 bg-green-50 rounded-lg'>
            <div className='text-2xl font-bold text-green-600'>{metrics.total_feedback}</div>
            <div className='text-sm text-gray-600 flex items-center justify-center'>
              <MessageSquare className='w-4 h-4 mr-1' />
              反馈总数
            </div>
          </div>

          {/* Useful Rate */}
          <div className='text-center p-3 bg-yellow-50 rounded-lg'>
            <div className='text-2xl font-bold text-yellow-600'>
              {Math.round(metrics.useful_rate * 100)}%
            </div>
            <div className='text-sm text-gray-600 flex items-center justify-center'>
              <Star className='w-4 h-4 mr-1' />
              有用率
            </div>
          </div>

          {/* Avg Response Time */}
          <div className='text-center p-3 bg-purple-50 rounded-lg'>
            <div className='text-2xl font-bold text-purple-600'>
              {formatResponseTime(metrics.avg_response_time_seconds)}
            </div>
            <div className='text-sm text-gray-600 flex items-center justify-center'>
              <Clock className='w-4 h-4 mr-1' />
              平均响应
            </div>
          </div>
        </div>

        {/* Feedback Details */}
        <div className='mb-6'>
          <h4 className='text-md font-semibold text-gray-700 mb-3'>反馈详情</h4>
          <div className='bg-gray-50 rounded-lg p-3'>
            <div className='flex items-center justify-between text-sm'>
              <span className='text-gray-600'>有用反馈</span>
              <span className='font-semibold text-green-600'>{metrics.useful_feedback}</span>
            </div>
            <div className='flex items-center justify-between text-sm mt-1'>
              <span className='text-gray-600'>无用反馈</span>
              <span className='font-semibold text-red-600'>
                {metrics.total_feedback - metrics.useful_feedback}
              </span>
            </div>
          </div>
        </div>

        {/* Usage by Kind */}
        {sortedKindData.length > 0 && (
          <div>
            <h4 className='text-md font-semibold text-gray-700 mb-3'>按功能分类</h4>
            <div className='space-y-2'>
              {sortedKindData.map(([kind, count]) => (
                <div
                  key={kind}
                  className='flex items-center justify-between p-2 bg-gray-50 rounded'
                >
                  <span className='text-sm text-gray-600'>{getKindLabel(kind)}</span>
                  <span className='text-sm font-semibold text-gray-800'>{count}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

// 使用React.memo优化性能
export const AIMetrics = React.memo(AIMetricsComponent, (prevProps, nextProps) => {
  return prevProps.className === nextProps.className && prevProps.days === nextProps.days;
});
