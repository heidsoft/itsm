'use client';

import { useEffect } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { Spin, App } from 'antd';

/**
 * /knowledge/:id 兼容路由。
 *
 * 历史原因:产品上有部分链接/分享的 URL 是 /knowledge/9 这种短路径,
 * 而详情页实际在 /knowledge/articles/[id]。这里做一次客户端重定向,
 * 保持原有 URL 可用。
 */
export default function KnowledgeIdRedirectPage() {
  const router = useRouter();
  const params = useParams();
  const id = (params?.id as string) || '';

  useEffect(() => {
    if (!id) return;
    router.replace(`/knowledge/articles/${encodeURIComponent(id)}`);
  }, [id, router]);

  return (
    <App>
      <div style={{ padding: 48, textAlign: 'center' }}>
        <Spin tip="正在跳转到知识详情..." />
      </div>
    </App>
  );
}
