'use client';

/**
 * 菜单项渲染组件
 */

import React, { useEffect } from 'react';
import { Menu, Badge } from 'antd';
import type { MenuProps } from 'antd';
// 必须与 Sidebar.tsx 使用同一份 CSS Module：CSS Modules 按文件哈希类名，
// 之前从父目录引入同名模块，导致本组件渲染的 .customMenu 永远匹配不到
// sidebar/Sidebar.module.css 中的暗色菜单项覆盖规则。
import styles from './Sidebar.module.css';
import type { MenuItem } from './menu-config';

import { useSidebarOpenKeys } from '@/lib/store/sidebar-open-keys-store';

interface MenuItemsProps {
  items: MenuItem[];
  selectedKeys: string[];
  onMenuClick: (key: string) => void;
  /** 侧栏是否收缩：收缩态由 antd 自行管理 popup 子菜单，我们不接管 openKeys。 */
  collapsed?: boolean;
}

type MenuItemRender = Required<MenuProps>['items'][number];

/** 取菜单项的纯文本标题：label 为 JSX 时，antd 收缩态无法自动生成
 * Tooltip，需显式传 title，否则收缩后悬浮只有图标没有文字。 */
function itemTitleText(item: MenuItem): string | undefined {
  return typeof item.label === 'string' ? item.label : undefined;
}

/**
 * 子菜单（antd SubMenu）的 key。由**父项自己的 key**（即 path）推导，
 * 而不是由渲染时临时分配的序号推导——否则同一棵树在不同渲染次序下
 * 会算出不同 key，展开态记忆就失效了。
 *
 * 以 `-group` 结尾的 path 直接复用，避免 `/x-group` 与 `/x` 的子 key 撞名。
 */
export function submenuKeyFor(itemKey: string): string {
  return itemKey.endsWith('-group') ? itemKey : `${itemKey}-group`;
}

/** 收集一棵菜单树里所有「有子项」的 submenu key（顺序无关，供 openKeys 使用）。 */
export function collectSubmenuKeys(items: MenuItem[]): string[] {
  const keys: string[] = [];
  const walk = (list: MenuItem[]) => {
    for (const item of list) {
      if (item.children && item.children.length > 0) {
        keys.push(submenuKeyFor(item.key));
        walk(item.children);
      }
    }
  };
  walk(items);
  return keys;
}

/**
 * 当前页面所在的子菜单祖先链（从最外层到最内层）。
 *
 * 用途：进入某个页面时，把它所在的菜单组**登记**为用户展开态。
 *
 * 为什么要主动登记：antd 自己会展开「含选中项」的那个组，所以「点进来后组是
 * 开着的」不需要我们做。但 antd 会把不在 `openKeys` 里的其它组全部收起——一旦
 * 用户离开这个页面，刚展开过的组就没了。登记之后它才被认定为「用户展开的组」，
 * 在后续导航中保持展开，这才是 `openKeys` 受控的实际价值。
 *
 * 返回空数组表示目标不在任何子菜单里（顶级页或不在菜单里）。
 */
export function ancestorGroupKeys(items: MenuItem[], targetPath: string): string[] {
  const chain: string[] = [];
  const walk = (list: MenuItem[], ancestors: string[]): boolean => {
    for (const item of list) {
      if (item.path === targetPath) {
        chain.push(...ancestors);
        return true;
      }
      if (item.children && item.children.length > 0) {
        if (walk(item.children, [...ancestors, submenuKeyFor(item.key)])) return true;
      }
    }
    return false;
  };
  walk(items, []);
  return chain;
}

/**
 * 渲染菜单项
 * 使用唯一的 key（前缀 + index）避免 Ant Design Menu 的 key 冲突
 */
