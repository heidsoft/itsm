import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

// 需要认证的路由
const protectedRoutes = [
  '/dashboard',
  '/tickets',
  '/incidents',
  '/problems',
  '/changes',
  '/assets',
  '/users',
  '/settings',
  '/reports',
];

// 公开路由（不需要认证）
const publicRoutes = [
  '/login',
  '/register',
  '/forgot-password',
  '/reset-password',
];

// API路由（需要特殊处理）
const apiRoutes = [
  '/api',
];

/**
 * Next.js 中间件
 * 处理路由保护和认证检查
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const token = request.cookies.get('auth-token')?.value;

  // 检查是否为API路由
  if (apiRoutes.some(route => pathname.startsWith(route))) {
    // API路由的认证检查由后端处理
    return NextResponse.next();
  }

  // 检查是否为受保护的路由
  const isProtectedRoute = protectedRoutes.some(route => 
    pathname.startsWith(route)
  );

  // 检查是否为公开路由
  const isPublicRoute = publicRoutes.some(route => 
    pathname.startsWith(route)
  );

  // 如果是受保护的路由但没有token，重定向到登录页
  if (isProtectedRoute && !token) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    return NextResponse.redirect(loginUrl);
  }

  // 如果已登录用户访问公开路由，重定向到仪表盘
  if (isPublicRoute && token) {
    return NextResponse.redirect(new URL('/dashboard', request.url));
  }

  // 根路径重定向
  if (pathname === '/') {
    if (token) {
      return NextResponse.redirect(new URL('/dashboard', request.url));
    } else {
      return NextResponse.redirect(new URL('/login', request.url));
    }
  }

  return NextResponse.next();
}

// 配置中间件匹配的路径
export const config = {
  matcher: [
    /*
     * 匹配所有路径除了:
     * - api (API routes)
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico (favicon file)
     * - public folder
     */
    '/((?!api|_next/static|_next/image|favicon.ico|public).*)',
  ],
};