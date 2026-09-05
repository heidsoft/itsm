'use client';

/**
 * 菜单项渲染组件
 */

import React from 'react';
import { Menu, Badge } from 'antd';
import type { MenuProps } from 'antd';
import styles from '../Sidebar.module.css';
import type { MenuItem } from './menu-config';

interface MenuItemsProps {
  items: MenuItem[];
  selectedKeys: string[];
  onMenuClick: (key: string) => void;
}

type MenuItemRender = Required<MenuProps>['items'][number];

/** 取菜单项的纯文本标题：label 为 JSX 时，antd 收缩态无法自动生成
 * Tooltip，需显式传 title，否则收缩后悬浮只有图标没有文字。 */
function itemTitleText(item: MenuItem): string | undefined {
  return typeof item.label === 'string' ? item.label : undefined;
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
      // 构建唯一父 key
      let parentKey = item.key;
      if (!item.key.endsWith('-group')) {
        parentKey = `${item.key}-group`;
      }
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
 */
export const MenuItems: React.FC<MenuItemsProps> = ({ items, selectedKeys, onMenuClick }) => {
  return (
    <Menu
      mode="inline"
      inlineIndent={16}
      selectedKeys={selectedKeys}
      items={renderMenuItems(items, onMenuClick)}
      theme="light"
      className={styles.customMenu}
      getPopupContainer={() => document.body}
    />
  );
};
