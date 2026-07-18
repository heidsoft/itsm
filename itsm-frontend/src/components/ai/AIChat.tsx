'use client';

/**
 * AI 智能助手 — 流式回答 + 引用来源
 *
 * 交互特性：
 *  - SSE 推送 token，助手气泡实时增长；期间用户可"停止生成"取消流。
 *  - 每条助手消息挂一组引用来源（RagAnswer），点击展开可查看片段与得分。
 *  - 流失败时自动降级为一次性 chat 调用，保证有可读回答。
 */

import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Avatar,
  Button,
  Card,
  Divider,
  Empty,
  Input,
  List,
  Space,
  Spin,
  Tag,
  Typography,
  message as antdMessage,
} from 'antd';
import { Bot, Eraser, LoaderCircle, Send, StopCircle, User } from 'lucide-react';

import { AIApi, aiChatStream, type RagAnswer } from '@/lib/api/ai-api';

const { Text, Paragraph } = Typography;

interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  createdAt: string;
  streaming?: boolean;
  sources?: RagAnswer[];
  error?: string;
}

const nextId = () => `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;

const AIChat: React.FC = () => {
  const [query, setQuery] = useState('');
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [convId, setConvId] = useState<number | undefined>(undefined);
  const [streaming, setStreaming] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);

  const scrollToBottom = useCallback(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages, scrollToBottom]);

  useEffect(() => {
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  const updateAssistant = useCallback((assistantId: string, patch: Partial<ChatMessage>) => {
    setMessages(prev => prev.map(m => (m.id === assistantId ? { ...m, ...patch } : m)));
  }, []);

  const appendAssistantContent = useCallback((assistantId: string, delta: string) => {
    setMessages(prev =>
      prev.map(m => (m.id === assistantId ? { ...m, content: m.content + delta } : m))
    );
  }, []);

  const runStreaming = useCallback(
    async (userMsg: ChatMessage, assistantId: string) => {
      const controller = new AbortController();
      abortRef.current = controller;
      setStreaming(true);

      try {
        const finalConvId = await aiChatStream(
          {
            query: userMsg.content,
            conversationId: convId,
            limit: 5,
            signal: controller.signal,
          },
          {
            onSources: sources => {
              updateAssistant(assistantId, { sources });
            },
            onDelta: delta => {
              appendAssistantContent(assistantId, delta);
            },
            onDone: newConvId => {
              if (newConvId) setConvId(newConvId);
              updateAssistant(assistantId, { streaming: false });
            },
            onError: msg => {
              updateAssistant(assistantId, { streaming: false, error: msg });
            },
          }
        );
        if (finalConvId) setConvId(finalConvId);
      } catch (err) {
        const aborted = (err as Error)?.name === 'AbortError';
        if (aborted) {
          updateAssistant(assistantId, { streaming: false, error: '已停止生成' });
          return;
        }
        // Fallback: try non-streaming chat so the user still gets an answer.
        try {
          const res = await AIApi.chat({
            query: userMsg.content,
            conversationId: convId,
            limit: 5,
          });
          const answers: unknown[] = Array.isArray(res?.answers) ? res.answers : [];
          const fallbackText = answers
            .map(a => (typeof a === 'string' ? a : JSON.stringify(a)))
            .join('\n\n');
          updateAssistant(assistantId, {
            streaming: false,
            content: fallbackText || '抱歉，我没有找到相关的答案。',
            sources: answers.filter((a): a is RagAnswer => typeof a === 'object' && a !== null),
          });
          if (res?.conversationId) setConvId(res.conversationId);
        } catch (fallbackErr) {
          const fallbackMsg =
            fallbackErr instanceof Error ? fallbackErr.message : '流式请求与降级请求均失败';
          updateAssistant(assistantId, { streaming: false, error: fallbackMsg });
          antdMessage.error('AI 回答失败，请稍后重试');
        }
      } finally {
        abortRef.current = null;
        setStreaming(false);
      }
    },
    [appendAssistantContent, convId, updateAssistant]
  );

  const handleSend = useCallback(() => {
    const trimmed = query.trim();
    if (!trimmed || streaming) return;

    const userMsg: ChatMessage = {
      id: nextId(),
      role: 'user',
      content: trimmed,
      createdAt: new Date().toISOString(),
    };
    const assistantMsg: ChatMessage = {
      id: nextId(),
      role: 'assistant',
      content: '',
      createdAt: new Date().toISOString(),
      streaming: true,
    };
    setMessages(prev => [...prev, userMsg, assistantMsg]);
    setQuery('');
    void runStreaming(userMsg, assistantMsg.id);
  }, [query, streaming, runStreaming]);

  const handleStop = useCallback(() => {
    abortRef.current?.abort();
  }, []);

  const handleClear = useCallback(() => {
    if (streaming) return;
    setMessages([]);
    setConvId(undefined);
  }, [streaming]);

  const isEmpty = useMemo(() => messages.length === 0, [messages]);

  return (
    <Card
      title={
        <Space size={8}>
          <Bot size={18} />
          <span>AI 助手</span>
          {streaming ? (
            <Tag color="processing" icon={<LoaderCircle size={12} className="animate-spin" />}>
              生成中
            </Tag>
          ) : null}
        </Space>
      }
      extra={
        <Space>
          {streaming ? (
            <Button size="small" danger icon={<StopCircle size={14} />} onClick={handleStop}>
              停止生成
            </Button>
          ) : null}
          <Button size="small" icon={<Eraser size={14} />} onClick={handleClear} disabled={streaming}>
            清空对话
          </Button>
        </Space>
      }
      style={{ height: 'calc(100vh - 150px)', display: 'flex', flexDirection: 'column' }}
      styles={{
        body: {
          flex: 1,
          overflow: 'hidden',
          display: 'flex',
          flexDirection: 'column',
          padding: 0,
        },
      }}
    >
      <div ref={scrollRef} style={{ flex: 1, overflowY: 'auto', padding: '16px' }}>
        {isEmpty ? (
          <Empty
            description="发起提问，AI 将结合知识库为你回答，并给出引用来源。"
            style={{ marginTop: 96 }}
          />
        ) : (
          <List
            itemLayout="horizontal"
            dataSource={messages}
            renderItem={item => (
              <List.Item style={{ border: 'none', padding: '12px 0', alignItems: 'flex-start' }}>
                <List.Item.Meta
                  avatar={
                    <Avatar
                      icon={item.role === 'user' ? <User size={16} /> : <Bot size={16} />}
                      style={{
                        backgroundColor: item.role === 'user' ? '#1890ff' : '#52c41a',
                      }}
                    />
                  }
                  title={
                    <Space size={6}>
                      <Text strong>{item.role === 'user' ? '你' : 'AI 助手'}</Text>
                      {item.streaming ? <Tag color="processing">流式</Tag> : null}
                      {item.error ? <Tag color="error">失败</Tag> : null}
                    </Space>
                  }
                  description={
                    <div
                      style={{
                        backgroundColor: item.role === 'user' ? '#f0f2f5' : '#f6ffed',
                        padding: '12px 14px',
                        borderRadius: 10,
                        color: '#000',
                        border: '1px solid #e6f4ff',
                      }}
                    >
                      {item.content ? (
                        <Paragraph
                          style={{ marginBottom: item.sources?.length ? 12 : 0, whiteSpace: 'pre-wrap' }}
                        >
                          {item.content}
                          {item.streaming ? <span className="ai-caret">▍</span> : null}
                        </Paragraph>
                      ) : item.streaming ? (
                        <Space size={6} align="center">
                          <Spin size="small" />
                          <Text type="secondary">检索知识库并生成回答…</Text>
                        </Space>
                      ) : null}
                      {item.error ? (
                        <Text type="danger" style={{ display: 'block', marginTop: 8 }}>
                          {item.error}
                        </Text>
                      ) : null}
                      {item.sources && item.sources.length > 0 ? (
                        <SourceList sources={item.sources} />
                      ) : null}
                    </div>
                  }
                />
              </List.Item>
            )}
          />
        )}
      </div>

      <Divider style={{ margin: 0 }} />

      <div style={{ padding: '16px' }}>
        <Space.Compact style={{ width: '100%' }}>
          <Input
            placeholder="请输入你的问题，Enter 发送"
            value={query}
            onChange={e => setQuery(e.target.value)}
            onPressEnter={handleSend}
            disabled={streaming}
          />
          <Button type="primary" icon={<Send size={14} />} onClick={handleSend} loading={streaming}>
            发送
          </Button>
        </Space.Compact>
      </div>
    </Card>
  );
};

const SourceList: React.FC<{ sources: RagAnswer[] }> = ({ sources }) => {
  return (
    <div style={{ marginTop: 6 }}>
      <Text type="secondary" style={{ fontSize: 12 }}>
        引用来源 · {sources.length}
      </Text>
      <ul style={{ paddingLeft: 18, marginTop: 6, marginBottom: 0 }}>
        {sources.map((s, idx) => {
          const title = s.title || s.snippet?.slice(0, 30) || `来源 ${idx + 1}`;
          const scoreLabel =
            typeof s.score === 'number' ? `${(s.score * 100).toFixed(0)}%` : undefined;
          return (
            <li key={`${s.objectType}-${s.id}-${idx}`} style={{ marginBottom: 6 }}>
              <Space size={6} wrap>
                <Tag color="blue">{s.objectType || 'source'}</Tag>
                <Text strong style={{ fontSize: 13 }}>
                  {title}
                </Text>
                {scoreLabel ? <Tag color="green">相关度 {scoreLabel}</Tag> : null}
              </Space>
              {s.snippet ? (
                <Paragraph
                  type="secondary"
                  style={{ marginTop: 4, marginBottom: 0, fontSize: 12 }}
                  ellipsis={{ rows: 2, expandable: true, symbol: '展开' }}
                >
                  {s.snippet}
                </Paragraph>
              ) : null}
            </li>
          );
        })}
      </ul>
    </div>
  );
};

export default AIChat;