export function renderMenuItems(
  items: MenuItem[],
  onMenuClick: (key: string) => void
): MenuItemRender[] {
  // 收集已使用的 key，确保唯一性
  const usedKeys = new Set<string>();

  return items.map((item, itemIndex) => {
    // 如果有子菜单
    if (item.children && item.children.length > 0) {
      // 子菜单 key 只由父项 path 推导（见 submenuKeyFor），与展开态记忆共用同一口径。
      const parentKey = submenuKeyFor(item.key);
      // 确保 key 唯一
      let uniqueParentKey = parentKey;
      let counter = 1;
      while (usedKeys.has(uniqueParentKey)) {
        uniqueParentKey = `${parentKey}-${counter++}`;
      }
      usedKeys.add(uniqueParentKey);

      return {
        key: uniqueParentKey,
        icon: item.icon,
        title: itemTitleText(item),
        label: (
          <div
            className={styles.menuItemLabel}
            title={
              typeof item.description === 'string'
                ? item.description
                : typeof item.label === 'string'
                  ? item.label
                  : undefined
            }
            onClick={() => {
              if (item.path) {
                onMenuClick(item.path);
              }
            }}
          >
            <span className="truncate">{item.label}</span>
            {item.badge && <Badge count={item.badge} size="small" className={styles.menuItemBadge} />}
          </div>
        ),
        children: item.children.map((child: MenuItem, childIndex: number) => {
          // 构建唯一子 key
          let uniqueChildKey = child.key;
          let childCounter = 0;
          while (usedKeys.has(uniqueChildKey)) {
            uniqueChildKey = `${child.key}-${++childCounter}`;
          }
          usedKeys.add(uniqueChildKey);

          return {
            key: uniqueChildKey,
            icon: child.icon,
            title: itemTitleText(child),
            label: (
              <div
                className={styles.menuItemLabel}
                // antd v6 items[].onClick 在 SubMenu 展开态下偶发不触发（rc-menu 内部
                // 事件冒泡与 title 切换竞争），保留 label 层 onClick 兜底，跟父项同
                // 一套机制，保证「变更管理 → 新建变更」这类子项一定可跳转。
                onClick={() => onMenuClick(child.path || child.key)}
              >
                <span className="truncate">{child.label}</span>
                {child.badge && <Badge count={child.badge} size="small" className={styles.menuItemBadge} />}
              </div>
            ),
            onClick: () => onMenuClick(child.path || child.key),
          };
        }),
      };
    }

    // 普通菜单项
    let uniqueKey = item.key;
    let counter = 0;
    while (usedKeys.has(uniqueKey)) {
      uniqueKey = `${item.key}-${++counter}`;
    }
    usedKeys.add(uniqueKey);

    return {
      key: uniqueKey,
      icon: item.icon,
      title: itemTitleText(item),
      label: (
        <div
          className={styles.menuItemLabel}
          onClick={() => onMenuClick(item.path || item.key)}
          title={
            typeof item.description === 'string'
              ? item.description
              : typeof item.label === 'string'
                ? item.label
                : undefined
          }
        >
          <span className="truncate">{item.label}</span>
          {item.badge && <Badge count={item.badge} size="small" className={styles.menuItemBadge} />}
        </div>
      ),
      onClick: () => onMenuClick(item.path || item.key),
      className: styles.menuItem,
    };
  });
}

/**
 * 菜单项列表组件
 *
 * 展开态记忆（openKeys 受控）：
 *  - 用户手动展开/收起某个组 → 由 antd 的 onOpenChange 整体回写进 store
 *    （跨导航、跨刷新保留）
 *  - 路由变化 → 把当前页面所在的祖先组并入展开集合。antd 自己会展开
 *    「含选中项」的那个组，但要把它**认定为用户展开态**，否则一旦导航到
 *    不包含它的页面就会被 antd 收起
 *  - 侧栏收缩态 → 不接管 openKeys：收缩后是 popup 子菜单，由 antd 自行管理，
 *    两边同时控制会互相打架
 *
 * 只记「用户操作过的组」而不把全树 openKeys 都存下来，是为了让新出现的
 * 菜单组保持默认收起（否则一次展开会把后续版本新增的所有组都连带打开）。
 */
export const MenuItems: React.FC<MenuItemsProps> = ({
  items,
  selectedKeys,
  onMenuClick,
  collapsed = false,
}) => {
  const openKeys = useSidebarOpenKeys();
  const currentPath = selectedKeys[0] ?? '';
  const activeChain = ancestorGroupKeys(items, currentPath);
  // 依赖用稳定的字符串而不是数组：activeChain 每次渲染都是新数组引用，
  // 直接放进 deps 会让每个渲染周期都跑一次 effect 并写 store。
  const activeChainKey = activeChain.join('|');
  const revealGroupChain = openKeys.revealGroupChain;

  useEffect(() => {
    if (collapsed) return;
    if (activeChainKey === '') return;
    revealGroupChain(activeChainKey.split('|'));
  }, [currentPath, activeChainKey, collapsed, revealGroupChain]);

  // 收缩态不接管 openKeys：antd 在 inlineCollapsed 下自动管理 popup，传 openKeys 会互相打架
  if (collapsed) {
    return (
      <Menu
        mode="inline"
        inlineIndent={16}
        selectedKeys={selectedKeys}
        items={renderMenuItems(items, onMenuClick)}
        className={styles.customMenu}
        getPopupContainer={() => document.body}
      />
    );
  }

  return (
    <Menu
      mode="inline"
      inlineIndent={16}
      selectedKeys={selectedKeys}
      openKeys={openKeys.openKeys}
      onOpenChange={keys => openKeys.setOpenKeys(keys as string[])}
      items={renderMenuItems(items, onMenuClick)}
      // 不写死亮色主题：让 Menu 继承 AntdProvider 的 ConfigProvider
      // （darkAlgorithm 跟随 useTheme().isDark）。写死亮色会让收缩态
      // submenu popup 与 inline 二级菜单在深色模式下出现白底块。
      className={styles.customMenu}
      getPopupContainer={() => document.body}
    />
  );
};
