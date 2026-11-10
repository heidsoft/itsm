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
        message.warning(data.message || 'AI summary service is temporarily unavailable');
      } else if (data.summary) {
        message.success('AI summary generated successfully');
      }
    } catch (e) {
      const err = e instanceof Error ? e.message : String(e);
      setState({ loading: false, error: err });
      message.error(`AI summary call failed:${err}`);
    }
  };

  return (
    <div className="py-2">
      <Alert
        type="info"
        showIcon
        title="AI Intelligent Summary"
        description="Automatically generate a summary based on the title, description and history to help quickly understand the full story of the event. Important information is subject to manual confirmation."
      />
      <Space className="mt-3 mb-3">
        <Button
          type="primary"
          icon={<Sparkles size={14} />}
          loading={state.loading}
          onClick={runSummarize}
        >

          Generate AI summary
        </Button>
        {state.summary && <Tag color="green">Generated</Tag>}
        {state.degraded && <Tag color="orange">Downgrade</Tag>}
      </Space>

      {state.loading && (
        <div className="py-8 text-center">
          <Spin tip="Generating summary..." />
        </div>
      )}

      {state.error && (
        <Alert
          type="error"
          showIcon
          title="AI summary call failed"
          description={state.error}
          className="mt-2"
        />
      )}

      {state.degraded && !state.loading && (
        <Alert
          type="warning"
          showIcon
          title={state.message || 'AI summary service is temporarily unavailable'}
          description="Possible reasons: The LLM gateway is not configured, the model call fails, or the network is abnormal. Please try again later or contact your administrator."
          className="mt-2"
        />
      )}

      {state.summary && !state.loading && (
        <Card size="small" className="mt-2 bg-gray-50">
          <pre className="whitespace-pre-wrap text-sm m-0">{state.summary}</pre>
        </Card>
      )}

      {!state.summary && !state.loading && !state.error && !state.degraded && (
        <Empty description="Click the button above to generate AI summary" />
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
      message.success('AI analysis completed');
    } catch (e) {
      const err = e instanceof Error ? e.message : String(e);
      setState({ loading: false, error: err });
      message.error(`AI analysis failed:${err}`);
    }
  };

  return (
    <div className="py-2">
      <Alert
        type="info"
        showIcon
        title="AI Impact Analysis"
        description="Based on the event content and historical data, analyze the potential impact scope, associated resources and disposal suggestions. Results are provided to assist decision-making only."
      />
      <Space className="mt-3 mb-3">
        <Button
          type="primary"
          icon={<Bot size={14} />}
          loading={state.loading}
          onClick={runAnalyze}
        >

          Perform AI analysis
        </Button>
      </Space>

      {state.loading && (
        <div className="py-8 text-center">
          <Spin tip="Performing AI analysis..." />
        </div>
      )}

      {state.error && (
        <Alert
          type="error"
          showIcon
          title="AI analysis failed"
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
        <Empty description="Click the button above to perform AI analysis" />
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

                      AI Intelligent Summary
                    </span>
                  ),
                  children: <AISummaryPanel incidentId={numericId} />,
                },
                {
                  key: 'ai-analyze',
                  label: (
                    <span>
                      <Bot size={14} className="inline mr-1" />

                      AI Impact Analysis
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
