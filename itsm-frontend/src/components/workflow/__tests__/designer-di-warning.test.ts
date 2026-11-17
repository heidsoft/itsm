import fs from 'fs';
import path from 'path';

// 设计器 DI 警示文案的语义守卫。
//
// 背景：BPMN 定义缺 <bpmndi:BPMNDiagram> 时仍然是有效的业务定义。画布在
// import.parse.complete(1500) 上用 bpmnAutoLayout.ensureBpmnDI 注入自动布局，
// 导入后立即 saveXML 回填父组件，因此缺 DI 的流程可正常渲染、编辑，保存即固化坐标。
//
// 历史问题：WorkflowDesigner 加载时会弹 "请导入含 DI 的文件后再编辑"。这句文案
// 在自动布局落地后已经失真——它把"已被自动处理"的情况描述成"你没法编辑"。
// prod 实测 20 个流程定义里有 10 个缺 DI，等于一半流程每次打开都收到错误指引。
//
// 守卫口径：精确解析 i18n 条目值再断言，不使用"整文件包含某词"的模糊匹配
// （同一文件里其它条目/注释也可能含同样词，模糊匹配会放过真实回归）。

const srcRoot = path.resolve(__dirname, '../../../..');

const readSrc = (relative: string) =>
  fs.readFileSync(path.join(srcRoot, relative), 'utf8');

/**
 * 去掉注释后再断言。
 *
 * 守卫判定的是「代码里写了什么文案」，不是「注释里讨论过什么文案」。
 * 不剥注释会让"禁止出现某句话"的断言被同文件的说明性注释误伤（
 * 注释里引用旧文案来警示恰恰是正常写法），也会掩盖真正留在代码里的旧文案。
 */
function stripComments(source: string): string {
  return source
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/^\s*\/\/.*$/gm, '');
}

/**
 * 从 i18n 源文件里取出 `missingDIAutoLayout` 的全部条目值（中英两个 locale）。
 *
 * 条目写法为跨行字符串字面量：
 *   missingDIAutoLayout:
 *     '……',
 * 因此按 key 定位后，取其后第一个单引号字符串字面量。
 */
function extractEntryValues(source: string, key: string): string[] {
  const values: string[] = [];
  let from = 0;
  while (true) {
    const at = source.indexOf(`${key}:`, from);
    if (at === -1) break;
    const firstQuote = source.indexOf("'", at);
    if (firstQuote === -1) break;
    const secondQuote = source.indexOf("'", firstQuote + 1);
    if (secondQuote === -1) break;
    values.push(source.slice(firstQuote + 1, secondQuote));
    from = secondQuote + 1;
  }
  return values;
}

describe('workflow designer DI warning semantics', () => {
  const designerSource = readSrc('src/components/workflow/designer/WorkflowDesigner.tsx');
  // 断言"不该出现什么文案"时只看代码，注释里引用旧文案是合法写法
  const designerCode = stripComments(designerSource);
  const i18nSource = readSrc('src/lib/i18n/translations.ts');

  it('加载缺 DI 的流程时必须使用「已自动布局」口径的文案 key', () => {
    // 仍要提示，因为用户需要知道原 XML 没有坐标
    expect(designerSource).toContain('message.warning(t(\'workflow.designer.missingDIAutoLayout\'))');
    // 触发条件必须是"原始 XML 不含 DI"
    expect(designerSource).toContain("!xmlContent.includes('<bpmndi:BPMNDiagram')");
  });

  it('不得回归「请导入含 DI 的文件后再编辑」这类让人以为流程坏了的文案', () => {
    expect(designerCode).not.toContain('请导入含 DI 的文件后再编辑');
    expect(designerCode).not.toContain('已保留原始 XML；请导入');
    // 更不能说成"无法渲染 / 无法编辑"——自动布局之后这两种说法都不成立
    expect(designerCode).not.toContain('无法渲染');
    expect(designerCode).not.toContain('无法编辑');
  });

  it('中英文两个 locale 的文案都必须声明自动布局，且不得残留旧指引', () => {
    const values = extractEntryValues(i18nSource, 'missingDIAutoLayout');

    // 两个 locale 各一条
    expect(values).toHaveLength(2);

    const zh = values.find(v => /[一-龥]/.test(v));
    const en = values.find(v => !/[一-龥]/.test(v));
    expect(zh).toBeDefined();
    expect(en).toBeDefined();

    // 中文必须点明"自动布局"和"保存即固化"
    expect(zh).toContain('自动布局');
    expect(zh).toContain('保存');
    expect(zh).not.toContain('请导入');

    // 英文同理
    expect(en).toContain('automatic layout');
    expect(en).toContain('saving');
    expect(en).not.toContain('import a file');
  });

  it('自动注入 DI 的两处调用点必须保留，否则上面的文案就变成谎话', () => {
    const designer = readSrc('src/components/workflow/BPMNDesigner.tsx');
    // 导入前注入，而不是渲染后补救
    expect(designer).toContain("'import.parse.complete'");
    expect(designer).toContain('ensureBpmnDI');
    expect(designer).toContain('hasBpmnDiagram');

    // 预览面（Viewer 没有自动布局能力）同样必须注入
    const preview = readSrc('src/components/workflow/BPMNDiagramPreview.tsx');
    expect(preview).toContain('ensureBpmnDI');
    expect(preview).toContain('hasBpmnDiagram');
  });

  it('注入优先级必须高于 bpmn-js 默认的 1000，否则先渲染再补 DI 必然报错', () => {
    const designer = readSrc('src/components/workflow/BPMNDesigner.tsx');
    // 形如 modeler.on('import.parse.complete', 1500, ...)
    const registration = designer.match(
      /on\(\s*\n?\s*'import\.parse\.complete',\s*\n?\s*(\d+)/,
    );
    expect(registration).not.toBeNull();
    expect(Number(registration?.[1])).toBeGreaterThan(1000);
  });

  it('导入后必须 saveXML 回填，保证坐标能随保存持久化', () => {
    const designer = readSrc('src/components/workflow/BPMNDesigner.tsx');

    // 组件里有两处 importXML：初始化时导入首帧 XML（模型尚未建好），
    // 以及 [xml] effect 里的重新导入。归一化发生在后者——它的特征是
    // 紧接着 importTokenRef 的并发令牌校验。只锚定这一处，避免用
    // "全文含 saveXML" 这种任何一处都能满足的弱断言。
    const normalizeAt = designer.indexOf('let normalizedXml');
    expect(normalizeAt).toBeGreaterThan(-1);

    // 归一化前必须已校验过并发令牌（否则旧导入会把新 XML 覆盖掉）
    const before = designer.slice(Math.max(0, normalizeAt - 900), normalizeAt);
    expect(before).toContain('importTokenRef');

    // 归一化块内必须 saveXML 并把结果回填父组件：
    // 没有这一步，自动布局只存在于内存，保存出去的 XML 仍然没有 DI。
    const window = designer.slice(normalizeAt, normalizeAt + 900);
    expect(window).toContain('saveXML');
    expect(window).toContain('emittedXmlRef');
    expect(window).toContain('onChangeRef.current');
  });
});
