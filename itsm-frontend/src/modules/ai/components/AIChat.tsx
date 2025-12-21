'use client';

/**
 * AI 智能助手组件
 */

import React, { useState, useEffect, useRef } from 'react';
import {
    Card, Input, Button, List, Avatar,
    Typography, Space, Spin, message, Divider
} from 'antd';
import {
    SendOutlined, RobotOutlined, UserOutlined,
    ClearOutlined
} from '@ant-design/icons';
import { AIApi } from '../api';
import type { AIMessage } from '../types';

const { Text, Paragraph } = Typography;

const AIChat: React.FC = () => {
    const [query, setQuery] = useState('');
    const [loading, setLoading] = useState(false);
    const [messages, setMessages] = useState<AIMessage[]>([]);
    const [convId, setConvId] = useState<number | undefined>(undefined);
    const scrollRef = useRef<HTMLDivElement>(null);

    const scrollToBottom = () => {
        if (scrollRef.current) {
            scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
        }
    };

    useEffect(() => {
        scrollToBottom();
    }, [messages]);

    const handleSend = async () => {
        if (!query.trim()) return;

        const userMsg: AIMessage = {
            id: Date.now(),
            conversation_id: convId || 0,
            role: 'user',
            content: query,
            created_at: new Date().toISOString()
        };

        setMessages(prev => [...prev, userMsg]);
        setQuery('');
        setLoading(true);

        try {
            const res = await AIApi.chat({
                query: userMsg.content,
                conversation_id: convId,
                limit: 5
            });

            setConvId(res.conversation_id);

            // Format answers from list to string for display
            const assistantContent = res.answers.map(a =>
                typeof a === 'string' ? a : JSON.stringify(a)
            ).join('\n\n');

            const assistantMsg: AIMessage = {
                id: Date.now() + 1,
                conversation_id: res.conversation_id,
                role: 'assistant',
                content: assistantContent || '抱歉，我没有找到相关的答案。',
                created_at: new Date().toISOString()
            };

            setMessages(prev => [...prev, assistantMsg]);
        } catch (error) {
            message.error('发送失败，请稍后重试');
        } finally {
            setLoading(false);
        }
    };

    const handleClear = () => {
        setMessages([]);
        setConvId(undefined);
    };

    return (
        <Card
            title={<span><RobotOutlined /> AI 助手</span>}
            extra={<Button size="small" icon={<ClearOutlined />} onClick={handleClear}>清空</Button>}
            style={{ height: 'calc(100vh - 150px)', display: 'flex', flexDirection: 'column' }}
            bodyStyle={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column', padding: 0 }}
        >
            <div
                ref={scrollRef}
                style={{ flex: 1, overflowY: 'auto', padding: '16px' }}
            >
                <List
                    itemLayout="horizontal"
                    dataSource={messages}
                    renderItem={(item) => (
                        <List.Item style={{ border: 'none', padding: '12px 0' }}>
                            <List.Item.Meta
                                avatar={
                                    <Avatar
                                        icon={item.role === 'user' ? <UserOutlined /> : <RobotOutlined />}
                                        style={{ backgroundColor: item.role === 'user' ? '#1890ff' : '#52c41a' }}
                                    />
                                }
                                title={<Text strong>{item.role === 'user' ? '你' : 'AI 助手'}</Text>}
                                description={
                                    <div style={{
                                        backgroundColor: item.role === 'user' ? '#f0f2f5' : '#e6f7ff',
                                        padding: '12px',
                                        borderRadius: '8px',
                                        color: '#000'
                                    }}>
                                        <Paragraph style={{ marginBottom: 0, whiteSpace: 'pre-wrap' }}>
                                            {item.content}
                                        </Paragraph>
                                    </div>
                                }
                            />
                        </List.Item>
                    )}
                />
                {loading && (
                    <div style={{ textAlign: 'center', margin: '20px 0' }}>
                        <Spin tip="思考中..." />
                    </div>
                )}
            </div>

            <Divider style={{ margin: 0 }} />

            <div style={{ padding: '16px' }}>
                <Space.Compact style={{ width: '100%' }}>
                    <Input
                        placeholder="请输入您的问题..."
                        value={query}
                        onChange={(e) => setQuery(e.target.value)}
                        onPressEnter={handleSend}
                        disabled={loading}
                    />
                    <Button
                        type="primary"
                        icon={<SendOutlined />}
                        onClick={handleSend}
                        loading={loading}
                    >
                        发送
                    </Button>
                </Space.Compact>
            </div>
        </Card>
    );
};

export default AIChat;
