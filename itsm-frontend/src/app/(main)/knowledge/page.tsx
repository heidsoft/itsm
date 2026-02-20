'use client';

import { useEffect } from 'react';
import ArticleList from '@/components/knowledge/ArticleList';

export default function KnowledgePage() {
    useEffect(() => {
        if (typeof document !== 'undefined') {
            document.title = '知识库 - ITSM 系统';
        }
    }, []);

    return <ArticleList />;
}
