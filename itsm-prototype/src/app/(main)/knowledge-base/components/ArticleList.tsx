'use client';

import React from 'react';
import Link from 'next/link';
import { Table, Tag, Button, Space, Badge } from 'antd';
import { Eye, Edit, MoreHorizontal, BookOpen } from 'lucide-react';
import { Problem, ProblemStatus, ProblemPriority } from '@/lib/services/problem-service';
import { useI18n } from '@/lib/i18n';

interface Article {
  id: string;
  title: string;
  summary: string;
  category: string;
  views: number;
  content?: string;
}

const getCategoryColor = (category: string) => {
  const colors: Record<string, string> = {
    账号管理: 'blue',
    故障排除: 'red',
    网络连接: 'green',
    流程指南: 'purple',
    系统配置: 'orange',
  };
  return colors[category] || 'default';
};

interface ArticleListProps {
  articles: Article[];
  loading: boolean;
  selectedRowKeys: React.Key[];
  onSelectedRowKeysChange: (keys: React.Key[]) => void;
}

export const ArticleList: React.FC<ArticleListProps> = ({
  articles,
  loading,
  selectedRowKeys,
  onSelectedRowKeysChange,
}) => {
    const { t } = useI18n();
  const columns = [
    {
      title: t('knowledgeBase.articleInfo'),
      key: 'article_info',
      width: 400,
      render: (_: unknown, record: Article) => (
        <div style={{ display: 'flex', alignItems: 'center' }}>
          <div
            style={{
              width: 40,
              height: 40,
              backgroundColor: '#e6f7ff',
              borderRadius: 8,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              marginRight: 12,
            }}
          >
            <BookOpen size={20} style={{ color: '#1890ff' }} />
          </div>
          <div>
            <div style={{ fontWeight: 'medium', color: '#000', marginBottom: 4 }}>
              <Link
                href={`/knowledge-base/${record.id}`}
                style={{ textDecoration: 'none', color: 'inherit' }}
              >
                {record.title}
              </Link>
            </div>
            <div style={{ fontSize: 'small', color: '#666' }}>{record.summary}</div>
          </div>
        </div>
      ),
    },
    {
      title: t('knowledgeBase.category'),
      dataIndex: 'category',
      key: 'category',
      width: 120,
      render: (category: string) => <Tag color={getCategoryColor(category)}>{category}</Tag>,
    },
    {
      title: t('knowledgeBase.views'),
      dataIndex: 'views',
      key: 'views',
      width: 100,
      render: (views: number) => <div style={{ fontSize: 'small' }}>{views}</div>,
    },
    {
      title: t('knowledgeBase.actions'),
      key: 'actions',
      width: 150,
      render: (_: unknown, record: Article) => (
        <Space size='small'>
          <Button
            type='text'
            size='small'
            icon={<Eye size={16} />}
            onClick={() => window.open(`/knowledge-base/${record.id}`)}
          />
          <Button
            type='text'
            size='small'
            icon={<Edit size={16} />}
            onClick={() => window.open(`/knowledge-base/${record.id}/edit`)}
          />
          <Button type='text' size='small' icon={<MoreHorizontal size={16} />} />
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 16 }}>
        <div>
          {selectedRowKeys.length > 0 && (
            <Badge count={selectedRowKeys.length} showZero style={{ backgroundColor: '#1890ff' }} />
          )}
        </div>
      </div>

      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: onSelectedRowKeysChange,
        }}
        columns={columns}
        dataSource={articles}
        rowKey='id'
        loading={loading}
        pagination={{
          pageSize: 10,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
        }}
        scroll={{ x: 1000 }}
      />
    </div>
  );
};
