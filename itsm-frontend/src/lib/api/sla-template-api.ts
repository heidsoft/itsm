/**
 * SLA 模板 API
 *
 * 后端 handlers/sla_template/handler.go，路由 /api/v1/sla/templates：
 *   GET    /             -> ListTemplates（含当前租户的安装状态）
 *   GET    /:key         -> GetTemplate
 *   POST   /:key/install -> InstallTemplate（按租户幂等）
 */

import { httpClient } from './http-client';
import type { SLABusinessHours, SLAEscalationLevel } from './sla-api';

const SLA_TEMPLATES_PATH = '/api/v1/sla/templates';

// 与后端 dto.SLATemplateResponse 逐字段一致：businessHours / escalationLevels
// 由后端投影成 camelCase，形状与已安装 SLA 定义相同。
export interface SLATemplate {
  key: string;
  name: string;
  description: string;
  serviceType: string;
  priority: string;
  responseTime: number; // 分钟
  resolutionTime: number; // 分钟
  businessHours?: SLABusinessHours;
  escalationLevels?: SLAEscalationLevel[];
  conditions: Record<string, unknown>;
  industry: string; // incident / change / service_request
  recommended: boolean;
  installed: boolean;
  slaDefinitionId?: number;
  installedAt?: string;
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
  items: SLATemplate[];
  total: number;
}

export interface TemplateInstallFailure {
  templateKey: string;
  reason: string;
}

// 批量安装逐项明细：失败模板必须点名，不能被「批量安装失败」一句话吞掉
export interface TemplateInstallSummary {
  results: TemplateInstallResult[];
  failures: TemplateInstallFailure[];
}

const failureReason = (error: unknown): string =>
  error instanceof Error && error.message ? error.message : '未知错误';

export class SLATemplateApi {
  /** 列出预置模板及其在当前租户的安装状态 */
  static async listTemplates(): Promise<SLATemplate[]> {
    const data = await httpClient.get<TemplateListResponse>(SLA_TEMPLATES_PATH);
    return data.items;
  }

  /** 获取单个 SLA 模板详情 */
  static async getTemplate(key: string): Promise<SLATemplate> {
    return httpClient.get<SLATemplate>(`${SLA_TEMPLATES_PATH}/${encodeURIComponent(key)}`);
  }

  /** 将模板安装到当前租户（幂等：已存在则返回 wasAlreadyExist=true） */
  static async installTemplate(key: string): Promise<TemplateInstallResult> {
    return httpClient.post<TemplateInstallResult>(
      `${SLA_TEMPLATES_PATH}/${encodeURIComponent(key)}/install`,
      {}
    );
  }

  /** 批量安装全部推荐模板，逐个提交并保留失败明细 */
  static async installAllRecommended(): Promise<TemplateInstallSummary> {
    const list = await this.listTemplates();
    const recommended = list.filter(t => t.recommended);
    const results: TemplateInstallResult[] = [];
    const failures: TemplateInstallFailure[] = [];

    for (const template of recommended) {
      try {
        results.push(await this.installTemplate(template.key));
      } catch (error) {
        failures.push({ templateKey: template.key, reason: failureReason(error) });
      }
    }

    return { results, failures };
  }
}

export default SLATemplateApi;
