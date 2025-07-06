'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { FormInput } from '../../components/FormInput';
import { FormTextarea } from '../../components/FormTextarea';

const CreateKnowledgeArticlePage = () => {
    const router = useRouter();
    const searchParams = useSearchParams();

    const [title, setTitle] = useState('');
    const [content, setContent] = useState('');
    const [category, setCategory] = useState('故障排除');

    useEffect(() => {
        const problemId = searchParams.get('fromProblemId');
        const problemTitle = searchParams.get('problemTitle');
        const problemSolution = searchParams.get('problemSolution');

        if (problemId) {
            setTitle(`问题 ${problemId} 解决方案: ${problemTitle || ''}`);
            setContent(`此文章由问题 ${problemId} 解决方案转化而来。\n\n**问题标题:** ${problemTitle || ''}\n\n**解决方案:**\n${problemSolution || ''}\n\n---`);
            setCategory('故障排除'); // 默认分类为故障排除
        }
    }, [searchParams]);

    const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        const formData = new FormData(e.currentTarget);
        const data = Object.fromEntries(formData.entries());
        console.log('新建知识文章数据:', JSON.stringify(data, null, 2));
        alert('知识文章已成功创建！');
        router.push('/knowledge-base'); // 提交后返回知识库列表
    };

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <button onClick={() => router.back()} className="flex items-center text-blue-600 hover:underline mb-4">
                    <ArrowLeft className="w-5 h-5 mr-2" />
                    返回知识库
                </button>
                <h2 className="text-4xl font-bold text-gray-800">新建知识文章</h2>
                <p className="text-gray-500 mt-1">记录和分享IT知识，提升自助服务能力</p>
            </header>

            <div className="bg-white p-8 rounded-lg shadow-md">
                <form onSubmit={handleSubmit}>
                    <div className="space-y-6">
                        <FormInput 
                            label="文章标题" 
                            id="title" 
                            name="title" 
                            type="text" 
                            value={title}
                            onChange={(e) => setTitle(e.target.value)}
                            required 
                            placeholder="例如：如何解决VPN连接问题" 
                        />
                        <div>
                            <label htmlFor="category" className="block text-sm font-medium text-gray-700">分类</label>
                            <select 
                                id="category" 
                                name="category" 
                                value={category}
                                onChange={(e) => setCategory(e.target.value)}
                                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                            >
                                <option>故障排除</option>
                                <option>操作指南</option>
                                <option>常见问题</option>
                                <option>账号管理</option>
                                <option>网络连接</option>
                                <option>云资源</option>
                            </select>
                        </div>
                        <FormTextarea 
                            label="文章内容 (支持Markdown)" 
                            id="content" 
                            name="content" 
                            rows={15} 
                            value={content}
                            onChange={(e) => setContent(e.target.value)}
                            required 
                            placeholder="在这里编写您的知识文章内容，支持Markdown格式。"
                        />
                    </div>
                    <div className="mt-8 flex justify-end">
                        <button type="button" onClick={() => router.back()} className="bg-gray-200 text-gray-700 font-semibold py-2 px-4 rounded-lg hover:bg-gray-300 mr-4">
                            取消
                        </button>
                        <button type="submit" className="bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700">
                            发布文章
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default CreateKnowledgeArticlePage;