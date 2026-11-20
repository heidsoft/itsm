import React from 'react';
import { App } from 'antd';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import WorkflowAIModal from '../WorkflowAIModal';
import { BPMNAIApi } from '@/lib/api/bpmn-ai-api';

// AI 创作工作流 / AI 建议审查回归：
// 1) lint category 英文枚举不得直接暴露给用户，必须中文化（structure/events/...）
// 2) lint 返回 200+null 时不得抛 "Cannot read properties of null"，也不得伪装成「0 条建议」
// 3) 入口 tab 直达：initialTab=optimize/compliance 时默认选中对应 tab
// 4) preview/template 的 validateFields 必须被 try/catch 包裹（校验失败不产生未捕获 rejection）

jest.mock('@/lib/api/bpmn-ai-api', () => ({
  BPMNAIApi: {
    generateBPMN: jest.fn(),
    previewBPMN: jest.fn(),
    getTemplateSuggestions: jest.fn(),
    lintBPMN: jest.fn(),
  },
}));

jest.mock('../WorkflowCanvas', () => ({
  getBpmnDesignerApi: jest.fn(() => ({ selectElement: jest.fn() })),
}));

// t() 对 category 系列返回真实中文，其余返回 key 末段，便于精确断言。
jest.mock('@/lib/i18n/useI18n', () => ({
  useI18n: () => ({
    t: (key: string) => {
      const categoryMap: Record<string, string> = {
        'workflow.aiModal.categoryStructure': '结构完整性',
        'workflow.aiModal.categoryEvents': '事件节点',
        'workflow.aiModal.categoryTasks': '任务节点',
        'workflow.aiModal.categoryFlows': '流程连线',
        'workflow.aiModal.categoryConnectivity': '连通性',
        'workflow.aiModal.categoryGateways': '网关分支',
        'workflow.aiModal.categoryApproval': '审批配置',
        'workflow.aiModal.categoryOther': '其他检查',
        'workflow.aiModal.lintEmptyResult': '分析结果为空',
        'workflow.aiModal.getOptimizationSuggestions': '获取优化建议',
        'workflow.aiModal.startComplianceCheck': '开始合规检查',
      };
      return categoryMap[key] ?? key.split('.').pop() ?? key;
    },
  }),
}));

describe('WorkflowAIModal - AI 创作/建议审查回归', () => {
  let consoleErrorSpy: jest.SpyInstance;

  beforeEach(() => {
    jest.clearAllMocks();
    consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => consoleErrorSpy.mockRestore());

  const renderModal = (initialTab?: 'generate' | 'optimize' | 'compliance') =>
    render(
      <App>
        <WorkflowAIModal
          visible
          initialTab={initialTab}
          onClose={jest.fn()}
          currentXML="<definitions />"
          workflowName="事件处理流程"
        />
      </App>
    );

  it('lint category 显示为中文，不暴露 structure 等英文枚举值', async () => {
    (BPMNAIApi.lintBPMN as jest.Mock).mockResolvedValueOnce({
      hasErrors: true,
      errorCount: 1,
      warningCount: 0,
      issues: [{ severity: 'error', category: 'structure', message: '缺少开始事件', elementId: 'StartEvent_1' }],
    });

    renderModal('optimize');
    fireEvent.click(screen.getByRole('button', { name: '获取优化建议' }));

    expect(await screen.findByText('结构完整性')).toBeInTheDocument();
    expect(screen.queryByText('structure')).not.toBeInTheDocument();
  });

  it('合规检查 tab 的 rule 也使用中文化 category', async () => {
    (BPMNAIApi.lintBPMN as jest.Mock).mockResolvedValueOnce({
      hasErrors: false,
      errorCount: 0,
      warningCount: 1,
      issues: [{ severity: 'warning', category: 'approval', message: '审批人未配置' }],
    });

    renderModal('compliance');
    fireEvent.click(screen.getByRole('button', { name: '开始合规检查' }));

    expect(await screen.findByText('审批配置')).toBeInTheDocument();
    expect(screen.queryByText('approval')).not.toBeInTheDocument();
  });

  it('lint 返回 null 时不崩溃，并给出空结果提示而非伪装成 0 条建议', async () => {
    (BPMNAIApi.lintBPMN as jest.Mock).mockResolvedValueOnce(null);

    renderModal('optimize');
    fireEvent.click(screen.getByRole('button', { name: '获取优化建议' }));

    await waitFor(() => expect(screen.getByText('分析结果为空')).toBeInTheDocument());
    expect(consoleErrorSpy).not.toHaveBeenCalled();
  });

  it('initialTab=optimize 时默认选中优化建议 tab', () => {
    renderModal('optimize');
    expect(screen.getByRole('button', { name: '获取优化建议' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: '生成流程' })).not.toBeInTheDocument();
  });

  it('initialTab=compliance 时默认选中合规检查 tab', () => {
    renderModal('compliance');
    expect(screen.getByRole('button', { name: '开始合规检查' })).toBeInTheDocument();
  });
});
