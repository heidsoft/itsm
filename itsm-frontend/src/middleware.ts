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
 * 历史遗留菜单路径 → 正确路由的映射表
 * 用于覆盖以下场景（Sidebar 客户端点击不会触发 middleware，只在这些场景触发）：
 * - 用户手动输入 URL / 直接访问书签
 * - 硬刷新页面（此时 Next.js 走服务端路由匹配）
 * - 其他站内链接通过 <a href> 的原生导航
 * 顺序：先做路径重定向，再做 auth 保护 —— 这样未登录用户会被先 307 到正确路径，
 * 然后在下一轮中间件检查后再跳 /login（redirect 参数会指向正确 URL）。
 */
const LEGACY_MENU_REDIRECTS: Record<string, string> = {
  // /list 后缀 → 模块根路径（App Router 下 xxx/page.tsx 即列表首页）
  '/service-requests/list': '/service-requests',
  '/incidents/list': '/incidents',
  '/problems/list': '/problems',
  '/changes/list': '/changes',
  '/knowledge/list': '/knowledge',
  '/service-catalog/list': '/service-catalog',
  '/assets/list': '/assets',
  '/workflow/list': '/workflow',
  '/ai/chat/list': '/ai/chat',
  '/msp/list': '/msp',
  '/releases/list': '/releases',
  // 命名错误：/admin/overview 页面加载后会客户端跳转到 /admin（系统管理首页），直接指向 /admin 避免两跳
  '/admin/index': '/admin',
  '/knowledge/articles/create': '/knowledge/articles/new',
  // 缺少独立页面的入口（模块主页本身就是概览/会话首页）
  '/sla/overview': '/sla',
  '/email-intake/conversations': '/email-intake',
  '/knowledge/articles': '/knowledge',
};

// 兜底：对任意 /xxx/list 路径（且不在显式映射中）也尝试剥离 /list
function tryStripListSuffix(pathname: string): string | null {
  if (pathname.endsWith('/list') && pathname.length > 6) {
    return pathname.slice(0, -5) || '/';
  }
  return null;
}

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
  const { pathname, search } = request.nextUrl;
  const token = getAuthToken(request);

  // 检查是否为API路由
  if (apiRoutes.some(route => pathname.startsWith(route))) {
    // API路由的认证检查由后端处理
    return NextResponse.next();
  }

  // ============== 第一层：历史遗留路径 307 重定向 ==============
  // 先做路径修正，再做 auth 保护，保证 bookmark/refresh/direct-URL 也能到达正确页面
  const exactRedirect = LEGACY_MENU_REDIRECTS[pathname];
  let correctedPath: string | null = exactRedirect ?? null;
  if (!correctedPath) {
    correctedPath = tryStripListSuffix(pathname);
  }
  if (correctedPath && correctedPath !== pathname) {
    // 保留原始 query string（例如 token / redirect / filter 等）
    const dest = new URL(correctedPath + search, request.url);
    return NextResponse.redirect(dest, 307);
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
