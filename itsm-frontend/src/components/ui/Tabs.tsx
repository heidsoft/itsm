'use client';

import React from 'react';
import { Tabs as AntTabs } from 'antd';
import type { TabsProps as AntTabsProps } from 'antd';

/**
 * Tabs - 兼容 shadcn 风格的 Tabs 组件
 * 基于 Ant Design Tabs 实现
 */

export interface TabsItemDescriptor {
  value: string;
  trigger?: React.ReactNode;
  content?: React.ReactNode;
  forceRender?: boolean;
}

export interface TabsContextValue {
  registerItem: (item: TabsItemDescriptor) => void;
  unregisterItem: (value: string) => void;
}

const TabsContext = React.createContext<TabsContextValue | null>(null);

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

export const Tabs: React.FC<TabsProps> = ({
  defaultValue,
  value,
  onValueChange,
  className,
  children,
  ...rest
}) => {
  const itemsRef = React.useRef<Map<string, TabsItemDescriptor>>(new Map());
  const [, force] = React.useReducer((x: number) => x + 1, 0);

  const registerItem = React.useCallback((item: TabsItemDescriptor) => {
    itemsRef.current.set(item.value, item);
    force();
  }, []);

  const unregisterItem = React.useCallback((val: string) => {
    itemsRef.current.delete(val);
    force();
  }, []);

  const ctxValue = React.useMemo<TabsContextValue>(
    () => ({ registerItem, unregisterItem }),
    [registerItem, unregisterItem],
  );

  const tabItems = Array.from(itemsRef.current.values()).map((it) => ({
    key: it.value,
    label: it.trigger,
    children: it.content,
    forceRender: it.forceRender,
  }));

  return (
    <TabsContext.Provider value={ctxValue}>
      <AntTabs
        {...rest}
        className={className}
        activeKey={value}
        defaultActiveKey={defaultValue}
        onChange={(k) => onValueChange?.(k)}
        items={tabItems}
      />
      {children}
    </TabsContext.Provider>
  );
};

export const TabsList: React.FC<TabsListProps> = ({ children }) => <>{children}</>;

export const TabsTrigger: React.FC<TabsTriggerProps> = ({ children }) => <>{children}</>;

export const TabsContent: React.FC<TabsContentProps> = ({ value, children, className, forceRender }) => {
  const ctx = React.useContext(TabsContext);
  React.useEffect(() => {
    if (!ctx) return;
    ctx.registerItem({ value, content: <div className={className}>{children}</div>, forceRender });
    return () => ctx.unregisterItem(value);
  }, [ctx, value, children, className, forceRender]);
  return null;
};

export default Tabs;