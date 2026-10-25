/**
 * 菜单图标前后端一致性回归测试
 *
 * 背景（2026-10-25 prod 栈实测）：侧边栏里有 16 个菜单渲染不出图标——
 * 报表中心、标准变更库、我的应用、项目管理、拓扑图、云资源列表、技能注册表…
 * 根因是 itsm-backend/pkg/menubaseline/baseline.go 的 Icon 取值（63 个 Lucide 名）
 * 与前端 iconMap 的注册项（59 个）漂移了，getIconByName 对未注册名返回 undefined，
 * AntD Menu 于是只渲染标签不渲染图标。
 *
 * 这个缺陷有两个加重因素：
 *   1. /admin/menus 的图标选择器只列 Object.keys(iconMap)，运营在 UI 里选不到
 *      那 16 个图标，也就无法自行补救，必须发版；
 *   2. 两侧都是手工维护的字符串清单，没有任何机制保证同步。
 *
 * 本测试把「后端用到的每个图标都必须在前端注册」固化成断言。
 * 新增菜单图标时若只改 baseline.go，这里会立刻红。
 */

import { readFileSync } from 'fs';
import { join } from 'path';

import { iconMap, getIconByName } from '../icons';

/** 从 baseline.go 抽出所有 Icon: "Xxx" 取值 */
function readBackendBaselineIcons(): string[] {
  const baselinePath = join(
    process.cwd(),
    '..',
    'itsm-backend',
    'pkg',
    'menubaseline',
    'baseline.go'
  );
  const source = readFileSync(baselinePath, 'utf-8');
  const icons = [...source.matchAll(/Icon:\s*"([^"]+)"/g)].map(m => m[1]);
  return [...new Set(icons)];
}

describe('sidebar iconMap 与后端菜单基线一致', () => {
  const backendIcons = readBackendBaselineIcons();
  const registered = Object.keys(iconMap);

  it('baseline.go 里确实声明了图标（防止路径写错导致空数组假通过）', () => {
    expect(backendIcons.length).toBeGreaterThan(50);
  });

  it('后端用到的每个图标都在前端注册（曾漏 16 个 → 菜单无图标）', () => {
    const missing = backendIcons.filter(name => !(name in iconMap));
    expect(missing).toEqual([]);
  });

  it('getIconByName 对基线图标全部返回可渲染节点', () => {
    for (const name of backendIcons) {
      expect(getIconByName(name)).toBeDefined();
    }
  });

  it('getIconByName 对未知名返回 undefined 而非抛错', () => {
    expect(getIconByName('__not_a_real_icon__')).toBeUndefined();
    expect(getIconByName(undefined)).toBeUndefined();
    expect(getIconByName('')).toBeUndefined();
  });

  it('前端没有为后端未使用的图标留死代码（超过 20 个视为失控）', () => {
    const unused = registered.filter(name => !backendIcons.includes(name));
    // 允许少量：菜单管理页的图标选择器会展示全部可选图标，
    // 运营建新菜单时需要一份可选池。
    expect(unused.length).toBeLessThanOrEqual(20);
  });
});
