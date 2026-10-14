/**
 * CSV 导出工具。
 *
 * 背景：problems/changes/incidents 列表页的「导出」长期是 message.info 占位
 * （后端没有对应导出端点），2026-10-14 审查后改为「按当前筛选条件全量拉取 +
 * 浏览器端生成 CSV」。口径：导出的就是用户当前看到的筛选结果，不多也不少。
 *
 * 两个细节：
 * - 带 UTF-8 BOM：Excel 直接打开中文 CSV 不乱码；
 * - 全部字段按字符串转义：公式注入防护（以 = + - @ 开头的值加前置单引号）。
 */

/** 单列定义：title 为表头，render 把行数据转为单元格文本。 */
export interface CsvColumn<T> {
  title: string;
  render: (row: T) => string | number | null | undefined;
}

/** 分页拉取结果的通用形状（与后端 {items,total} 信封对齐）。 */
export interface PageResult<T> {
  items: T[];
  total: number;
}

/**
 * 按页拉取全部匹配记录（受 maxPages 保护，避免筛选条件异常时无限翻页）。
 * 后端 pageSize 上限为 100（common/constants.go MaxPageSize）。
 */
export async function fetchAllPages<T>(
  fetchPage: (page: number, pageSize: number) => Promise<PageResult<T>>,
  pageSize = 100,
  maxPages = 50,
): Promise<T[]> {
  const all: T[] = [];
  for (let page = 1; page <= maxPages; page++) {
    const resp = await fetchPage(page, pageSize);
    const items = resp.items ?? [];
    all.push(...items);
    // total 为 0 是合法值（空结果），不能用 || 兜底成当前页条数
    if (all.length >= resp.total || items.length < pageSize) {
      break;
    }
  }
  return all;
}

/** CSV 日期单元格：本地时间字符串，空值/非法值输出空串。 */
export function csvDate(value?: string | null): string {
  if (!value) {
    return '';
  }
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '' : date.toLocaleString('zh-CN', { hour12: false });
}

/** CSV 单元格转义：包裹引号、屏蔽公式注入。 */
function escapeCsvValue(value: unknown): string {
  if (value === null || value === undefined) {
    return '';
  }
  let text = typeof value === 'string' ? value : String(value);
  // 公式注入防护：Excel/WPS 会把以这些字符开头的单元格当公式执行
  if (/^[=+\-@\t\r]/.test(text)) {
    text = `'${text}`;
  }
  if (/[",\r\n]/.test(text)) {
    text = `"${text.replace(/"/g, '""')}"`;
  }
  return text;
}

/** 生成 CSV 文本（含 BOM 与 CRLF 行尾，Excel 兼容）。 */
export function buildCsv<T>(columns: CsvColumn<T>[], rows: T[]): string {
  const header = columns.map((col) => escapeCsvValue(col.title)).join(',');
  const lines = rows.map((row) => columns.map((col) => escapeCsvValue(col.render(row))).join(','));
  return `\uFEFF${[header, ...lines].join('\r\n')}\r\n`;
}

/** 触发浏览器下载。文件名自动补 .csv 后缀。 */
export function downloadCsv(filename: string, csv: string): void {
  const name = filename.toLowerCase().endsWith('.csv') ? filename : `${filename}.csv`;
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = name;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}
