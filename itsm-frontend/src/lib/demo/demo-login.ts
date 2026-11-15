/**
 * 演示环境一键直达（SaaS only）。
 *
 * 设计约束：
 * 1. 官网链接只带开关 `?demo=1`，不带任何凭据 —— URL 里永远看不到密码。
 * 2. demo 凭据在 next build 阶段通过 NEXT_PUBLIC_DEMO_* 注入；开源版这些
 *    变量为空，功能整体关闭，登录页保持原样（不会暴露任何演示账号）。
 * 3. demo 账号是 audit_admin 只读角色（见 internal/authz/roles.go），即使
 *    凭据公开也只能看、不能改，无需担心被滥用写数据。
 *
 * 本文件只放纯函数，方便单测与反向验证；不 import React/antd/next/navigation。
 */

export interface DemoCredentials {
  username: string;
  password: string;
  tenantCode: string;
}

/** URL 参数名：官网一键直达链接形如 https://itsm.cloudmesh.top/login?demo=1 */
export const DEMO_LOGIN_PARAM = 'demo';

/**
 * 从构建期环境变量读取演示凭据。
 * 只有「显式开启 + 用户名密码都非空」才启用；任一缺失即视为未配置，
 * 保证开源版/自建环境默认关闭，且不会用空凭据去撞登录接口。
 */
export function readDemoCredentials(env: Record<string, string | undefined> = process.env): DemoCredentials | null {
  const enabled = env.NEXT_PUBLIC_DEMO_ENABLED === 'true';
  const username = (env.NEXT_PUBLIC_DEMO_USER || '').trim();
  const password = env.NEXT_PUBLIC_DEMO_PASSWORD || '';
  const tenantCode = (env.NEXT_PUBLIC_DEMO_TENANT || 'default').trim();

  if (!enabled || !username || !password) {
    return null;
  }
  return { username, password, tenantCode: tenantCode || 'default' };
}

/**
 * 判断当前登录是否为「一键直达演示」请求。
 * 必须同时满足：URL 带 demo=1 开关、且构建期配置了演示凭据。
 * 单独带参数但没配置凭据（开源版）时不触发，回退到普通登录页。
 */
export function shouldAutoDemoLogin(
  searchParams: URLSearchParams | null,
  env: Record<string, string | undefined> = process.env
): boolean {
  if (!searchParams || searchParams.get(DEMO_LOGIN_PARAM) !== '1') {
    return false;
  }
  return readDemoCredentials(env) !== null;
}
