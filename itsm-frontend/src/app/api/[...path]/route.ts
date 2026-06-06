import { NextRequest, NextResponse } from 'next/server';

const BACKEND_BASE_URL = process.env.ITSM_BACKEND_URL || 'http://localhost:8090';

async function proxyRequest(request: NextRequest, params: Promise<{ path: string[] }>) {
  const { path } = await params;
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
