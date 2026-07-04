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
            onClick={e => {
              if (item.path) {
                e.stopPropagation();
                onMenuClick(item.path);
              }
            }}
          >
            <span className="truncate">{item.label}</span>
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
            label: (
              <div className={styles.menuItemLabel}>
                <span className="truncate">{child.label}</span>
              </div>
            ),
            onClick: () => onMenuClick(child.key),
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
        >
          <span className="truncate">{item.label}</span>
          {item.badge && <Badge count={item.badge} size="small" className={styles.menuItemBadge} />}
        </div>
      ),
      onClick: () => onMenuClick(item.key),
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
    />
  );
};
