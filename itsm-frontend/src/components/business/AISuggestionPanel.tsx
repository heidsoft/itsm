/**
 * AISuggestionPanel - AI-powered ticket classification suggestions
 *
 * Shows AI-suggested category, priority, and assignee for a ticket.
 * Allows user to accept or dismiss suggestions.
 */

'use client';

import React, { useState, useEffect } from 'react';
import { Card, Typography, Tag, Button, Space, Spin, Progress, Alert } from 'antd';
import { Sparkles, Check, X, RefreshCw, AlertCircle } from 'lucide-react';
import { aiTriage, type TriageResult } from '@/lib/api/ai-api';

const { Text, Paragraph } = Typography;

interface AISuggestionPanelProps {
  title: string;
  description?: string;
  onAccept?: (suggestion: TriageResult) => void;
  onDismiss?: () => void;
  initialSuggestion?: TriageResult | null;
}

const categoryColors: Record<string, string> = {
  database: 'blue',
  network: 'cyan',
  server: 'purple',
  application: 'green',
  security: 'red',
  storage: 'orange',
  user_access: 'gold',
  general: 'default',
};

const priorityColors: Record<string, string> = {
  critical: 'red',
  high: 'orange',
  medium: 'blue',
  low: 'green',
};

const categoryLabels: Record<string, string> = {
  database: '数据库',
  network: '网络',
  server: '服务器',
  application: '应用',
  security: '安全',
  storage: '存储',
  user_access: '用户访问',
  general: '通用',
};

const priorityLabels: Record<string, string> = {
  critical: '紧急',
  high: '高',
  medium: '中',
  low: '低',
};

export function AISuggestionPanel({
  title,
  description,
  onAccept,
  onDismiss,
  initialSuggestion,
}: AISuggestionPanelProps) {
  const [loading, setLoading] = useState(false);
  const [suggestion, setSuggestion] = useState<TriageResult | null>(initialSuggestion ?? null);
  const [error, setError] = useState<string | null>(null);
  const [collapsed, setCollapsed] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  // Fetch AI suggestion when title/description change
  useEffect(() => {
    if (!title || dismissed) return;

    const fetchSuggestion = async () => {
      setLoading(true);
      setError(null);
      try {
        const result = await aiTriage(title, description || '');
        setSuggestion(result);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'AI分析失败');
        setSuggestion(null);
      } finally {
        setLoading(false);
      }
    };

    // Debounce the API call
    const timer = setTimeout(fetchSuggestion, 800);
    return () => clearTimeout(timer);
  }, [title, description, dismissed]);

  const handleAccept = () => {
    if (suggestion && onAccept) {
      onAccept(suggestion);
    }
  };

  const handleDismiss = () => {
    setDismissed(true);
    setSuggestion(null);
    onDismiss?.();
  };

  const handleRefresh = () => {
    setDismissed(false);
    setSuggestion(null);
    setError(null);
  };

  // Don't render if dismissed
  if (dismissed) {
    return (
      <Card
        size="small"
        className="mb-4 border-dashed border-2 border-gray-300 bg-gray-50"
        bodyStyle={{ padding: '12px' }}
      >
        <div className="flex items-center justify-between">
          <Space>
            <Sparkles className="w-4 h-4 text-gray-400" />
            <Text type="secondary" className="text-sm">
              AI建议已忽略
            </Text>
          </Space>
          <Button
            type="link"
            size="small"
            icon={<RefreshCw className="w-3 h-3" />}
            onClick={handleRefresh}
          >
            重新分析
          </Button>
        </div>
      </Card>
    );
  }

  // Loading state
  if (loading) {
    return (
      <Card
        size="small"
        className="mb-4 bg-gradient-to-r from-blue-50 to-indigo-50 border-blue-200"
        bodyStyle={{ padding: '12px' }}
      >
        <div className="flex items-center gap-3">
          <Spin size="small" />
          <div>
            <Text className="text-sm font-medium text-blue-700">AI智能分析中...</Text>
            <Paragraph type="secondary" className="text-xs mb-0">
              基于标题和描述分析工单分类
            </Paragraph>
          </div>
        </div>
      </Card>
    );
  }

  // Error state
  if (error) {
    return (
      <Card
        size="small"
        className="mb-4 bg-gray-50 border-gray-200"
        bodyStyle={{ padding: '12px' }}
      >
        <div className="flex items-center justify-between">
          <Space>
            <AlertCircle className="w-4 h-4 text-gray-400" />
            <Text type="secondary" className="text-sm">
              AI分析暂不可用
            </Text>
          </Space>
          <Button
            type="link"
            size="small"
            icon={<RefreshCw className="w-3 h-3" />}
            onClick={handleRefresh}
          >
            重试
          </Button>
        </div>
      </Card>
    );
  }

  // No suggestion yet
  if (!suggestion) {
    return null;
  }

  const confidencePercent = Math.round(suggestion.confidence * 100);
  const confidenceStatus =
    confidencePercent >= 70 ? 'success' : confidencePercent >= 40 ? 'normal' : 'exception';

  return (
    <Card
      size="small"
      className="mb-4 bg-gradient-to-r from-indigo-50 to-purple-50 border-indigo-200"
      bodyStyle={{ padding: '12px' }}
      title={
        <Space>
          <Sparkles className="w-4 h-4 text-indigo-600" />
          <span className="font-medium text-indigo-700">AI智能分析建议</span>
        </Space>
      }
      extra={
        <Button
          type="text"
          size="small"
          icon={<X className="w-3 h-3" />}
          onClick={() => setCollapsed(!collapsed)}
        />
      }
    >
      {collapsed ? (
        <Text type="secondary" className="text-xs">
          点击展开查看AI分析详情
        </Text>
      ) : (
        <>
          <div className="grid grid-cols-3 gap-3 mb-3">
            {/* Category */}
            <div className="text-center">
              <Text type="secondary" className="text-xs block mb-1">
                建议分类
              </Text>
              <Tag color={categoryColors[suggestion.category] || 'default'} className="text-sm">
                {categoryLabels[suggestion.category] || suggestion.category}
              </Tag>
            </div>

            {/* Priority */}
            <div className="text-center">
              <Text type="secondary" className="text-xs block mb-1">
                建议优先级
              </Text>
              <Tag color={priorityColors[suggestion.priority] || 'default'} className="text-sm">
                {priorityLabels[suggestion.priority] || suggestion.priority}
              </Tag>
            </div>

            {/* Confidence */}
            <div className="text-center">
              <Text type="secondary" className="text-xs block mb-1">
                置信度
              </Text>
              <Progress
                percent={confidencePercent}
                size="small"
                status={confidenceStatus as 'success' | 'normal' | 'exception'}
                strokeColor={
                  confidencePercent >= 70
                    ? '#52c41a'
                    : confidencePercent >= 40
                      ? '#1890ff'
                      : '#ff4d4f'
                }
                className="mb-0"
              />
            </div>
          </div>

          {/* Explanation */}
          {suggestion.explanation && (
            <Paragraph type="secondary" className="text-xs mb-3 bg-white/50 rounded p-2">
              {suggestion.explanation}
            </Paragraph>
          )}

          {/* Action buttons */}
          <div className="flex justify-end gap-2">
            <Button size="small" icon={<X className="w-3 h-3" />} onClick={handleDismiss}>
              忽略
            </Button>
            <Button
              type="primary"
              size="small"
              icon={<Check className="w-3 h-3" />}
              onClick={handleAccept}
            >
              采纳建议
            </Button>
          </div>
        </>
      )}
    </Card>
  );
}
