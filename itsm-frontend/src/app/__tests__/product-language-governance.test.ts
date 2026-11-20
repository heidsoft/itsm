/**
 * 产品语言与 UI 命名治理守卫（对应 RULES/UI-DEVELOPMENT.md §12）
 *
 * 只守「用户可见层」：菜单 Name、i18n 词条、页面 title/label/description 文案。
 * 技术标识（文件路径、后端 API 路径、类型名、代码注释）不在扫描范围——
 * 后端资源叫 escalation-matrices 是接口契约，不因业务名调整而改动。
 *
 * 反向验证：往 baseline.go 任一 Name 插入「智能中枢」，或往 translations.ts
 * 放回 escalationMatrices 词条块，本套件应转红。
 */
import fs from 'node:fs';
import path from 'node:path';

const frontendRoot = process.cwd();
const srcRoot = path.join(frontendRoot, 'src');
const baselineFile = path.join(frontendRoot, '..', 'itsm-backend', 'pkg', 'menubaseline', 'baseline.go');
const translationsFile = path.join(srcRoot, 'lib', 'i18n', 'translations.ts');

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
});
