import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 应用与服务管理（/applications）类型列回归（D14）：
 * 类型必须与表单 Select 同口径中文（Web应用/移动应用/后端服务），
 * 不得出现 (type||'UNKNOWN').toUpperCase() 的原始大写值。
 */

jest.mock('@/lib/user-preferences', () => ({
  userPreferences: {
    get: () => ({ language: 'zh-CN' }),
    subscribe: () => () => {},
  },
}));

jest.mock('@/lib/services/application-service', () => ({
  applicationService: {
    listApplications: jest.fn().mockResolvedValue([
      { id: 1, name: '门户', code: 'portal', type: 'web', projectId: 1, edges: { microservices: [] } },
      { id: 2, name: '移动端', code: 'app', type: 'mobile', projectId: 1, edges: { microservices: [] } },
      { id: 3, name: '未标类型', code: 'none', type: '', projectId: 1, edges: { microservices: [] } },
    ]),
  },
}));

jest.mock('@/lib/services/project-service', () => ({
  projectService: { listProjects: jest.fn().mockResolvedValue([]) },
}));

import ApplicationsPage from '../page';
import { applicationService } from '@/lib/services/application-service';

describe('应用管理类型列（D14）', () => {
  beforeEach(() => jest.clearAllMocks());

  it('类型显示中文映射，空值显示 -，不出现 WEB/UNKNOWN', async () => {
    render(
      <App>
        <ApplicationsPage />
      </App>
    );
    await waitFor(() => expect(applicationService.listApplications).toHaveBeenCalled());

    expect(await screen.findByText("Web Application")).toBeInTheDocument();
    expect(screen.getByText("Mobile App")).toBeInTheDocument();

    const body = document.body.textContent || '';
    expect(body).not.toContain('UNKNOWN');
    expect(body).not.toMatch(/\b(WEB|MOBILE|BACKEND)\b/);
  });
});
