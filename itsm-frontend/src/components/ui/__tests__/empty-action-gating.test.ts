/**
 * 空态入口门控守卫
 *
 * 背景（2026-10-26 浏览器实测）：列表页的「新建」入口有两处——页面 `primaryAction`
 * 与表格 `emptyText` 里的 `LoadingEmptyError.empty.actionText`。前者已被
 * `useOperationPermissions` 门控，后者写死 `showAction: true` 就成了绕过门控的
 * 第二入口：end_user 在 /assets 看不到页头按钮，却仍能在空态里点到「新增资产」，
 * 点进去 POST /assets 直接 403。
 *
 * 本守卫锁定「空态动作必须由权限推导」这一约定：`showAction` 只允许写成表达式，
 * 不允许字面量 true。新增列表页请按 AssetList/ServiceRequestList 的写法接
 * usePermissions 后传布尔值。
 */

import fs from 'fs';
import path from 'path';

const SRC_ROOT = path.join(__dirname, '..', '..', '..');

function collectSourceFiles(dir: string, out: string[] = []): string[] {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === '__tests__' || entry.name === 'node_modules') continue;
      collectSourceFiles(full, out);
    } else if (/\.(tsx|ts)$/.test(entry.name) && !/\.(test|spec)\.(tsx|ts)$/.test(entry.name)) {
      out.push(full);
    }
  }
  return out;
}

describe('空态入口门控', () => {
  it('源码中不得出现写死的 showAction: true', () => {
    const offenders: string[] = [];
    for (const file of collectSourceFiles(SRC_ROOT)) {
      const content = fs.readFileSync(file, 'utf8');
      if (/showAction:\s*true\b/.test(content)) {
        offenders.push(path.relative(SRC_ROOT, file));
      }
    }

    expect(offenders).toEqual([]);
  });

  it('空态动作确实由权限码推导（阳性对照，防止守卫被清空后静默通过）', () => {
    const assetList = fs.readFileSync(
      path.join(SRC_ROOT, 'components', 'asset', 'AssetList.tsx'),
      'utf8'
    );
    expect(assetList).toMatch(/showAction:\s*canCreateAsset\b/);
    expect(assetList).toMatch(/hasPermission\('asset',\s*'write'\)/);

    const serviceRequestList = fs.readFileSync(
      path.join(SRC_ROOT, 'components', 'service-request', 'ServiceRequestList.tsx'),
      'utf8'
    );
    expect(serviceRequestList).toMatch(/showAction:\s*canCreateServiceRequest\b/);
    expect(serviceRequestList).toMatch(/hasPermission\('service_request',\s*'write'\)/);
  });
});
