import fs from 'fs';
import path from 'path';

// 流程设计器三块交互缺陷的源码契约守卫（版本控制 / 节点属性 / 流程校验）。
//
// 这些都是纯逻辑/状态管理修复，无法在 jsdom 里稳定挂载 bpmn-js 验证，
// 因此沿用项目已有的文本契约惯例（designer-di-warning.test.ts 等）：
// 直接断言源码里的关键不变量，配合 workflowValidation.test.ts 的纯函数单测。

const srcRoot = path.resolve(__dirname, '../../../../..');
const readSrc = (relative: string) => fs.readFileSync(path.join(srcRoot, relative), 'utf8');

// 「不得包含旧代码」类断言必须先剥注释：注释里引用旧写法（如 `return []`、
// `slaConfig && workflow`）来警示恰恰是正常写法，不剥会让断言自匹配假红。
const stripComments = (source: string): string =>
  source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/.*$/gm, '');

const designer = readSrc('src/components/workflow/designer/WorkflowDesigner.tsx');
const bpmnDesigner = readSrc('src/components/workflow/BPMNDesigner.tsx');
const versionsPage = readSrc('src/app/(main)/workflow/versions/page.tsx');
const i18n = readSrc('src/lib/i18n/translations.ts');

describe('流程校验门禁 fail-closed', () => {
  it('设计器未就绪 / 本地校验抛错时返回阻断 issue，绝不 return 空数组', () => {
    // 两条失败路径都必须构造 validationUnavailableIssue，而不是裸 []
    const unavailableBuilds = designer.match(/validationUnavailableIssue\(/g) ?? [];
    expect(unavailableBuilds.length).toBeGreaterThanOrEqual(3);

    // 关键：validateWorkflow 函数体内不得再出现裸 `return [];`
    const start = designer.indexOf('const validateWorkflow');
    const end = designer.indexOf('// 保存工作流', start);
    expect(start).toBeGreaterThan(-1);
    expect(end).toBeGreaterThan(start);
    const body = stripComments(designer.slice(start, end));
    // 唯一的 `return []` 只能出现在「校验真正跑完且无任何 issue」的通过分支；
    // 失败/不可用分支必须返回 validationUnavailableIssue 构造的阻断数组。
    const bareReturns = body.match(/return\s+\[\s*\]\s*;/g) ?? [];
    expect(bareReturns).toHaveLength(1);
    expect(body).toContain('return blocked;');
  });

  it('后端 lint 不可用升级为阻断性 error，不再标记 severity info', () => {
    const start = designer.indexOf('const validateWorkflow');
    const body = designer.slice(start, designer.indexOf('// 保存工作流', start));
    expect(body).toContain('validationUnavailableIssue(t(\'workflow.designer.validationBackendUnavailableBlocked\'))');
    expect(body).not.toContain("severity: 'info'");
  });

  it('部署与保存部署入口在 await 校验前用 ref 同步置位重入守卫', () => {
    expect(designer).toContain('const deployInFlightRef = useRef(false)');
    // 两个入口都要先检查再置位
    const guardHits = designer.match(/if \(deployInFlightRef\.current\) return;/g) ?? [];
    expect(guardHits.length).toBeGreaterThanOrEqual(2);
    const setHits = designer.match(/deployInFlightRef\.current = true;/g) ?? [];
    expect(setHits.length).toBeGreaterThanOrEqual(2);
    // finally 必须释放，否则一次失败后永久锁死
    const clearHits = designer.match(/deployInFlightRef\.current = false;/g) ?? [];
    expect(clearHits.length).toBeGreaterThanOrEqual(2);
  });

  it('部署/发布在门禁拦截时区分「校验不可用」与「流程有错」', () => {
    const unavailableHits = designer.match(/isValidationUnavailable\(issues\)/g) ?? [];
    expect(unavailableHits.length).toBeGreaterThanOrEqual(2);
  });

  it('中英文都声明了校验不可用的阻断文案', () => {
    const zh = (i18n.match(/validationUnavailable:/g) ?? []).length;
    expect(zh).toBeGreaterThanOrEqual(2); // 中英两个 locale
  });
});

describe('连通性校验接入生产', () => {
  it('BPMNDesigner 生产校验调用 buildConnectivityIssues', () => {
    expect(bpmnDesigner).toContain("from './designer/localValidation'");
    // 必须把 elementRegistry 元素映射成 ConnectivityElement 再传入
    expect(bpmnDesigner).toContain('const connectivityElements: ConnectivityElement[]');
    // 关键：必须把结果合并进 errors，否则又变成只 import 不生效的死代码。
    expect(bpmnDesigner).toContain('errors.push(...buildConnectivityIssues(connectivityElements, t))');
  });

  it('连通性规则的 5 个文案 key 中英文都已声明', () => {
    const keys = [
      'startEventNoOutgoing',
      'endEventNoIncoming',
      'flowMissingEndpoint',
      'nodeUnreachable',
      'genericTaskUnsupported',
    ];
    for (const key of keys) {
      expect((i18n.match(new RegExp(`${key}:`, 'g')) ?? []).length).toBeGreaterThanOrEqual(2);
    }
  });
});

describe('版本控制竞态与错误态', () => {
  it('版本页用 seq 守卫丢弃过期响应，并有独立 loadError 态', () => {
    expect(versionsPage).toContain('const loadSeqRef = useRef(0)');
    expect(versionsPage).toContain('const [listError, setListError] = useState(false)');
    // catch 里必须置 listError，渲染里必须消费
    expect(versionsPage).toContain('setListError(true)');
    expect(versionsPage).toContain("listError ? 'error'");
  });

  it('designer 三条加载链路各有独立 generation ref，互不干扰', () => {
    expect(designer).toContain('const workflowSeqRef = useRef(0)');
    expect(designer).toContain('const workflowVersionsSeqRef = useRef(0)');
    expect(designer).toContain('const workflowConfigSeqRef = useRef(0)');
  });

  it('版本加载失败清空列表，不保留上一个流程的版本', () => {
    const start = designer.indexOf('const loadWorkflowVersions');
    const body = designer.slice(start, designer.indexOf('// 加载工作流配置', start));
    expect(body).toContain('setWorkflowVersions([])');
  });
});

describe('节点属性与 SLA 回填', () => {
  it('节点面板按 selection.id 加 key，切节点强制 remount 清防抖态', () => {
    const inspectorAt = designer.indexOf('<WorkflowNodeInspector');
    const window = designer.slice(inspectorAt, inspectorAt + 400);
    expect(window).toContain('key={selectedNode?.id');
  });

  it('SLA 回填用函数式更新，消除闭包内 workflow 恒 null 的 stale closure', () => {
    const start = designer.indexOf('const loadWorkflowConfig');
    const body = designer.slice(start, designer.indexOf('// 初始化加载', start));
    expect(body).toContain('setWorkflow(prev =>');
    // 不得再出现 `slaConfig && workflow` 这种恒假条件（剥注释避免自匹配）
    expect(stripComments(body)).not.toContain('slaConfig && workflow');
  });
});
