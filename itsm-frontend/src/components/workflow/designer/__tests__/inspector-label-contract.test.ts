import fs from 'fs';
import path from 'path';

// 节点属性面板文案契约守卫：字段标签不得再暴露 BPMN 引擎技术名，
// 人员字段须用一致的业务白话（标签与说明口径统一）。
// 沿用项目已有的文本契约惯例（designer-di-warning.test.ts /
// workflow-designer-fixes-contract.test.ts），断言源码关键不变量。

const srcRoot = path.resolve(__dirname, '../../../../..');
const inspector = fs.readFileSync(
  path.join(srcRoot, 'src/components/workflow/designer/WorkflowNodeInspector.tsx'),
  'utf8'
);

//  assertions on source text only; JSX 注释里引用旧写法会自匹配，先剥注释。
const stripComments = (source: string): string =>
  source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/.*$/gm, '');

const body = stripComments(inspector);

// 曾被拼在中文标签后面的 13 个引擎字段名（含重复出现的 messageRef）。
const removedTechNames = [
  'assignee',
  'candidateUsers',
  'candidateGroups',
  'formKey',
  'priority',
  'dueDate',
  'followUpDate',
  'ruleRef',
  'messageRef',
  'operation',
  'signalRef',
  'errorCode',
  'escalationCode',
];

describe('节点属性面板字段标签文案', () => {
  it('字段名后不得再拼 BPMN 引擎技术名（中文 + 空格 + 括号英文名）', () => {
    // 匹配形如「受理人 (assignee)」「表单键 (formKey)」的中英混排标签。
    const mixed = body.match(/[\u4e00-\u9fa5]\s*\([a-zA-Z][a-zA-Z0-9]*\)/g);
    expect(mixed).toBeNull();
  });

  it('13 个引擎字段名不再出现在任何字段标题里', () => {
    for (const tech of removedTechNames) {
      // 只约束「中文标签 + 括号」这一展示形态，不禁止变量/属性名本身。
      expect(body).not.toMatch(
        new RegExp(`[\\u4e00-\\u9fa5]\\s*\\(${tech}\\)`)
      );
    }
  });

  it('三个人员字段使用一致的业务白话，且标签与说明口径不矛盾', () => {
    expect(body).toContain('指定处理人');
    expect(body).toContain('可选处理人');
    expect(body).toContain('审批组');
    // 旧的 CAD 术语不得残留为字段标题。
    expect(body).not.toMatch(/受理人\s*\(/);
    expect(body).not.toMatch(/候选人\s*\(/);
    expect(body).not.toMatch(/候选组\s*\(/);
    // 候选人字段的说明不再是「候选人」，审批组 placeholder 与标签一致。
    expect(body).toContain('这些人中任意一位都能领取并处理该任务');
    expect(body).toContain('placeholder="选择可选处理人（多选）"');
    expect(body).toContain('placeholder="选择审批组（多选）"');
  });

  it('人员字段的写入属性保持 BPMN 契约不变（文案改动不影响数据流）', () => {
    // 标签改名只是展示层；onChange 写入的仍是 assignee/candidateUsers/candidateGroups。
    expect(body).toContain('apply({ assignee:');
    expect(body).toContain('apply({ candidateUsers:');
    expect(body).toContain('apply({ candidateGroups:');
  });
});
