'use client';

import { useState, useEffect } from 'react';
import { KnowledgeApi } from '../lib/knowledge-api';

interface Article {
  id: string;
  title: string;
  summary: string;
  category: string;
  views: number;
  content?: string;
}

export const useKnowledgeBaseData = () => {
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchText, setSearchText] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('');
  const [stats, setStats] = useState({
    total: 0,
    accountManagement: 0,
    troubleshooting: 0,
    network: 0,
  });

  useEffect(() => {
    loadArticles();
    loadStats();
  }, []);

  const loadArticles = async () => {
    setLoading(true);
    try {
      const mockArticles: Article[] = [
        {
          id: '1',
          title: '如何重置企业邮箱密码',
          summary: '本文介绍了如何通过自助服务重置企业邮箱密码的详细步骤。',
          category: '账号管理',
          views: 128,
        },
        {
          id: '2',
          title: 'VPN连接常见问题及解决方案',
          summary: '整理了VPN连接过程中可能遇到的问题及其解决方案。',
          category: '故障排除',
          views: 96,
        },
        {
          id: '3',
          title: '内部系统访问权限申请流程',
          summary: '详细说明了如何申请和审批内部系统的访问权限。',
          category: '流程指南',
          views: 75,
        },
      ];
      setArticles(mockArticles);
    } catch (error) {
      console.error('加载知识库文章失败:', error);
    } finally {
      setLoading(false);
    }
  };

  const loadStats = async () => {
    try {
      // 模拟统计数据
      setStats({
        total: 42,
        accountManagement: 12,
        troubleshooting: 18,
        network: 8,
      });
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  };

  const filteredArticles = articles.filter(article => {
    const matchesSearch = searchText
      ? article.title.toLowerCase().includes(searchText.toLowerCase()) ||
        article.summary.toLowerCase().includes(searchText.toLowerCase())
      : true;
    const matchesCategory = categoryFilter ? article.category === categoryFilter : true;
    return matchesSearch && matchesCategory;
  });

  return {
    articles: filteredArticles,
    loading,
    stats,
    setSearchText,
    setCategoryFilter,
    loadArticles,
  };
};
