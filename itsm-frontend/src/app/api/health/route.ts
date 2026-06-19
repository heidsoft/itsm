import { NextResponse } from 'next/server';

/**
 * 前端健康检查端点
 * GET /api/health
 * 返回应用运行状态、时间戳和版本信息
 */
export async function GET() {
  return NextResponse.json({
    status: 'ok',
    timestamp: new Date().toISOString(),
    version: '1.0.0',
  });
}
