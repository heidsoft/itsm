
'use client';

import { Search, Filter, PlusCircle } from 'lucide-react';

import React, { useState } from 'react';
import Link from 'next/link';
import  from 'lucide-react';

// 模拟知识文章数据
const mockArticles = [
    { id: 'KB-001', title: '如何重置您的域账户密码', category: '账号管理', author: 'IT服务台', publishedDate: '2024-01-15', views: 1250 },
    { id: 'KB-002', title: 'Web服务器CPU高占用率排查指南', category: '故障排除', author: '运维团队', publishedDate: '2024-03-20', views: 890 },
    { id: 'KB-003', title: 'VPN连接常见问题及解决方案', category: '网络连接', author: '网络团队', publishedDate: '2024-02-10', views: 1500 },
    { id: 'KB-004', title: '申请云虚拟机最佳实践', category: '云资源', author: '云平台团队', publishedDate: '2024-05-01', views: 560 },
];

const KnowledgeBaseListPage = () => {
    const [searchTerm, setSearchTerm] = useState('');
    const [filterCategory, setFilterCategory] = useState('全部');

    const filteredArticles = mockArticles.filter(article => {
        const matchesSearch = article.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
                              article.category.toLowerCase().includes(searchTerm.toLowerCase());
        const matchesCategory = filterCategory === '全部' || article.category === filterCategory;
        return matchesSearch && matchesCategory;
    });

    const categories = ['全部', ...new Set(mockArticles.map(a => a.category))];

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8 flex justify-between items-center">
                <div>
                    <h2 className="text-4xl font-bold text-gray-800">知识库</h2>
                    <p className="text-gray-500 mt-1">自助解决问题，沉淀IT经验</p>
                </div>
                <Link href="/knowledge-base/new" passHref>
                    <button className="flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors">
                        <PlusCircle className="w-5 h-5 mr-2" />
                        新建文章
                    </button>
                </Link>
            </header>

            {/* 搜索与筛选 */}
            <div className="mb-6 bg-white p-4 rounded-lg shadow-sm flex flex-wrap items-center gap-4">
                <div className="relative flex-grow">
                    <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                    <input 
                        type="text"
                        placeholder="搜索知识文章..."
                        className="w-full pl-12 pr-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        onChange={(e) => setSearchTerm(e.target.value)}
                    />
                </div>
                <div className="flex items-center">
                    <Filter className="w-5 h-5 text-gray-500 mr-2" />
                    <span className="text-sm font-semibold mr-2">分类:</span>
                    <select 
                        className="px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                        onChange={(e) => setFilterCategory(e.target.value)}
                        value={filterCategory}
                    >
                        {categories.map(cat => (
                            <option key={cat} value={cat}>{cat}</option>
                        ))}
                    </select>
                </div>
            </div>

            {/* 知识文章列表 */}
            <div className="bg-white rounded-lg shadow-md overflow-hidden">
                <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                        <tr>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">文章ID</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">标题</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">分类</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">作者</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">发布日期</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">浏览量</th>
                        </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                        {filteredArticles.length > 0 ? filteredArticles.map(article => (
                            <tr key={article.id} className="hover:bg-gray-50">
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <Link href={`/knowledge-base/${article.id}`} className="text-blue-600 font-semibold hover:underline">
                                        {article.id}
                                    </Link>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap max-w-sm truncate">{article.title}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{article.category}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{article.author}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{article.publishedDate}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{article.views}</td>
                            </tr>
                        )) : (
                            <tr>
                                <td colSpan={6} className="px-6 py-4 whitespace-nowrap text-center text-sm text-gray-500">
                                    没有找到匹配的文章。
                                </td>
                            </tr>
                        )}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default KnowledgeBaseListPage;
