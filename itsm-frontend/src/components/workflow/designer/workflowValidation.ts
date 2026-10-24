import type { BPMNLintIssue } from '@/lib/api/bpmn-ai-api';

export interface ValidationIssue {
  type: 'error' | 'warning' | 'info';
  message: string;
  elementId?: string;
  elementType?: string;
  elementName?: string;
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
