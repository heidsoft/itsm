'use client';

import { useState, useEffect } from 'react';
import {
  problemService,
  Problem,
  ProblemStatus,
  ListProblemsParams,
} from '@/lib/services/problem-service';
import { message } from 'antd';

export const useProblemsData = () => {
  const [problems, setProblems] = useState<Problem[]>([]);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<string>('');
  const [searchText, setSearchText] = useState('');
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });
  const [stats, setStats] = useState({
    total: 0,
    open: 0,
    inProgress: 0,
    resolved: 0,
  });

  useEffect(() => {
    fetchProblems();
    fetchStats();
  }, [pagination.current, pagination.pageSize, filter, searchText]);

  const fetchProblems = async () => {
    setLoading(true);
    try {
      const params: ListProblemsParams = {
        page: pagination.current,
        page_size: pagination.pageSize,
      };

      if (filter) {
        Object.assign(params, { status: filter as ProblemStatus });
      }

      if (searchText) {
        Object.assign(params, { keyword: searchText });
      }

      const response = await problemService.listProblems(params);
      setProblems(response.problems);
      setPagination(prev => ({ ...prev, total: response.total }));
    } catch {
      message.error('获取问题列表失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      // TODO: 实现统计数据获取
      console.log('获取统计数据');
    } catch {
      message.error('获取统计数据失败');
    }
  };

  const handleTableChange = (page: number, pageSize?: number) => {
    setPagination(prev => ({
      ...prev,
      current: page,
      pageSize: pageSize || prev.pageSize,
    }));
  };

  return {
    problems,
    loading,
    filter,
    setFilter,
    searchText,
    setSearchText,
    pagination,
    handleTableChange,
    stats,
    fetchProblems,
  };
};
