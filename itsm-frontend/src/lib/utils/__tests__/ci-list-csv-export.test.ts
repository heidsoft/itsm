/**
 * R5 回归锁：CMDB 配置项列表的 CSV 导出必须复用统一入口。
 *
 * 旧实现手写 `.map(...).join(',')`，只把引号翻倍，绕过了
 * `buildCsv` 的公式注入中性化：CI 名称写成 `=HYPERLINK(...)` 时，
 * 导出的文件在 Excel/WPS 打开即触发公式。incidents/changes/problems
 * 三个列表页早已切到 buildCsv，CMDB 是漏网的一条。
 */

import fs from 'fs';
import path from 'path';

import { buildCsv } from '../csv-export';

const ciListSource = fs.readFileSync(
  path.join(process.cwd(), 'src/components/cmdb/CIList.tsx'),
  'utf8'
);

describe('CMDB CIList CSV 导出统一入口', () => {
  it('通过 buildCsv + downloadCsv 导出，不再手写 CSV 字符串', () => {
    expect(ciListSource).toContain("from '@/lib/utils/csv-export'");
    expect(ciListSource).toContain('buildCsv<ConfigurationItem>');
    expect(ciListSource).toContain('downloadCsv(');
    expect(ciListSource).not.toMatch(/\.join\(','\)/);
    expect(ciListSource).not.toMatch(/replace\(\/"\/g/);
    expect(ciListSource).not.toMatch(/new Blob\(\['\\ufeff'/);
  });

  it('统一入口会把公式形态的 CI 名称中性化并带 BOM', () => {
    const csv = buildCsv<{ name: string }>(
      [{ title: '名称', render: row => row.name }],
      [{ name: '=HYPERLINK("http://evil.test","click")' }]
    );
    expect(csv.startsWith('\uFEFF')).toBe(true);
    expect(csv).toContain("'=HYPERLINK");
  });
});
