/**
 * AI 工具审批按钮的权限码一致性（回归锁）。
 *
 * 后端 `router/ai_routes.go` 的 `POST /agent/tools/:id/approve` 挂的是
 * `RequirePermission("ai", "approve")`，即 **ai:approve**。该码在 2026-11-17
 * 从 ai:write 拆出并只授管理与一线角色（`authz/catalog.go:169` 明写「ai:approve
 * 只授管理/一线」，最终用户不得审批 AI 代人写操作）。
 *
 * 前端页面曾用 `PERM.ai.write` 判按钮可见性，后果是双向错位：
 *   - 持 ai:approve 的审批人（一线/管理）如果只有 approve 而无 write，看不到按钮；
 *   - 只有 ai:write 的账号看到按钮，点下去被后端 403。
 *
 * 本测试把前端**源码里**的判定表达式与后端**路由常量**对齐：不是断言字符串出现，
 * 而是分别解析两侧，再断言两者取的是同一个码。任一侧被改单边，断言即红。
 */

import fs from 'node:fs';
import path from 'node:path';

const FRONTEND_ROOT = process.cwd();
const REPO_ROOT = path.resolve(FRONTEND_ROOT, '..');
const PAGE = path.join(FRONTEND_ROOT, 'src/app/(main)/ai/approval/page.tsx');
const ROUTES = path.join(REPO_ROOT, 'itsm-backend/router/ai_routes.go');

/** 从后端路由文件里取 approve 端点挂的权限对（resource, action）。 */
function backendApproveCode(): string {
  const src = fs.readFileSync(ROUTES, 'utf8');
  // 端点行：agentGrp.POST("/tools/:id/approve", middleware.RequirePermission("ai", "approve"), ...
  const line = src
    .split('\n')
    .find(l => l.includes('/tools/:id/approve') && l.includes('RequirePermission'));
  if (!line) throw new Error('未找到 approve 路由行，无法校验权限码');
  const m = /RequirePermission\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)/.exec(line);
  if (!m) throw new Error(`approve 路由行未使用 RequirePermission 字面量：${line.trim()}`);
  return `${m[1]}:${m[2]}`;
}

/** 从页面源码里取 canApprove 实际使用的 PERM 路径。 */
function frontendApproveCode(): string {
  const src = fs.readFileSync(PAGE, 'utf8');
  const m = /const\s+canApprove\s*=\s*hasPermission\(\s*PERM\.([A-Za-z0-9_.]+)\s*\)/.exec(src);
  if (!m) throw new Error('未找到 canApprove 的 hasPermission 判定，无法校验权限码');
  return m[1].split('.').join(':');
}

describe('AI 工具审批权限码（前后端同码）', () => {
  it('页面判定用的权限码等于后端 approve 路由的权限码', () => {
    expect(frontendApproveCode()).toBe(backendApproveCode());
  });

  it('后端 approve 路由确实挂的是 ai:approve（不是 ai:write）', () => {
    // 显式锚定一次：即使有人「同步把两侧都改成 ai:write」，这条也会红——
    // 因为 catalog/roles 的拆分决策不该被回退。
    expect(backendApproveCode()).toBe('ai:approve');
  });

  it('页面里不得再出现「需 ai:write」这类与实际码不符的提示文案', () => {
    const src = fs.readFileSync(PAGE, 'utf8');
    const hits = src
      .split('\n')
      .filter(l => !/^\s*(\/\/|\*|\/\*)/.test(l))
      .filter(l => l.includes('ai:write'));
    expect(hits).toEqual([]);
  });
});
