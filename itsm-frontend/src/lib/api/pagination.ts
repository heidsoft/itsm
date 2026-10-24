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
