/**
 * SLA 模板 API
 *
 * 对应后端 controller/sla_template_controller.go
 * 路由：/api/v1/sla/templates
 *   GET    /                     -> ListTemplates
 *   GET    /:key                 -> GetTemplate
 *   POST   /:key/install         -> InstallTemplate
 */

import { httpClient } from './http-client';

// SLA 模板（开箱即用预置）
export interface SLATemplate {
  key: string;
  name: string;
  description: string;
  serviceType: string;
  priority: string;
  responseTime: number; // 分钟
  resolutionTime: number; // 分钟
  businessHours: Record<string, unknown>;
  escalationRules: Record<string, unknown>;
  conditions: Record<string, unknown>;
  industry: string; // incident / change / service_request
  recommended: boolean;
}

// 模板安装结果
export interface TemplateInstallResult {
  templateKey: string;
  slaDefinitionId: number;
  created: boolean;
  wasAlreadyExist: boolean;
  message: string;
}

export interface TemplateListResponse {
  templates: SLATemplate[];
  total: number;
}

export class SLATemplateApi {
  /** 列出所有预置 SLA 模板 */
  static async listTemplates(): Promise<SLATemplate[]> {
    const data = await httpClient.get<TemplateListResponse | SLATemplate[]>(
      '/api/v1/sla/templates'
    );
    if (Array.isArray(data)) return data;
    return (data as TemplateListResponse).templates ?? [];
  }

  /** 获取单个 SLA 模板详情 */
  static async getTemplate(key: string): Promise<SLATemplate> {
    return httpClient.get<SLATemplate>(`/api/v1/sla/templates/${encodeURIComponent(key)}`);
  }

  /** 将模板安装到当前租户（幂等：已存在则返回 wasAlreadyExist=true） */
  static async installTemplate(key: string): Promise<TemplateInstallResult> {
    return httpClient.post<TemplateInstallResult>(
      `/api/v1/sla/templates/${encodeURIComponent(key)}/install`,
      {}
    );
  }

  /** 批量安装全部推荐模板 */
  static async installAllRecommended(): Promise<TemplateInstallResult[]> {
    const list = await this.listTemplates();
    const recommended = list.filter(t => t.recommended);
    const results: TemplateInstallResult[] = [];
    for (const t of recommended) {
      try {
        // eslint-disable-next-line no-await-in-loop
        const r = await this.installTemplate(t.key);
        results.push(r);
      } catch (err) {
        // 单个失败不影响其他
        // eslint-disable-next-line no-console
        console.error(`install template ${t.key} failed`, err);
      }
    }
    return results;
  }
}

export default SLATemplateApi;