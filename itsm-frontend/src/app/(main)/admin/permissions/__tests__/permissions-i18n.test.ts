/**
 * 回归测试：权限模块 i18n key 未翻译（浏览器测试 bug #28：11/23 模块显示原始 key）
 *
 * 修复前页面用 `moduleKey.toUpperCase()`（权限码，如 ticket → TICKET）查翻译，
 * 而字典 key 是常量名（TICKETS/KNOWLEDGE_BASE/WORKFLOWS），导致 11 个模块
 * 渲染出原始 key。本测试直接引用页面生产代码中的映射常量，验证：
 * 1. 每个权限码都有显式 i18n key 映射（不再依赖 toUpperCase 巧合）；
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
    // 修复前的错误形态：ticket → TICKET（字典中不存在）
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
