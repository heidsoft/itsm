'use client';

import React, { useState, useCallback } from 'react';
import { Card, Button, Progress, Tag, Collapse, Alert, Statistic, Row, Col } from 'antd';
import { Play, CheckCircle, XCircle, Clock, AlertTriangle, FileText, Code } from 'lucide-react';

const { Panel } = Collapse;

// 测试结果类型定义
interface TestResult {
  id: string;
  name: string;
  status: 'passed' | 'failed' | 'pending' | 'skipped';
  duration: number;
  error?: string;
  description?: string;
}

interface TestSuite {
  id: string;
  name: string;
  description: string;
  tests: TestResult[];
  status: 'passed' | 'failed' | 'running' | 'pending';
  duration: number;
}

interface TestRunnerState {
  isRunning: boolean;
  progress: number;
  suites: TestSuite[];
  summary: {
    total: number;
    passed: number;
    failed: number;
    skipped: number;
    duration: number;
  };
}

// 模拟测试数据
const mockTestSuites: TestSuite[] = [
  {
    id: 'auth-tests',
    name: '认证模块测试',
    description: '测试用户认证、权限验证等功能',
    status: 'pending',
    duration: 0,
    tests: [
      {
        id: 'auth-1',
        name: '用户登录测试',
        status: 'pending',
        duration: 0,
        description: '测试用户登录功能是否正常工作',
      },
      {
        id: 'auth-2',
        name: '权限验证测试',
        status: 'pending',
        duration: 0,
        description: '测试用户权限验证逻辑',
      },
      {
        id: 'auth-3',
        name: 'Token刷新测试',
        status: 'pending',
        duration: 0,
        description: '测试JWT Token自动刷新机制',
      },
    ],
  },
  {
    id: 'dashboard-tests',
    name: '仪表板组件测试',
    description: '测试仪表板页面的各个组件',
    status: 'pending',
    duration: 0,
    tests: [
      {
        id: 'dashboard-1',
        name: 'KPI指标卡片渲染测试',
        status: 'pending',
        duration: 0,
        description: '测试KPI指标卡片是否正确渲染',
      },
      {
        id: 'dashboard-2',
        name: '系统状态监控测试',
        status: 'pending',
        duration: 0,
        description: '测试系统状态监控组件',
      },
      {
        id: 'dashboard-3',
        name: '数据获取Hook测试',
        status: 'pending',
        duration: 0,
        description: '测试useDashboardData Hook的数据获取逻辑',
      },
    ],
  },
  {
    id: 'ticket-tests',
    name: '工单管理测试',
    description: '测试工单创建、更新、查询等功能',
    status: 'pending',
    duration: 0,
    tests: [
      {
        id: 'ticket-1',
        name: '工单创建测试',
        status: 'pending',
        duration: 0,
        description: '测试工单创建表单和API调用',
      },
      {
        id: 'ticket-2',
        name: '工单列表查询测试',
        status: 'pending',
        duration: 0,
        description: '测试工单列表的分页和筛选功能',
      },
      {
        id: 'ticket-3',
        name: '工单状态更新测试',
        status: 'pending',
        duration: 0,
        description: '测试工单状态变更逻辑',
      },
    ],
  },
  {
    id: 'api-tests',
    name: 'API接口测试',
    description: '测试前端API调用层',
    status: 'pending',
    duration: 0,
    tests: [
      {
        id: 'api-1',
        name: 'HTTP客户端测试',
        status: 'pending',
        duration: 0,
        description: '测试HTTP客户端的请求和响应处理',
      },
      {
        id: 'api-2',
        name: '错误处理测试',
        status: 'pending',
        duration: 0,
        description: '测试API错误处理和重试机制',
      },
      {
        id: 'api-3',
        name: '缓存机制测试',
        status: 'pending',
        duration: 0,
        description: '测试API响应缓存功能',
      },
    ],
  },
];

// 模拟测试执行
const simulateTestExecution = async (
  test: TestResult,
  onProgress: (progress: number) => void
): Promise<TestResult> => {
  const duration = Math.random() * 2000 + 500; // 0.5-2.5秒
  const startTime = Date.now();
  
  // 模拟测试执行过程
  const interval = setInterval(() => {
    const elapsed = Date.now() - startTime;
    const progress = Math.min(elapsed / duration, 1);
    onProgress(progress);
  }, 100);

  await new Promise(resolve => setTimeout(resolve, duration));
  clearInterval(interval);

  // 随机生成测试结果
  const success = Math.random() > 0.2; // 80%成功率
  
  return {
    ...test,
    status: success ? 'passed' : 'failed',
    duration: Math.round(duration),
    error: success ? undefined : '模拟测试失败: 断言不匹配',
  };
};

