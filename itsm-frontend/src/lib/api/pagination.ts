/**
 * 分页上限契约。
 *
 * 后端 `common.MaxPageSize = 100`，且 `GetPaginationFromQuery` 会把更大的
 * pageSize **静默截断**成 100（itsm-backend/common/pagination.go）：请求
 * pageSize=200 时，响应回的 pageSize 是 100，items 也只有 100 条，没有任何
 * 错误或警告字段。
 *
 * 前端曾经在多处直接写 `pageSize: 200` / `500` / `1000`，读代码的人会以为
 * 真的取到了那么多条——大租户下表现为「选人下拉里找不到某个人」「对账漏行」，
 * 而且界面上完全看不出来是被截断了。统一改用本常量，让上限在调用处显式可见。
 *
 * 需要真正全量数据时，不要调大这个值（后端不会放行），而是分页循环拉取：
 * 见 `@/lib/utils/csv-export` 导出的 `fetchAllPages`。
 */
export const BACKEND_MAX_PAGE_SIZE = 100;

/**
 * 判断一次列表请求是否被后端静默截断。
 *
 * 后端只做「大于 100 就压成 100」，不返回任何 `truncated` 标记，信封里的
 * `total` 仍是**真实总数**。因此判据是「真实总数 > 本次实际拿到的条数」，
 * 而不是「请求 pageSize > 100」——后者在 total 恰好等于 100 时会误报。
 *
 * 典型受害场景是选人/选 CI 这类**非分页下拉**：写 `pageSize: 100` 看着像
 * 「取全量」，300 人的租户里只会加载前 100 个，且界面上完全看不出来，
 * 表现为「下拉里找不到某个人」。调用方应据此显式提示，而不是静默少给。
 *
 * `total` 缺失（旧端点/非标准信封）时返回 false——宁可不提示，也不要
 * 在有完整数据的时候谎报截断。
 */
export function isTruncated(total: number | undefined | null, returned: number): boolean {
  if (typeof total !== 'number' || !Number.isFinite(total)) return false;
  return total > returned;
}

/**
 * 截断提示文案。返回 null 表示未截断，调用方无需渲染提示。
 */
export function truncationHint(total: number | undefined | null, returned: number): string | null {
  if (!isTruncated(total, returned)) return null;
  return `仅显示前 ${returned} 条（共 ${total} 条），请用搜索缩小范围`;
}
