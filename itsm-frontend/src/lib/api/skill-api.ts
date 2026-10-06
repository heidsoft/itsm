/**
 * AI 技能注册表 API
 *
 * 对应后端 handlers/skill/handler.go 的 /api/v1/skills 与 /api/v1/admin/skills。
 * 复用 skill_registry.go (in-process registry) 与 skill_manifest.go 的 manifest 校验。
 *
 * Sprint 2 Task 2: 把 registry 的管理面板暴露为 admin 路由，让 marketplace:write 角色
 * 能浏览、注册、更新、提升 (promote) 与禁用技能。
 */

import { httpClient } from '@/lib/api/http-client';

export type SkillCategory = 'ga' | 'pilot' | 'experimental';
export type SkillStatus = 'active' | 'disabled';

export interface SkillManifest {
  iconUrl?: string;
  screenshots?: string[];
  inputSchema?: Record<string, unknown>;
  outputSchema?: Record<string, unknown>;
  minSystemVersion?: string;
  rating?: number;
  installCount?: number;
}

export interface SkillMetrics {
  totalCalls: number;
  successRate: number;
  avgLatencyMs: number;
  errorCount: number;
  lastUsedAt?: string;
}

export interface SkillEntry {
  code: string;
  name: string;
  version: string;
  title: string;
  provider: string;
  description?: string;
  longDescription?: string;
  category: SkillCategory;
  tags: string[];
  capabilities: string[];
  requiredPermissions: string[];
  author?: string;
  isOfficial: boolean;
  checksum: string;
  isBuiltin: boolean;
  status: SkillStatus;
  manifest: SkillManifest;
  metrics: SkillMetrics;
}

export interface SkillListResponse {
  items: SkillEntry[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export interface SkillUpsertRequest {
  code: string;
  version: string;
  title: string;
  description?: string;
  longDescription?: string;
  category: SkillCategory;
  tags: string[];
  capabilities: string[];
  requiredPermissions: string[];
  provider: string;
  author?: string;
  inputSchema?: Record<string, unknown>;
  outputSchema?: Record<string, unknown>;
  executor?: Record<string, unknown>;
}

export interface SkillListParams {
  tag?: string;
  category?: SkillCategory;
  status?: SkillStatus;
  q?: string;
  page?: number;
  pageSize?: number;
}

export const skillApi = {
  async list(params: SkillListParams = {}): Promise<SkillListResponse> {
    const search = new URLSearchParams();
    if (params.tag) search.set('tag', params.tag);
    if (params.category) search.set('category', params.category);
    if (params.status) search.set('status', params.status);
    if (params.q) search.set('q', params.q);
    if (params.page !== undefined) search.set('page', String(params.page));
    if (params.pageSize !== undefined) search.set('pageSize', String(params.pageSize));
    const query = search.toString();
    return httpClient.get<SkillListResponse>(
      `/api/v1/admin/skills${query ? `?${query}` : ''}`,
    );
  },

  async get(code: string): Promise<SkillEntry> {
    return httpClient.get<SkillEntry>(`/api/v1/admin/skills/${encodeURIComponent(code)}`);
  },

  async create(body: SkillUpsertRequest): Promise<SkillEntry> {
    return httpClient.post<SkillEntry>('/api/v1/admin/skills', body);
  },

  async update(code: string, body: SkillUpsertRequest): Promise<{ updated: true; manifest: SkillManifest }> {
    return httpClient.put<{ updated: true; manifest: SkillManifest }>(
      `/api/v1/admin/skills/${encodeURIComponent(code)}`,
      body,
    );
  },

  async promote(code: string): Promise<{ category: 'ga' }> {
    return httpClient.post<{ category: 'ga' }>(
      `/api/v1/admin/skills/${encodeURIComponent(code)}/promote`,
      {},
    );
  },

  async disable(code: string): Promise<{ disabled: true }> {
    return httpClient.delete<{ disabled: true }>(
      `/api/v1/admin/skills/${encodeURIComponent(code)}`,
    );
  },
};

export default skillApi;