export const TestRunner: React.FC = () => {
  const [state, setState] = useState<TestRunnerState>({
    isRunning: false,
    progress: 0,
    suites: mockTestSuites,
    summary: {
      total: 0,
      passed: 0,
      failed: 0,
      skipped: 0,
      duration: 0,
    },
  });

  const runAllTests = useCallback(async () => {
    setState(prev => ({
      ...prev,
      isRunning: true,
      progress: 0,
      suites: prev.suites.map(suite => ({
        ...suite,
        status: 'pending',
        duration: 0,
        tests: suite.tests.map(test => ({
          ...test,
          status: 'pending',
          duration: 0,
          error: undefined,
        })),
      })),
    }));

    const totalTests = mockTestSuites.reduce((sum, suite) => sum + suite.tests.length, 0);
    let completedTests = 0;
    const startTime = Date.now();

    const updatedSuites: TestSuite[] = [];

    for (const suite of mockTestSuites) {
      const suiteStartTime = Date.now();
      
      // 更新套件状态为运行中
      setState(prev => ({
        ...prev,
        suites: prev.suites.map(s => 
          s.id === suite.id ? { ...s, status: 'running' } : s
        ),
      }));

      const updatedTests: TestResult[] = [];
      
      for (const test of suite.tests) {
        const result = await simulateTestExecution(test, () => {
          // 更新进度
          const progress = (completedTests / totalTests) * 100;
          setState(prev => ({ ...prev, progress }));
        });
        
        updatedTests.push(result);
        completedTests++;
        
        // 更新单个测试结果
        setState(prev => ({
          ...prev,
          progress: (completedTests / totalTests) * 100,
          suites: prev.suites.map(s => 
            s.id === suite.id 
              ? {
                  ...s,
                  tests: s.tests.map(t => t.id === test.id ? result : t),
                }
              : s
          ),
        }));
      }

      const suiteDuration = Date.now() - suiteStartTime;
      const suiteStatus = updatedTests.every(t => t.status === 'passed') ? 'passed' : 'failed';
      
      const updatedSuite: TestSuite = {
        ...suite,
        status: suiteStatus,
        duration: suiteDuration,
        tests: updatedTests,
      };
      
      updatedSuites.push(updatedSuite);
      
      // 更新套件状态
      setState(prev => ({
        ...prev,
        suites: prev.suites.map(s => 
          s.id === suite.id ? updatedSuite : s
        ),
      }));
    }

    // 计算总结
    const totalDuration = Date.now() - startTime;
    const allTests = updatedSuites.flatMap(s => s.tests);
    const summary = {
      total: allTests.length,
      passed: allTests.filter(t => t.status === 'passed').length,
      failed: allTests.filter(t => t.status === 'failed').length,
      skipped: allTests.filter(t => t.status === 'skipped').length,
      duration: totalDuration,
    };

    setState(prev => ({
      ...prev,
      isRunning: false,
      progress: 100,
      summary,
    }));
  }, []);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'passed':
        return <CheckCircle size={16} className="text-green-500" />;
      case 'failed':
        return <XCircle size={16} className="text-red-500" />;
      case 'running':
        return <Clock size={16} className="text-blue-500" />;
      case 'skipped':
        return <AlertTriangle size={16} className="text-yellow-500" />;
      default:
        return <Clock size={16} className="text-gray-400" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'passed':
        return 'success';
      case 'failed':
        return 'error';
      case 'running':
        return 'processing';
      case 'skipped':
        return 'warning';
      default:
        return 'default';
    }
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
            <Code size={24} />
            测试运行器
          </h1>
          <p className="text-gray-600 mt-1">
            运行前端组件和API接口的自动化测试
          </p>
        </div>
        
        <Button
          type="primary"
          size="large"
          icon={<Play size={16} />}
          onClick={runAllTests}
          loading={state.isRunning}
          disabled={state.isRunning}
        >
          {state.isRunning ? '运行中...' : '运行所有测试'}
        </Button>
      </div>

      {/* 测试进度 */}
      {state.isRunning && (
        <Card>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium">测试进度</span>
              <span className="text-sm text-gray-500">
                {Math.round(state.progress)}%
              </span>
            </div>
            <Progress 
              percent={state.progress} 
              status={state.isRunning ? 'active' : 'success'}
              strokeColor={{
                '0%': '#108ee9',
                '100%': '#87d068',
              }}
            />
          </div>
        </Card>
      )}

      {/* 测试摘要 */}
      {state.summary.total > 0 && (
        <Card title="测试摘要" extra={<FileText size={16} />}>
          <Row gutter={16}>
            <Col span={6}>
              <Statistic
                title="总测试数"
                value={state.summary.total}
                valueStyle={{ color: '#1890ff' }}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="通过"
                value={state.summary.passed}
                valueStyle={{ color: '#52c41a' }}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="失败"
                value={state.summary.failed}
                valueStyle={{ color: '#f5222d' }}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="执行时间"
                value={state.summary.duration}
                suffix="ms"
                valueStyle={{ color: '#722ed1' }}
              />
            </Col>
          </Row>
          
          {state.summary.failed > 0 && (
            <Alert
              message="部分测试失败"
              description={`${state.summary.failed} 个测试未通过，请检查详细信息`}
              type="warning"
              showIcon
              className="mt-4"
            />
          )}
        </Card>
      )}

      {/* 测试套件详情 */}
      <Card title="测试套件">
        <Collapse>
          {state.suites.map(suite => (
            <Panel
              key={suite.id}
              header={
                <div className="flex items-center justify-between w-full pr-4">
                  <div className="flex items-center gap-3">
                    {getStatusIcon(suite.status)}
                    <span className="font-medium">{suite.name}</span>
                    <Tag color={getStatusColor(suite.status)}>
                      {suite.tests.filter(t => t.status === 'passed').length}/{suite.tests.length}
                    </Tag>
                  </div>
                  {suite.duration > 0 && (
                    <span className="text-sm text-gray-500">
                      {suite.duration}ms
                    </span>
                  )}
                </div>
              }
            >
              <div className="space-y-3">
                <p className="text-gray-600 text-sm mb-4">
                  {suite.description}
                </p>
                
                {suite.tests.map(test => (
                  <div
                    key={test.id}
                    className="flex items-start justify-between p-3 bg-gray-50 rounded-lg"
                  >
                    <div className="flex items-start gap-3 flex-1">
                      {getStatusIcon(test.status)}
                      <div className="flex-1">
                        <div className="font-medium text-sm">{test.name}</div>
                        {test.description && (
                          <div className="text-xs text-gray-500 mt-1">
                            {test.description}
                          </div>
                        )}
                        {test.error && (
                          <div className="text-xs text-red-600 mt-2 p-2 bg-red-50 rounded">
                            {test.error}
                          </div>
                        )}
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-2 text-xs text-gray-500">
                      {test.duration > 0 && (
                        <span>{test.duration}ms</span>
                      )}
                      <Tag color={getStatusColor(test.status)}>
                        {test.status}
                      </Tag>
                    </div>
                  </div>
                ))}
              </div>
            </Panel>
          ))}
        </Collapse>
      </Card>

      {/* 测试覆盖率信息 */}
      <Card title="测试覆盖率" className="mb-6">
        <div className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="text-center p-4 bg-blue-50 rounded-lg">
              <div className="text-2xl font-bold text-blue-600">85%</div>
              <div className="text-sm text-gray-600">语句覆盖率</div>
            </div>
            <div className="text-center p-4 bg-green-50 rounded-lg">
              <div className="text-2xl font-bold text-green-600">78%</div>
              <div className="text-sm text-gray-600">分支覆盖率</div>
            </div>
            <div className="text-center p-4 bg-purple-50 rounded-lg">
              <div className="text-2xl font-bold text-purple-600">92%</div>
              <div className="text-sm text-gray-600">函数覆盖率</div>
            </div>
          </div>
          
          <Alert
            message="测试覆盖率建议"
            description="建议保持语句覆盖率在80%以上，分支覆盖率在75%以上，以确保代码质量。"
            type="info"
            showIcon
          />
        </div>
      </Card>
    </div>
  );
};

export default TestRunner;