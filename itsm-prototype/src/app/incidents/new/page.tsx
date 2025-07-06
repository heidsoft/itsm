'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { FormInput } from '../../components/FormInput';
import { FormTextarea } from '../../components/FormTextarea';

const CreateIncidentPage = () => {
    const router = useRouter();

    const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        const formData = new FormData(e.currentTarget);
        const data = Object.fromEntries(formData.entries());
        console.log('新建事件数据:', JSON.stringify(data, null, 2));
        alert('事件已成功创建！');
        router.push('/incidents'); // 提交后返回事件列表
    };

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <button onClick={() => router.back()} className="flex items-center text-blue-600 hover:underline mb-4">
                    <ArrowLeft className="w-5 h-5 mr-2" />
                    返回事件列表
                </button>
                <h2 className="text-4xl font-bold text-gray-800">新建事件</h2>
                <p className="text-gray-500 mt-1">手动记录新的IT事件</p>
            </header>

            <div className="bg-white p-8 rounded-lg shadow-md">
                <form onSubmit={handleSubmit}>
                    <div className="space-y-6">
                        <FormInput label="事件标题" id="title" name="title" type="text" required placeholder="简要描述事件内容" />
                        <FormTextarea label="详细描述" id="description" name="description" rows={6} required placeholder="请提供事件的详细信息，包括影响范围、发生时间等..." />
                        <div>
                            <label htmlFor="priority" className="block text-sm font-medium text-gray-700">优先级</label>
                            <select id="priority" name="priority" className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" defaultValue="中">
                                <option>高</option>
                                <option>中</option>
                                <option>低</option>
                            </select>
                        </div>
                        <div>
                            <label htmlFor="source" className="block text-sm font-medium text-gray-700">事件来源</label>
                            <select id="source" name="source" className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" defaultValue="服务台">
                                <option>服务台</option>
                                <option>监控系统</option>
                                <option>安全中心</option>
                                <option>用户报告</option>
                            </select>
                        </div>
                    </div>
                    <div className="mt-8 flex justify-end">
                        <button type="button" onClick={() => router.back()} className="bg-gray-200 text-gray-700 font-semibold py-2 px-4 rounded-lg hover:bg-gray-300 mr-4">
                            取消
                        </button>
                        <button type="submit" className="bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700">
                            创建事件
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default CreateIncidentPage;