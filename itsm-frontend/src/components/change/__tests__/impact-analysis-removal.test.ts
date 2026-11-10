import fs from 'fs';
import path from 'path';

// 变更「影响分析」结构化表单的后端资源从未存在（读写 /changes/:id/impact 均 404），
// 2026-11-07 经确认整体摘除。真实影响能力保留在 CMDB 影响面板（/cmdb-impact）
// 与风险评估的 impactAnalysis 文本字段。此守卫防止死面被复制回来。

const srcRoot = path.resolve(__dirname, '../../../..');

describe('change impact-analysis dead surface removal', () => {
  it('change-api.ts must not call the never-registered /changes/:id/impact resource', () => {
    const source = fs.readFileSync(path.join(srcRoot, 'src/lib/api/change-api.ts'), 'utf8');
    expect(source).not.toContain('/impact`');
    expect(source).not.toContain('getImpactAnalysis');
    expect(source).not.toContain('updateImpactAnalysis');
    // 真实的 CMDB 影响摘要必须保留
    expect(source).toContain('/cmdb-impact`');
  });

  it('ChangeImpactAnalysis component must stay deleted', () => {
    expect(
      fs.existsSync(path.join(srcRoot, 'src/components/change/ChangeImpactAnalysis.tsx')),
    ).toBe(false);
  });

  it('ChangeDetail must not reference the removed tab or handlers', () => {
    const source = fs.readFileSync(
      path.join(srcRoot, 'src/components/change/ChangeDetail.tsx'),
      'utf8',
    );
    expect(source).not.toContain('ChangeImpactAnalysis');
    expect(source).not.toContain('loadImpactAnalysis');
    expect(source).not.toContain('handleSaveImpactAnalysis');
  });
});