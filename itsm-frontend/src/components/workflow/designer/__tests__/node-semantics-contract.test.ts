import fs from 'fs';
import path from 'path';

// 节点语义修复的源码契约守卫：A 类误导文案 + B1 ServiceTask handler 名。
// 沿用项目已有的文本契约惯例（inspector-label-contract.test.ts 等）。
// B1 额外做跨文件断言：前端下拉写入的 handler 名必须命中后端真实注册名，
// 否则流程执行时 serviceTaskReference 拿它寻址会 100% dead_letter。

const frontendRoot = path.resolve(__dirname, '../../../../..'); // -> itsm-frontend/
const repoRoot = path.resolve(__dirname, '../../../../../..'); // -> itsm/
const readFrontend = (relative: string) => fs.readFileSync(path.join(frontendRoot, relative), 'utf8');
const readRepo = (relative: string) => fs.readFileSync(path.join(repoRoot, relative), 'utf8');

const inspector = readFrontend('src/components/workflow/designer/WorkflowNodeInspector.tsx');
const bpmnDesigner = readFrontend('src/components/workflow/BPMNDesigner.tsx');
// 后端 handler 源码：注册名在这里定义，作为「真实存在」的真相源。
const webhookHandler = readRepo('itsm-backend/service/bpmn/webhook_handler.go');
const genericHandler = readRepo('itsm-backend/service/bpmn/generic_handler.go');

const stripComments = (s: string): string =>
  s.replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/.*$/gm, '');
const inspectorBody = stripComments(inspector);
const designerBody = stripComments(bpmnDesigner);

describe('A 类：误导性文案', () => {
  it('网关默认分支 placeholder 指明填连线 ID，不再说节点 ID', () => {
    // 写入逻辑 normalizeDefaultFlow 会把值 resolve 成 sequenceFlow，填节点 ID 会保存失败。
    const hits = inspectorBody.match(/placeholder="输入默认分支连线的 ID"/g);
    expect(hits).toHaveLength(2); // 排他 + 包容两个网关
    expect(inspectorBody).not.toContain('placeholder="输入默认流转的节点ID"');
  });

  it('开始事件定时开关文案不再宣称「按周期」，与默认 PT1H 一次性语义一致', () => {
    expect(inspectorBody).toContain('开启后将在设定时间自动启动流程一次');
    expect(inspectorBody).not.toContain('开启后按周期自动启动流程');
  });

  it('审批组说明不再写死「任一成员」，通过规则交由审批模式决定', () => {
    expect(inspectorBody).toContain('通过规则（任一/全部/阈值/顺序）由上方审批模式决定');
    expect(inspectorBody).not.toContain('审批组中任一成员审批即视为该节点通过');
  });
});

describe('B1：ServiceTask handler 名对齐后端注册名', () => {
  it('下拉写入的 webhook/generic_handler 旧死信值已被真实注册名替换', () => {
    expect(inspectorBody).toContain("value: 'webhook_task'");
    expect(inspectorBody).toContain("value: 'generic_task'");
    // 通知渠道里的 webhook 是另一种语义，仍应保留，不能一刀切删除。
    expect(inspectorBody).toContain("{ label: 'Webhook', value: 'webhook' }");
  });

  it('前端下拉的 handler 名确实存在于后端注册（消除 dead_letter）', () => {
    // serviceTaskReference 把 implementation 值当 handler ID 去注册表查；
    // webhook_handler.go / generic_handler.go 的 GetTaskType 返回名即注册键。
    expect(webhookHandler).toContain('return "webhook_task"');
    expect(genericHandler).toContain('return "generic_task"');
  });

  it('本地发布校验的 supported 集合包含下拉真实写入的两个新名', () => {
    const supportedLine = designerBody.split('\n').find(l => l.includes('const supported = new Set'));
    expect(supportedLine).toBeDefined();
    expect(supportedLine).toContain("'webhook_task'");
    expect(supportedLine).toContain("'generic_task'");
    // 历史值保留以兼容已有流程，不得被误删。
    expect(supportedLine).toContain("'webhook'");
    expect(supportedLine).toContain("'generic_handler'");
  });

  it('抄送任务识别仍依赖 cc_handler，未被 handler 名改动误伤', () => {
    expect(inspectorBody).toContain("bo.implementation as string) === 'cc_handler'");
  });
});
