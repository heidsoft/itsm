/**
 * 侧栏菜单展开态记忆：行为测试。
 *
 * 缺陷背景（审计实测）：
 *  `MenuItems` 的 `<Menu>` 只传 `selectedKeys`，没有 `openKeys`，于是
 *  - 每次导航或刷新后所有二级菜单全部收起；
 *  - 归位页（URL 不在父级路径下，共 18 个，如 /admin/ticket-categories 挂在
 *    「工单管理」下）的选中项直接藏在折叠的组里，用户看不到自己在哪；
 *  - 工作流域 10 个子项每次都要重新展开一次。
 *
 * 断言口径说明：这里直接渲染真实 `MenuItems` 组件、读写真实 zustand store，
 * 不做自制副本。反向验证时把 `openKeys` 传参删掉，本套件必须转红。
 */

import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { Menu } from 'antd';

import { MenuItems, ancestorGroupKeys, collectSubmenuKeys, submenuKeyFor } from '../MenuItems';
import type { MenuItem } from '../menu-config';
import { useSidebarOpenKeys } from '@/lib/store/sidebar-open-keys-store';

/** 复刻真实菜单形状：工单管理（2 子项，其中一个是归位页）+ 一个顶级页。 */
const ITEMS: MenuItem[] = [
  {
    key: '/tickets',
    label: '工单管理',
    path: '/tickets',
    children: [
      { key: '/admin/ticket-categories', label: '工单分类', path: '/admin/ticket-categories' },
      { key: '/tickets/templates', label: '工单模板', path: '/tickets/templates' },
    ],
  },
  { key: '/dashboard', label: '服务台', path: '/dashboard' },
];

