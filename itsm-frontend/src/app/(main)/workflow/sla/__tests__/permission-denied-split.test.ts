/**
 * 流程 SLA 监控页的「违规数据无权限」分流（回归锁）。
 *
 * 这一页的菜单准入码是 `bpmn:read`，但违规列表来自 SLA 域的
 * `GET /api/v1/sla/violations`，该端点挂的是 `sla:read`
 * （router/sla_routes.go:45，rbac_precheck_gen.go:430）。
 *
 * technician 与 security_admin 持 bpmn:read 却不持 sla:read，于是能进页面、
 * 违规区永远加载失败。此前失败一律渲染成「SLA 违规数据加载失败，请稍后重试」
 * ——对一个权限问题给出「重试」的建议，用户会一直点刷新。
 *
 * 本测试锁三件事：① 页面确实用 isPermissionDeniedError 分流；
 * ② 无权限分支的文案明确提到权限码，而不是误导性地说「稍后重试」；
 * ③ 两种失败的 Alert 类型不同（warning vs error），肉眼可区分。
 */

import fs from 'node:fs';
import path from 'node:path';

const FRONTEND_ROOT = process.cwd();
const REPO_ROOT = path.resolve(FRONTEND_ROOT, '..');
const PAGE = path.join(FRONTEND_ROOT, 'src/app/(main)/workflow/sla/page.tsx');
const ROUTES = path.join(REPO_ROOT, 'itsm-backend/router/sla_routes.go');

/** 后端违规列表端点的权限码。 */
function backendViolationCode(): string {
  const src = fs.readFileSync(ROUTES, 'utf8');
  const line = src
    .split('\n')
    .find(l => l.includes('/violations"') && l.includes('RequirePermission') && l.includes('GET'));
  if (!line) throw new Error('未找到 GET /violations 路由行');
  const m = /RequirePermission\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)/.exec(line);
  if (!m) throw new Error(`路由行未使用 RequirePermission 字面量：${line.trim()}`);
  return `${m[1]}:${m[2]}`;
}

function pageSource(): string {
  return fs.readFileSync(PAGE, 'utf8');
}

describe('流程 SLA 监控页的权限失败分流', () => {
  it('后端违规端点的权限码是 sla:read（与菜单的 bpmn:read 不同源）', () => {
    // 显式锚定这处不同源，避免有人误以为「同域就该同码」而把页面改成 bpmn:read。
    expect(backendViolationCode()).toBe('sla:read');
  });

  it('页面用 isPermissionDeniedError 区分无权限与真失败', () => {
    const src = pageSource();
    expect(src).toContain('isPermissionDeniedError');
    expect(/setViolationsDenied\(\s*isPermissionDeniedError\(/.test(src)).toBe(true);
  });

  it('无权限分支的文案点明权限码，且不把「重试」当建议', () => {
    const src = pageSource();
    // 只取 true 分支的字面量，不能把 else 分支（「请稍后重试」）一起框进来：
    // 那是给真失败用的建议，属于正确文案。
    const deniedDesc = src.match(
      /description=\{[\s\S]*?violationsDenied\s*\n?\s*\?\s*'([^']*)'/
    );
    expect(deniedDesc).not.toBeNull();
    const desc = deniedDesc![1];
    expect(desc).toContain('sla:read');
    // 关键反向断言：这条分支里不得出现「稍后重试」——权限问题重试无用。
    expect(desc).not.toContain('稍后重试');
  });

  it('两种失败用不同的 Alert 类型（warning vs error），肉眼可区分', () => {
    const src = pageSource();
    expect(/type=\{violationsDenied \? 'warning' : 'error'\}/.test(src)).toBe(true);
  });
});
