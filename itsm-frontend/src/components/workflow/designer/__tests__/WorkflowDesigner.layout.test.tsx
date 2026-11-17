import React from 'react';
import { App } from 'antd';
import { render, waitFor } from '@testing-library/react';
import WorkflowDesigner from '../WorkflowDesigner';

// D51 回归：/workflow/designer 根容器布局契约。
// 全局 Header 是 sticky（56px），main-content 有上下 padding，另有 footer。
// 设计器根容器若用 h-screen（100vh），文档会比视口高约 136px；
// 页面滚动后 sticky Header 会遮住本页 Tabs（「版本管理」「流程配置」不可点击）。
// 修复：根容器高度 = 100vh - 全局chrome，Tabs 通过 workflow-designer-tabs
// 类把高度链传递到 tabpane（见 globals.css），内容在 Tab 内滚动。

jest.mock('next/navigation', () => ({
  useRouter: () => ({ push: jest.fn(), replace: jest.fn(), prefetch: jest.fn() }),
  useSearchParams: () => ({ get: () => null }),
}));

jest.mock('@/lib/i18n/useI18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

jest.mock('@/lib/api/workflow-api', () => ({
  WorkflowAPI: {
    getProcessDefinition: jest.fn(),
    listVersions: jest.fn(),
    createProcessDefinition: jest.fn(),
    updateProcessDefinition: jest.fn(),
    deployProcessDefinition: jest.fn(),
    createAndPublishProcessDefinition: jest.fn(),
    publishProcessDefinition: jest.fn(),
    createWorkflowVersion: jest.fn(),
  },
}));

jest.mock('@/lib/api/bpmn-ai-api', () => ({
  BPMNAIApi: { lintBPMN: jest.fn() },
}));

jest.mock('@/lib/api/user-api', () => ({
  UserApi: { getUsers: jest.fn().mockResolvedValue({ items: [], total: 0 }) },
}));

jest.mock('@/lib/api/role-api', () => ({
  RoleAPI: { getRoles: jest.fn().mockResolvedValue({ items: [], total: 0 }) },
}));

jest.mock('@/lib/api/group-api', () => ({
  GroupAPI: { getGroups: jest.fn().mockResolvedValue({ items: [], total: 0 }) },
}));

jest.mock('@/lib/api/http-client', () => ({
  httpClient: { getTenantId: jest.fn(() => 1) },
}));

jest.mock('@/lib/api/pagination', () => ({ BACKEND_MAX_PAGE_SIZE: 100 }));

// 重型子组件全部打桩：本测试只关心布局高度契约
jest.mock('../WorkflowToolbar', () => () => <div data-testid="toolbar" />);
jest.mock('../WorkflowCanvas', () => ({
  __esModule: true,
  default: () => <div data-testid="canvas" />,
  getBpmnDesignerApi: jest.fn(() => null),
}));
jest.mock('../WorkflowNodeInspector', () => () => <div data-testid="inspector" />);
jest.mock('../WorkflowProperties', () => () => <div data-testid="properties" />);
jest.mock('../WorkflowNewModal', () => () => null);
jest.mock('../WorkflowVersionModal', () => () => null);
jest.mock('../WorkflowSettingsModal', () => () => null);
jest.mock('../WorkflowMetadataModal', () => () => null);
jest.mock('../WorkflowAIModal', () => () => null);
jest.mock('../../BPMNDiagramPreview', () => () => null);

describe('WorkflowDesigner 布局契约（D51 回归）', () => {
  it('根容器扣除全局 chrome 而非 h-screen，Tabs 使用填充高度链', async () => {
    const { container } = render(
      <App>
        <WorkflowDesigner />
      </App>
    );

    await waitFor(() => {
      expect(container.querySelector('.ant-tabs')).toBeTruthy();
    });

    const rootLayout = container.querySelector('.ant-layout');
    expect(rootLayout).toBeTruthy();
    const rootClass = rootLayout!.className;
    // 修复前为 h-screen（=100vh）：文档溢出，sticky Header 遮挡 Tabs
    expect(rootClass).not.toContain('h-screen');
    // 修复后：100vh - 56px(Header) - 24px(移动padding) / 32px(桌面padding) - 48px(footer)
    expect(rootClass).toContain('calc(100vh-128px)');
    expect(rootClass).toContain('calc(100vh-136px)');

    // Tabs 高度链 hook：globals.css 依此把高度逐级传到 tabpane
    const tabs = container.querySelector('.ant-tabs');
    expect(tabs).toBeTruthy();
    expect(tabs!.className).toContain('workflow-designer-tabs');

    // Tab 内容不再使用会漂移的 100vh 魔数高度
    expect(container.innerHTML).not.toContain('100vh-200px');
    expect(container.innerHTML).not.toContain('100vh-220px');
  });
});
