'use client';

import React, { useState, useCallback } from 'react';
import { useHotkeys } from 'react-hotkeys-hook';
import { Modal, List, Tag, Typography } from 'antd';
import { useRouter } from 'next/navigation';
import { useLayoutStore } from '@/lib/store/layout-store';

const { Title, Text } = Typography;

// 快捷键配置
const SHORTCUTS = [
  {
    category: '导航',
    items: [
      { key: 'g d', description: '跳转到仪表盘', action: '/dashboard' },
      { key: 'g t', description: '跳转到工单列表', action: '/tickets' },
      { key: 'g c', description: '跳转到CMDB', action: '/cmdb' },
      { key: 'g k', description: '跳转到知识库', action: '/knowledge' },
      { key: 'g w', description: '跳转到工作流', action: '/workflows' },
    ],
  },
  {
    category: '操作',
    items: [
      { key: 'ctrl+n / cmd+n', description: '新建工单', action: '/tickets/new' },
      { key: 'ctrl+k / cmd+k', description: '打开全局搜索', action: 'search' },
      { key: 'ctrl+b / cmd+b', description: '切换侧边栏折叠', action: 'toggleSidebar' },
      { key: 'ctrl+r / cmd+r', description: '刷新当前页面数据', action: 'refresh' },
    ],
  },
  {
    category: '帮助',
    items: [
      { key: '? / ctrl+/ / cmd+/', description: '显示快捷键帮助', action: 'help' },
      { key: 'esc', description: '关闭弹窗/取消操作', action: 'close' },
    ],
  },
];

interface GlobalShortcutProviderProps {
  children: React.ReactNode;
}

export default function GlobalShortcutProvider({ children }: GlobalShortcutProviderProps) {
  const [helpVisible, setHelpVisible] = useState(false);
  const [searchVisible, setSearchVisible] = useState(false);
  const router = useRouter();
  const { collapsed, setCollapsed } = useLayoutStore();

  // 打开帮助
  const openHelp = useCallback(() => {
    setHelpVisible(true);
  }, []);

  // 打开全局搜索
  const openSearch = useCallback(() => {
    setSearchVisible(true);
    // 这里可以后续对接全局搜索组件
  }, []);

  // 切换侧边栏
  const toggleSidebar = useCallback(() => {
    setCollapsed(!collapsed);
  }, [collapsed, setCollapsed]);

  // 注册快捷键
  useHotkeys('g d', () => router.push('/dashboard'), { preventDefault: true });
  useHotkeys('g t', () => router.push('/tickets'), { preventDefault: true });
  useHotkeys('g c', () => router.push('/cmdb'), { preventDefault: true });
  useHotkeys('g k', () => router.push('/knowledge'), { preventDefault: true });
  useHotkeys('g w', () => router.push('/workflows'), { preventDefault: true });
  useHotkeys('ctrl+n, cmd+n', () => router.push('/tickets/new'), { preventDefault: true });
  useHotkeys('ctrl+k, cmd+k', openSearch, { preventDefault: true });
  useHotkeys('ctrl+b, cmd+b', toggleSidebar, { preventDefault: true });
  useHotkeys('ctrl+/, cmd+/, ?', openHelp, { preventDefault: true });

  return (
    <>
      {children}
      
      {/* 快捷键帮助弹窗 */}
      <Modal
        title="键盘快捷键"
        open={helpVisible}
        onCancel={() => setHelpVisible(false)}
        footer={null}
        width={600}
      >
        <div className="space-y-6">
          {SHORTCUTS.map((category) => (
            <div key={category.category}>
              <Title level={5} className="!mb-3 !text-gray-800">
                {category.category}
              </Title>
              <List
                dataSource={category.items}
                renderItem={(item) => (
                  <List.Item className="!py-2 !px-0 flex justify-between items-center">
                    <Text className="text-gray-700">{item.description}</Text>
                    <div className="flex gap-2">
                      {item.key.split(' / ').map((k) => (
                        <Tag key={k} color="blue" className="font-mono text-xs">
                          {k}
                        </Tag>
                      ))}
                    </div>
                  </List.Item>
                )}
              />
            </div>
          ))}
        </div>
      </Modal>
    </>
  );
}
