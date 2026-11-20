/**
 * 产品语言与 UI 命名治理守卫（对应 RULES/UI-DEVELOPMENT.md §12）
 *
 * 只守「用户可见层」：菜单 Name、i18n 词条、页面 title/label/description 文案，
 * 以及导航/面包屑/顶栏入口的可见性口径。技术标识（文件路径、后端 API 路径、
 * 类型名、代码注释）不在扫描范围——后端资源叫 escalation-matrices 是接口契约，
 * 不因业务名调整而改动。
 *
 * 反向验证：往 baseline.go 任一 Name 插入「智能中枢」，或放回「AI 中心」一级行，
 * 或把顶栏按钮改回只按权限码渲染，本套件应转红。
 */
import fs from 'node:fs';
import path from 'node:path';

const frontendRoot = process.cwd();
const srcRoot = path.join(frontendRoot, 'src');
const backendRoot = path.join(frontendRoot, '..', 'itsm-backend');
const baselineFile = path.join(backendRoot, 'pkg', 'menubaseline', 'baseline.go');
const translationsFile = path.join(srcRoot, 'lib', 'i18n', 'translations.ts');
const routeConfigFile = path.join(srcRoot, 'lib', 'router', 'route-config.ts');
const breadcrumbFile = path.join(srcRoot, 'components', 'layout', 'header', 'breadcrumb-utils.ts');
const headerFile = path.join(srcRoot, 'components', 'layout', 'header', 'Header.tsx');
const sidebarFile = path.join(srcRoot, 'components', 'layout', 'sidebar', 'Sidebar.tsx');
const menuConfigFile = path.join(srcRoot, 'components', 'layout', 'sidebar', 'menu-config.ts');
const visibilityFile = path.join(srcRoot, 'components', 'layout', 'header', 'ai-assistant-visibility.ts');
const migrationUp = path.join(backendRoot, 'migrations', '20261121_retire_ai_center_menu.sql');
const migrationDown = path.join(backendRoot, 'migrations', '20261121_retire_ai_center_menu_down.sql');

/** §12.3：禁止用这些词包装普通业务功能，也禁止作为一级导航名 */
const BANNED = [
  '智能中枢', '智慧运营', 'AI 工作台', '智能洞察', '智能决策中心', '全域智能',
  '智能生态', '数字员工中心', '全链路智能', '统一智能空间', '超级 Agent 中心',
  '赋能', '闭环', '全域', '生态', '中枢', '智慧',
];

/** §12.3：普通功能不得机械挂这些前缀；具体 AI 能力名（AI 工单摘要等）是合规的 */
const MECHANICAL_PREFIXES = ['智能工单', '智慧资产', '智能资产', '智慧运营', '自动驾驶'];

const readBaseline = () => fs.readFileSync(baselineFile, 'utf8');
const menuNames = () => [...readBaseline().matchAll(/Name:\s*"([^"]+)"/g)].map(m => m[1]);
// 只取菜单自身 Path；`, Path:` 前缀用于避开 ParentPath
const menuPaths = () => [...readBaseline().matchAll(/,\s*Path:\s*"([^"]+)"/g)].map(m => m[1]);

interface MenuRow {
  name: string;
  path: string;
  parentPath: string;
}

/** 逐行解析基线里的 MenuSpec，ParentPath 缺省（顶级）视为空串 */
const menuRows = (): MenuRow[] =>
  readBaseline()
    .split('\n')
    .filter(line => line.includes('Name: "') && line.includes('Path: "'))
    .map(line => ({
      name: /Name:\s*"([^"]+)"/.exec(line)?.[1] ?? '',
      path: /,\s*Path:\s*"([^"]+)"/.exec(line)?.[1] ?? '',
      parentPath: /ParentPath:\s*"([^"]*)"/.exec(line)?.[1] ?? '',
    }));

