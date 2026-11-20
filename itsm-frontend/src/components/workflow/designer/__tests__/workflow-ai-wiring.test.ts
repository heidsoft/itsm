import fs from 'fs';
import path from 'path';

// 纯文本契约：锁定「源码结构」层面的修复，避免行为测试覆盖不到的接线回归。
// 相对路径基准：本文件位于 src/components/workflow/designer/__tests__/
const componentDir = path.resolve(__dirname, '..');
const toolbarSrc = fs.readFileSync(path.join(componentDir, 'WorkflowToolbar.tsx'), 'utf8');
const modalSrc = fs.readFileSync(path.join(componentDir, 'WorkflowAIModal.tsx'), 'utf8');
const designerSrc = fs.readFileSync(path.join(componentDir, 'WorkflowDesigner.tsx'), 'utf8');
const i18nSrc = fs.readFileSync(path.resolve(__dirname, '../../../../lib/i18n/translations.ts'), 'utf8');

const stripComments = (s: string) => s.replace(/\/\/.*$/gm, '').replace(/\/\*[\s\S]*?\*\//g, '');

describe('AI 创作/建议接线契约', () => {
  it('工具栏三个 AI 菜单项分别传 generate / optimize / compliance，不再共用无参回调', () => {
    const src = stripComments(toolbarSrc);
    expect(src).toMatch(/onAIClick\?\.\('generate'\)/);
    expect(src).toMatch(/onAIClick\?\.\('optimize'\)/);
    expect(src).toMatch(/onAIClick\?\.\('compliance'\)/);
    expect(src).not.toMatch(/onClick:\s*onAIClick\s*[,}]/);
  });

  it('设计器把入口 tab 存入状态并传给 Modal', () => {
    const src = stripComments(designerSrc);
    expect(src).toMatch(/setAiModalTab\(tab\)/);
    expect(src).toMatch(/initialTab=\{aiModalTab\}/);
  });

  it('Modal 用 categoryLabel 映射，且映射表覆盖后端全部 category 枚举', () => {
    const src = stripComments(modalSrc);
    expect(src).toMatch(/title:\s*categoryLabel\(issue\.category\)/);
    expect(src).toMatch(/rule:\s*categoryLabel\(issue\.category\)/);
    expect(src).not.toMatch(/title:\s*issue\.category/);
    expect(src).not.toMatch(/rule:\s*issue\.category/);
    for (const c of ['structure', 'events', 'tasks', 'flows', 'connectivity', 'gateways', 'approval']) {
      expect(src).toContain(`${c}:`);
    }
  });

  it('lint 结果统一走 readLintIssues 防御，不再直接 result.issues.map', () => {
    const src = stripComments(modalSrc);
    expect(src).toMatch(/const issues = readLintIssues\(result\)/);
    expect(src).not.toMatch(/result\.issues\.map/);
  });

  it('preview 与 template 的 validateFields 被 try/catch 包裹', () => {
    const src = stripComments(modalSrc);
    // 两处 validateFields 前都应有 try {
    const matches = src.match(/try\s*\{\s*values\s*=\s*await\s+form\.validateFields\(/g);
    expect(matches).not.toBeNull();
    expect(matches!.length).toBeGreaterThanOrEqual(2);
  });

  it('中英文翻译文件都补齐 category 与 lintEmptyResult key', () => {
    for (const key of [
      'categoryStructure', 'categoryEvents', 'categoryTasks', 'categoryFlows',
      'categoryConnectivity', 'categoryGateways', 'categoryApproval', 'categoryOther',
      'lintEmptyResult',
    ]) {
      // 每个 key 在中英两处各出现一次（键名相同，共 2 次）
      expect(i18nSrc.split(key).length - 1).toBeGreaterThanOrEqual(2);
    }
  });
});
