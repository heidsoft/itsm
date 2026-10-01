import type { NextRequest} from 'next/server';
import { NextResponse } from 'next/server';

const BACKEND_BASE_URL = process.env.ITSM_BACKEND_URL || 'http://localhost:8090';

// 禁止代理的敏感路径（防止SSRF和路径遍历）
const BLOCKED_PATHS = [
  '/api/v1/admin/users', // 用户管理敏感操作
  '/api/v1/admin/config', // 配置敏感操作
  '/api/v1/system', // 系统敏感操作
];

function isPathBlocked(path: string[]): boolean {
  const fullPath = '/' + path.join('/');
  return BLOCKED_PATHS.some(blocked => fullPath.startsWith(blocked));
}

async function proxyRequest(request: NextRequest, params: Promise<{ path: string[] }>) {
  const { path } = await params;

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
  // host/content-length are connection-specific and must be recomputed by the
  // backend's HTTP client.
  headers.delete('host');
  headers.delete('content-length');

  // Preserve client IP across the Next.js → backend hop so backend audit logs
  // (Fix #5) can record the real browser IP instead of the frontend container IP.
  // Without this, requests that hit `localhost:3000` (frontend) instead of
  // `localhost:80` (nginx) leak itsm-frontend's 172.28.0.7 into audit logs.
  // Priority chain (most specific wins):
  //   1. X-Forwarded-For from upstream proxy (e.g. nginx) — trusted
  //   2. X-Real-IP from upstream proxy — trusted
  //   3. request.ip / request.socket.remoteAddress — direct TCP source
  // The backend's trusted-proxies list includes the frontend container's CIDR
  // (RFC1918), so the chain is honored end-to-end.
  const upstreamXFF = request.headers.get('x-forwarded-for')?.split(',')[0]?.trim();
  const upstreamRealIP = request.headers.get('x-real-ip')?.trim();
  // Try multiple paths to the real TCP source IP. Next.js's NextRequest may
  // expose it as `.ip` (Edge runtime), or we may need to reach into the
  // underlying Node IncomingMessage via the request meta store.
  const reqAny = request as unknown as Record<string, unknown>;
  const tcpIP =
    (typeof reqAny.ip === 'string' && reqAny.ip) ||
    (typeof (reqAny as { socket?: { remoteAddress?: string } }).socket?.remoteAddress === 'string'
      ? (reqAny as { socket?: { remoteAddress?: string } }).socket?.remoteAddress
      : undefined) ||
    // Underlying Node IncomingMessage is exposed by Next.js as `originalRequest`
    // (internal; may be renamed across versions). Walk all string-typed props
    // that look like IPs to be robust against renames.
    (() => {
      for (const k of Object.keys(reqAny)) {
        const v = (reqAny as Record<string, unknown>)[k];
        if (typeof v === 'string' && /^\d+\.\d+\.\d+\.\d+$/.test(v)) return v;
        if (v && typeof v === 'object') {
          const sock = (v as { remoteAddress?: unknown }).remoteAddress;
          if (typeof sock === 'string') return sock;
        }
      }
      return undefined;
    })();
  const clientIP = upstreamXFF || upstreamRealIP || tcpIP || '';
  if (clientIP) {
    const existingXFF = request.headers.get('x-forwarded-for');
    // Append tcpIP if it's not already in the chain (dedupe by exact match).
    const ips = existingXFF
      ? existingXFF.split(',').map(s => s.trim()).filter(Boolean)
      : [];
    if (tcpIP && !ips.includes(tcpIP) && clientIP === tcpIP) {
      ips.push(tcpIP);
    }
    if (ips.length === 0) ips.push(clientIP);
    headers.set('x-forwarded-for', ips.join(', '));
    if (!headers.has('x-real-ip')) {
      headers.set('x-real-ip', clientIP);
    }
  }
  headers.set('x-forwarded-proto', request.headers.get('x-forwarded-proto') || request.nextUrl.protocol.replace(':', ''));
  headers.set('x-forwarded-host', request.headers.get('host') || '');

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

    // 修复 Set-Cookie 头丢失问题：
    // fetch() 返回的 Headers 对象按规范会过滤掉 Set-Cookie 头（防止 XSS 窃取），
    // 但 NextResponse 直接构造时可以重新写入。Node.js 18+ 提供 getSetCookie() 获取原始数组。
    // 必须转发后端 Set-Cookie（如 access_token/refresh_token httpOnly cookie），
    // 否则登录后浏览器无法收到 cookie，导致后续请求 401。
    const setCookies = (response.headers as unknown as { getSetCookie?: () => string[] }).getSetCookie?.() ?? [];
    if (setCookies.length > 0) {
      // 先删除可能从 Headers 复制过来的单个 Set-Cookie（值为合并字符串，浏览器无法解析）
      responseHeaders.delete('set-cookie');
      for (const cookie of setCookies) {
        responseHeaders.append('set-cookie', cookie);
      }
    }

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
