'use client';

import React, { useState, useCallback } from 'react';
import { Card, Button, Input, Select, Space, Form, message, Divider } from 'antd';
import { PlusOutlined, DeleteOutlined, SaveOutlined, PlayCircleOutlined } from '@ant-design/icons';

const { Option } = Select;
const { TextArea } = Input;

interface WorkflowStep {
  id: string;
  name: string;
  type: 'start' | 'end' | 'task' | 'approval' | 'condition' | 'action';
  description?: string;
  assignee?: string;
  timeout?: number;
  conditions?: string[];
  actions?: string[];
  nextSteps?: string[];
}

interface WorkflowDefinition {
  id?: string;
  name: string;
  description?: string;
  version: string;
  tenantId: number;
  steps: WorkflowStep[];
  variables: Record<string, any>;
  metadata?: Record<string, any>;
}

const WorkflowDesigner: React.FC = () => {
  const [workflow, setWorkflow] = useState<WorkflowDefinition>({
    name: '',
    description: '',
    version: '1.0.0',
    tenantId: 1,
    steps: [],
    variables: {},
    metadata: {}
  });

  const [selectedStep, setSelectedStep] = useState<WorkflowStep | null>(null);

  const addStep = useCallback(() => {
    const newStep: WorkflowStep = {
      id: `step_${Date.now()}`,
      name: `新步骤 ${workflow.steps.length + 1}`,
      type: 'task',
      description: '',
      assignee: '',
      timeout: 24,
      conditions: [],
      actions: [],
      nextSteps: []
    };

    setWorkflow(prev => ({
      ...prev,
      steps: [...prev.steps, newStep]
    }));
  }, [workflow.steps.length]);

  const updateStep = useCallback((stepId: string, updates: Partial<WorkflowStep>) => {
    setWorkflow(prev => ({
      ...prev,
      steps: prev.steps.map(step => 
        step.id === stepId ? { ...step, ...updates } : step
      )
    }));
  }, []);

  const deleteStep = useCallback((stepId: string) => {
    setWorkflow(prev => ({
      ...prev,
      steps: prev.steps.filter(step => step.id !== stepId)
    }));
    if (selectedStep?.id === stepId) {
      setSelectedStep(null);
    }
  }, [selectedStep]);

  const saveWorkflow = useCallback(async () => {
    try {
      // TODO: 调用API保存工作流定义
      message.success('工作流保存成功');
    } catch (error) {
      message.error('保存失败: ' + error);
    }
  }, [workflow]);

  const startWorkflow = useCallback(async () => {
    try {
      // TODO: 调用API启动工作流
      message.success('工作流启动成功');
    } catch (error) {
      message.error('启动失败: ' + error);
    }
  }, [workflow]);

  const stepTypes = [
    { value: 'start', label: '开始', color: '#52c41a' },
    { value: 'end', label: '结束', color: '#ff4d4f' },
    { value: 'task', label: '任务', color: '#1890ff' },
    { value: 'approval', label: '审批', color: '#fa8c16' },
    { value: 'condition', label: '条件', color: '#722ed1' },
    { value: 'action', label: '动作', color: '#13c2c2' }
  ];

  return (
    <div className="p-6">
      <Card title="工作流设计器" className="mb-4">
        <Form layout="vertical">
          <Form.Item label="工作流名称" required>
            <Input
              value={workflow.name}
              onChange={(e) => setWorkflow(prev => ({ ...prev, name: e.target.value }))}
              placeholder="请输入工作流名称"
            />
          </Form.Item>
          
          <Form.Item label="描述">
            <TextArea
              value={workflow.description}
              onChange={(e) => setWorkflow(prev => ({ ...prev, description: e.target.value }))}
              placeholder="请输入工作流描述"
              rows={3}
            />
          </Form.Item>

          <Form.Item label="版本">
            <Input
              value={workflow.version}
              onChange={(e) => setWorkflow(prev => ({ ...prev, version: e.target.value }))}
              placeholder="版本号"
            />
          </Form.Item>
        </Form>

        <Space className="mb-4">
          <Button type="primary" icon={<SaveOutlined />} onClick={saveWorkflow}>
            保存工作流
          </Button>
          <Button icon={<PlayCircleOutlined />} onClick={startWorkflow}>
            启动工作流
          </Button>
        </Space>
      </Card>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* 步骤列表 */}
        <Card title="工作流步骤" className="lg:col-span-1">
          <Button 
            type="dashed" 
            icon={<PlusOutlined />} 
            onClick={addStep}
            className="w-full mb-4"
          >
            添加步骤
          </Button>

          <div className="space-y-2">
            {workflow.steps.map((step) => (
              <div
                key={step.id}
                className={`p-3 border rounded cursor-pointer hover:bg-gray-50 ${
                  selectedStep?.id === step.id ? 'border-blue-500 bg-blue-50' : 'border-gray-200'
                }`}
                onClick={() => setSelectedStep(step)}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center space-x-2">
                    <div
                      className="w-3 h-3 rounded-full"
                      style={{ backgroundColor: stepTypes.find(t => t.value === step.type)?.color }}
                    />
                    <span className="font-medium">{step.name}</span>
                  </div>
                  <Button
                    type="text"
                    icon={<DeleteOutlined />}
                    size="small"
                    onClick={(e) => {
                      e.stopPropagation();
                      deleteStep(step.id);
                    }}
                  />
                </div>
                <div className="text-sm text-gray-500 mt-1">
                  {stepTypes.find(t => t.value === step.type)?.label}
                </div>
              </div>
            ))}
          </div>
        </Card>

        {/* 步骤配置 */}
        <Card title="步骤配置" className="lg:col-span-2">
          {selectedStep ? (
            <Form layout="vertical">
              <Form.Item label="步骤名称">
                <Input
                  value={selectedStep.name}
                  onChange={(e) => updateStep(selectedStep.id, { name: e.target.value })}
                />
              </Form.Item>

              <Form.Item label="步骤类型">
                <Select
                  value={selectedStep.type}
                  onChange={(value) => updateStep(selectedStep.id, { type: value })}
                >
                  {stepTypes.map(type => (
                    <Option key={type.value} value={type.value}>
                      <div className="flex items-center space-x-2">
                        <div
                          className="w-3 h-3 rounded-full"
                          style={{ backgroundColor: type.color }}
                        />
                        <span>{type.label}</span>
                      </div>
                    </Option>
                  ))}
                </Select>
              </Form.Item>

              <Form.Item label="描述">
                <TextArea
                  value={selectedStep.description}
                  onChange={(e) => updateStep(selectedStep.id, { description: e.target.value })}
                  rows={3}
                />
              </Form.Item>

              {['task', 'approval'].includes(selectedStep.type) && (
                <>
                  <Form.Item label="负责人">
                    <Input
                      value={selectedStep.assignee}
                      onChange={(e) => updateStep(selectedStep.id, { assignee: e.target.value })}
                      placeholder="请输入负责人"
                    />
                  </Form.Item>

                  <Form.Item label="超时时间（小时）">
                    <Input
                      type="number"
                      value={selectedStep.timeout}
                      onChange={(e) => updateStep(selectedStep.id, { timeout: Number(e.target.value) })}
                      placeholder="24"
                    />
                  </Form.Item>
                </>
              )}

              {selectedStep.type === 'condition' && (
                <Form.Item label="条件表达式">
                  <TextArea
                    value={selectedStep.conditions?.join('\n')}
                    onChange={(e) => updateStep(selectedStep.id, { 
                      conditions: e.target.value.split('\n').filter(c => c.trim()) 
                    })}
                    rows={3}
                    placeholder="每行一个条件表达式"
                  />
                </Form.Item>
              )}

              {selectedStep.type === 'action' && (
                <Form.Item label="动作配置">
                  <TextArea
                    value={selectedStep.actions?.join('\n')}
                    onChange={(e) => updateStep(selectedStep.id, { 
                      actions: e.target.value.split('\n').filter(a => a.trim()) 
                    })}
                    rows={3}
                    placeholder="每行一个动作配置"
                  />
                </Form.Item>
              )}
            </Form>
          ) : (
            <div className="text-center text-gray-500 py-8">
              请选择一个步骤进行配置
            </div>
          )}
        </Card>
      </div>

      {/* 工作流变量配置 */}
      <Card title="工作流变量" className="mt-4">
        <div className="space-y-4">
          {Object.entries(workflow.variables).map(([key, value]) => (
            <div key={key} className="flex items-center space-x-2">
              <Input
                value={key}
                onChange={(e) => {
                  const newVariables = { ...workflow.variables };
                  delete newVariables[key];
                  newVariables[e.target.value] = value;
                  setWorkflow(prev => ({ ...prev, variables: newVariables }));
                }}
                placeholder="变量名"
                className="w-32"
              />
              <span>:</span>
              <Input
                value={value}
                onChange={(e) => {
                  setWorkflow(prev => ({
                    ...prev,
                    variables: { ...prev.variables, [key]: e.target.value }
                  }));
                }}
                placeholder="变量值"
              />
              <Button
                type="text"
                icon={<DeleteOutlined />}
                onClick={() => {
                  const newVariables = { ...workflow.variables };
                  delete newVariables[key];
                  setWorkflow(prev => ({ ...prev, variables: newVariables }));
                }}
              />
            </div>
          ))}
          
          <Button
            type="dashed"
            icon={<PlusOutlined />}
            onClick={() => {
              setWorkflow(prev => ({
                ...prev,
                variables: { ...prev.variables, [`var_${Date.now()}`]: '' }
              }));
            }}
          >
            添加变量
          </Button>
        </div>
      </Card>
    </div>
  );
};

export default WorkflowDesigner;
