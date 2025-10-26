'use client';

import React, { useState, useCallback } from 'react';
import {
  Card,
  Button,
  Input,
  Select,
  Tag,
  Spin,
  Alert,
  Collapse,
  Progress,
  Typography,
  Space,
  Divider,
  List,
  Avatar,
} from 'antd';
import { Bot, Lightbulb, Users, Search, CheckCircle, Clock, Target } from 'lucide-react';
import {
  AIService,
  type TicketAnalysisRequest,
  type AssigneeRecommendation,
  type SolutionSuggestion,
} from '../lib/ai-service';

const { Text } = Typography;
const { TextArea } = Input;
const { Option } = Select;
const { Panel } = Collapse;

interface AIAssistantProps {
  initialData?: {
    title?: string;
    description?: string;
    category?: string;
  };
  onRecommendationSelect?: (recommendation: AssigneeRecommendation | SolutionSuggestion) => void;
  className?: string;
}

interface AIAnalysisState {
  classification: {
    category: string;
    priority: string;
    urgency: string;
    confidence: number;
    reasoning: string;
    suggestions?: string[];
  } | null;
  assigneeRecommendations: AssigneeRecommendation[];
  solutionSuggestions: SolutionSuggestion[];
  loading: boolean;
  error: string | null;
}

export const AIAssistant: React.FC<AIAssistantProps> = ({
  initialData,
  onRecommendationSelect,
  className = '',
}) => {
  const [analysisState, setAnalysisState] = useState<AIAnalysisState>({
    classification: null,
    assigneeRecommendations: [],
    solutionSuggestions: [],
    loading: false,
    error: null,
  });

  const [inputData, setInputData] = useState({
    title: initialData?.title || '',
    description: initialData?.description || '',
    category: initialData?.category || '',
  });

  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<any>(null);
  const [searchLoading, setSearchLoading] = useState(false);

  // 执行AI分析
  const performAnalysis = useCallback(async () => {
    if (!inputData.title || !inputData.description) {
      setAnalysisState(prev => ({
        ...prev,
        error: '请提供标题和描述信息',
      }));
      return;
    }

    setAnalysisState(prev => ({
      ...prev,
      loading: true,
      error: null,
    }));

    try {
      const request: TicketAnalysisRequest = {
        title: inputData.title,
        description: inputData.description,
        userContext: {
          department: 'IT部门',
          role: '用户',
          location: '总部',
        },
      };

      // 并行执行多个AI分析
      const [classification, assigneeRecommendations, solutionSuggestions] = await Promise.all([
        AIService.classifyTicket(request),
        AIService.recommendAssignee({
          category: inputData.category || 'general',
          priority: 'medium',
          description: inputData.description,
          requiredSkills: [],
        }),
        AIService.suggestSolutions({
          query: `${inputData.title} ${inputData.description}`,
          category: inputData.category,
          limit: 5,
        }),
      ]);

      setAnalysisState({
        classification,
        assigneeRecommendations,
        solutionSuggestions,
        loading: false,
        error: null,
      });
    } catch (error) {
      console.error('AI分析失败:', error);
      setAnalysisState(prev => ({
        ...prev,
        loading: false,
        error: error instanceof Error ? error.message : 'AI分析失败，请稍后重试',
      }));
    }
  }, [inputData]);

  // 智能搜索
  const performIntelligentSearch = useCallback(async () => {
    if (!searchQuery.trim()) return;

    setSearchLoading(true);
    try {
      const results = await AIService.intelligentSearch(searchQuery, {
        type: 'all',
      });
      setSearchResults(results);
    } catch (error) {
      console.error('智能搜索失败:', error);
    } finally {
      setSearchLoading(false);
    }
  }, [searchQuery]);

  // 渲染分类结果
  const renderClassification = () => {
    if (!analysisState.classification) return null;

    const { classification } = analysisState;
    return (
      <Card size='small' className='mb-4'>
        <div className='flex items-center justify-between mb-3'>
          <div className='flex items-center space-x-2'>
            <Target size={16} className='text-blue-600' />
            <Text className='font-semibold'>智能分类</Text>
          </div>
          <Progress
            type='circle'
            size={40}
            percent={Math.round(classification.confidence * 100)}
            format={() => `${Math.round(classification.confidence * 100)}%`}
          />
        </div>

        <Space wrap>
          <Tag color='blue'>类别: {classification.category}</Tag>
          <Tag
            color={
              classification.priority === 'critical'
                ? 'red'
                : classification.priority === 'high'
                ? 'orange'
                : classification.priority === 'medium'
                ? 'yellow'
                : 'green'
            }
          >
            优先级: {classification.priority}
          </Tag>
          <Tag color='purple'>紧急程度: {classification.urgency}</Tag>
        </Space>

        <Divider className='my-3' />

        <div>
          <Text className='text-sm text-gray-600 block mb-2'>分析依据:</Text>
          <Text className='text-sm'>{classification.reasoning}</Text>
        </div>

        {classification.suggestions && classification.suggestions.length > 0 && (
          <div className='mt-3'>
            <Text className='text-sm text-gray-600 block mb-2'>建议:</Text>
            <ul className='text-sm space-y-1'>
              {classification.suggestions.map((suggestion: string, index: number) => (
                <li key={index} className='flex items-start space-x-2'>
                  <Lightbulb size={12} className='text-yellow-500 mt-1 flex-shrink-0' />
                  <span>{suggestion}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </Card>
    );
  };

  // 渲染处理人推荐
  const renderAssigneeRecommendations = () => {
    if (analysisState.assigneeRecommendations.length === 0) return null;

    return (
      <Card size='small' className='mb-4'>
        <div className='flex items-center space-x-2 mb-3'>
          <Users size={16} className='text-green-600' />
          <Text className='font-semibold'>推荐处理人</Text>
        </div>

        <List
          size='small'
          dataSource={analysisState.assigneeRecommendations}
          renderItem={item => (
            <List.Item
              actions={[
                <Button
                  key='select'
                  type='link'
                  size='small'
                  onClick={() => onRecommendationSelect?.(item)}
                >
                  选择
                </Button>,
              ]}
            >
              <List.Item.Meta
                avatar={<Avatar className='bg-blue-500'>{item.userName.charAt(0)}</Avatar>}
                title={
                  <div className='flex items-center space-x-2'>
                    <span>{item.userName}</span>
                    <Tag
                      color={
                        item.availability === 'available'
                          ? 'green'
                          : item.availability === 'busy'
                          ? 'orange'
                          : 'red'
                      }
                    >
                      {item.availability === 'available'
                        ? '空闲'
                        : item.availability === 'busy'
                        ? '忙碌'
                        : '不可用'}
                    </Tag>
                    <Progress
                      size='small'
                      percent={Math.round(item.confidence * 100)}
                      showInfo={false}
                      strokeWidth={4}
                    />
                  </div>
                }
                description={
                  <div className='text-xs space-y-1'>
                    <div>
                      部门: {item.department} | 工作负载: {item.workload}%
                    </div>
                    <div>专长: {item.expertise.join(', ')}</div>
                    <div className='text-gray-500'>{item.reasoning}</div>
                  </div>
                }
              />
            </List.Item>
          )}
        />
      </Card>
    );
  };

  // 渲染解决方案建议
  const renderSolutionSuggestions = () => {
    if (analysisState.solutionSuggestions.length === 0) return null;

    return (
      <Card size='small' className='mb-4'>
        <div className='flex items-center space-x-2 mb-3'>
          <CheckCircle size={16} className='text-purple-600' />
          <Text className='font-semibold'>解决方案建议</Text>
        </div>

        <Collapse size='small'>
          {analysisState.solutionSuggestions.map((solution, index) => (
            <Panel
              key={solution.solutionId}
              header={
                <div className='flex items-center justify-between w-full pr-4'>
                  <span className='font-medium'>{solution.title}</span>
                  <div className='flex items-center space-x-2'>
                    <Tag color='blue'>成功率: {Math.round(solution.successRate * 100)}%</Tag>
                    <Tag color='green'>
                      <Clock size={10} className='mr-1' />
                      {solution.estimatedTime}分钟
                    </Tag>
                  </div>
                </div>
              }
            >
              <div className='space-y-3'>
                <Text className='text-sm text-gray-600'>{solution.description}</Text>

                <div>
                  <Text className='text-sm font-medium block mb-2'>处理步骤:</Text>
                  <ol className='text-sm space-y-1 pl-4'>
                    {solution.steps.map((step, stepIndex) => (
                      <li key={stepIndex}>{step}</li>
                    ))}
                  </ol>
                </div>

                {solution.relatedKnowledge.length > 0 && (
                  <div>
                    <Text className='text-sm font-medium block mb-2'>相关知识:</Text>
                    <Space wrap>
                      {solution.relatedKnowledge.map((knowledge, knowledgeIndex) => (
                        <Tag key={knowledgeIndex} color='cyan' className='cursor-pointer'>
                          {knowledge}
                        </Tag>
                      ))}
                    </Space>
                  </div>
                )}

                <div className='flex justify-between items-center pt-2 border-t'>
                  <Text className='text-xs text-gray-500'>
                    置信度: {Math.round(solution.confidence * 100)}%
                  </Text>
                  <Button
                    type='primary'
                    size='small'
                    onClick={() => onRecommendationSelect?.(solution)}
                  >
                    采用此方案
                  </Button>
                </div>
              </div>
            </Panel>
          ))}
        </Collapse>
      </Card>
    );
  };

  return (
    <div className={`ai-assistant ${className}`}>
      <Card
        title={
          <div className='flex items-center space-x-2'>
            <Bot size={20} className='text-blue-600' />
            <span>AI 智能助手</span>
          </div>
        }
        size='small'
      >
        {/* 输入区域 */}
        <div className='space-y-3 mb-4'>
          <Input
            placeholder='请输入问题标题'
            value={inputData.title}
            onChange={e => setInputData(prev => ({ ...prev, title: e.target.value }))}
          />

          <TextArea
            placeholder='请详细描述问题...'
            rows={3}
            value={inputData.description}
            onChange={e => setInputData(prev => ({ ...prev, description: e.target.value }))}
          />

          <div className='flex space-x-2'>
            <Select
              placeholder='选择类别'
              value={inputData.category}
              onChange={value => setInputData(prev => ({ ...prev, category: value }))}
              className='flex-1'
            >
              <Option value='network'>网络问题</Option>
              <Option value='software'>软件问题</Option>
              <Option value='hardware'>硬件问题</Option>
              <Option value='access'>权限问题</Option>
              <Option value='general'>其他</Option>
            </Select>

            <Button
              type='primary'
              onClick={performAnalysis}
              loading={analysisState.loading}
              disabled={!inputData.title || !inputData.description}
            >
              AI 分析
            </Button>
          </div>
        </div>

        {/* 错误提示 */}
        {analysisState.error && (
          <Alert
            message={analysisState.error}
            type='error'
            closable
            className='mb-4'
            onClose={() => setAnalysisState(prev => ({ ...prev, error: null }))}
          />
        )}

        {/* 加载状态 */}
        {analysisState.loading && (
          <div className='text-center py-8'>
            <Spin size='large' />
            <Text className='block mt-2 text-gray-500'>AI 正在分析中...</Text>
          </div>
        )}

        {/* 分析结果 */}
        {!analysisState.loading && (
          <div>
            {renderClassification()}
            {renderAssigneeRecommendations()}
            {renderSolutionSuggestions()}
          </div>
        )}

        {/* 智能搜索 */}
        <Divider />
        <div className='space-y-3'>
          <div className='flex items-center space-x-2'>
            <Search size={16} className='text-gray-600' />
            <Text className='font-semibold'>智能搜索</Text>
          </div>

          <div className='flex space-x-2'>
            <Input
              placeholder='输入自然语言查询...'
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              onPressEnter={performIntelligentSearch}
            />
            <Button
              onClick={performIntelligentSearch}
              loading={searchLoading}
              disabled={!searchQuery.trim()}
            >
              搜索
            </Button>
          </div>

          {searchResults && (
            <div className='mt-3'>
              {searchResults.suggestions.length > 0 && (
                <div className='mb-3'>
                  <Text className='text-sm text-gray-600 block mb-2'>搜索建议:</Text>
                  <Space wrap>
                    {searchResults.suggestions.map((suggestion: string, index: number) => (
                      <Tag
                        key={index}
                        className='cursor-pointer'
                        onClick={() => setSearchQuery(suggestion)}
                      >
                        {suggestion}
                      </Tag>
                    ))}
                  </Space>
                </div>
              )}

              <Text className='text-sm text-gray-600'>
                找到{' '}
                {searchResults.tickets.length +
                  searchResults.knowledge.length +
                  searchResults.incidents.length}{' '}
                条相关结果
              </Text>
            </div>
          )}
        </div>
      </Card>
    </div>
  );
};

export default AIAssistant;
