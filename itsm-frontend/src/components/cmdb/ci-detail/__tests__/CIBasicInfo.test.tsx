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
    ci_type_id: 1,
    asset_tag: 'WEB-001',
    serial_number: 'ABC123456',
    model: 'PowerEdge R740',
    vendor: 'Dell',
    location: 'Data Center A, Rack 12',
    environment: 'production',
    criticality: 'high',
    assigned_to: 'DevOps Team',
    owned_by: 'John Doe',
    discovery_source: 'auto-discovery',
    source: 'manual',
    cloud_provider: 'AWS',
    cloud_account_id: 'aws-123456',
    cloud_region: 'us-east-1',
    cloud_zone: 'us-east-1a',
    cloud_resource_id: 'i-1234567890abcdef0',
    cloud_resource_type: 'EC2 Instance',
    cloud_resource_ref_id: 123456,
    cloud_sync_status: 'success',
    cloud_sync_time: '2024-01-20T15:30:00Z',
    tenant_id: 1,
    created_at: '2024-01-15T10:00:00Z',
    updated_at: '2024-01-20T15:30:00Z',
    description: 'Production web server for customer portal',
  } as ConfigurationItem;

  const defaultProps = {
    ci: mockCI,
  };

  it('应该显示 CI 基本信息字段', () => {
    render(<CIBasicInfo {...defaultProps} />);

    // 验证关键字段存在
    expect(screen.getByText(mockCI.asset_tag!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.serial_number!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.model!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.vendor!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.location!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.environment!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.assigned_to!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.owned_by!)).toBeInTheDocument();
  });

  it('应该显示云相关信息', () => {
    render(<CIBasicInfo {...defaultProps} />);

    expect(screen.getByText(mockCI.cloud_provider!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.cloud_region!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.cloud_resource_type!)).toBeInTheDocument();
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

    expect(screen.getByText(String(mockCI.tenant_id))).toBeInTheDocument();
  });

  it('应该处理可选字段为空值', () => {
    const minimalCI: ConfigurationItem = {
      id: 1,
      name: 'Minimal CI',
      type: 'server',
      status: CIStatus.ACTIVE,
      ci_type_id: 1,
      tenant_id: 1,
      created_at: '2024-01-15T10:00:00Z',
      updated_at: '2024-01-20T15:30:00Z',
      description: 'Minimal CI for testing',
      asset_tag: undefined,
      serial_number: undefined,
      model: undefined,
      vendor: undefined,
      location: undefined,
      environment: undefined,
      criticality: undefined,
      assigned_to: undefined,
      owned_by: undefined,
      discovery_source: undefined,
      source: undefined,
      cloud_provider: undefined,
      cloud_account_id: undefined,
      cloud_region: undefined,
      cloud_zone: undefined,
      cloud_resource_id: undefined,
      cloud_resource_type: undefined,
      cloud_resource_ref_id: undefined,
      cloud_sync_status: undefined,
      cloud_sync_time: undefined,
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

    expect(screen.getByText(mockCI.discovery_source!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.source!)).toBeInTheDocument();
  });
});
