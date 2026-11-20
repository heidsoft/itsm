import fs from 'node:fs';
import path from 'node:path';

// 2026-11-20 决策：管理/配置类页面顶部的「XX 怎么用」UsageGuideCard、
// 「流程配置路线」WorkflowConfigGuide，以及「本页配置/本页展示…」定位说明
// Alert 全站统一下线。它们默认折叠但展开后每页 4–6 段操作说明，或常驻一行
// 冗长的职责解释，在演示环境里喧宾夺主；页面顶部一句话说明、页面内真实业务
// Alert 与表单字段提示才是就近可用的信息。此守卫防止死面被复制回来。

const srcRoot = path.join(process.cwd(), 'src');
const SELF = path.basename(__filename);

const walk = (dir: string): string[] =>
  fs.readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return walk(full);
    return /\.(ts|tsx)$/.test(entry.name) ? [full] : [];
  });

describe('用法引导卡片全站下线', () => {
  it('两个组件文件必须保持删除', () => {
    expect(fs.existsSync(path.join(srcRoot, 'components/common/UsageGuideCard.tsx'))).toBe(false);
    expect(fs.existsSync(path.join(srcRoot, 'components/workflow/WorkflowConfigGuide.tsx'))).toBe(false);
  });

  it('任何源码都不得再次 import 或渲染已下线的卡片', () => {
    const offenders = walk(srcRoot)
      .filter(file => path.basename(file) !== SELF)
      .filter(file => /UsageGuideCard|WorkflowConfigGuide|流程配置路线/.test(fs.readFileSync(file, 'utf8')))
      .map(file => path.relative(srcRoot, file));

    expect(offenders).toEqual([]);
  });

  it('已下线的卡片标题文案不得在页面里复活', () => {
    const titles = [
      '工作流管理怎么用',
      '系统配置怎么用',
      'CI 类型怎么用',
      'SLA 模板怎么用',
      '怎么理解和使用升级矩阵',
      '怎么配置、怎么测试',
      '启用后如何测试和使用连接器',
      '先设计流程并编排审批节点',
      '本页配置 BPMN 审批节点的审批人规则',
      '本页配置的是 SLA 阈值告警，不是升级矩阵',
      '本页展示后端当前生效的升级矩阵',
    ];
    // 2026-11-21 升级矩阵页已整体删除（并入「超时升级」双 Tab），路由目录不得复活
    expect(
      fs.existsSync(path.join(srcRoot, 'app/(main)/admin/escalation-matrices')),
    ).toBe(false);
    const offenders = walk(path.join(srcRoot, 'app'))
      .filter(file => path.basename(file) !== SELF)
      .filter(file => {
        const source = fs.readFileSync(file, 'utf8');
        return titles.some(title => source.includes(title));
      })
      .map(file => path.relative(srcRoot, file));

    expect(offenders).toEqual([]);
  });
});
