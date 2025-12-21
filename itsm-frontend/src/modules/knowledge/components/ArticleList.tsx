'use client';

/**
 * 知识库文章列表组件
 */

import React, { useState, useEffect } from 'react';
import {
    Table, Tag, Button, Card, Space, Tooltip,
    Input, Select, Form, message, Modal, Breadcrumb
} from 'antd';
import {
    SearchOutlined, EyeOutlined, EditOutlined,
    PlusOutlined, ReloadOutlined, DeleteOutlined
} from '@ant-design/icons';
import { useRouter } from 'next/navigation';
import dayjs from 'dayjs';

import { KnowledgeApi } from '../api';
import { KnowledgeStatus, KnowledgeStatusLabels, KnowledgeStatusColors } from '../constants';
import type { KnowledgeArticle, ArticleQuery } from '../types';

const { Option } = Select;

const ArticleList: React.FC = () => {
    const router = useRouter();
    const [loading, setLoading] = useState(false);
    const [data, setData] = useState<KnowledgeArticle[]>([]);
    const [total, setTotal] = useState(0);
    const [categories, setCategories] = useState<string[]>([]);
    const [form] = Form.useForm();

    const [query, setQuery] = useState<ArticleQuery>({
        page: 1,
        page_size: 10,
    });

    const loadCategories = async () => {
        try {
            const res = await KnowledgeApi.getCategories();
            setCategories(res || []);
        } catch (e) {
            console.error(e);
        }
    };

    const loadData = async () => {
        setLoading(true);
        try {
            const values = await form.validateFields();
            const resp = await KnowledgeApi.getArticles({
                ...query,
                ...values,
            });
            setData(resp.articles || []);
            setTotal(resp.total || 0);
        } catch (error) {
            console.error(error);
            message.error('加载文章列表失败');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        loadCategories();
        loadData();
    }, [query]);

    const handleSearch = () => {
        setQuery(prev => ({ ...prev, page: 1 }));
    };

    const handleDelete = (id: number) => {
        Modal.confirm({
            title: '确定要删除此文章吗？',
            content: '删除后无法恢复。',
            onOk: async () => {
                try {
                    await KnowledgeApi.deleteArticle(id);
                    message.success('删除成功');
                    loadData();
                } catch (e) {
                    message.error('删除失败');
                }
            }
        });
    };

    const columns = [
        {
            title: 'ID',
            dataIndex: 'id',
            width: 70,
        },
        {
            title: '标题',
            dataIndex: 'title',
            ellipsis: true,
            render: (text: string, record: KnowledgeArticle) => (
                <a onClick={() => router.push(`/knowledge/articles/${record.id}`)}>{text}</a>
            )
        },
        {
            title: '分类',
            dataIndex: 'category',
            width: 120,
        },
        {
            title: '标签',
            dataIndex: 'tags',
            render: (tags: string[]) => (
                <>
                    {tags?.map(tag => (
                        <Tag key={tag}>{tag}</Tag>
                    ))}
                </>
            ),
        },
        {
            title: '状态',
            dataIndex: 'is_published',
            width: 100,
            render: (is_published: boolean) => {
                const status = is_published ? KnowledgeStatus.PUBLISHED : KnowledgeStatus.DRAFT;
                return (
                    <Tag color={KnowledgeStatusColors[status]}>
                        {KnowledgeStatusLabels[status]}
                    </Tag>
                );
            },
        },
        {
            title: '更新时间',
            dataIndex: 'updated_at',
            width: 160,
            render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
        },
        {
            title: '操作',
            key: 'action',
            width: 120,
            render: (_: any, record: KnowledgeArticle) => (
                <Space size="small">
                    <Tooltip title="编辑">
                        <Button
                            type="text"
                            icon={<EditOutlined />}
                            onClick={() => router.push(`/knowledge/articles/${record.id}/edit`)}
                        />
                    </Tooltip>
                    <Tooltip title="删除">
                        <Button
                            type="text"
                            danger
                            icon={<DeleteOutlined />}
                            onClick={() => handleDelete(record.id)}
                        />
                    </Tooltip>
                </Space>
            ),
        },
    ];

    return (
        <Card bordered={false}>
            <Breadcrumb style={{ marginBottom: 16 }}>
                <Breadcrumb.Item>首页</Breadcrumb.Item>
                <Breadcrumb.Item>知识库</Breadcrumb.Item>
                <Breadcrumb.Item>文章列表</Breadcrumb.Item>
            </Breadcrumb>

            <Form form={form} layout="inline" style={{ marginBottom: 24 }}>
                <Form.Item name="search">
                    <Input placeholder="搜索标题/内容" allowClear prefix={<SearchOutlined />} />
                </Form.Item>
                <Form.Item name="category">
                    <Select placeholder="分类" style={{ width: 140 }} allowClear>
                        {categories.map(c => (
                            <Option key={c} value={c}>{c}</Option>
                        ))}
                    </Select>
                </Form.Item>
                <Form.Item name="status">
                    <Select placeholder="状态" style={{ width: 110 }} allowClear>
                        <Option value="published">已发布</Option>
                        <Option value="draft">草稿</Option>
                    </Select>
                </Form.Item>
                <Form.Item>
                    <Space>
                        <Button type="primary" onClick={handleSearch}>查询</Button>
                        <Button icon={<ReloadOutlined />} onClick={loadData} />
                    </Space>
                </Form.Item>
                <div style={{ flex: 1, textAlign: 'right' }}>
                    <Button
                        type="primary"
                        icon={<PlusOutlined />}
                        onClick={() => router.push('/knowledge/articles/create')}
                    >
                        新建文章
                    </Button>
                </div>
            </Form>

            <Table
                rowKey="id"
                columns={columns as any}
                dataSource={data}
                loading={loading}
                pagination={{
                    current: query.page,
                    pageSize: query.page_size,
                    total: total,
                    onChange: (page, page_size) => setQuery(prev => ({ ...prev, page, page_size })),
                }}
            />
        </Card>
    );
};

export default ArticleList;
