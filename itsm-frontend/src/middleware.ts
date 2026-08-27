import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import { isValidJwtToken } from '@/lib/auth/jwt-decoder';

// 需要认证的路由
const protectedRoutes = [
  '/dashboard',
  '/tickets',
  '/incidents',
  '/problems',
  '/changes',
  '/assets',
  '/releases',
  '/licenses',
  '/cmdb',
  '/service-catalog',
  '/knowledge-base',
  '/knowledge',
  '/sla',
  '/sla-dashboard',
  '/sla-monitor',
  '/audit-logs',
  '/email-intake',
  '/reports',
  '/workflow',
  '/users',
  '/settings',
  '/admin',
  '/enterprise',
  '/projects',
  '/applications',
  '/tags',
  '/msp',
  '/ai',
  '/approvals',
  '/improvements',
  '/installations',
  '/marketplace',
  '/my-requests',
  '/notifications',
  '/profile',
  '/service-requests',
  '/standard-changes',
  '/system',
  '/teams',
  '/templates',
  '/agent-ops-demo',
];

// 公开路由（不需要认证）
const publicRoutes = ['/login', '/register', '/forgot-password', '/reset-password'];

// API路由（需要特殊处理）
const apiRoutes = ['/api'];

/**
 * 从请求中获取认证 Token
 * 支持多种方式：Cookie、Authorization Header
 */
function getAuthToken(request: NextRequest): string | null {
  // 1. 权威来源：后端下发的 httpOnly cookie（浏览器访问）
  const accessToken = request.cookies.get('access_token')?.value;
  if (accessToken) {
    return accessToken;
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

  // 4. 历史兼容分支：极旧后端可能把真值 JWT 写入 auth-token cookie。
  //    现行栈中：后端只下发 access_token（httpOnly），前端 auth-service 只写 auth-token=1 标记位，
  //    因此该分支在现行登录流程下不会触发，仅作为遗留后端部署的兜底；切勿当作 JWT 校验入口。
  const legacyToken = request.cookies.get('auth-token')?.value;
  if (legacyToken && legacyToken !== '1') {
    return legacyToken;
  }

  return null;
}

/**
 * 验证 Token 格式是否有效（JWT 应该有3个部分）
 * 委托给 lib/auth/jwt-decoder，使用 atob() 解码 base64url，
 * 避免 Edge Runtime 报 "Code generation from strings disallowed for this context"。
 */
const isValidToken = isValidJwtToken;

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
  const isProtectedRoute = protectedRoutes.some(route => pathname.startsWith(route));

  // 检查是否为公开路由
  const isPublicRoute = publicRoutes.some(route => pathname.startsWith(route));

  // 检查是否为客户端导航（从本应用其他页面导航）
  // 客户端导航时，Referer header会指向本应用的页面
  const referer = request.headers.get('referer');
  const refererUrl = referer ? new URL(referer) : null;
  const isClientSideNavigation = refererUrl && refererUrl.origin === request.nextUrl.origin;

  // 验证 token 格式有效性（JWT 检查）
  const isValid = isValidToken(token);

  // 如果是受保护的路由但没有有效token，重定向到登录页
  if (isProtectedRoute && !isValid) {
    const loginUrl = new URL('/login', request.url);
    loginUrl.searchParams.set('redirect', pathname);
    // 如果 token 存在但无效（过期），标记 expired
    if (token) {
      loginUrl.searchParams.set('expired', 'true');
    }
    return NextResponse.redirect(loginUrl);
  }

  // 如果已登录用户访问公开路由，重定向到仪表盘
  if (isPublicRoute && isValid) {
    return NextResponse.redirect(new URL('/dashboard', request.url));
  }

  // 根路径不再重定向 — 介绍页对所有用户可见
  // 已登录用户访问根路径时，由客户端组件显示"进入系统"按钮

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
