/**
 * CIBasicInfo - CI 基本信息组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { CIBasicInfo } from '../sections/CIBasicInfo';
import type { CIBasicInfoProps } from '../types';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('CIBasicInfo', () => {
  const mockCI = {
    id: 'ci-001',
    name: 'Web Server 1',
    type: 'server',
    status: 'active',
    description: 'Production web server for customer portal',
    ipAddress: '192.168.1.100',
    macAddress: '00:1B:44:11:3A:B7',
    location: 'Data Center A, Rack 12',
    manufacturer: 'Dell',
    model: 'PowerEdge R740',
    serialNumber: 'ABC123456',
    team: 'DevOps',
    environment: 'production',
    owner: 'John Doe',
    createdAt: '2024-01-15T10:00:00Z',
    updatedAt: '2024-01-20T15:30:00Z',
  };

  const defaultProps: CIBasicInfoProps = {
    ci: mockCI,
  };

  it('应该显示 CI 名称', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.name)).toBeInTheDocument();
  });

  it('应该显示所有基本信息字段', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(/id/i)).toBeInTheDocument();
    expect(screen.getByText(mockCI.id)).toBeInTheDocument();
    expect(screen.getByText(/type/i)).toBeInTheDocument();
    expect(screen.getByText(mockCI.type)).toBeInTheDocument();
    expect(screen.getByText(/status/i)).toBeInTheDocument();
    expect(screen.getByText(mockCI.status)).toBeInTheDocument();
  });

  it('应该显示 IP 和 MAC 地址', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.ipAddress)).toBeInTheDocument();
    expect(screen.getByText(mockCI.macAddress)).toBeInTheDocument();
  });

  it('应该显示制造商和型号', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.manufacturer)).toBeInTheDocument();
    expect(screen.getByText(mockCI.model)).toBeInTheDocument();
  });

  it('应该显示序列号', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.serialNumber)).toBeInTheDocument();
  });

  it('应该显示位置信息', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.location)).toBeInTheDocument();
  });

  it('应该显示团队和负责人', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.team)).toBeInTheDocument();
    expect(screen.getByText(mockCI.owner)).toBeInTheDocument();
  });

  it('应该显示环境', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.environment)).toBeInTheDocument();
  });

  it('应该显示创建和更新时间', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(/2024|Jan|15/)).toBeInTheDocument();
    expect(screen.getByText(/2024|Jan|20/)).toBeInTheDocument();
  });

  it('应该格式化描述文本', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.description)).toBeInTheDocument();
  });

  it('应该显示状态指示器', () => {
    render(<CIBasicInfo {...defaultProps} />);

    const statusElement = screen.getByText(mockCI.status).closest('span');
    expect(statusElement).toHaveClass('status-indicator');
  });

  it('应该处理可选字段为空', () => {
    const minimalCI = {
      ...mockCI,
      manufacturer: null,
      model: null,
      serialNumber: null,
    };

    render(<CIBasicInfo ci={minimalCI} />);

    // 不应崩溃，可选字段不显示
    expect(screen.getByText(mockCI.name)).toBeInTheDocument();
  });

  it('应该显示 IP 地址格式', () => {
    render(<CIBasicInfo {...defaultProps} />);

    const ipElement = screen.getByText(mockCI.ipAddress);
    expect(ipElement).toHaveClass('ip-address');
  });

  it('应该显示 MAC 地址格式', () => {
    render(<CIBasicInfo {...defaultProps} />);

    const macElement = screen.getByText(mockCI.macAddress);
    expect(macElement).toHaveClass('mac-address');
  });

  it('应该根据状态应用不同颜色', () => {
    render(<CIBasicInfo {...defaultProps} />);

    const statusElement = screen.getByText('active');
    expect(statusElement.closest('span')).toHaveClass('status-active');
  });

  it('应该显示部分 ID 用于显示', () => {
    render(<CIBasicInfo {...defaultProps} />);

    // 可能只显示部分 ID
    const idElement = screen.getByText(mockCI.id);
    expect(idElement).toBeInTheDocument();
  });
});
