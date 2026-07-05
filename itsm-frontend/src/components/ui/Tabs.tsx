import React from 'react';
import { Tabs as AntTabs } from 'antd';

interface TabsProps {
  defaultValue?: string;
  value?: string;
  onValueChange?: (value: string) => void;
  children?: React.ReactNode;
  className?: string;
}

interface TabsListProps {
  className?: string;
  children?: React.ReactNode;
}

interface TabsTriggerProps {
  value: string;
  children?: React.ReactNode;
  className?: string;
}

interface TabsContentProps {
  value: string;
  children?: React.ReactNode;
  className?: string;
}

export const Tabs = ({ defaultValue, value, onValueChange, children, className }: TabsProps) => {
  // Extract TabsList triggers for tab items and TabsContent for panels
  const items: { key: string; label: React.ReactNode; children: React.ReactNode }[] = [];

  const triggers: { value: string; label: React.ReactNode }[] = [];
  const contents: { value: string; children: React.ReactNode }[] = [];

  React.Children.forEach(children, (child) => {
    if (React.isValidElement<TabsListProps>(child)) {
      if (child.type === TabsList) {
        React.Children.forEach(child.props.children, (trigger) => {
          if (React.isValidElement<TabsTriggerProps>(trigger) && trigger.type === TabsTrigger) {
            triggers.push({ value: trigger.props.value, label: trigger.props.children });
          }
        });
      }
    }
    if (React.isValidElement<TabsContentProps>(child) && child.type === TabsContent) {
      contents.push({ value: child.props.value, children: child.props.children });
    }
  });

  // Merge triggers and contents into Ant Design items
  triggers.forEach((trigger) => {
    const content = contents.find((c) => c.value === trigger.value);
    items.push({
      key: trigger.value,
      label: trigger.label,
      children: content?.children || null,
    });
  });

  return (
    <AntTabs
      activeKey={value || defaultValue}
      onChange={(key) => onValueChange?.(key)}
      items={items}
      className={className}
    />
  );
};

export const TabsList = ({ children }: TabsListProps) => <>{children}</>;
export const TabsTrigger = ({ children }: TabsTriggerProps) => <>{children}</>;
export const TabsContent = ({ children }: TabsContentProps) => <>{children}</>;
