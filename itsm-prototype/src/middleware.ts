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
 * 从请求中获取认证 Token
 * 支持多种方式：Cookie、Authorization Header
 */
function getAuthToken(request: NextRequest): string | null {
  // 1. 优先从 Cookie 读取（浏览器访问）
  const cookieToken = request.cookies.get('auth-token')?.value;
  if (cookieToken) {
    return cookieToken;
  }

  // 2. 从 Authorization Header 读取（API 调用）
  const authHeader = request.headers.get('Authorization');
  if (authHeader) {
    // 支持 Bearer Token 格式
    if (authHeader.startsWith('Bearer ')) {
      return authHeader.substring(7);
    }
    // 支持直接传递 Token
    return authHeader;
  }

  // 3. 从自定义 Header 读取
  const customToken = request.headers.get('X-Auth-Token');
  if (customToken) {
    return customToken;
  }

  // 4. 兼容旧的 access_token cookie
  const accessToken = request.cookies.get('access_token')?.value;
  if (accessToken) {
    return accessToken;
  }

  return null;
}

/**
 * Next.js 中间件
 * 处理路由保护和认证检查
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const token = getAuthToken(request);

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