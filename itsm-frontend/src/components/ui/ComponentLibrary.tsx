'use client';

import React from 'react';
import { Button } from './Button';
import { Card, StatCard, InfoCard } from './Card';
import { EnhancedInput, EnhancedTextArea } from './EnhancedInput';
import { cn } from '@/lib/utils';

/**
 * 组件库展示和示例
 * 展示所有可用的组件变体和使用方法
 */
export const ComponentLibrary: React.FC = () => {
  return (
    <div className="p-8 space-y-12 bg-gray-50 min-h-screen">
      <div className="max-w-7xl mx-auto">
        {/* 标题 */}
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-gray-900 mb-4">ITSM 组件库</h1>
          <p className="text-lg text-gray-600">完整的UI组件集合，支持企业级应用开发</p>
        </div>

        {/* 按钮组件示例 */}
        <section className="bg-white rounded-xl p-8 shadow-sm">
          <h2 className="text-2xl font-semibold text-gray-900 mb-6">按钮组件</h2>
          
          <div className="space-y-6">
            {/* 变体展示 */}
            <div>
              <h3 className="text-lg font-medium text-gray-700 mb-4">变体样式</h3>
              <div className="flex flex-wrap gap-4">
                <Button variant="primary">Primary</Button>
                <Button variant="secondary">Secondary</Button>
                <Button variant="outline">Outline</Button>
                <Button variant="ghost">Ghost</Button>
                <Button variant="danger">Danger</Button>
                <Button variant="success">Success</Button>
                <Button variant="link">Link</Button>
                <Button variant="dashed">Dashed</Button>
                <Button variant="text">Text</Button>
              </div>
            </div>

            {/* 尺寸展示 */}
            <div>
              <h3 className="text-lg font-medium text-gray-700 mb-4">尺寸变体</h3>
              <div className="flex items-center gap-4">
                <Button size="xs" variant="primary">Extra Small</Button>
                <Button size="sm" variant="primary">Small</Button>
                <Button size="md" variant="primary">Medium</Button>
                <Button size="lg" variant="primary">Large</Button>
                <Button size="xl" variant="primary">Extra Large</Button>
              </div>
            </div>

            {/* 状态展示 */}
            <div>
              <h3 className="text-lg font-medium text-gray-700 mb-4">状态样式</h3>
              <div className="flex flex-wrap gap-4">
                <Button variant="primary" loading>Loading</Button>
                <Button variant="primary" disabled>Disabled</Button>
                <Button variant="primary" icon={<span>📎</span>}>With Icon</Button>
                <Button variant="primary" fullWidth>Full Width</Button>
              </div>
            </div>
          </div>
        </section>

        {/* 卡片组件示例 */}
        <section className="bg-white rounded-xl p-8 shadow-sm">
          <h2 className="text-2xl font-semibold text-gray-900 mb-6">卡片组件</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {/* 基础卡片 */}
            <Card title="基础卡片" description="这是一个基础的卡片组件">
              <p className="text-gray-600">卡片内容区域，可以放置任何内容。</p>
            </Card>

            {/* 可悬停卡片 */}
            <Card 
              title="可悬停卡片" 
              description="鼠标悬停有交互效果"
              hoverable
              onClick={() => console.log('Card clicked')}
            >
              <p className="text-gray-600">支持点击事件和悬停效果。</p>
            </Card>

            {/* 带封面卡片 */}
            <Card 
              title="封面卡片" 
              description="带有封面图片的卡片"
              cover={
                <div className="h-32 bg-gradient-to-br from-blue-400 to-blue-600 flex items-center justify-center">
                  <span className="text-white text-2xl font-bold">Cover</span>
                </div>
              }
            >
              <p className="text-gray-600">可以添加封面图片或自定义内容。</p>
            </Card>

            {/* 统计卡片 */}
            <StatCard
              title="总工单数"
              value="1,234"
              description="较上月增长 12%"
              trend={{ value: 12, type: 'up' }}
              color="blue"
              icon={<span className="text-2xl">📊</span>}
            />

            <StatCard
              title="完成率"
              value="89.5%"
              description="目标完成度良好"
              trend={{ value: 5.2, type: 'up' }}
              color="green"
              icon={<span className="text-2xl">✅</span>}
            />

            <StatCard
              title="待处理"
              value="23"
              description="需要优先处理"
              trend={{ value: 8, type: 'down' }}
              color="red"
              icon={<span className="text-2xl">⚠️</span>}
            />

            {/* 信息卡片 */}
            <InfoCard
              title="系统提示"
              content="系统将于今晚 22:00 进行维护，预计持续 2 小时。"
              status="info"
              icon={<span>ℹ️</span>}
            />

            <InfoCard
              title="操作成功"
              content="数据已成功保存到系统中。"
              status="success"
              icon={<span>✅</span>}
            />

            <InfoCard
              title="警告信息"
              content="磁盘空间使用率已达到 85%，请及时清理。"
              status="warning"
              icon={<span>⚠️</span>}
            />
          </div>
        </section>

        {/* 输入框组件示例 */}
        <section className="bg-white rounded-xl p-8 shadow-sm">
          <h2 className="text-2xl font-semibold text-gray-900 mb-6">输入框组件</h2>
          
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
            {/* 增强型输入框 */}
            <div className="space-y-6">
              <h3 className="text-lg font-medium text-gray-700">增强型输入框</h3>
              
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">默认输入框</label>
                  <EnhancedInput placeholder="请输入内容" />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">填充样式</label>
                  <EnhancedInput variant="filled" placeholder="填充样式输入框" />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">轮廓样式</label>
                  <EnhancedInput variant="outlined" placeholder="轮廓样式输入框" />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">下划线样式</label>
                  <EnhancedInput variant="underlined" placeholder="下划线样式输入框" />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">带验证</label>
                  <EnhancedInput
                    placeholder="输入邮箱"
                    validation={{
                      status: 'success',
                      message: '邮箱格式正确'
                    }}
                    showCount
                    maxLength={50}
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">自动完成</label>
                  <EnhancedInput
                    placeholder="搜索工单..."
                    options={[
                      { value: '工单-001', label: '工单-001', description: '系统维护工单' },
                      { value: '工单-002', label: '工单-002', description: '网络故障工单' },
                      { value: '工单-003', label: '工单-003', description: '用户权限申请' },
                    ]}
                  />
                </div>
              </div>
            </div>

            {/* 文本域组件 */}
            <div className="space-y-6">
              <h3 className="text-lg font-medium text-gray-700">增强型文本域</h3>
              
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">基础文本域</label>
                  <EnhancedTextArea
                    placeholder="请输入详细描述..."
                    rows={4}
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">带字符计数</label>
                  <EnhancedTextArea
                    placeholder="限制200字符..."
                    rows={4}
                    showCount
                    maxLength={200}
                    placeholderHint="请详细描述问题和需求"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">自动调整高度</label>
                  <EnhancedTextArea
                    placeholder="自动调整高度的文本域..."
                    autoSize={{ minRows: 2, maxRows: 6 }}
                  />
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* 组合示例 */}
        <section className="bg-white rounded-xl p-8 shadow-sm">
          <h2 className="text-2xl font-semibold text-gray-900 mb-6">组合示例</h2>
          
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* 工单表单示例 */}
            <Card title="创建工单" description="填写工单信息">
              <div className="space-y-4">
                <EnhancedInput
                  variant="filled"
                  placeholder="工单标题"
                  validation={{ status: 'success' }}
                />
                <EnhancedTextArea
                  placeholder="工单描述"
                  autoSize={{ minRows: 3, maxRows: 6 }}
                  showCount
                  maxLength={500}
                />
                <div className="flex gap-3 pt-2">
                  <Button variant="primary" size="md">提交工单</Button>
                  <Button variant="ghost" size="md">保存草稿</Button>
                </div>
              </div>
            </Card>

            {/* 搜索框示例 */}
            <Card title="高级搜索" description="多条件搜索">
              <div className="space-y-4">
                <EnhancedInput
                  placeholder="关键词搜索"
                  prefixIcon={<span>🔍</span>}
                  options={[
                    { value: '故障', label: '故障' },
                    { value: '维护', label: '维护' },
                    { value: '升级', label: '升级' },
                  ]}
                />
                <div className="grid grid-cols-2 gap-3">
                  <EnhancedInput placeholder="开始日期" inputType="date" />
                  <EnhancedInput placeholder="结束日期" inputType="date" />
                </div>
                <div className="flex gap-3">
                  <Button variant="primary" size="sm" fullWidth>搜索</Button>
                  <Button variant="outline" size="sm" fullWidth>重置</Button>
                </div>
              </div>
            </Card>
          </div>
        </section>

        {/* 使用指南 */}
        <section className="bg-blue-50 rounded-xl p-8">
          <h2 className="text-2xl font-semibold text-blue-900 mb-6">使用指南</h2>
          
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div className="bg-white rounded-lg p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-3">🎨 设计一致性</h3>
              <p className="text-gray-600">所有组件遵循统一的设计系统，确保视觉一致性和品牌识别度。</p>
            </div>
            
            <div className="bg-white rounded-lg p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-3">📱 移动端优化</h3>
              <p className="text-gray-600">所有组件都经过移动端优化，支持触摸交互和响应式布局。</p>
            </div>
            
            <div className="bg-white rounded-lg p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-3">♿ 可访问性</h3>
              <p className="text-gray-600">组件支持键盘导航、屏幕阅读器和色彩对比度标准。</p>
            </div>
            
            <div className="bg-white rounded-lg p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-3">🔧 高度可定制</h3>
              <p className="text-gray-600">提供丰富的配置选项，满足不同场景的使用需求。</p>
            </div>
            
            <div className="bg-white rounded-lg p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-3">⚡ 性能优化</h3>
              <p className="text-gray-600">组件经过性能优化，支持懒加载和虚拟滚动等特性。</p>
            </div>
            
            <div className="bg-white rounded-lg p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-3">📖 类型安全</h3>
              <p className="text-gray-600">完整的TypeScript类型定义，提供开发时的智能提示。</p>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
};

export default ComponentLibrary;