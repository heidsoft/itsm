'use client';

import React from 'react';
import { Tabs as AntTabs } from 'antd';
import type { TabsProps as AntTabsProps } from 'antd';

/**
 * Tabs - 兼容 shadcn 风格的 Tabs 组件
 * 基于 Ant Design Tabs 实现
 *
 * shadcn 写法是「标记式」的：
 *   <Tabs><TabsList><TabsTrigger value="a">A</TabsTrigger></TabsList>
 *         <TabsContent value="a">内容</TabsContent></Tabs>
 * 这些子元素本身不渲染任何东西，只作为 Ant Design `items` 的描述来源。
 *
 * 之前用 effect + ref 注册子项，有两个问题：
 * 1. TabsTrigger 从未注册，AntTabs 只有内容没有 label，触发按钮只能靠
 *    原样渲染 children 兜底，导致多个触发文字串连在一起且点击无效；
 * 2. children 被渲染两次（AntTabs 之外再渲染一次），触发文字重复出现。
 *
 * 现在改为在渲染时用 React.Children 递归收集（包装层如 CardHeader 也能穿透），
 * 只把收集结果交给 AntTabs，原始 children 不再渲染。
 */

export interface TabsProps extends Omit<AntTabsProps, 'activeKey' | 'defaultActiveKey' | 'onChange' | 'items' | 'children'> {
  defaultValue?: string;
  value?: string;
  onValueChange?: (value: string) => void;
  className?: string;
  children?: React.ReactNode;
}

export interface TabsListProps {
  children?: React.ReactNode;
}

export interface TabsTriggerProps {
  value: string;
  children?: React.ReactNode;
}

export interface TabsContentProps {
  value: string;
  className?: string;
  children?: React.ReactNode;
  forceRender?: boolean;
}

interface CollectedTrigger {
  value: string;
  label?: React.ReactNode;
}

interface CollectedContent {
  node: React.ReactNode;
  forceRender?: boolean;
}

interface CollectedTabs {
  triggers: CollectedTrigger[];
  contents: Map<string, CollectedContent>;
}

interface MarkerProps {
  value?: unknown;
  children?: React.ReactNode;
  className?: string;
  forceRender?: boolean;
}

const collectTabs = (children: React.ReactNode, acc: CollectedTabs): void => {
  React.Children.forEach(children, (child) => {
    if (!React.isValidElement(child)) return;
    const props = child.props as MarkerProps;

    if (child.type === TabsTrigger) {
      const value = typeof props.value === 'string' ? props.value : '';
      if (value) acc.triggers.push({ value, label: props.children });
      return;
    }

    if (child.type === TabsContent) {
      const value = typeof props.value === 'string' ? props.value : '';
      if (value) {
        acc.contents.set(value, {
          node: <div className={props.className}>{props.children}</div>,
          forceRender: props.forceRender,
        });
      }
      return;
    }

    // TabsList / CardHeader / Fragment 等包装层：继续向下收集。
    if (props.children) collectTabs(props.children, acc);
  });
};

export const Tabs: React.FC<TabsProps> = ({
  defaultValue,
  value,
  onValueChange,
  className,
  children,
  ...rest
}) => {
  const collected: CollectedTabs = { triggers: [], contents: new Map() };
  collectTabs(children, collected);

  const items = collected.triggers.map((trigger) => {
    const content = collected.contents.get(trigger.value);
    return {
      key: trigger.value,
      label: trigger.label,
      children: content?.node,
      forceRender: content?.forceRender,
    };
  });

  return (
    <AntTabs
      {...rest}
      className={className}
      activeKey={value}
      defaultActiveKey={defaultValue}
      onChange={(key) => onValueChange?.(key)}
      items={items}
    />
  );
};

// 以下三个组件只是标记，内容由 Tabs 在渲染时收集。
export const TabsList: React.FC<TabsListProps> = () => null;

export const TabsTrigger: React.FC<TabsTriggerProps> = () => null;

export const TabsContent: React.FC<TabsContentProps> = () => null;

export default Tabs;
