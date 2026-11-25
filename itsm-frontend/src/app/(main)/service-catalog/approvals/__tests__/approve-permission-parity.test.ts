/**
 * 服务请求审批按钮的权限码一致性（回归锁）。
 *
 * 后端 `router/service_request_routes.go` 的 `POST /service-requests/:id/approval`
 * 挂的是 `RequirePermission("service_request", "approve")`，即
 * **service_request:approve**。而「目录审批」这条菜单的准入码是
 * `service_request:read`（见 pkg/menubaseline/baseline.go）。
 *
 * 两者不是同一个码：service_catalog_admin / technician / end_user / audit_admin
 * 只持 read 不持 approve。页面曾无条件渲染审批/驳回按钮，结果是这些账号能进
 * 列表、点下去必被后端 403，且失败提示只有一句「审批失败」，看不出是权限问题。
 *
 * 本测试分别解析两侧源码再断言同码：单边改动（无论改路由还是改页面判定）即红。
 */

import fs from 'node:fs';
import path from 'node:path';

const FRONTEND_ROOT = process.cwd();
const REPO_ROOT = path.resolve(FRONTEND_ROOT, '..');
const PAGE = path.join(FRONTEND_ROOT, 'src/app/(main)/service-catalog/approvals/page.tsx');
const ROUTES = path.join(REPO_ROOT, 'itsm-backend/router/service_request_routes.go');

/** 从后端路由文件里取审批端点挂的权限对（resource, action）。 */
function backendApproveCode(): string {
  const src = fs.readFileSync(ROUTES, 'utf8');
  // 端点行：sr.POST("/:id/approval", middleware.RequirePermission("service_request", "approve"), ...
  const line = src
    .split('\n')
    .find(l => l.includes('"/:id/approval"') && l.includes('RequirePermission'));
  if (!line) throw new Error('未找到服务请求审批路由行，无法校验权限码');
  const m = /RequirePermission\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)/.exec(line);
  if (!m) throw new Error(`审批路由行未使用 RequirePermission 字面量：${line.trim()}`);
  return `${m[1]}:${m[2]}`;
}

/** 从页面源码里取 canApprove 实际使用的 PERM 路径。 */
function frontendApproveCode(): string {
  const src = fs.readFileSync(PAGE, 'utf8');
  const m = /const\s+canApprove\s*=\s*hasPermission\(\s*PERM\.([A-Za-z0-9_.]+)\s*\)/.exec(src);
  if (!m) throw new Error('未找到 canApprove 的 hasPermission 判定，无法校验权限码');
  return m[1].split('.').join(':');
}

/** 源码里除注释外的有效行，避免注释中出现的码误伤断言。 */
function codeLines(file: string): string[] {
  return fs
    .readFileSync(file, 'utf8')
    .split('\n')
    .filter(l => !/^\s*(\/\/|\*|\/\*)/.test(l));
}

describe('服务请求审批权限码（前后端同码）', () => {
  it('页面判定用的权限码等于后端审批路由的权限码', () => {
    expect(frontendApproveCode()).toBe(backendApproveCode());
  });

  it('后端审批路由确实挂的是 service_request:approve', () => {
    // 显式锚定：即使有人「同步把两侧都改成 read」，这条也会红——
    // 审批动作与只读动作的分离是 catalog 里的既定决策，不该被回退。
    expect(backendApproveCode()).toBe('service_request:approve');
  });

  it('审批与驳回按钮都受 canApprove 门控（不能只门控其中一个）', () => {
    const src = fs.readFileSync(PAGE, 'utf8');
    // 两个按钮的 JSX 块都必须在 `{canApprove && (` 之内。
    const approveBlocks = src.match(/canApprove\s*&&\s*\(/g) ?? [];
    expect(approveBlocks.length).toBeGreaterThanOrEqual(2);
  });

  it('无审批权限时要显式说明原因，而不是让按钮消失得无声无息', () => {
    const lines = codeLines(PAGE);
    const hint = lines.filter(l => l.includes('没有审批权限') || l.includes('service_request:approve'));
    expect(hint.length).toBeGreaterThan(0);
  });
});
