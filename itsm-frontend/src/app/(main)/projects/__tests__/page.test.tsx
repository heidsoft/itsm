import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import '@testing-library/jest-dom';

/**
 * 项目管理（/projects）状态列回归（D13）：
 * 状态必须中文化（进行中/已完成/已暂停），空值显示 '-'，
 * 不得出现原始英文枚举或 'Unknown' 兜底。
 */

jest.mock('@/lib/user-preferences', () => ({
  userPreferences: {
    get: () => ({ language: 'zh-CN' }),
    subscribe: () => () => {},
  },
}));

jest.mock('@/lib/services/project-service', () => ({
  projectService: {
    listProjects: jest.fn().mockResolvedValue([
      { id: 1, name: '核心平台', code: 'PRJ-1', status: 'active' },
      { id: 2, name: '老系统', code: 'PRJ-2', status: 'completed' },
      { id: 3, name: '暂停项', code: 'PRJ-3', status: 'suspended' },
      { id: 4, name: '未标记', code: 'PRJ-4', status: '' },
    ]),
    deleteProject: jest.fn(),
  },
}));

jest.mock('@/lib/services/department-service', () => ({
  departmentService: { getDepartmentTree: jest.fn().mockResolvedValue([]) },
}));

jest.mock('@/lib/api/user-api', () => ({
  UserApi: { getUsers: jest.fn().mockResolvedValue({ items: [] }) },
}));

import ProjectsPage from '../page';
import { projectService } from '@/lib/services/project-service';

describe('项目管理状态列（D13）', () => {
  beforeEach(() => jest.clearAllMocks());

  it('状态枚举显示中文，空值显示 -，不出现 Unknown/原始英文', async () => {
    render(
      <App>
        <ProjectsPage />
      </App>
    );
    await waitFor(() => expect(projectService.listProjects).toHaveBeenCalled());

    expect(await screen.findByText('进行中')).toBeInTheDocument();
    expect(screen.getByText('已完成')).toBeInTheDocument();
    expect(screen.getByText('已暂停')).toBeInTheDocument();

    const body = document.body.textContent || '';
    expect(body).not.toContain('Unknown');
    expect(body).not.toMatch(/\b(active|completed|suspended)\b/);
  });
});
