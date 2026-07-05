/**
 * CIBasicInfo - CI 基本信息组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { CIBasicInfo } from '../sections/CIBasicInfo';
import { CIStatus } from '@/constants/cmdb';
import type { ConfigurationItem } from '@/types/biz/cmdb';

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}));

describe('CIBasicInfo', () => {
  const mockCI = {
    // ConfigurationItem required fields
    id: 1,
    name: 'Web Server 1',
    type: 'server',
    status: CIStatus.ACTIVE,
    // ConfigurationItem 接口字段
    ciTypeId: 1,
    assetTag: 'WEB-001',
    serialNumber: 'ABC123456',
    model: 'PowerEdge R740',
    vendor: 'Dell',
    location: 'Data Center A, Rack 12',
    environment: 'production',
    criticality: 'high',
    assignedTo: 'DevOps Team',
    ownedBy: 'John Doe',
    discoverySource: 'auto-discovery',
    source: 'manual',
    cloudProvider: 'AWS',
    cloudAccountId: 'aws-123456',
    cloudRegion: 'us-east-1',
    cloudZone: 'us-east-1a',
    cloudResourceId: 'i-1234567890abcdef0',
    cloudResourceType: 'EC2 Instance',
    cloudResourceRefId: 123456,
    cloudSyncStatus: 'success',
    cloudSyncTime: '2024-01-20T15:30:00Z',
    tenantId: 1,
    createdAt: '2024-01-15T10:00:00Z',
    updatedAt: '2024-01-20T15:30:00Z',
    description: 'Production web server for customer portal',
  } as ConfigurationItem;

  const defaultProps = {
    ci: mockCI,
  };

  it('应该显示 CI 基本信息字段', () => {
    render(<CIBasicInfo {...defaultProps} />);

    // 验证关键字段存在
    expect(screen.getByText(mockCI.assetTag!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.serialNumber!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.model!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.vendor!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.location!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.environment!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.assignedTo!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.ownedBy!)).toBeInTheDocument();
  });

  it('应该显示云相关信息', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.cloudProvider!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.cloudRegion!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.cloudResourceType!)).toBeInTheDocument();
  });

  it('应该显示时间信息', () => {
    render(<CIBasicInfo {...defaultProps} />);

    // 检查日期信息存在
    expect(screen.getAllByText(/2024/).length).toBeGreaterThan(0);
  });

  it('应该显示描述', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.description)).toBeInTheDocument();
  });

  it('应该显示所属租户', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(String(mockCI.tenantId))).toBeInTheDocument();
  });

  it('应该处理可选字段为空值', () => {
    const minimalCI: ConfigurationItem = {
      id: 1,
      name: 'Minimal CI',
      type: 'server',
      status: CIStatus.ACTIVE,
      ciTypeId: 1,
      tenantId: 1,
      createdAt: '2024-01-15T10:00:00Z',
      updatedAt: '2024-01-20T15:30:00Z',
      description: 'Minimal CI for testing',
      assetTag: undefined,
      serialNumber: undefined,
      model: undefined,
      vendor: undefined,
      location: undefined,
      environment: undefined,
      criticality: undefined,
      assignedTo: undefined,
      ownedBy: undefined,
      discoverySource: undefined,
      source: undefined,
      cloudProvider: undefined,
      cloudAccountId: undefined,
      cloudRegion: undefined,
      cloudZone: undefined,
      cloudResourceId: undefined,
      cloudResourceType: undefined,
      cloudResourceRefId: undefined,
      cloudSyncStatus: undefined,
      cloudSyncTime: undefined,
    };

    render(<CIBasicInfo ci={minimalCI} />);

    // 空值应显示占位符 '-'，不应崩溃
    expect(screen.getAllByText('-').length).toBeGreaterThan(0);
  });

  it('应该显示资产分类', () => {
    render(<CIBasicInfo {...defaultProps} />);

    // 资产分类显示为 "类型 {ci_type_id}"
    expect(screen.getByText(/类型 1/i)).toBeInTheDocument();
  });

  it('应该显示标签和重要性', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.criticality!)).toBeInTheDocument();
  });

  it('应该显示发现源和数据来源', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.discoverySource!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.source!)).toBeInTheDocument();
  });
});
