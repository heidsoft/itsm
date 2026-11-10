import React from 'react';
import { App } from 'antd';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import WorkflowAIModal from '../WorkflowAIModal';
import { BPMNAIApi } from '@/lib/api/bpmn-ai-api';

jest.mock('@/lib/api/bpmn-ai-api', () => ({
  BPMNAIApi: {
    generateBPMN: jest.fn(),
    previewBPMN: jest.fn(),
    getTemplateSuggestions: jest.fn(),
  },
}));

jest.mock('../WorkflowCanvas', () => ({
  getBpmnDesignerApi: jest.fn(() => ({
    selectElement: jest.fn(),
  })),
}));

describe('WorkflowAIModal', () => {
  let consoleErrorSpy: jest.SpyInstance;

  beforeEach(() => {
    jest.clearAllMocks();
    consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
  });

  afterEach(() => {
    consoleErrorSpy.mockRestore();
  });

  const renderModal = (onApplyGeneratedProcess = jest.fn()) => {
    render(
      <App>
        <WorkflowAIModal
          visible
          onClose={jest.fn()}
          currentXML="<existing />"
          workflowName="事件处理流程"
          onApplyGeneratedProcess={onApplyGeneratedProcess}
        />
      </App>
    );
    return { onApplyGeneratedProcess };
  };

  it('calls the real BPMN AI generate endpoint and applies generated XML', async () => {
    const generatedXml = '<?xml version="1.0"?><bpmn:definitions />';
    (BPMNAIApi.generateBPMN as jest.Mock).mockResolvedValueOnce({
      bpmnXml: generatedXml,
      processId: 'incident_flow',
      processName: '事件处理流程',
      processDescription: 'AI generated workflow',
      version: '1.0.0',
      nodeCount: 4,
      complexity: 'medium',
      explanation: '包含事件分派与处理节点',
    });

    const { onApplyGeneratedProcess } = renderModal();

    fireEvent.change(screen.getByLabelText("Requirement Description"), {
      target: {
        value: '用户提交事件，服务台分派给工程师，工程师处理后关闭。',
      },
    });
    fireEvent.click(screen.getByRole('button', { name: /Generate Process/ }));

    await waitFor(() => expect(BPMNAIApi.generateBPMN).toHaveBeenCalledTimes(1));
    expect(BPMNAIApi.generateBPMN).toHaveBeenCalledWith({
      requirement: '事件处理流程\n\n用户提交事件，服务台分派给工程师，工程师处理后关闭。',
      processType: 'custom',
      enterpriseType: 'cn_enterprise',
      includeSla: true,
      includeNotifications: true,
      includeApprovals: true,
    });

    expect(await screen.findByText(generatedXml)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /Apply to Canvas/ }));
    expect(onApplyGeneratedProcess).toHaveBeenCalledWith(generatedXml);
  }, 40_000);

  it('renders a clean error when generation fails', async () => {
    (BPMNAIApi.generateBPMN as jest.Mock).mockRejectedValueOnce(new Error('AI服务暂不可用'));

    renderModal();

    fireEvent.change(screen.getByLabelText("Requirement Description"), {
      target: {
        value: '用户提交事件，服务台分派给工程师，工程师处理后关闭。',
      },
    });
    fireEvent.click(screen.getByRole('button', { name: /Generate Process/ }));

    expect(await screen.findByText("Generation Failed")).toBeInTheDocument();
    expect(screen.getByText('AI服务暂不可用')).toBeInTheDocument();
  });
});
