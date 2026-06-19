'use client';

import React from 'react';
import { Skeleton, Layout } from 'antd';
import { LAYOUT_CONFIG } from '@/config/layout.config';

/**
 * 主布局加载状态
 * 匹配主布局结构：侧边栏占位 + 内容区骨架屏
 */
export default function Loading() {
  return (
    <Layout className="min-h-screen bg-[#f5f7fb]">
      {/* 侧边栏占位 */}
      <div
        style={{
          position: 'fixed',
          left: 0,
          top: 0,
          bottom: 0,
          width: LAYOUT_CONFIG.sider.width,
          background: '#ffffff',
          borderRight: '1px solid #f0f0f0',
          zIndex: 1000,
          padding: '16px 12px',
        }}
      >
        {/* Logo 占位 */}
        <Skeleton.Input active size="small" style={{ width: '80%', height: 32, marginBottom: 24 }} />
        {/* 菜单项占位 */}
        {Array.from({ length: 6 }).map((_, index) => (
          <Skeleton.Input
            key={index}
            active
            size="small"
            style={{ width: '100%', height: 36, marginBottom: 8 }}
          />
        ))}
      </div>

      {/* 主内容区 */}
      <Layout
        style={{
          paddingLeft: LAYOUT_CONFIG.sider.width,
          background: '#f5f7fb',
        }}
      >
        {/* Header 占位 */}
        <div
          style={{
            height: 64,
            background: '#ffffff',
            borderBottom: '1px solid #f0f0f0',
            display: 'flex',
            alignItems: 'center',
            padding: '0 24px',
          }}
        >
          <Skeleton.Input active size="small" style={{ width: 200, height: 24 }} />
        </div>

        {/* 内容骨架屏 */}
        <div style={{ padding: '16px' }}>
          <Skeleton active paragraph={{ rows: 2 }} style={{ marginBottom: 24 }} />
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16, marginBottom: 24 }}>
            {Array.from({ length: 4 }).map((_, index) => (
              <div key={index}>
                <Skeleton active paragraph={{ rows: 3 }} />
              </div>
            ))}
          </div>
          <Skeleton active paragraph={{ rows: 8 }} />
        </div>
      </Layout>
    </Layout>
  );
}
