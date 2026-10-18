/**
 * CSV 导出工具单测。
 *
 * 覆盖三条真实缺陷面：翻页终止条件（total=0 是合法值，不能 || 兜底）、
 * 公式注入转义、下载时 blob URL 的申请与释放。
 */
import { buildCsv, csvDate, downloadCsv, fetchAllPages, type PageResult } from '../csv-export';

const BOM = '\uFEFF';
const CRLF = '\r\n';

interface Row {
  name: string;
  qty: number;
  note?: string | null;
}

const columns = [
  { title: '名称', render: (row: Row) => row.name },
  { title: '数量', render: (row: Row) => row.qty },
  { title: '备注', render: (row: Row) => row.note },
];

describe('fetchAllPages', () => {
  it('按 total 拉满并合并各页', async () => {
    const pages: Record<number, PageResult<number>> = {
      1: { items: [1, 2], total: 5 },
      2: { items: [3, 4], total: 5 },
      3: { items: [5], total: 5 },
    };
    const fetchPage = jest.fn((page: number) => Promise.resolve(pages[page]));

    await expect(fetchAllPages(fetchPage, 2, 10)).resolves.toEqual([1, 2, 3, 4, 5]);
    expect(fetchPage).toHaveBeenCalledTimes(3);
  });

  it('total 为 0 时只取一页，不把 0 兜底成当前页条数', async () => {
    const fetchPage = jest.fn(() => Promise.resolve({ items: [], total: 0 }));

    await expect(fetchAllPages(fetchPage, 100, 10)).resolves.toEqual([]);
    expect(fetchPage).toHaveBeenCalledTimes(1);
  });

  it('条数少于 pageSize 即停止，即使 total 仍声称更多', async () => {
    const fetchPage = jest.fn(() => Promise.resolve({ items: ['a'], total: 999 }));

    await expect(fetchAllPages(fetchPage, 100, 10)).resolves.toEqual(['a']);
    expect(fetchPage).toHaveBeenCalledTimes(1);
  });

  it('每页都满且 total 虚高时受 maxPages 保护', async () => {
    const fetchPage = jest.fn(() => Promise.resolve({ items: ['x', 'y'], total: 10_000 }));

    const all = await fetchAllPages(fetchPage, 2, 3);

    expect(all).toHaveLength(6);
    expect(fetchPage).toHaveBeenCalledTimes(3);
  });

  it('缺 items 的响应按空页处理而不是抛错', async () => {
    const fetchPage = jest.fn(() => Promise.resolve({ total: 3 } as PageResult<string>));

    await expect(fetchAllPages(fetchPage, 2, 5)).resolves.toEqual([]);
    expect(fetchPage).toHaveBeenCalledTimes(1);
  });

  it('默认 pageSize 100 传给 fetchPage', async () => {
    const fetchPage = jest.fn(() => Promise.resolve({ items: ['a'], total: 1 }));

    await fetchAllPages(fetchPage);

    expect(fetchPage).toHaveBeenCalledWith(1, 100);
  });
});

describe('csvDate', () => {
  it('空值输出空串', () => {
    expect(csvDate(undefined)).toBe('');
    expect(csvDate(null)).toBe('');
    expect(csvDate('')).toBe('');
  });

  it('非法时间输出空串而不是 Invalid Date', () => {
    expect(csvDate('not-a-date')).toBe('');
  });

  it('合法时间用 zh-CN 本地串', () => {
    const iso = '2026-10-18T01:30:00Z';
    expect(csvDate(iso)).toBe(new Date(iso).toLocaleString('zh-CN', { hour12: false }));
  });
});

