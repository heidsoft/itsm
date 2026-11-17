/**
 * 租户类型映射回归锁（缺陷 O8）。
 *
 * 修复前有三处同源问题，全部被一个 `as Tenant[]` 强转掩盖：
 *   1. 映射表键写成 camelCase（`saasCustomer` / `mspProvider`），
 *      而后端 `/api/v1/tenants` 返回的 `type` 是 snake_case，恒不命中，
 *      标签一直回退显示原始枚举；
 *   2. 页面本地另立一份 `Tenant` 类型，声明了后端根本不存在的 `userCount`；
 *   3. 列表端点不做聚合，字段恒缺席，前端 `record.userCount || 0` 把
 *      「未计算」显示成「0 用户 0 工单」。
 *
 * 本文件锁 1 与 3 的前端口径（后端聚合由 Go 侧
 * `internal/services/tenant_usage_stats_test.go` 锁）。
 *
 * 关键断言是**值域本身**（snake_case），不是「渲染出来的标签好看」——
 * 后者在键写错时同样能通过（因为 resolveTenantType 会原样回退枚举）。
 */
import { translations } from '@/lib/i18n/translations';
import {
  TENANT_TYPE_VALUES,
  TENANT_TYPE_LABEL_KEYS,
  getTenantTypeConfig,
  resolveTenantType,
  type TenantTypeConfigMap,
} from '../tenant-display';

/** 用真实词表构造 t，避免 mock 一个「什么都返回 key」的假翻译掩盖缺词条。 */
const makeTranslator = (language: 'zh-CN' | 'en-US') => {
  const catalogs: Record<string, Record<string, unknown>> = translations;
  const dict = catalogs[language];
  return (key: string): string => {
    const segments = key.split('.');
    let cursor: unknown = dict;
    for (const segment of segments) {
      if (cursor === null || typeof cursor !== 'object') return key;
      cursor = (cursor as Record<string, unknown>)[segment];
    }
    return typeof cursor === 'string' ? cursor : key;
  };
};

describe('tenant-display 值域与标签', () => {
  describe('映射表键必须是后端 snake_case 值域', () => {
    it('包含后端返回的 snake_case 复合类型', () => {
      expect(TENANT_TYPE_VALUES).toEqual(
        expect.arrayContaining(['saas_customer', 'msp_provider', 'msp_customer'])
      );
    });

    // 反向锁：这三个 camelCase 写法正是修复前的错误键。它们必须不在值域里，
    // 否则说明有人又按 i18n 命名的直觉去改映射表了。
    it.each(['saasCustomer', 'mspProvider', 'mspCustomer'])(
      '%s 不得出现在值域里（camelCase 是修复前的错误写法）',
      wrongKey => {
        expect(TENANT_TYPE_VALUES).not.toContain(wrongKey);
      }
    );

    it('所有键都是 snake_case 或单段小写词（不含大写字母）', () => {
      for (const type of TENANT_TYPE_VALUES) {
        expect(type).not.toMatch(/[A-Z]/);
      }
    });
  });

  describe('getTenantTypeConfig 的键与定义表一一对应', () => {
    it.each(['zh-CN', 'en-US'] as const)('%s：配置键集合等于值域', language => {
      const config = getTenantTypeConfig(makeTranslator(language));
      expect(Object.keys(config).sort()).toEqual([...TENANT_TYPE_VALUES].sort());
    });

    // zh-CN 与 en-US 必须双侧同步：缺词条时 useI18n 的 fallback 会静默降级成
    // 中文（或直接返回 key），界面看着「能用」但其实是漏的。
    it.each(['zh-CN', 'en-US'] as const)('%s：每个 labelKey 都真实命中词条', language => {
      const t = makeTranslator(language);
      for (const labelKey of TENANT_TYPE_LABEL_KEYS) {
        expect(t(`tenants.typeLabels.${labelKey}`)).not.toBe(`tenants.typeLabels.${labelKey}`);
      }
    });

    it('中英两套词条值不同（防止 en-US 直接复制中文）', () => {
      const zh = getTenantTypeConfig(makeTranslator('zh-CN'));
      const en = getTenantTypeConfig(makeTranslator('en-US'));
      const sameValues = TENANT_TYPE_VALUES.filter(
        type => zh[type].label === en[type].label
      );
      // 允许极少数同形词（如 MSP），但不应全部相同
      expect(sameValues.length).toBeLessThan(TENANT_TYPE_VALUES.length);
    });
  });

  describe('resolveTenantType 命中与回退', () => {
    let config: TenantTypeConfigMap;

    beforeAll(() => {
      config = getTenantTypeConfig(makeTranslator('zh-CN'));
    });

    it('snake_case 类型命中 i18n 标签而不是原样枚举', () => {
      const resolved = resolveTenantType(config, 'msp_provider');
      expect(resolved.label).toBe('MSP服务商');
      expect(resolved.label).not.toBe('msp_provider');
      expect(resolved.color).toBe('gold');
    });

    it.each(['standard', 'internal', 'saas_customer', 'msp_customer', 'msp', 'customer'])(
      '%s 均命中词条（不透传原始枚举）',
      type => {
        expect(resolveTenantType(config, type).label).not.toBe(type);
      }
    );

    // api-config 的 Tenant.type 还含三个历史兼容值，后端不会返回。
    // 命中不到时按「原样显示 + 默认灰色」处理 —— 不静默归类成某一类，
    // 否则脏数据会看起来像正常数据。
    it.each(['trial', 'professional', 'enterprise'])(
      '历史兼容值 %s 回退为原样枚举 + 默认灰',
      legacyType => {
        expect(resolveTenantType(config, legacyType)).toEqual({
          color: 'default',
          label: legacyType,
        });
      }
    );

    it('空字符串也不触发静默归类', () => {
      expect(resolveTenantType(config, '')).toEqual({ color: 'default', label: '' });
    });

    // 防止有人把 hasOwnProperty 改成宽松索引：原型链上的键不得被当成合法类型。
    it('不把原型链属性误判为命中', () => {
      expect(resolveTenantType(config, 'constructor')).toEqual({
        color: 'default',
        label: 'constructor',
      });
    });
  });
});
