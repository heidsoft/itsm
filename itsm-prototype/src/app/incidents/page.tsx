"use client";

import { useEffect, useState } from "react";
import { fetchSLADefinitions, deleteSLADefinition } from "./api";
import Link from "next/link";

export default function SLADefinitionListPage() {
  const [data, setData] = useState<unknown[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);

  useEffect(() => {
    fetchSLADefinitions(page, 10).then((res) => {
      setData(res.definitions);
      setTotal(res.total);
    });
  }, [page]);

  return (
    <div>
      <h1 className="text-xl font-bold mb-4">SLA定义管理</h1>
      <Link href="/admin/sla-definitions/new" className="btn btn-primary mb-2">
        新建SLA
      </Link>
      <table className="table-auto w-full">
        <thead>
          <tr>
            <th>名称</th>
            <th>服务类型</th>
            <th>优先级</th>
            <th>影响范围</th>
            <th>响应时间(分钟)</th>
            <th>解决时间(分钟)</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          {data.map((sla) => (
            <tr key={sla.id}>
              <td>{sla.name}</td>
              <td>{sla.service_type}</td>
              <td>{sla.priority}</td>
              <td>{sla.impact}</td>
              <td>{sla.response_time}</td>
              <td>{sla.resolution_time}</td>
              <td>
                <Link href={`/admin/sla-definitions/edit?id=${sla.id}`}>
                  编辑
                </Link>
                <button
                  onClick={() => {
                    deleteSLADefinition(sla.id).then(() =>
                      window.location.reload()
                    );
                  }}
                >
                  删除
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {/* 分页组件略 */}
    </div>
  );
}
