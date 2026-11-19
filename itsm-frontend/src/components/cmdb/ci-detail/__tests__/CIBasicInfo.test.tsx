/**
 * CIBasicInfo - CI 基本信息组件测试
 */

import React from 'react';
import { render, screen } from '@testing-library/react';
import { CIBasicInfo } from '../sections/CIBasicInfo';
import { CIStatus } from '@/constants/cmdb';
import { translations } from '@/lib/i18n/translations';
import type { ConfigurationItem, CIType } from '@/types/biz/cmdb';

// 用真实的 zh-CN 词表 mock t，而不是 `key => key`：
// 后者会让枚举反查拿到 i18n key，测试就永远看不到「生产/中」这类真实文案，
// 也就锁不住「详情页显示 production/medium」这个历史缺陷。
const zhCN = translations['zh-CN'] as unknown as Record<string, unknown>;
const resolve = (key: string): unknown => {
  let value: unknown = zhCN;
  for (const part of key.split('.')) {
    if (value && typeof value === 'object' && part in value) {
      value = (value as Record<string, unknown>)[part];
    } else {
      return undefined;
    }
  }
  return value;
};

jest.mock('@/lib/i18n', () => ({
  useI18n: () => ({
    t: (key: string) => {
      const value = resolve(key);
      return typeof value === 'string' ? value : key;
    },
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

    // 验证关键字段存在（非枚举的直出字段）
    expect(screen.getByText(mockCI.assetTag!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.serialNumber!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.model!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.vendor!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.location!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.assignedTo!)).toBeInTheDocument();
    expect(screen.getByText(mockCI.ownedBy!)).toBeInTheDocument();
  });

  it('应该把枚举值翻译成中文，而不是直出后端原始值', () => {
    render(<CIBasicInfo {...defaultProps} />);

    // 历史缺陷：环境/重要性/数据来源/云厂商/同步状态曾直接输出
    // production / high / manual / AWS / success，与 create/edit 表单文案不一致。
    expect(screen.getByText('生产')).toBeInTheDocument();
    expect(screen.getByText('高')).toBeInTheDocument();
    expect(screen.getByText('手工录入')).toBeInTheDocument();
    // AWS 在词表中本身就是 AWS，用 success→成功 验证同步状态走的是词表
    expect(screen.getByText('成功')).toBeInTheDocument();
    // 原始枚举值不得出现在详情页
    expect(screen.queryByText('production')).not.toBeInTheDocument();
    expect(screen.queryByText('high')).not.toBeInTheDocument();
    expect(screen.queryByText('success')).not.toBeInTheDocument();
  });

  it('未知枚举值应原样回退，不被吞成空白', () => {
    render(
      <CIBasicInfo
        ci={{ ...mockCI, environment: 'edge-case-env', criticality: 'unknown-value' }}
      />
    );

    expect(screen.getByText('edge-case-env')).toBeInTheDocument();
    expect(screen.getByText('unknown-value')).toBeInTheDocument();
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

  it('应该显示资产分类：优先中文 description，无 typeInfo 时回退类型 ID', () => {
    // 有 typeInfo 时展示中文名，而不是 ci_types.name 里的英文代码
    render(
      <CIBasicInfo
        ci={mockCI}
        typeInfo={
          {
            id: 1,
            name: 'server',
            description: '服务器',
            attributeSchema: '',
            isActive: true,
            tenantId: 1,
          } as CIType
        }
      />
    );
    expect(screen.getByText('服务器')).toBeInTheDocument();
    expect(screen.queryByText('server')).not.toBeInTheDocument();
  });

  it('无 typeInfo 时资产分类回退为「类型 {id}」', () => {
    render(<CIBasicInfo {...defaultProps} />);
    expect(screen.getByText(/类型 1/i)).toBeInTheDocument();
  });

  it('应该按类型模板显示动态属性并格式化布尔值', () => {
    render(
      <CIBasicInfo
        ci={{
          ...mockCI,
          attributes: { cpuCores: 16, monitored: true },
        }}
        typeInfo={{
          id: 1,
          name: '服务器',
          description: '',
          attributeSchema: JSON.stringify({
            fields: [
              { key: 'cpuCores', label: 'CPU 核数', type: 'number' },
              { key: 'monitored', label: '纳入监控', type: 'boolean' },
            ],
          }),
          isActive: true,
          tenantId: 1,
        }}
      />
    );

    expect(screen.getByText('CPU 核数')).toBeInTheDocument();
    expect(screen.getByText('16')).toBeInTheDocument();
    expect(screen.getByText('纳入监控')).toBeInTheDocument();
    expect(screen.getByText('是')).toBeInTheDocument();
  });
});
