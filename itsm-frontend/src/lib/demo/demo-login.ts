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

/**
 * 构建期绑定版本 —— 客户端组件必须调用这一组，不能用上面的默认参数版本。
 *
 * 原因（踩坑记录）：Next.js 的 DefinePlugin 只把「字面的」
 * `process.env.NEXT_PUBLIC_XXX` 成员访问在 next build 阶段替换成字符串常量。
 * 若写成 `readDemoCredentials()` 依赖默认参数 `env = process.env`，编译后是
 * `e.NEXT_PUBLIC_DEMO_USER`（e 为参数，默认 C.env），DefinePlugin 不会内联，
 * 浏览器端 process.env 不含这些键 → 凭据恒为空 → 一键直达永远不触发。
 *
 * 这里对每个变量都用字面 `process.env.NEXT_PUBLIC_*` 访问，确保被静态内联进
 * 公开 bundle；仅把内联后的基本类型值交给纯函数做判定，保持逻辑可单测。
 */
export function readBuiltinDemoCredentials(): DemoCredentials | null {
  return readDemoCredentials({
    NEXT_PUBLIC_DEMO_ENABLED: process.env.NEXT_PUBLIC_DEMO_ENABLED,
    NEXT_PUBLIC_DEMO_USER: process.env.NEXT_PUBLIC_DEMO_USER,
    NEXT_PUBLIC_DEMO_PASSWORD: process.env.NEXT_PUBLIC_DEMO_PASSWORD,
    NEXT_PUBLIC_DEMO_TENANT: process.env.NEXT_PUBLIC_DEMO_TENANT,
  });
}

/** 构建期绑定版本的一键直达判定；用法同上，客户端组件必须调用本函数。 */
export function shouldAutoDemoLoginBuiltin(searchParams: URLSearchParams | null): boolean {
  if (!searchParams || searchParams.get(DEMO_LOGIN_PARAM) !== '1') {
    return false;
  }
  return readBuiltinDemoCredentials() !== null;
}
