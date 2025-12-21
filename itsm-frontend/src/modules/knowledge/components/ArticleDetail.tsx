'use client';

/**
 * 知识库文章详情组件
 */

import React, { useState, useEffect } from 'react';
import {
    Card, Tag, Button, Skeleton, Result,
    Typography, Space, Breadcrumb, message, Divider
} from 'antd';
import { ArrowLeftOutlined, EditOutlined, UserOutlined, CalendarOutlined, FolderOutlined } from '@ant-design/icons';
import { useParams, useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { KnowledgeApi } from '../api';
import { KnowledgeStatus, KnowledgeStatusLabels, KnowledgeStatusColors } from '../constants';
import type { KnowledgeArticle } from '../types';

const { Title, Paragraph, Text } = Typography;

const ArticleDetail: React.FC = () => {
    const { id } = useParams() as { id: string };
    const router = useRouter();
    const [loading, setLoading] = useState(true);
    const [article, setArticle] = useState<KnowledgeArticle | null>(null);

    useEffect(() => {
        if (id) {
            loadDetail();
        }
    }, [id]);

    const loadDetail = async () => {
        setLoading(true);
        try {
            const data = await KnowledgeApi.getArticle(id!);
            setArticle(data);
        } catch (error) {
            console.error(error);
            message.error('加载文章详情失败');
        } finally {
            setLoading(false);
        }
    };

    if (loading) return <Card><Skeleton active /></Card>;

    if (!article) {
        return (
            <Card>
                <Result
                    status="404"
                    title="404"
                    subTitle="抱歉，您访问的文章不存在"
                    extra={<Button type="primary" onClick={() => router.push('/knowledge')}>返回列表</Button>}
                />
            </Card>
        );
    }

    const status = article.is_published ? KnowledgeStatus.PUBLISHED : KnowledgeStatus.DRAFT;

    return (
        <div style={{ padding: '0 0 24px' }}>
            <Breadcrumb style={{ marginBottom: 16 }}>
                <Breadcrumb.Item>首页</Breadcrumb.Item>
                <Breadcrumb.Item>知识库</Breadcrumb.Item>
                <Breadcrumb.Item onClick={() => router.push('/knowledge')}>文章列表</Breadcrumb.Item>
                <Breadcrumb.Item>文章详情</Breadcrumb.Item>
            </Breadcrumb>

            <Card>
                <div style={{ marginBottom: 24 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                        <Button
                            icon={<ArrowLeftOutlined />}
                            onClick={() => router.push('/knowledge')}
                        >
                            返回列表
                        </Button>
                        <Button
                            type="primary"
                            icon={<EditOutlined />}
                            onClick={() => router.push(`/knowledge/articles/${article.id}/edit`)}
                        >
                            编辑文章
                        </Button>
                    </div>

                    <Title level={2}>{article.title}</Title>

                    <Space split={<Divider type="vertical" />} wrap>
                        <Space><UserOutlined /><Text type="secondary">作者 ID: {article.author_id}</Text></Space>
                        <Space><CalendarOutlined /><Text type="secondary">{dayjs(article.created_at).format('YYYY-MM-DD HH:mm')}</Text></Space>
                        <Space><FolderOutlined /><Text type="secondary">{article.category || '未分类'}</Text></Space>
                        <Tag color={KnowledgeStatusColors[status]}>{KnowledgeStatusLabels[status]}</Tag>
                    </Space>
                </div>

                <div style={{ marginBottom: 16 }}>
                    {article.tags?.map(tag => (
                        <Tag key={tag}>{tag}</Tag>
                    ))}
                </div>

                <Divider />

                <div className="article-content" style={{ minHeight: 400 }}>
                    <Paragraph>
                        {article.content.split('\n').map((line, i) => (
                            <React.Fragment key={i}>
                                {line}
                                <br />
                            </React.Fragment>
                        ))}
                    </Paragraph>
                </div>
            </Card>
        </div>
    );
};

export default ArticleDetail;
