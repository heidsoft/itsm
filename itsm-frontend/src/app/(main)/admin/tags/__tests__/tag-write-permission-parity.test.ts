/**
 * 全局标签页的写权限码一致性（回归锁）。
 *
 * 菜单「全局标签」的准入码是 `system_config:read`（pkg/menubaseline/baseline.go），
 * 但页面的写操作真实落在工单标签域的细分动作上：
 *   POST   /api/v1/ticket-tags      → ticket_tag:create
 *   PUT    /api/v1/ticket-tags/:id  → ticket_tag:update
 *   DELETE /api/v1/ticket-tags/:id  → ticket_tag:delete
 * （router/ticket_tag_routes.go）
 *
 * audit_admin / security_admin 恰好只持 system_config:read 而不持这三个写码，
 * 于是「菜单放得进来、新建与删除一律 403」。页面曾无条件渲染新建/编辑/删除。
 *
 * 本测试解析后端路由文件取每个写动作的真实码，再与页面判定表达式里的 PERM 路径
 * 逐一比对。任一侧单边改动即红。
 */

import fs from 'node:fs';
import path from 'node:path';

const FRONTEND_ROOT = process.cwd();
const REPO_ROOT = path.resolve(FRONTEND_ROOT, '..');
const PAGE = path.join(FRONTEND_ROOT, 'src/app/(main)/admin/tags/page.tsx');
const ROUTES = path.join(REPO_ROOT, 'itsm-backend/router/ticket_tag_routes.go');

/** 从后端路由文件里取某 HTTP 方法在某路径上挂的权限码。 */
function backendCode(method: string, pathFragment: string): string {
  const src = fs.readFileSync(ROUTES, 'utf8');
  const line = src
    .split('\n')
    .find(
      l =>
        l.includes(`tags.${method}(`) &&
        l.includes(`"${pathFragment}"`) &&
        l.includes('RequirePermission')
    );
  if (!line) throw new Error(`未找到 ${method} ${pathFragment} 的路由行`);
  const m = /RequirePermission\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)/.exec(line);
  if (!m) throw new Error(`路由行未使用 RequirePermission 字面量：${line.trim()}`);
  return `${m[1]}:${m[2]}`;
}

/** 取页面里某个判定变量的 PERM 路径转成的码。 */
function frontendCode(variable: string): string {
  const src = fs.readFileSync(PAGE, 'utf8');
  const re = new RegExp(
    `const\\s+${variable}\\s*=\\s*hasPermission\\(\\s*PERM\\.([A-Za-z0-9_.]+)\\s*\\)`
  );
  const m = re.exec(src);
  if (!m) throw new Error(`未找到 ${variable} 的 hasPermission 判定`);
  return m[1].split('.').join(':');
}

describe('全局标签写权限码（前后端同码）', () => {
  it('新建判定等于后端 POST /ticket-tags 的权限码', () => {
    expect(frontendCode('canCreate')).toBe(backendCode('POST', ''));
  });

  it('编辑判定等于后端 PUT /ticket-tags/:id 的权限码', () => {
    expect(frontendCode('canUpdate')).toBe(backendCode('PUT', '/:id'));
  });

  it('删除判定等于后端 DELETE /ticket-tags/:id 的权限码', () => {
    expect(frontendCode('canDelete')).toBe(backendCode('DELETE', '/:id'));
  });

  it('三个写动作不能塌缩成同一个码（否则「只能编辑不能删」的组合会失效）', () => {
    const codes = [frontendCode('canCreate'), frontendCode('canUpdate'), frontendCode('canDelete')];
    expect(new Set(codes).size).toBe(3);
  });

  it('页面不得用菜单准入码 system_config:read 去门控写按钮', () => {
    const src = fs.readFileSync(PAGE, 'utf8');
    const writes = src
      .split('\n')
      .filter(l => !/^\s*(\/\/|\*|\/\*)/.test(l))
      .filter(l => l.includes('hasPermission') && l.includes('system_config'));
    expect(writes).toEqual([]);
  });
});
