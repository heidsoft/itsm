import { NextRequest, NextResponse } from 'next/server';

const BACKEND_BASE_URL = process.env.ITSM_BACKEND_URL || 'http://localhost:8090';

// 禁止代理的敏感路径（防止SSRF和路径遍历）
const BLOCKED_PATHS = [
  '/api/v1/admin/users', // 用户管理敏感操作
  '/api/v1/admin/config', // 配置敏感操作
  '/api/v1/system', // 系统敏感操作
];

// 公开路径（无需认证即可访问，用于登录、注册、刷新token等）
const PUBLIC_PATHS = [
  '/api/v1/auth/login',
  '/api/v1/auth/register',
  '/api/v1/auth/refresh',
  '/api/v1/auth/forgot-password',
  '/api/v1/auth/reset-password',
  '/api/v1/auth/sso',
  '/api/v1/health',
  '/api/v1/connectors', // 连接器市场列表（公开）
  '/api/v1/connectors/health', // 连接器健康（公开）
];

function getAuthToken(request: NextRequest): string | null {
  const cookieToken = request.cookies.get('auth-token')?.value;
  if (cookieToken) return cookieToken;

  const authHeader = request.headers.get('Authorization');
  if (authHeader) {
    if (authHeader.startsWith('Bearer ')) return authHeader.substring(7);
    return authHeader;
  }

  const customToken = request.headers.get('X-Auth-Token');
  if (customToken) return customToken;

  const accessToken = request.cookies.get('access_token')?.value;
  if (accessToken) return accessToken;

  return null;
}

function isValidToken(token: string | null): boolean {
  if (!token) return false;
  const parts = token.split('.');
  if (parts.length !== 3) return false;
  try {
    const payload = JSON.parse(Buffer.from(parts[1], 'base64').toString('utf-8'));
    if (payload.exp) {
      const currentTime = Math.floor(Date.now() / 1000);
      if (payload.exp < currentTime) return false;
    }
    return true;
  } catch {
    return false;
  }
}

function isPathBlocked(path: string[]): boolean {
  const fullPath = '/' + path.join('/');
  return BLOCKED_PATHS.some(blocked => fullPath.startsWith(blocked));
}

async function proxyRequest(request: NextRequest, params: Promise<{ path: string[] }>) {
  const { path } = await params;

  // 公开路径直接放行（不需要 token）
  // path 数组来自 [...path] 捕获，不含 /api 前缀
  // 例如请求 /api/v1/auth/login 时 path = ['v1', 'auth', 'login']
  const fullPath = '/api/' + path.join('/');
  if (PUBLIC_PATHS.some(p => fullPath === p || fullPath.startsWith(p + '/'))) {
    // 跳过认证检查，继续代理
  } else {
  // 认证检查
  const token = getAuthToken(request);
  if (!isValidToken(token)) {
    return NextResponse.json(
      { code: 2001, message: 'Unauthorized: authentication required' },
      { status: 401 }
    );
  }
  }

  // 敏感路径检查
  if (isPathBlocked(path)) {
    return NextResponse.json(
      { code: 2003, message: 'Forbidden: this endpoint cannot be accessed through the proxy' },
      { status: 403 }
    );
  }

  const backendURL = new URL(`/api/${path.join('/')}`, BACKEND_BASE_URL);
  backendURL.search = request.nextUrl.search;

  const headers = new Headers(request.headers);
  headers.delete('host');
  headers.delete('content-length');

  const init: RequestInit = {
    method: request.method,
    headers,
    redirect: 'manual',
  };

  if (!['GET', 'HEAD'].includes(request.method)) {
    init.body = await request.text();
  }

  try {
    const response = await fetch(backendURL, init);
    const responseHeaders = new Headers(response.headers);
    responseHeaders.delete('content-encoding');
    responseHeaders.delete('content-length');

    return new NextResponse(response.body, {
      status: response.status,
      headers: responseHeaders,
    });
  } catch {
    return NextResponse.json({ code: 5001, message: 'Backend request failed' }, { status: 500 });
  }
}

type RouteContext = { params: Promise<{ path: string[] }> };

export async function GET(request: NextRequest, context: RouteContext) {
  return proxyRequest(request, context.params);
}

export async function POST(request: NextRequest, context: RouteContext) {
  return proxyRequest(request, context.params);
}

export async function PUT(request: NextRequest, context: RouteContext) {
  return proxyRequest(request, context.params);
}

export async function PATCH(request: NextRequest, context: RouteContext) {
  return proxyRequest(request, context.params);
}

export async function DELETE(request: NextRequest, context: RouteContext) {
  return proxyRequest(request, context.params);
}

export const dynamic = 'force-dynamic';
