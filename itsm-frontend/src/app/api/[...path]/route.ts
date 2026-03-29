import { NextRequest, NextResponse } from 'next/server';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8090';

// 禁用代理的配置
const fetchOptions: RequestInit = {
  credentials: 'include',
};

// 通过环境变量禁用代理
if (process.env.NO_PROXY || process.env.no_proxy) {
  // 确保本地地址不走代理
}

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const { path } = await params;
  const url = `${API_BASE_URL}/${path.join('/')}`;
  const searchParams = request.nextUrl.searchParams.toString();
  const fullUrl = searchParams ? `${url}?${searchParams}` : url;

  const cookieHeader = request.headers.get('cookie') || '';
  // Get auth from header first, then fallback to localStorage (for cross-origin proxy)
  const authHeader = request.headers.get('Authorization') ||
    (request.cookies.get('access_token')?.value ? `Bearer ${request.cookies.get('access_token').value}` : '');

  try {
    const response = await fetch(fullUrl, {
      ...fetchOptions,
      method: 'GET',
      headers: {
        'Cookie': cookieHeader,
        'Authorization': authHeader,
        'Content-Type': 'application/json',
      },
    });

    const data = await response.json();
    const newCookies = response.headers.get('set-cookie');

    const headers = new Headers();
    if (newCookies) {
      headers.set('Set-Cookie', newCookies);
    }

    return NextResponse.json(data, { status: response.status, headers });
  } catch (error) {
    return NextResponse.json(
      { code: 500, message: 'Backend request failed' },
      { status: 500 }
    );
  }
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const { path } = await params;
  const url = `${API_BASE_URL}/${path.join('/')}`;

  const cookieHeader = request.headers.get('cookie') || '';
  const body = await request.json();

  try {
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Cookie': cookieHeader,
        'Authorization': request.headers.get('Authorization') || '',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
      credentials: 'include',
    });

    const data = await response.json();
    const newCookies = response.headers.get('set-cookie');

    const headers = new Headers();
    if (newCookies) {
      headers.set('Set-Cookie', newCookies);
    }

    return NextResponse.json(data, { status: response.status, headers });
  } catch (error) {
    return NextResponse.json(
      { code: 500, message: 'Backend request failed' },
      { status: 500 }
    );
  }
}

export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const { path } = await params;
  const url = `${API_BASE_URL}/${path.join('/')}`;

  const cookieHeader = request.headers.get('cookie') || '';
  const body = await request.json();

  try {
    const response = await fetch(url, {
      method: 'PUT',
      headers: {
        'Cookie': cookieHeader,
        'Authorization': request.headers.get('Authorization') || '',
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
      credentials: 'include',
    });

    const data = await response.json();
    const newCookies = response.headers.get('set-cookie');

    const headers = new Headers();
    if (newCookies) {
      headers.set('Set-Cookie', newCookies);
    }

    return NextResponse.json(data, { status: response.status, headers });
  } catch (error) {
    return NextResponse.json(
      { code: 500, message: 'Backend request failed' },
      { status: 500 }
    );
  }
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const { path } = await params;
  const url = `${API_BASE_URL}/${path.join('/')}`;

  const cookieHeader = request.headers.get('cookie') || '';

  try {
    const response = await fetch(url, {
      method: 'DELETE',
      headers: {
        'Cookie': cookieHeader,
        'Authorization': request.headers.get('Authorization') || '',
        'Content-Type': 'application/json',
      },
      credentials: 'include',
    });

    const data = await response.json();
    const newCookies = response.headers.get('set-cookie');

    const headers = new Headers();
    if (newCookies) {
      headers.set('Set-Cookie', newCookies);
    }

    return NextResponse.json(data, { status: response.status, headers });
  } catch (error) {
    return NextResponse.json(
      { code: 500, message: 'Backend request failed' },
      { status: 500 }
    );
  }
}