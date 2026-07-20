import { httpClient } from './http-client';

export interface DomainConfig {
  id: number;
  configKey: string;
  configType: string;
  configValue: Record<string, unknown>;
  inheritMode: 'inherit' | 'override' | 'extend';
  tenantId: number;
  departmentId: number;
  teamId: number;
  version: number;
  isActive: boolean;
  description?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface DomainConfigPayload {
  configType: string;
  configKey: string;
  configValue: Record<string, unknown>;
  inheritMode?: 'inherit' | 'override' | 'extend';
  departmentId?: number;
  teamId?: number;
  description?: string;
}

export interface EffectiveConfig {
  key: string;
  value: Record<string, unknown>;
  source: string;
  inheritMode: string;
  version: number;
}

export interface EffectiveConfigQuery {
  configType: string;
  configKey: string;
  departmentId?: number;
  teamId?: number;
}

type RawDomainConfig = Partial<DomainConfig> & {
  configKey?: string;
  configType?: string;
  configValue?: Record<string, unknown>;
  inheritMode?: 'inherit' | 'override' | 'extend';
  tenantId?: number;
  departmentId?: number;
  teamId?: number;
  isActive?: boolean;
  createdAt?: string;
  updatedAt?: string;
};

type RawEffectiveConfig = Partial<EffectiveConfig> & {
  inheritMode?: string;
};

function normalizeDomainConfig(config: RawDomainConfig): DomainConfig {
  return {
    id: config.id || 0,
    configKey: config.configKey || '',
    configType: config.configType || '',
    configValue: config.configValue || {},
    inheritMode: config.inheritMode || 'inherit',
    tenantId: config.tenantId ?? 0,
    departmentId: config.departmentId ?? 0,
    teamId: config.teamId ?? 0,
    version: config.version ?? 1,
    isActive: config.isActive ?? true,
    description: config.description,
    createdAt: config.createdAt,
    updatedAt: config.updatedAt,
  };
}

function normalizeEffectiveConfig(config: RawEffectiveConfig | null | undefined): EffectiveConfig | null {
  if (!config) return null;
  return {
    key: config.key || '',
    value: config.value || {},
    source: config.source || '',
    inheritMode: config.inheritMode || '',
    version: config.version ?? 0,
  };
}

export class DomainConfigApi {
  static async list(configType?: string): Promise<DomainConfig[]> {
    const response = await httpClient.get<RawDomainConfig[]>(
      '/api/v1/domain-configs',
      configType ? { configType: configType } : undefined
    );
    return (response || []).map(normalizeDomainConfig);
  }

  static async save(payload: DomainConfigPayload): Promise<void> {
    await httpClient.post('/api/v1/domain-configs', payload);
  }

  static async getEffective(query: EffectiveConfigQuery): Promise<EffectiveConfig | null> {
    const response = await httpClient.get<RawEffectiveConfig>('/api/v1/domain-configs/effective', {
      configType: query.configType,
      configKey: query.configKey,
      ...(query.departmentId ? { departmentId: query.departmentId } : {}),
      ...(query.teamId ? { teamId: query.teamId } : {}),
    });
    return normalizeEffectiveConfig(response);
  }
}
