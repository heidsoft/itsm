import { httpClient } from './http-client';

export interface DomainConfig {
  id: number;
  config_key: string;
  config_type: string;
  config_value: Record<string, unknown>;
  inherit_mode: 'inherit' | 'override' | 'extend';
  tenant_id: number;
  department_id: number;
  team_id: number;
  version: number;
  is_active: boolean;
  description?: string;
  created_at?: string;
  updated_at?: string;
}

export interface DomainConfigPayload {
  config_type: string;
  config_key: string;
  config_value: Record<string, unknown>;
  inherit_mode?: 'inherit' | 'override' | 'extend';
  department_id?: number;
  team_id?: number;
  description?: string;
}

export interface EffectiveConfig {
  key: string;
  value: Record<string, unknown>;
  source: string;
  inherit_mode: string;
  version: number;
}

export interface EffectiveConfigQuery {
  config_type: string;
  config_key: string;
  department_id?: number;
  team_id?: number;
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
    config_key: config.config_key || config.configKey || '',
    config_type: config.config_type || config.configType || '',
    config_value: config.config_value || config.configValue || {},
    inherit_mode: config.inherit_mode || config.inheritMode || 'inherit',
    tenant_id: config.tenant_id ?? config.tenantId ?? 0,
    department_id: config.department_id ?? config.departmentId ?? 0,
    team_id: config.team_id ?? config.teamId ?? 0,
    version: config.version ?? 1,
    is_active: config.is_active ?? config.isActive ?? true,
    description: config.description,
    created_at: config.created_at || config.createdAt,
    updated_at: config.updated_at || config.updatedAt,
  };
}

function normalizeEffectiveConfig(config: RawEffectiveConfig | null | undefined): EffectiveConfig | null {
  if (!config) return null;
  return {
    key: config.key || '',
    value: config.value || {},
    source: config.source || '',
    inherit_mode: config.inherit_mode || config.inheritMode || '',
    version: config.version ?? 0,
  };
}

export class DomainConfigApi {
  static async list(configType?: string): Promise<DomainConfig[]> {
    const response = await httpClient.get<RawDomainConfig[]>(
      '/api/v1/domain-configs',
      configType ? { config_type: configType } : undefined
    );
    return (response || []).map(normalizeDomainConfig);
  }

  static async save(payload: DomainConfigPayload): Promise<void> {
    await httpClient.post('/api/v1/domain-configs', payload);
  }

  static async getEffective(query: EffectiveConfigQuery): Promise<EffectiveConfig | null> {
    const response = await httpClient.get<RawEffectiveConfig>('/api/v1/domain-configs/effective', {
      config_type: query.config_type,
      config_key: query.config_key,
      ...(query.department_id ? { department_id: query.department_id } : {}),
      ...(query.team_id ? { team_id: query.team_id } : {}),
    });
    return normalizeEffectiveConfig(response);
  }
}
