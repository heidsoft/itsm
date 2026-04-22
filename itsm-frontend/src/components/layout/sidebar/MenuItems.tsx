'use client';

/**
 * 菜单项渲染组件
 */

import React from 'react';
import { Menu, Badge } from 'antd';
import type { MenuProps } from 'antd';
import styles from '../Sidebar.module.css';
import { MenuItem } from './menu-config';

interface MenuItemsProps {
  items: MenuItem[];
  selectedKeys: string[];
  onMenuClick: (key: string) => void;
}

type MenuItemRender = Required<MenuProps>['items'][number];

/**
 * 渲染菜单项
 */
export function renderMenuItems(
  items: MenuItem[],
  onMenuClick: (key: string) => void
): MenuItemRender[] {
  return items.map(item => {
    // 如果有子菜单
    if (item.children) {
      // 确保父菜单 key 与子菜单不重复：添加 "-group" 后缀
      const parentKey = item.key.endsWith('-group') ? item.key : `${item.key}-group`;
      return {
        key: parentKey,
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
        children: item.children.map((child: MenuItem) => ({
          key: child.key,
          icon: child.icon,
          label: (
            <div className={styles.menuItemLabel}>
              <span className="truncate">{child.label}</span>
            </div>
          ),
          onClick: () => onMenuClick(child.key),
        })),
      };
    }

    // 普通菜单项
    return {
      key: item.key,
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
