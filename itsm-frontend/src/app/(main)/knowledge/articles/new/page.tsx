'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

export default function NewArticleRedirect() {
  const router = useRouter();

  useEffect(() => {
    // 重定向到正确的创建页面
    router.replace('/knowledge/articles/create');
  }, [router]);

  return null;
}
