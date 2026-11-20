/**
 * 权限模块的 i18n key 契约。
 *
 * 页面若用 `moduleKey.toUpperCase()`（权限码，如 ticket → TICKET）查翻译，
 * 而字典 key 是常量名（TICKETS/KNOWLEDGE_BASE/WORKFLOWS），模块就会直接
 * 渲染出原始 key。本测试引用页面生产代码里的映射常量，验证：
 * 1. 每个权限码都有显式 i18n key 映射（不依赖 toUpperCase 巧合）；
 * 2. zh-CN / en-US 字典中对应 label 与 description 完整存在。
 */
import { PERMISSION_MODULES, MODULE_I18N_KEYS } from '../page';
import { translations } from '@/lib/i18n/translations';

type Dict = Record<string, unknown>;

function lookup(dict: Dict, keyPath: string): unknown {
  return keyPath.split('.').reduce<unknown>((acc, k) => {
    if (acc && typeof acc === 'object') return (acc as Dict)[k];
    return undefined;
  }, dict);
}

describe('permissions 模块 i18n 契约', () => {
  const moduleCodes = Object.values(PERMISSION_MODULES);

  it('模块清单应为 23 个权限码', () => {
    expect(moduleCodes).toHaveLength(23);
  });

  it.each(moduleCodes)('权限码 %s 必须有显式 i18n key 映射', (code) => {
    const i18nKey = MODULE_I18N_KEYS[code];
    expect(typeof i18nKey).toBe('string');
    expect(i18nKey).toBeTruthy();
  });

  it.each(moduleCodes)('权限码 %s 的翻译在 zh-CN/en-US 均存在', (code) => {
    const i18nKey = MODULE_I18N_KEYS[code];
    for (const locale of ['zh-CN', 'en-US'] as const) {
      const dict = translations[locale] as unknown as Dict;
      const label = lookup(dict, `permissions.modules.${i18nKey}.label`);
      const description = lookup(dict, `permissions.modules.${i18nKey}.description`);
      expect(typeof label).toBe('string');
      expect((label as string).length).toBeGreaterThan(0);
      expect(typeof description).toBe('string');
      expect((description as string).length).toBeGreaterThan(0);
    }
  });

  it('页面渲染路径解析出的 key 不得回退为原始权限码大写', () => {
    // 错误形态：ticket → TICKET，字典里没有这个 key
    const dict = translations['zh-CN'] as unknown as Dict;
    const brokenKeys = moduleCodes
      .map((code) => code.toUpperCase())
      .filter((upper) => lookup(dict, `permissions.modules.${upper}.label`) === undefined);
    // 若页面仍用 toUpperCase()，这组 key 将直接渲染为原始文本
    expect(brokenKeys.length).toBeGreaterThan(0); // 证明错误形态确实存在过
    // 而生产映射必须全部命中
    for (const code of moduleCodes) {
      const key = MODULE_I18N_KEYS[code];
      expect(lookup(dict, `permissions.modules.${key}.label`)).toBeDefined();
    }
  });
});

/**
 * 角色权限页 / 权限矩阵页共用的展示名映射完整性。
 * 根因：后端权限目录有 58 resource / 24 action，而前端 roles.resourceLabels、
 * roles.actionLabels 旧版只覆盖 23/12 个，未命中的直接暴露英文 code（ticket_tag、
 * escalate 等），造成「有的中文有的英文」。本契约读后端 catalog 权威源，确保
 * 每一个 resource/action 在中英字典里都有翻译。
 */
describe('权限展示名 i18n 覆盖全部后端权限码', () => {
  const fs = require('fs') as typeof import('fs');
  const path = require('path') as typeof import('path');
  // 从 __tests__ 向上找到同时含 itsm-backend 与 itsm-frontend 的仓库根。
  let repoRoot = __dirname;
  for (let i = 0; i < 12; i++) {
    if (
      fs.existsSync(path.join(repoRoot, 'itsm-backend')) &&
      fs.existsSync(path.join(repoRoot, 'itsm-frontend'))
    ) {
      break;
    }
    repoRoot = path.dirname(repoRoot);
  }
  const catalogSrc = fs.readFileSync(
    path.join(repoRoot, 'itsm-backend/internal/authz/catalog.go'),
    'utf-8'
  );
  const entries = [...catalogSrc.matchAll(/\{"([^"]+)",\s*"[^"]+",\s*"([^"]+)",\s*"([^"]+)",\s*"[^"]*"\}/g)].map(
    (m) => ({ code: m[1], resource: m[2], action: m[3] })
  );

  it('能从后端权威源解析出权限码（防止正则失效导致契约空跑）', () => {
    expect(entries.length).toBeGreaterThan(100);
  });

  const resources = [...new Set(entries.map((e) => e.resource))];
  const actions = [...new Set(entries.map((e) => e.action))];

  for (const locale of ['zh-CN', 'en-US'] as const) {
    const dict = translations[locale] as unknown as Dict;
    it(`[${locale}] 全部 ${resources.length} 个 resource 都有中文/英文资源名`, () => {
      const missing = resources.filter((r) => {
        const v = lookup(dict, `roles.resourceLabels.${r}`);
        return typeof v !== 'string' || v.length === 0;
      });
      expect(missing).toEqual([]);
    });

    it(`[${locale}] 全部 ${actions.length} 个 action 都有可读操作名`, () => {
      const missing = actions.filter((a) => {
        const v = lookup(dict, `roles.actionLabels.${a}`);
        return typeof v !== 'string' || v.length === 0;
      });
      expect(missing).toEqual([]);
    });
  }
});
