'use client';

import React, { useState } from 'react';
import { App, Button, Card, Tabs, Alert, Spin, Empty, Space, Tag } from 'antd';
import { ArrowLeft, MessageSquare, Clock as HistoryIcon, Bot, Sparkles } from 'lucide-react';
import { useRouter, useParams } from 'next/navigation';
import IncidentDetail from '@/components/incident/IncidentDetail';
import {
  CommentPanel,
  HistoryTimeline,
  incidentCommentAdapter,
  fetchAuditLogHistory,
} from '@/components/business/detail-tabs';
import { useAuthStore } from '@/lib/store/auth-store';
import { useI18n } from '@/lib/i18n/useI18n';
import { AIApi } from '@/lib/api/ai-api';
import dayjs from 'dayjs';

const formatDateTime = (v?: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm') : '-');

interface AISummaryState {
  loading: boolean;
  summary?: string;
  degraded?: boolean;
  message?: string;
  error?: string;
}

interface AIAnalyzeState {
  loading: boolean;
  result?: unknown;
  error?: string;
}

function AISummaryPanel({ incidentId }: { incidentId: number }) {
  const { t } = useI18n();
  const { message } = App.useApp();
  const [state, setState] = useState<AISummaryState>({ loading: false });

  const runSummarize = async () => {
    setState({ loading: true });
    try {
      const data = await AIApi.summarizeTicket(incidentId);
      setState({
        loading: false,
        summary: data.summary,
        degraded: data.degraded,
        message: data.message,
      });
      if (data.degraded) {
        message.warning(data.message || 'AI 摘要服务暂时不可用');
      } else if (data.summary) {
        message.success('AI 摘要生成成功');
      }
    } catch (e) {
      const err = e instanceof Error ? e.message : String(e);
      setState({ loading: false, error: err });
      message.error(`AI 摘要调用失败：${err}`);
    }
  };

  return (
    <div className="py-2">
      <Alert
        type="info"
        showIcon
        message="AI 智能摘要"
        description="基于事件标题、描述、评论与历史，自动生成结构化摘要，便于快速了解事件全貌。AI 摘要属于辅助决策，重要信息请以人工确认为准。"
      />
      <Space className="mt-3 mb-3">
        <Button
          type="primary"
          icon={<Sparkles size={14} />}
          loading={state.loading}
          onClick={runSummarize}
        >
          生成 AI 摘要
        </Button>
        {state.summary && <Tag color="green">已生成</Tag>}
        {state.degraded && <Tag color="orange">降级</Tag>}
      </Space>

      {state.loading && (
        <div className="py-8 text-center">
          <Spin tip="正在生成摘要..." />
        </div>
      )}

      {state.error && (
        <Alert
          type="error"
          showIcon
          message="AI 摘要调用失败"
          description={state.error}
          className="mt-2"
        />
      )}

      {state.degraded && !state.loading && (
        <Alert
          type="warning"
          showIcon
          message={state.message || 'AI 摘要服务暂时不可用'}
          description="可能原因：LLM 网关未配置、模型调用失败或网络异常。请稍后重试或联系管理员。"
          className="mt-2"
        />
      )}

      {state.summary && !state.loading && (
        <Card size="small" className="mt-2 bg-gray-50">
          <pre className="whitespace-pre-wrap text-sm m-0">{state.summary}</pre>
        </Card>
      )}

      {!state.summary && !state.loading && !state.error && !state.degraded && (
        <Empty description="点击上方按钮生成 AI 摘要" />
      )}
    </div>
  );
}

function AIAnalyzePanel({ incidentId }: { incidentId: number }) {
  const { t } = useI18n();
  const { message } = App.useApp();
  const [state, setState] = useState<AIAnalyzeState>({ loading: false });

  const runAnalyze = async () => {
    setState({ loading: true });
    try {
      const result = await AIApi.analyzeIncident(incidentId);
      setState({ loading: false, result });
      message.success('AI 分析完成');
    } catch (e) {
      const err = e instanceof Error ? e.message : String(e);
      setState({ loading: false, error: err });
      message.error(`AI 分析失败：${err}`);
    }
  };

  return (
    <div className="py-2">
      <Alert
        type="info"
        showIcon
        message="AI 影响分析"
        description="基于事件内容与历史数据，分析潜在影响范围、关联资源与处置建议。结果仅供辅助决策。"
      />
      <Space className="mt-3 mb-3">
        <Button
          type="primary"
          icon={<Bot size={14} />}
          loading={state.loading}
          onClick={runAnalyze}
        >
          执行 AI 分析
        </Button>
      </Space>

      {state.loading && (
        <div className="py-8 text-center">
          <Spin tip="正在执行 AI 分析..." />
        </div>
      )}

      {state.error && (
        <Alert
          type="error"
          showIcon
          message="AI 分析失败"
          description={state.error}
          className="mt-2"
        />
      )}

      {state.result !== undefined && !state.loading && (
        <Card size="small" className="mt-2 bg-gray-50">
          <pre className="whitespace-pre-wrap text-sm m-0">
            {typeof state.result === 'string'
              ? state.result
              : JSON.stringify(state.result, null, 2)}
          </pre>
        </Card>
      )}

      {state.result === undefined && !state.loading && !state.error && (
        <Empty description="点击上方按钮执行 AI 分析" />
      )}
    </div>
  );
}

export default function IncidentDetailPage() {
  const router = useRouter();
  const params = useParams();
  const id = params?.id as string;
  const numericId = Number(id);
  const { user } = useAuthStore();
  const { t } = useI18n();

  return (
    <App>
      <div style={{ padding: 24 }}>
        <div style={{ marginBottom: 16 }}>
          <Button
            type="link"
            icon={<ArrowLeft />}
            onClick={() => router.back()}
            style={{ paddingLeft: 0, color: '#666' }}
          >
            {t('common.back')}
          </Button>
        </div>
        <IncidentDetail id={id} />

        {Number.isFinite(numericId) && numericId > 0 && (
          <Card className="mt-4 rounded-lg shadow-sm border border-gray-200">
            <Tabs
              defaultActiveKey="comments"
              items={[
                {
                  key: 'comments',
                  label: (
                    <span>
                      <MessageSquare size={14} className="inline mr-1" />
                      {t('detailTabs.comments')}
                    </span>
                  ),
                  children: (
                    <CommentPanel
                      targetType="incident"
                      targetId={numericId}
                      adapter={incidentCommentAdapter}
                      currentUserId={user?.id}
                      formatDateTime={formatDateTime}
                      showInternalToggle={false}
                    />
                  ),
                },
                {
                  key: 'ai-summary',
                  label: (
                    <span>
                      <Sparkles size={14} className="inline mr-1" />
                      AI 智能摘要
                    </span>
                  ),
                  children: <AISummaryPanel incidentId={numericId} />,
                },
                {
                  key: 'ai-analyze',
                  label: (
                    <span>
                      <Bot size={14} className="inline mr-1" />
                      AI 影响分析
                    </span>
                  ),
                  children: <AIAnalyzePanel incidentId={numericId} />,
                },
                {
                  key: 'history',
                  label: (
                    <span>
                      <HistoryIcon size={14} className="inline mr-1" />
                      {t('detailTabs.history')}
                    </span>
                  ),
                  children: (
                    <HistoryTimeline
                      targetType="incident"
                      targetId={numericId}
                      fetchAuditLog={fetchAuditLogHistory}
                      formatDateTime={formatDateTime}
                    />
                  ),
                },
              ]}
            />
          </Card>
        )}
      </div>
    </App>
  );
}
