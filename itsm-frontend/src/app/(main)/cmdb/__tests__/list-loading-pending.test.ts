/**
 * D56 回归：CMDB 列表加载态必须用 isPending
 *
 * React Query 的 `isLoading = isPending && isFetching`。App Router SSR/首帧订阅
 * 尚未发生，`isFetching=false` 会让 `isLoading` 判成非加载态，空数据先被当成
 * 最终结果渲染（「暂无数据」/「共 0 个」）。D55 已在 cloud-accounts 与 useTickets
 * 上验证该机制，本守卫锁定 A 类 8 个 CMDB 列表查询点，防止回退为 isLoading。
 */

import fs from 'node:fs';
import path from 'node:path';

const SRC = path.join(process.cwd(), 'src');

type Case = {
  file: string;
  pending: string[];
  loading: string[];
};

const cases: Case[] = [
  {
    file: 'app/(main)/cmdb/cloud-services/page.tsx',
    pending: ['servicesQuery.isPending'],
    loading: ['servicesQuery.isLoading'],
  },
  {
    file: 'app/(main)/cmdb/relationships/page.tsx',
    pending: ['cisQuery.isPending'],
    loading: ['cisQuery.isLoading'],
  },
  {
    file: 'app/(main)/cmdb/reconciliation/page.tsx',
    pending: ['reconQuery.isPending', 'servicesQuery.isPending'],
    loading: ['reconQuery.isLoading', 'servicesQuery.isLoading'],
  },
  {
    file: 'app/(main)/cmdb/cloud-resources/page.tsx',
    pending: ['resourcesQuery.isPending'],
    loading: ['resourcesQuery.isLoading'],
  },
  {
    file: 'app/(main)/cmdb/cloud-signals/page.tsx',
    pending: ['syncQuery.isPending', 'alertsQuery.isPending', 'findingsQuery.isPending'],
    loading: ['syncQuery.isLoading', 'alertsQuery.isLoading', 'findingsQuery.isLoading'],
  },
  {
    file: 'app/(main)/cmdb/registry/page.tsx',
    pending: ['q.isPending'],
    loading: ['q.isLoading'],
  },
  {
    file: 'components/cmdb/CIList.tsx',
    pending: ['listQuery.isPending'],
    loading: ['listQuery.isLoading'],
  },
  {
    file: 'components/cmdb/CIRelationshipManager.tsx',
    pending: ['relationsQuery.isPending'],
    loading: ['relationsQuery.isLoading'],
  },
];

describe('CMDB 列表首屏加载态（D56 回归）', () => {
  it.each(cases)('$file 使用 isPending 且不再用 isLoading', ({ file, pending, loading }) => {
    const source = fs.readFileSync(path.join(SRC, file), 'utf8');

    for (const token of pending) {
      expect(source).toContain(token);
    }
    for (const token of loading) {
      expect(source).not.toContain(token);
    }
  });
});
