import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: '登录 - ITSM Platform',
  description: 'IT服务管理平台 - 用户登录',
};

/**
 * 认证路由组布局
 * 用于登录、注册等认证相关页面
 * 提供简洁的全屏布局，无需导航栏和侧边栏
 */
export default function AuthLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <div className='auth-layout'>
      {/* 认证页面不需要额外的布局结构 */}
      {/* 每个认证页面自己控制全屏布局和样式 */}
      {children}
    </div>
  );
}

