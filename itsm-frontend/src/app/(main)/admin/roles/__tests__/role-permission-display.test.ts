/**
 * 角色权限展示回归锁。
 *
 * 修复前三个现象：
 * 1. 分组标题取「该组第一条权限的 name」（如「查看工单」），与组内 action 语义错位；
 * 2. i18n 未命中时直接暴露英文 code（ticket_tag / escalate 等）；
 * 3. 50+ 资源卡片全展开且无高度约束，把弹窗拉得很长。
 *
 * 本测试直接对从 page.tsx 导出的纯函数断言，并用真实中英字典构造 t()，
 * 确保「展示名口径」与「i18n 覆盖」两件事都被锁住。
 */
import { translations } from '@/lib/i18n/translations';
import { actionDisplay, derivePermissionModules } from '../page';
import type { PermissionCatalogItem } from '@/lib/api/api-config';

type Dict = Record<string, unknown>;

function makeT(locale: 'zh-CN' | 'en-US') {
  const dict = translations[locale] as unknown as Dict;
  return (key: string): string => {
    const value = key.split('.').reduce<unknown>((acc, k) => {
      if (acc && typeof acc === 'object') return (acc as Dict)[k];
      return undefined;
    }, dict);
    return typeof value === 'string' ? value : key;
  };
}

function item(code: string, name: string): PermissionCatalogItem {
  const [resource, action] = code.split(':');
  return { id: 1, code, name, resource, action } as PermissionCatalogItem;
}

describe('derivePermissionModules 权限矩阵派生', () => {
  const t = makeT('zh-CN');

  it('分组标题取资源名，不取首条权限的动作名', () => {
    const catalog = [
      item('ticket:read', '查看工单'),
      item('ticket:write', '管理工单'),
      item('incident:read', '查看事件'),
    ];
    const modules = derivePermissionModules(catalog, t);
    expect(modules).toHaveLength(2);
    // ticket 的资源名是「工单管理」，不是首条 name「查看工单」
    expect(modules.find((m) => m.resource === 'ticket')?.label).toBe('工单管理');
    expect(modules.find((m) => m.resource === 'incident')?.label).toBe('事件管理');
  });

  it('i18n 未收录的资源兜底到后端中文 name 去动作前缀，绝不暴露英文 code', () => {
    // escal 类资源故意用一个字典里没有的 code，验证「中文 name 去前缀」兜底
    const catalog = [item('custom_xyz:read', '查看自定义模块')];
    const modules = derivePermissionModules(catalog, t);
    expect(modules[0].label).toBe('自定义模块');
    expect(modules[0].label).not.toBe('custom_xyz');
  });

  it('同资源多 action 去重且按出现顺序保留', () => {
    const catalog = [
      item('ticket:read', '查看工单'),
      item('ticket:write', '管理工单'),
      item('ticket:read', '查看工单'),
      item('ticket:delete', '删除工单'),
    ];
    const modules = derivePermissionModules(catalog, t);
    expect(modules[0].actions).toEqual(['read', 'write', 'delete']);
  });

  it('空 catalog 返回空矩阵，不抛错', () => {
    expect(derivePermissionModules([], t)).toEqual([]);
  });

  it('缺 resource/action 的脏数据被跳过', () => {
    const dirty = [
      { id: 1, code: ':read', name: 'x', resource: '', action: 'read' },
      { id: 2, code: 'ticket:', name: 'x', resource: 'ticket', action: '' },
    ] as PermissionCatalogItem[];
    expect(derivePermissionModules(dirty, t)).toEqual([]);
  });
});

describe('actionDisplay 操作名展示', () => {
  it('中文：新 action 显示中文而非英文 code', () => {
    const t = makeT('zh-CN');
    expect(actionDisplay('escalate', t)).toBe('升级');
    expect(actionDisplay('rollback', t)).toBe('回滚');
    expect(actionDisplay('force-update', t)).toBe('强制更新');
    expect(actionDisplay('read', t)).toBe('读取');
  });

  it('英文：全部 action 都有英文名，不残留 code', () => {
    const t = makeT('en-US');
    for (const code of [
      'read',
      'write',
      'escalate',
      'rollback',
      'force-update',
      'provision',
      'triage',
      'override',
    ]) {
      expect(actionDisplay(code, t)).not.toBe(code);
    }
  });

  it('未知 action 才兜底为 code（可诊断，不白屏）', () => {
    const t = makeT('zh-CN');
    expect(actionDisplay('totally_new_action', t)).toBe('totally_new_action');
  });
});
