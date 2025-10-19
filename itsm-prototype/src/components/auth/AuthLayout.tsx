'use client';

import React from 'react';
import { Server, BarChart3, Shield, Zap, Sparkles } from 'lucide-react';
import { cn } from '@/lib/utils';

/**
 * 认证页面布局组件
 * 提供统一的认证页面布局，包括品牌区域和表单区域
 */
export interface AuthLayoutProps {
  children: React.ReactNode;
  title?: string;
  subtitle?: string;
  showBranding?: boolean;
}

export const AuthLayout: React.FC<AuthLayoutProps> = ({
  children,
  title = 'ITSM Pro',
  subtitle = '智能IT服务管理平台',
  showBranding = true,
}) => {
  return (
    <div
      className='min-h-screen flex'
      style={{
        background: `linear-gradient(135deg, #f8fafc 0%, #e0e7ff 100%)`,
      }}
    >
      {/* 左侧品牌区域 */}
      {showBranding && (
        <div className='hidden lg:flex lg:w-1/2 relative overflow-hidden'>
          {/* 背景装饰 - 使用设计系统颜色 */}
          <div
            className='absolute inset-0'
            style={{
              background: `linear-gradient(135deg, #1890ff 0%, #2563eb 50%, #1d4ed8 100%)`,
            }}
          ></div>
          <div className='absolute inset-0 bg-black/20'></div>

          {/* 装饰性几何图形 */}
          <div className='absolute top-20 left-20 w-32 h-32 bg-white/10 rounded-full blur-xl'></div>
          <div className='absolute bottom-40 right-20 w-48 h-48 bg-white/5 rounded-full blur-2xl'></div>
          <div className='absolute top-1/2 left-1/3 w-24 h-24 bg-yellow-400/20 rounded-full blur-lg'></div>

          {/* 主要内容 */}
          <div className='relative z-10 flex flex-col justify-center px-16 py-12 text-white'>
            {/* Logo和标题 */}
            <div className='mb-12'>
              <div className='flex items-center mb-6'>
                <div
                  className='w-16 h-16 rounded-2xl flex items-center justify-center mr-4'
                  style={{ backgroundColor: 'rgba(255,255,255,0.2)' }}
                >
                  <Server className='w-8 h-8 text-white' />
                </div>
                <div>
                  <h1 className='text-3xl font-bold'>{title}</h1>
                  <p className='text-white/80 text-lg'>{subtitle}</p>
                </div>
              </div>
            </div>

            {/* 特性介绍 */}
            <div className='space-y-6 mb-12'>
              <div className='flex items-center space-x-4'>
                <div className='w-12 h-12 bg-white/10 rounded-lg flex items-center justify-center'>
                  <BarChart3 className='w-6 h-6 text-white' />
                </div>
                <div>
                  <h3 className='text-lg font-semibold'>智能分析</h3>
                  <p className='text-white/70'>AI驱动的数据分析和预测</p>
                </div>
              </div>

              <div className='flex items-center space-x-4'>
                <div className='w-12 h-12 bg-white/10 rounded-lg flex items-center justify-center'>
                  <Shield className='w-6 h-6 text-white' />
                </div>
                <div>
                  <h3 className='text-lg font-semibold'>安全可靠</h3>
                  <p className='text-white/70'>企业级安全防护和权限管理</p>
                </div>
              </div>

              <div className='flex items-center space-x-4'>
                <div className='w-12 h-12 bg-white/10 rounded-lg flex items-center justify-center'>
                  <Zap className='w-6 h-6 text-white' />
                </div>
                <div>
                  <h3 className='text-lg font-semibold'>高效协作</h3>
                  <p className='text-white/70'>团队协作和流程自动化</p>
                </div>
              </div>
            </div>

            {/* 底部信息 */}
            <div className='flex items-center justify-between text-sm text-white/60'>
              <div className='flex items-center space-x-4'>
                <span>© 2024 ITSM Pro</span>
                <span>•</span>
                <span>企业级服务</span>
              </div>
              <div className='flex items-center space-x-2'>
                <Sparkles className='w-4 h-4' />
                <span>持续创新</span>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 右侧表单区域 */}
      <div
        className={`flex-1 flex items-center justify-center px-6 py-12 lg:px-8 ${
          showBranding ? '' : 'w-full'
        }`}
      >
        <div className='w-full max-w-md'>{children}</div>
      </div>
    </div>
  );
};

/**
 * 认证卡片组件
 * 提供统一的认证表单卡片样式
 */
export interface AuthCardProps {
  children: React.ReactNode;
  title?: string;
  subtitle?: string;
  showMobileLogo?: boolean;
  /** 卡片类型 */
  variant?: 'default' | 'elevated' | 'outlined' | 'filled';
  /** 卡片尺寸 */
  size?: 'sm' | 'md' | 'lg';
  /** 是否显示边框 */
  bordered?: boolean;
  /** 是否可悬停 */
  hoverable?: boolean;
  /** 自定义样式 */
  style?: React.CSSProperties;
  /** 自定义类名 */
  className?: string;
  /** 头部额外内容 */
  extra?: React.ReactNode;
  /** 底部内容 */
  footer?: React.ReactNode;
  /** 是否加载中 */
  loading?: boolean;
}

export const AuthCard: React.FC<AuthCardProps> = ({
  children,
  title,
  subtitle,
  showMobileLogo = true,
  variant = 'default',
  size = 'md',
  bordered = true,
  hoverable = false,
  style,
  className,
  extra,
  footer,
  loading = false,
}) => {
  // 卡片变体样式
  const getVariantStyles = () => {
    switch (variant) {
      case 'elevated':
        return {
          backgroundColor: '#ffffff',
          boxShadow: '0 10px 15px -3px rgb(0 0 0 / 0.08)',
          border: 'none',
        };
      case 'outlined':
        return {
          backgroundColor: '#ffffff',
          boxShadow: 'none',
          border: `2px solid #e2e8f0`,
        };
      case 'filled':
        return {
          backgroundColor: '#f1f5f9',
          boxShadow: 'none',
          border: 'none',
        };
      default:
        return {
          backgroundColor: '#ffffff',
          boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.08)',
          border: `1px solid #e2e8f0`,
        };
    }
  };

  // 卡片尺寸样式
  const getSizeStyles = () => {
    switch (size) {
      case 'sm':
        return {
          padding: 16,
          borderRadius: 6,
        };
      case 'lg':
        return {
          padding: 24,
          borderRadius: 8,
        };
      default:
        return {
          padding: 24,
          borderRadius: 8,
        };
    }
  };

  // 加载骨架屏
  const LoadingSkeleton = () => (
    <div className='animate-pulse'>
      <div className='h-4 bg-gray-200 rounded w-3/4 mb-4'></div>
      <div className='space-y-3'>
        <div className='h-3 bg-gray-200 rounded w-full'></div>
        <div className='h-3 bg-gray-200 rounded w-5/6'></div>
        <div className='h-3 bg-gray-200 rounded w-4/6'></div>
      </div>
    </div>
  );

  return (
    <>
      {/* 移动端Logo */}
      {showMobileLogo && (
        <div className='lg:hidden text-center mb-8'>
          <div
            className='inline-flex items-center justify-center w-16 h-16 rounded-2xl mb-4'
            style={{ backgroundColor: '#1890ff' }}
          >
            <Server className='w-8 h-8 text-white' />
          </div>
          <h1 className='text-2xl font-bold' style={{ color: '#1e293b' }}>
            ITSM Pro
          </h1>
          <p style={{ color: '#64748b' }}>智能IT服务管理平台</p>
        </div>
      )}

      {/* 认证卡片 */}
      <div
        className={cn(
          'transition-all duration-200',
          hoverable && 'hover:shadow-lg hover:scale-[1.02] cursor-pointer',
          className
        )}
        style={{
          ...getVariantStyles(),
          ...getSizeStyles(),
          ...style,
        }}
      >
        {/* 卡片头部 */}
        {(title || subtitle || extra) && (
          <div className='mb-6'>
            <div className='flex items-center justify-between mb-2'>
              {title && (
                <h2 className='text-xl font-semibold' style={{ color: '#1e293b' }}>
                  {title}
                </h2>
              )}
              {extra && <div className='ml-4'>{extra}</div>}
            </div>
            {subtitle && <p style={{ color: '#64748b' }}>{subtitle}</p>}
          </div>
        )}

        {/* 卡片内容 */}
        <div className='relative'>{loading ? <LoadingSkeleton /> : children}</div>

        {/* 卡片底部 */}
        {footer && (
          <div className='mt-6 pt-4 border-t' style={{ borderColor: '#e2e8f0' }}>
            {footer}
          </div>
        )}
      </div>
    </>
  );
};

export default AuthLayout;