/** 整行注释不参与「旧名复活」判定，否则解释性注释会自匹配 */
const stripComments = (text: string): string =>
  text
    .split('\n')
    .filter(line => !/^\s*(\/\/|\*|\/\*)/.test(line))
    .join('\n');

const walkTsx = (dir: string): string[] =>
  fs.readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) return walkTsx(full);
    return entry.name.endsWith('.tsx') ? [full] : [];
  });

describe('产品语言与 UI 命名治理', () => {
  it('菜单名称不得使用 AI 化 / 装饰性命名词', () => {
    const offenders = menuNames().filter(name =>
      BANNED.some(word => name.includes(word)) ||
      MECHANICAL_PREFIXES.some(prefix => name.includes(prefix)),
    );
    expect(offenders).toEqual([]);
  });

  it('不得出现重复菜单入口（不同名称指向同一 path）', () => {
    const paths = menuPaths();
    const duplicates = [...new Set(paths.filter((p, i) => paths.indexOf(p) !== i))];
    expect(duplicates).toEqual([]);
  });

  it('同一业务概念单一叫法：/admin/escalation-rules 的菜单名固定为「超时升级」', () => {
    const line = readBaseline().split('\n').find(l => l.includes('"/admin/escalation-rules"'));
    expect(line).toBeDefined();
    expect(line).toContain('Name: "超时升级"');
  });

  it('已下线旧名「升级矩阵」不得在菜单与 i18n 词条里复活', () => {
    expect(menuNames().some(name => name.includes('升级矩阵'))).toBe(false);
    // 中英两份 escalationMatrices 词条块曾是 0 消费的死词条，且英文块中英夹杂
    expect(fs.readFileSync(translationsFile, 'utf8')).not.toContain('escalationMatrices:');
  });

  it('已下线旧名不得作为页面用户可见文案（title / label / description / placeholder）复活', () => {
    const self = path.basename(__filename);
    const visibleText = /(?:title|label|description|placeholder)\s*[:=]\s*(['"`])([^'"`]*升级矩阵[^'"`]*)\1/g;
    const offenders: string[] = [];
    for (const file of walkTsx(srcRoot)) {
      if (path.basename(file) === self || file.includes(`${path.sep}__tests__${path.sep}`)) continue;
      const text = fs.readFileSync(file, 'utf8');
      for (const match of text.matchAll(visibleText)) {
        offenders.push(`${path.relative(srcRoot, file)}: ${match[2]}`);
      }
    }
    expect(offenders).toEqual([]);
  });

  // ===== 2026-11-21 方案 A：「AI 中心」一级下线 + 能力按业务归位 =====

  it('一级导航不得按技术能力聚合命名（§12.3 Business / Function-over-Technology First）', () => {
    const offenders = menuRows()
      .filter(row => row.parentPath === '')
      .filter(row =>
        /^(AI|智能|智慧|Agent|Copilot|GPT)/.test(row.name) ||
        ['AI 中心', '智能助手', 'AI 助手', '智能中枢'].some(word => row.name.includes(word))
      )
      .map(row => row.name);
    expect(offenders).toEqual([]);
  });

  it('/ai/chat 不得再作为菜单 Path：一级技术聚合入口已下线，问答只留顶栏全局入口', () => {
    expect(menuRows().filter(row => row.path === '/ai/chat').map(row => row.name)).toEqual([]);
  });

  it('三个 AI 能力页按业务归位且名称单一，旧名不得复活', () => {
    const rows = menuRows();
    const want: Record<string, { name: string; parent: string }> = {
      '/tickets/ai-create': { name: 'AI 创建工单', parent: '/tickets' },
      '/ai/approval': { name: '工具调用审批', parent: '/approvals' },
      '/ai/audit': { name: 'AI 效果评估', parent: '/audit-logs' },
    };
    for (const [p, expected] of Object.entries(want)) {
      const row = rows.find(r => r.path === p);
      if (!row) throw new Error(`菜单基线缺少归位后的 ${p}`);
      expect(row.name).toBe(expected.name);
      expect(row.parentPath).toBe(expected.parent);
    }
    const oldNames = ['AI 中心', 'AI 评估与审计', 'AI 审批', '智能助手', 'AI 问答'];
    expect(menuNames().filter(name => oldNames.includes(name))).toEqual([]);
  });

  it('旧名不得在路由死配置 / 面包屑 / 顶栏按钮文案里复活', () => {
    const banned = ['智能助手', 'AI 问答', 'AI 中心', 'AI 评估与审计', 'AI 审批', 'AI助手', 'AI对话'];
    for (const file of [routeConfigFile, breadcrumbFile, headerFile]) {
      const text = stripComments(fs.readFileSync(file, 'utf8'));
      expect({ file: path.relative(srcRoot, file), hits: banned.filter(w => text.includes(w)) })
        .toEqual({ file: path.relative(srcRoot, file), hits: [] });
    }
    // 顶栏按钮统一叫法
    expect(fs.readFileSync(headerFile, 'utf8')).toContain('title="AI 助手"');
    // 段兜底叫法与菜单名一致
    const breadcrumb = fs.readFileSync(breadcrumbFile, 'utf8');
    expect(breadcrumb).toMatch(/ai:\s*'AI 助手'/);
    expect(breadcrumb).toMatch(/chat:\s*'AI 助手'/);
    expect(breadcrumb).toMatch(/'ai-create':\s*'AI 创建工单'/);
  });

  it('业务归位后的面包屑不暴露已下线的 /ai 技术聚合层', () => {
    const breadcrumb = fs.readFileSync(breadcrumbFile, 'utf8');
    expect(breadcrumb).toContain("'/ai/chat': [{ title: 'AI 助手' }]");
    expect(breadcrumb).toContain("'/ai/audit': [");
    expect(breadcrumb).toContain('AI 效果评估');
    expect(breadcrumb).toContain('工具调用审批');
  });

  it('顶栏 AI 助手按钮接入 ai capability 门控，与侧栏同源（统一可见性）', () => {
    const header = fs.readFileSync(headerFile, 'utf8');
    expect(header).toContain('shouldShowAIAssistant');
    // 渲染条件必须依赖判定结果，不能退回「只按权限码」的裸条件
    expect(header).toMatch(/\{showAIAssistant\s*\?/);
    expect(stripComments(header)).not.toContain('hasPermission(PERM.ai.read) || isAdmin() ? (');

    const visibility = fs.readFileSync(visibilityFile, 'utf8');
    expect(visibility).toContain('isCapabilityVisible');
    expect(visibility).toContain('capabilityForPath(AI_ASSISTANT_PATH)');

    // 侧栏复用同一个判据函数，不另起一套口径
    expect(stripComments(fs.readFileSync(sidebarFile, 'utf8')))
      .toContain('isCapabilityVisible(capability)');
  });

  it('AI 能力入口的能力门控规则覆盖顶栏与三个归位页面', () => {
    const rules = [...fs.readFileSync(menuConfigFile, 'utf8').matchAll(/\[\s*'([^']+)'\s*,\s*'([^']+)'\s*\]/g)]
      .map(m => ({ pathPrefix: m[1], key: m[2] }));
    const aiPrefixes = rules.filter(r => r.key === 'ai').map(r => r.pathPrefix);
    expect(aiPrefixes).toEqual(expect.arrayContaining(['/ai', '/tickets/ai-create']));
  });

  it('存量迁移 up/down 成对，停用 /ai/chat 旧一级行', () => {
    expect(fs.existsSync(migrationUp)).toBe(true);
    expect(fs.existsSync(migrationDown)).toBe(true);
    const up = fs.readFileSync(migrationUp, 'utf8');
    expect(up).toContain("path = '/ai/chat'");
    expect(up).toContain('is_enabled = false');
    expect(fs.readFileSync(migrationDown, 'utf8')).toContain('is_enabled = true');
  });
});
