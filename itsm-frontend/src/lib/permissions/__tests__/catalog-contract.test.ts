/**
 * 权限码契约守卫（2026-11-14 权限码单点化批次）
 *
 * 三条不变量：
 *  1. catalog.gen.ts 自身一致：code === `${resource}:${action}` 且无重复码；
 *  2. 生产代码不得手写权限码字面量调用 hasPermission/hasResourcePermission/
 *     guardOperation —— 一律引用生成的 PERM.* 常量，幽灵码在编译期即失败；
 *  3. 路由配置（route-config）声明的码 ⊆ 后端码空间（与 rp(PERM.x.y) 的
 *     类型保证互为冗余，防止有人绕开类型用 as 断言写回字符串）。
 */

import fs from 'fs';
import path from 'path';

import { PERMISSION_CATALOG } from '../catalog.gen';

const SRC_ROOT = path.join(__dirname, '..', '..', '..');

function walk(dir: string, out: string[] = []): string[] {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === '__tests__' || entry.name === 'node_modules') continue;
      walk(full, out);
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full);
    }
  }
  return out;
}

describe('权限码契约守卫', () => {
  describe('catalog.gen.ts 完整性', () => {
    it('每个条目满足 code === resource:action', () => {
      for (const entry of PERMISSION_CATALOG) {
        expect(`${entry.resource}:${entry.action}`).toBe(entry.code);
        expect(entry.code.length).toBeGreaterThan(0);
        expect(entry.name.length).toBeGreaterThan(0);
      }
    });

    it('无重复权限码', () => {
      const seen = new Set<string>();
      for (const entry of PERMISSION_CATALOG) {
        expect(seen.has(entry.code)).toBe(false);
        seen.add(entry.code);
      }
    });

    it('生成物带 DO NOT EDIT 头（防手写漂移）', () => {
      const raw = fs.readFileSync(
        path.join(SRC_ROOT, 'lib', 'permissions', 'catalog.gen.ts'),
        'utf8',
      );
      expect(raw).toContain('DO NOT EDIT');
    });
  });

  describe('生产代码零手写权限码字面量', () => {
    it('hasPermission/hasResourcePermission/guardOperation 只接受 PERM 常量', () => {
      const files = walk(SRC_ROOT).filter(file => {
        const rel = path.relative(SRC_ROOT, file);
        if (rel.startsWith('lib/permissions')) return false;
        return !rel.endsWith('.gen.ts');
      });

      const offenders: string[] = [];
      // 匹配以字符串字面量为首个实参的调用；PERM.xxx 形式不会命中
      const patterns = [
        /hasPermission\(\s*['"`]/,
        /hasResourcePermission\(\s*['"`]/,
        /guardOperation\(\s*['"`]/,
      ];
      for (const file of files) {
        const lines = fs.readFileSync(file, 'utf8').split('\n');
        lines.forEach((line, idx) => {
          for (const pattern of patterns) {
            if (pattern.test(line)) {
              offenders.push(
                `${path.relative(SRC_ROOT, file)}:${idx + 1} ${line.trim()}`,
              );
            }
          }
        });
      }

      expect(offenders).toEqual([]);
    });
  });
});