describe('菜单展开态记忆', () => {
  beforeEach(() => {
    useSidebarOpenKeys.setState({ openKeys: [] });
  });

  describe('key 推导（展开态与渲染共用同一口径）', () => {
    it('submenuKeyFor 只由父项 path 推导，带 -group 后缀的复用不二次追加', () => {
      expect(submenuKeyFor('/tickets')).toBe('/tickets-group');
      expect(submenuKeyFor('/tickets-group')).toBe('/tickets-group');
    });

    it('collectSubmenuKeys 收集全部有子项的组', () => {
      expect(collectSubmenuKeys(ITEMS)).toEqual(['/tickets-group']);
    });
  });

  describe('当前路径的祖先链', () => {
    it('归位页的祖先链来自菜单归属，而不是 URL 前缀', () => {
      // /admin/ticket-categories 的 URL 前缀是 /admin，但它在「工单管理」下
      expect(ancestorGroupKeys(ITEMS, '/admin/ticket-categories')).toEqual(['/tickets-group']);
    });

    it('子菜单内的路径同样能查到', () => {
      expect(ancestorGroupKeys(ITEMS, '/tickets/templates')).toEqual(['/tickets-group']);
    });

    it('顶级页面没有祖先链', () => {
      expect(ancestorGroupKeys(ITEMS, '/dashboard')).toEqual([]);
    });

    it('不在菜单里的路径返回空，不瞎猜', () => {
      expect(ancestorGroupKeys(ITEMS, '/somewhere-else')).toEqual([]);
    });
  });

  describe('渲染行为：路由进入后自动展开所在组', () => {
    it('进入归位页时，父组被写进展开态（后续进入其它页也不会被 antd 收起）', () => {
      // 说明：antd 自身会展开包含选中项的子菜单，因此「点进来后组是开着的」本身
      // 不需要我们做。我们主动写入的意义在于：把该组登记为用户展开态，此后导航到
      // 不再包含它的页面时它仍保持展开（见下一条 DOM 级用例）。
      const { rerender } = render(
        <MenuItems items={ITEMS} selectedKeys={['/dashboard']} onMenuClick={() => {}} />
      );
      expect(useSidebarOpenKeys.getState().openKeys).toEqual([]);

      rerender(
        <MenuItems
          items={ITEMS}
          selectedKeys={['/admin/ticket-categories']}
          onMenuClick={() => {}}
        />
      );
      expect(useSidebarOpenKeys.getState().openKeys).toContain('/tickets-group');
    });

    it('自动展开是「合并」而不是「覆盖」，用户手动展开的其它组不被收起', () => {
      useSidebarOpenKeys.setState({ openKeys: ['/other-group'] });
      render(
        <MenuItems
          items={ITEMS}
          selectedKeys={['/admin/ticket-categories']}
          onMenuClick={() => {}}
        />
      );
      const keys = useSidebarOpenKeys.getState().openKeys;
      expect(keys).toContain('/other-group');
      expect(keys).toContain('/tickets-group');
    });

    it('展开态跨组件卸载保留（刷新后由 persist 恢复，此处验证 store 不被重置）', () => {
      const { unmount } = render(
        <MenuItems
          items={ITEMS}
          selectedKeys={['/admin/ticket-categories']}
          onMenuClick={() => {}}
        />
      );
      expect(useSidebarOpenKeys.getState().openKeys).toContain('/tickets-group');
      unmount();
      expect(useSidebarOpenKeys.getState().openKeys).toContain('/tickets-group');
    });

    it('收缩态不接管 openKeys（交给 antd 的 popup 子菜单自行管理）', () => {
      useSidebarOpenKeys.setState({ openKeys: ['/tickets-group'] });
      const { container } = render(
        <MenuItems
          items={ITEMS}
          selectedKeys={['/admin/ticket-categories']}
          onMenuClick={() => {}}
          collapsed
        />
      );
      // 收缩态渲染的仍是可用的 Menu（不因传了 openKeys 而互相打架）
      expect(container.querySelector('.ant-menu')).not.toBeNull();
      // store 未被收缩态改动
      expect(useSidebarOpenKeys.getState().openKeys).toEqual(['/tickets-group']);
    });

    it('🔴 用户手动展开的组在导航到别处后仍然展开（DOM 级，不是只看 store）', () => {
      // 这是 openKeys 受控的**唯一真实价值**所在：antd 自己会展开「含选中项」
      // 的那个组，但会把它不认识的其它组全部收起。所以「记忆」必须落到 DOM 上验证——
      // 只断言 store 里存了 key 是不够的（已实测：去掉 openKeys={} 传参，store 断言全绿）。
      useSidebarOpenKeys.setState({ openKeys: [] });
      const { container, rerender } = render(
        <MenuItems
          items={ITEMS}
          selectedKeys={['/admin/ticket-categories']}
          onMenuClick={() => {}}
        />
      );
      // 用户手动展开「工单管理」后在别的组里操作：这里用 store 直接模拟手动展开，
      // 并导航到不含该组的页面（/dashboard）。
      useSidebarOpenKeys.setState({ openKeys: ['/tickets-group'] });
      rerender(<MenuItems items={ITEMS} selectedKeys={['/dashboard']} onMenuClick={() => {}} />);

      const submenu = container.querySelector('li.ant-menu-submenu');
      expect(submenu).not.toBeNull();
      // 该组不在当前路径的祖先链里，却仍然展开 → 证明 openKeys 真的接到了 Menu 上
      expect(submenu!.className).toContain('ant-menu-submenu-open');
      expect(submenu!.querySelector('.ant-menu-submenu-title')?.getAttribute('aria-expanded'))
        .toBe('true');
    });
  });

  describe('store 语义', () => {
    it('setOpenKeys 完全替换（用户收起某组时该组必须消失）', () => {
      useSidebarOpenKeys.setState({ openKeys: ['/tickets-group'] });
      useSidebarOpenKeys.getState().setOpenKeys([]);
      expect(useSidebarOpenKeys.getState().openKeys).toEqual([]);
    });

    it('revealGroupChain 幂等，不产生重复 key', () => {
      useSidebarOpenKeys.getState().revealGroupChain(['/tickets-group']);
      useSidebarOpenKeys.getState().revealGroupChain(['/tickets-group']);
      expect(useSidebarOpenKeys.getState().openKeys).toEqual(['/tickets-group']);
    });

    it('clear 用于切换账号，避免上一个人的展开态串到下一个账号', () => {
      useSidebarOpenKeys.setState({ openKeys: ['/tickets-group'] });
      useSidebarOpenKeys.getState().clear();
      expect(useSidebarOpenKeys.getState().openKeys).toEqual([]);
    });
  });
});