describe('buildCsv', () => {
  it('带 BOM、CRLF 行尾与表头', () => {
    expect(buildCsv(columns, [{ name: '重置密码', qty: 2 }])).toBe(
      `${BOM}名称,数量,备注${CRLF}重置密码,2,${CRLF}`
    );
  });

  it('空行集仍输出表头', () => {
    expect(buildCsv(columns, [])).toBe(`${BOM}名称,数量,备注${CRLF}`);
  });

  it('null/undefined 单元格输出空串', () => {
    expect(buildCsv(columns, [{ name: 'a', qty: 1, note: null }])).toBe(
      `${BOM}名称,数量,备注${CRLF}a,1,${CRLF}`
    );
  });

  it('数字 0 不被当作空值', () => {
    expect(buildCsv([{ title: 'n', render: (row: Row) => row.qty }], [{ name: '', qty: 0 }])).toBe(
      `${BOM}n${CRLF}0${CRLF}`
    );
  });

  it('含逗号/引号/换行的值整体加引号并把引号翻倍', () => {
    const csv = buildCsv(columns, [
      { name: 'A,B', qty: 1, note: '他说"好"' },
      { name: '多行', qty: 2, note: '第一行\n第二行' },
    ]);

    expect(csv).toBe(
      `${BOM}名称,数量,备注${CRLF}` +
        `"A,B",1,"他说""好"""${CRLF}` +
        `多行,2,"第一行\n第二行"${CRLF}`
    );
  });

  it('以 = + - @ 制表符 开头的值加前置单引号防公式注入', () => {
    const csv = buildCsv(
      [{ title: 'v', render: (row: Row) => row.name }],
      [
        { name: '=1+1', qty: 1 },
        { name: '+8613800000000', qty: 1 },
        { name: '-模板', qty: 1 },
        { name: '@cmd', qty: 1 },
        { name: '\t隐藏制表', qty: 1 },
      ]
    );

    expect(csv).toBe(
      `${BOM}v${CRLF}` +
        `'=1+1${CRLF}` +
        `'+8613800000000${CRLF}` +
        `'-模板${CRLF}` +
        `'@cmd${CRLF}` +
        `'\t隐藏制表${CRLF}`
    );
  });

  it('前置单引号后仍含回车时再整体加引号', () => {
    const csv = buildCsv([{ title: 'v', render: (row: Row) => row.name }], [
      { name: '\r回车', qty: 1 },
    ]);

    expect(csv).toBe(`${BOM}v${CRLF}"'${'\r'}回车"${CRLF}`);
  });

  it('公式前缀与逗号同时命中时先加前缀再整体转义', () => {
    const csv = buildCsv([{ title: 'v', render: (row: Row) => row.name }], [
      { name: '=1,2', qty: 1 },
    ]);

    expect(csv).toBe(`${BOM}v${CRLF}"'=1,2"${CRLF}`);
  });

  it('表头本身含逗号也走同一转义', () => {
    const csv = buildCsv([{ title: 'a,b', render: (row: Row) => row.name }], [
      { name: 'x', qty: 1 },
    ]);

    expect(csv).toBe(`${BOM}"a,b"${CRLF}x${CRLF}`);
  });
});

describe('downloadCsv', () => {
  let createdBlob: Blob | undefined;
  const createObjectURL = jest.fn((blob: Blob) => {
    createdBlob = blob;
    return 'blob:mock-url';
  });
  const revokeObjectURL = jest.fn();
  const clicked: HTMLAnchorElement[] = [];
  const originalCreate = URL.createObjectURL;
  const originalRevoke = URL.revokeObjectURL;

  // jest.config 开了 restoreMocks，spy 必须在每个用例前重新装，否则 beforeAll 装的会被还原。
  beforeEach(() => {
    createObjectURL.mockClear();
    revokeObjectURL.mockClear();
    clicked.length = 0;
    createdBlob = undefined;
    URL.createObjectURL = createObjectURL;
    URL.revokeObjectURL = revokeObjectURL;
    jest
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(function (this: HTMLAnchorElement) {
        clicked.push(this);
      });
  });

  afterEach(() => {
    URL.createObjectURL = originalCreate;
    URL.revokeObjectURL = originalRevoke;
  });

  it('补 .csv 后缀、按 text/csv 建 blob、点击后移除锚点并释放 URL', () => {
    downloadCsv('工单导出', `${BOM}a${CRLF}`);

    expect(createObjectURL).toHaveBeenCalledTimes(1);
    expect(createdBlob).toBeInstanceOf(Blob);
    expect(createdBlob?.type).toBe('text/csv;charset=utf-8');

    expect(clicked).toHaveLength(1);
    const anchor = clicked[0];
    expect(anchor.getAttribute('href')).toBe('blob:mock-url');
    expect(anchor.download).toBe('工单导出.csv');
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:mock-url');
    expect(anchor.parentNode).toBeNull();
  });

  it('已带 .csv（大小写不敏感）的后缀不重复追加', () => {
    downloadCsv('tickets.CSV', 'a');

    expect(clicked[0].download).toBe('tickets.CSV');
  });
});
