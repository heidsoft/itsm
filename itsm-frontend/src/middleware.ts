import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

/**
 * 历史遗留菜单路径 → 正确路由的映射表
 * 用于覆盖以下场景（Sidebar 客户端点击不会触发 middleware，只在这些场景触发）：
 * - 用户手动输入 URL / 直接访问书签
 * - 硬刷新页面（此时 Next.js 走服务端路由匹配）
 * - 其他站内链接通过 <a href> 的原生导航
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
 * Next.js 中间件
 *
 * 只做 URL 修正，不做登录判断。边缘层无法校验会话：cookie 是 httpOnly 的，
 * 这里既拿不到凭证内容也无法确认它未被吊销，历史上靠 JWT 字符串形状判定「已登录」
 * 会把过期凭证当成有效会话，也会让 /login 页在真实会话存在时被弹走。
 * 会话真相由布局层调用 GET /api/v1/auth/session 判定并负责跳转 /login。
 */
export function middleware(request: NextRequest) {
  const { pathname, search } = request.nextUrl;

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

  return NextResponse.next();
}

export const config = {
  matcher: ['/((?!api|_next/static|_next/image|favicon.ico|public).*)'],
};
