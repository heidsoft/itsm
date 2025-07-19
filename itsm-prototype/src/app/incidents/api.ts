import { httpClient } from "@/app/lib/http-client";

// 获取SLA定义列表
export async function fetchSLADefinitions(page = 1, pageSize = 10) {
  // 正确的后端接口路径
  return httpClient.get("/api/sla/definitions", { page, page_size: pageSize });
}

// 删除SLA定义
export async function deleteSLADefinition(id: string) {
  // 正确的后端接口路径
  return httpClient.delete(`/api/sla/definitions/${id}`);
} 