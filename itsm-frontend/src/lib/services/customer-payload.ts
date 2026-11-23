/**
 * 客户/分支表单提交体构造。
 *
 * 抽成独立模块而不是页面内联的原因：这里的类型契约（别名为 `string[]`）是与后端
 * `email_intake.customerRequest` 的接口约定，一旦被页面里随手写的 `?? ''` 覆盖，
 * 表现是「保存失败」一句笼统报错，排查成本极高。独立模块才能用行为测试锁住。
 */

export type CustomerFormValues = {
  name: string;
  shortName?: string;
  /** tags 模式收集的多值；后端字段类型是 []string */
  aliases?: string[];
};

export type CustomerPayload = {
  name: string;
  shortName: string;
  aliases: string[];
  historicalNames: string[];
  status: string;
};

export type BranchPayload = {
  name: string;
  aliases: string[];
  customerId: number;
  status: string;
};

/** 与 Select tags 控件一致的输入提示：回车或粘逗号都能拆成多条。 */
export const CUSTOMER_FORM_EMPTY_HINT = '输入后回车，或粘贴逗号分隔的多条';

/**
 * 构造客户提交体：只取接口声明字段。
 *
 * 不回传 id/createdAt/updatedAt——这些是服务端回读字段，回灌会顺带覆盖服务端
 * 自己维护的值（例如 normalizedName 的派生逻辑虽在服务端，但任何多余字段都让
 * 请求体语义变得不可预测）。
 */
export function toCustomerPayload(
  values: CustomerFormValues,
  status: string,
  historicalNames: string[] = []
): CustomerPayload {
  return {
    name: values.name?.trim() ?? '',
    shortName: values.shortName?.trim() ?? '',
    aliases: values.aliases ?? [],
    historicalNames,
    status,
  };
}

export function toBranchPayload(
  values: CustomerFormValues,
  customerId: number,
  status: string
): BranchPayload {
  return {
    name: values.name?.trim() ?? '',
    aliases: values.aliases ?? [],
    customerId,
    status,
  };
}
