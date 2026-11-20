import type { BPMNLintIssue } from '@/lib/api/bpmn-ai-api';

export interface ValidationIssue {
  type: 'error' | 'warning' | 'info';
  message: string;
  elementId?: string;
  elementType?: string;
  elementName?: string;
  /** 来源分类：backend=后端 lint；validation-unavailable=校验服务自身不可用（fail-closed） */
  category?: string;
}

/**
 * 校验流程本身不可用时的阻断性 issue。
 *
 * 绝不能用空数组表达「校验失败」——部署/发布门禁用 issues.some(type==='error')
 * 判定放行，空数组会被当成「校验通过」，让破坏性流程绕过门禁上线。
 */
export function validationUnavailableIssue(message: string): ValidationIssue {
  return {
    type: 'error',
    category: 'validation-unavailable',
    message,
  };
}

/** 校验流程自身失败（服务不可用），门禁必须据此阻断部署/发布。 */
export function isValidationUnavailable(issues: ValidationIssue[]): boolean {
  return issues.some(issue => issue.category === 'validation-unavailable');
}

const issueKey = (elementId: string | undefined, message: string) => `${elementId ?? ''}|${message}`;

/**
 * 合并画布本地校验与后端 lint。
 * 后端是语义规则的唯一来源；本地结果先展示，重复项按元素+文案去重。
 */
export function mergeLintIssues(
  localIssues: ValidationIssue[],
  lintIssues: BPMNLintIssue[]
): ValidationIssue[] {
  const merged = [...localIssues];
  const seen = new Set(merged.map((issue) => issueKey(issue.elementId, issue.message)));

  for (const issue of lintIssues) {
    const key = issueKey(issue.elementId, issue.message);
    if (seen.has(key)) continue;
    seen.add(key);
    merged.push({
      type: issue.severity === 'error' ? 'error' : issue.severity === 'warning' ? 'warning' : 'info',
      message: issue.message,
      elementId: issue.elementId,
      elementName: issue.elementName,
    });
  }

  return merged;
}
