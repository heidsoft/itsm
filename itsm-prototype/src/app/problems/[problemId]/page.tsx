'use client';

import { Tag as TagIcon, ArrowLeft } from 'lucide-react';

import React, { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { Card, Button, message, Spin, Tag, Input } from 'antd';

const { TextArea } = Input;
import {
  problemService,
  Problem,
  ProblemStatus,
  ProblemPriority,
} from '../../lib/services/problem-service';

const ProblemDetailPage = () => {
  const params = useParams();
  const router = useRouter();
  const problemId = parseInt(params.problemId as string);
  const [problem, setProblem] = useState<Problem | null>(null);
  const [loading, setLoading] = useState(true);
  const [newComment, setNewComment] = useState('');

  // 加载问题详情
  useEffect(() => {
    if (problemId) {
      loadProblemDetail();
    }
  }, [problemId]);

  const loadProblemDetail = async () => {
    setLoading(true);
    try {
      const data = await problemService.getProblem(problemId);
      setProblem(data);
    } catch (error) {
      console.error('加载问题详情失败:', error);
      message.error('加载问题详情失败');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateChange = () => {
    if (problem) {
      router.push(
        `/changes/new?fromProblem=${problem.id}&problemTitle=${encodeURIComponent(problem.title)}`
      );
    }
  };

  const handlePublishToKB = () => {
    if (problem) {
      router.push(
        `/knowledge-base/new?fromProblemId=${problem.id}&problemTitle=${encodeURIComponent(
          problem.title
        )}`
      );
    }
  };

  const handleAddComment = () => {
    if (newComment.trim() && problem) {
      // TODO: 实现添加评论的API调用
      message.info('评论功能待实现');
      setNewComment('');
    }
  };

  if (loading) {
    return (
      <div className='p-10 bg-gray-50 min-h-full flex items-center justify-center'>
        <Spin size='large' />
      </div>
    );
  }

  if (!problem) {
    return (
      <div className='p-10 bg-gray-50 min-h-full'>
        <div className='text-center'>
          <AlertTriangle className='w-16 h-16 text-red-500 mx-auto mb-4' />
          <h2 className='text-2xl font-bold text-gray-800 mb-2'>问题不存在</h2>
          <p className='text-gray-600 mb-4'>该问题可能已被删除或ID无效</p>
          <Button type='primary' onClick={() => router.push('/problems')}>
            返回问题列表
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className='p-10 bg-gray-50 min-h-full'>
      <header className='mb-8'>
        <button
          onClick={() => router.back()}
          className='flex items-center text-blue-600 hover:underline mb-4'
        >
          <ArrowLeft className='w-5 h-5 mr-2' />
          返回问题列表
        </button>
        <div className='flex justify-between items-start'>
          <div>
            <h2 className='text-4xl font-bold text-gray-800'>问题：{problem.title}</h2>
            <p className='text-gray-500 mt-1'>问题ID: PRB-{String(problem.id).padStart(5, '0')}</p>
          </div>
          <div className='flex space-x-4'>
            <button
              onClick={handleCreateChange}
              className='flex items-center bg-purple-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-purple-700 transition-colors'
            >
              <GitMerge className='w-5 h-5 mr-2' />
              创建变更
            </button>
            <button
              onClick={handlePublishToKB}
              className='flex items-center bg-green-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-green-700 transition-colors'
            >
              <BookOpen className='w-5 h-5 mr-2' />
              发布到知识库
            </button>
          </div>
        </div>
      </header>

      <div className='grid grid-cols-1 lg:grid-cols-3 gap-8'>
        {/* 左侧：问题详情和处理日志 */}
        <div className='lg:col-span-2 bg-white p-8 rounded-lg shadow-md'>
          <h3 className='text-xl font-semibold text-gray-700 mb-4'>问题描述</h3>
          <p className='text-gray-600 mb-8'>{problem.description}</p>

          <h3 className='text-xl font-semibold text-gray-700 mb-4'>根本原因分析 (RCA)</h3>
          <p className='text-gray-600 mb-8'>{problem.root_cause}</p>

          <h3 className='text-xl font-semibold text-gray-700 mb-4'>影响范围</h3>
          <p className='text-gray-600 mb-8'>{problem.impact}</p>

          {/* 评论/备注区域 */}
          <div className='mt-8 pt-8 border-t border-gray-200'>
            <h3 className='text-xl font-semibold text-gray-700 mb-4 flex items-center'>
              <MessageSquare className='w-5 h-5 mr-2' /> 内部评论/备注
            </h3>
            <div className='space-y-4 mb-4'>
              <p className='text-sm text-gray-500'>暂无评论。</p>
            </div>
            <TextArea
              rows={3}
              placeholder='添加内部评论或备注...'
              value={newComment}
              onChange={e => setNewComment(e.target.value)}
            />
            <Button type='primary' onClick={handleAddComment} className='mt-2'>
              添加评论
            </Button>
          </div>
        </div>

        {/* 右侧：元数据和联动功能 */}
        <div className='space-y-8'>
          <Card title='问题信息' className='shadow-md'>
            <div className='space-y-3 text-sm'>
              <div className='flex justify-between'>
                <span>状态:</span>
                <Tag color={problemService.getStatusColor(problem.status)}>
                  {problemService.getStatusLabel(problem.status)}
                </Tag>
              </div>
              <div className='flex justify-between'>
                <span>优先级:</span>
                <Tag color={problemService.getPriorityColor(problem.priority)}>
                  {problemService.getPriorityLabel(problem.priority)}
                </Tag>
              </div>
              <div className='flex justify-between'>
                <span>分类:</span>
                <span className='font-semibold'>{problem.category}</span>
              </div>
              <div className='flex justify-between'>
                <span>负责人:</span>
                <span className='font-semibold'>{problem.assignee?.name || '-'}</span>
              </div>
              <div className='flex justify-between'>
                <span>创建时间:</span>
                <span className='font-semibold'>
                  {new Date(problem.created_at).toLocaleString('zh-CN')}
                </span>
              </div>
              <div className='flex justify-between'>
                <span>更新时间:</span>
                <span className='font-semibold'>
                  {new Date(problem.updated_at).toLocaleString('zh-CN')}
                </span>
              </div>
            </div>
          </Card>

          <Card title='关联事件' className='shadow-md'>
            <div className='space-y-3'>
              <p className='text-sm text-gray-500'>暂无关联事件。</p>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default ProblemDetailPage;
