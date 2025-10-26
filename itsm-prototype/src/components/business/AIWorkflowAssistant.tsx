'use client';

import React, { useState, useEffect } from 'react';
import {
  Card,
  Button,
  Input,
  Select,
  message,
  Space,
  Typography,
  Tag,
  Progress,
  Alert,
  Tooltip,
  Divider,
} from 'antd';
import {
  Bot,
  Zap,
  Lightbulb,
  TrendingUp,
  Clock,
  CheckCircle,
  AlertTriangle,
  Brain,
  Workflow,
  BookOpen,
} from 'lucide-react';
import { aiTriage, aiSearchKB, aiSimilarIncidents } from '@/lib/api/ai-api';

const { TextArea } = Input;
const { Title, Text } = Typography;
const { Option } = Select;

interface AIWorkflowAssistantProps {
  onSuggestion?: (suggestion: any) => void;
  className?: string;
}

interface WorkflowSuggestion {
  category: string;
  priority: string;
  assignee: string;
  estimatedTime: string;
  workflow: string[];
  confidence: number;
  reasoning: string;
}

export const AIWorkflowAssistant: React.FC<AIWorkflowAssistantProps> = ({
  onSuggestion,
  className = '',
}) => {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(false);
  const [suggestion, setSuggestion] = useState<WorkflowSuggestion | null>(null);
  const [knowledgeResults, setKnowledgeResults] = useState<any[]>([]);
  const [similarIncidents, setSimilarIncidents] = useState<any[]>([]);

  const handleAnalyze = async () => {
    if (!title.trim()) {
      message.warning('请输入工单标题');
      return;
    }

    setLoading(true);
    try {
      // 并行执行多个AI分析
      const [triageResult, kbResults, incidentResults] = await Promise.all([
        aiTriage(title, description),
        aiSearchKB(title, 3),
        aiSimilarIncidents(title, 3),
      ]);

      // 构建工作流建议
      const workflowSuggestion: WorkflowSuggestion = {
        category: triageResult.category,
        priority: triageResult.priority,
        assignee: `用户ID: ${triageResult.assignee_id}`,
        estimatedTime: getEstimatedTime(triageResult.priority),
        workflow: generateWorkflow(triageResult.category, triageResult.priority),
        confidence: triageResult.confidence,
        reasoning: triageResult.explanation,
      };

      setSuggestion(workflowSuggestion);
      setKnowledgeResults(kbResults.answers || []);
      setSimilarIncidents(incidentResults.incidents || []);

      if (onSuggestion) {
        onSuggestion(workflowSuggestion);
      }

      message.success('AI分析完成');
    } catch (error) {
      console.error('AI分析失败:', error);
      message.error('AI分析失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  const getEstimatedTime = (priority: string): string => {
    switch (priority) {
      case 'urgent':
        return '2-4小时';
      case 'high':
        return '4-8小时';
      case 'medium':
        return '8-24小时';
      case 'low':
        return '24-48小时';
      default:
        return '8-24小时';
    }
  };

  const generateWorkflow = (category: string, priority: string): string[] => {
    const baseWorkflow = [
      '工单创建',
      '初步分析',
      '解决方案制定',
      '实施解决',
      '验证确认',
      '工单关闭',
    ];

    if (priority === 'urgent') {
      return ['紧急响应', '立即分析', '快速解决', '验证确认', '事后总结'];
    }

    if (category === 'database') {
      return ['数据库问题确认', '备份检查', '问题分析', '修复实施', '数据验证', '监控确认'];
    }

    if (category === 'network') {
      return ['网络连通性测试', '设备状态检查', '配置分析', '修复实施', '连通性验证'];
    }

    return baseWorkflow;
  };

  const getPriorityColor = (priority: string): string => {
    switch (priority) {
      case 'urgent':
        return 'red';
      case 'high':
        return 'orange';
      case 'medium':
        return 'blue';
      case 'low':
        return 'green';
      default:
        return 'default';
    }
  };

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case 'database':
        return '🗄️';
      case 'network':
        return '🌐';
      case 'software':
        return '💻';
      case 'hardware':
        return '🔧';
      default:
        return '📋';
    }
  };

  return (
    <div className={`space-y-6 ${className}`}>
      <Card
        title={
          <Space>
            <Bot className='text-blue-600' />
            <span>AI工作流助手</span>
          </Space>
        }
        extra={
          <Tooltip title='AI将分析工单内容并提供智能建议'>
            <Lightbulb className='text-yellow-500' />
          </Tooltip>
        }
      >
        <div className='space-y-4'>
          <div>
            <Text strong>工单标题 *</Text>
            <Input
              placeholder='请输入工单标题...'
              value={title}
              onChange={e => setTitle(e.target.value)}
              className='mt-2'
            />
          </div>

          <div>
            <Text strong>工单描述</Text>
            <TextArea
              placeholder='请详细描述问题...'
              value={description}
              onChange={e => setDescription(e.target.value)}
              rows={3}
              className='mt-2'
            />
          </div>

          <Button
            type='primary'
            icon={<Brain />}
            loading={loading}
            onClick={handleAnalyze}
            disabled={!title.trim()}
            className='w-full'
          >
            {loading ? 'AI分析中...' : '开始AI分析'}
          </Button>
        </div>
      </Card>

      {suggestion && (
        <Card
          title={
            <Space>
              <Workflow className='text-green-600' />
              <span>AI工作流建议</span>
              <Tag color={getPriorityColor(suggestion.priority)}>
                {suggestion.priority.toUpperCase()}
              </Tag>
            </Space>
          }
        >
          <div className='space-y-4'>
            <div className='grid grid-cols-2 gap-4'>
              <div>
                <Text type='secondary'>分类</Text>
                <div className='flex items-center mt-1'>
                  <span className='text-lg mr-2'>{getCategoryIcon(suggestion.category)}</span>
                  <Text strong>{suggestion.category}</Text>
                </div>
              </div>
              <div>
                <Text type='secondary'>预计处理时间</Text>
                <div className='flex items-center mt-1'>
                  <Clock className='mr-2 text-blue-500' />
                  <Text strong>{suggestion.estimatedTime}</Text>
                </div>
              </div>
            </div>

            <div>
              <Text type='secondary'>置信度</Text>
              <div className='flex items-center mt-1'>
                <Progress
                  percent={Math.round(suggestion.confidence * 100)}
                  size='small'
                  className='flex-1 mr-2'
                />
                <Text strong>{Math.round(suggestion.confidence * 100)}%</Text>
              </div>
            </div>

            <div>
              <Text type='secondary'>推荐工作流</Text>
              <div className='mt-2 space-y-2'>
                {suggestion.workflow.map((step, index) => (
                  <div key={index} className='flex items-center'>
                    <div className='w-6 h-6 bg-blue-100 text-blue-600 rounded-full flex items-center justify-center text-xs font-medium mr-3'>
                      {index + 1}
                    </div>
                    <Text>{step}</Text>
                  </div>
                ))}
              </div>
            </div>

            <div>
              <Text type='secondary'>AI推理</Text>
              <div className='mt-1 p-3 bg-gray-50 rounded'>
                <Text>{suggestion.reasoning}</Text>
              </div>
            </div>
          </div>
        </Card>
      )}

      {knowledgeResults.length > 0 && (
        <Card
          title={
            <Space>
              <BookOpen className='text-purple-600' />
              <span>相关知识库</span>
            </Space>
          }
        >
          <div className='space-y-3'>
            {knowledgeResults.map((item, index) => (
              <div key={index} className='p-3 border rounded hover:bg-gray-50'>
                <div className='flex items-start justify-between'>
                  <div className='flex-1'>
                    <Text strong className='block mb-1'>
                      {item.title || `知识库条目 ${index + 1}`}
                    </Text>
                    <Text type='secondary' className='text-sm'>
                      {item.snippet}
                    </Text>
                  </div>
                  <Tag color='purple'>{Math.round((item.score || 0) * 100)}%</Tag>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}

      {similarIncidents.length > 0 && (
        <Card
          title={
            <Space>
              <TrendingUp className='text-orange-600' />
              <span>相似事件</span>
            </Space>
          }
        >
          <div className='space-y-3'>
            {similarIncidents.map((incident, index) => (
              <div key={index} className='p-3 border rounded hover:bg-gray-50'>
                <div className='flex items-start justify-between'>
                  <div className='flex-1'>
                    <Text strong className='block mb-1'>
                      {incident.title || `事件 ${incident.id}`}
                    </Text>
                    <Text type='secondary' className='text-sm'>
                      {incident.snippet}
                    </Text>
                  </div>
                  <Tag color='orange'>{Math.round((incident.score || 0) * 100)}%</Tag>
                </div>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
};
