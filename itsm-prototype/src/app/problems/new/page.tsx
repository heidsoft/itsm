'use client';

import React, { useState, useEffect } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { FormInput } from '../../components/FormInput';
import { FormTextarea } from '../../components/FormTextarea';

const CreateProblemPage = () => {
    const router = useRouter();
    const searchParams = useSearchParams();

    const [title, setTitle] = useState('');
    const [description, setDescription] = useState('');
    const [relatedIncidentId, setRelatedIncidentId] = useState('');

    useEffect(() => {
        const incidentId = searchParams.get('fromIncidentId');
        const incidentTitle = searchParams.get('incidentTitle');
        const incidentDescription = searchParams.get('incidentDescription');

        if (incidentId) {
            setRelatedIncidentId(incidentId);
            setTitle(`由事件 ${incidentId} 引起的问题: ${incidentTitle || ''}`);
            setDescription(`此问题由以下事件引发：\n事件ID: ${incidentId}\n事件标题: ${incidentTitle || ''}\n事件描述: ${incidentDescription || ''}\n\n请在此处填写问题的详细描述和根本原因分析...`);
        }
    }, [searchParams]);

    const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        const formData = new FormData(e.currentTarget);
        const data = Object.fromEntries(formData.entries());
        console.log('新建问题数据:', JSON.stringify(data, null, 2));
        alert('问题已成功创建！');
        router.push('/problems'); // 提交后返回问题列表
    };

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <button onClick={() => router.back()} className="flex items-center text-blue-600 hover:underline mb-4">
                    <ArrowLeft className="w-5 h-5 mr-2" />
                    返回问题列表
                </button>
                <h2 className="text-4xl font-bold text-gray-800">新建问题</h2>
                <p className="text-gray-500 mt-1">识别、分析和解决IT服务的根本原因</p>
            </header>

            <div className="bg-white p-8 rounded-lg shadow-md">
                <form onSubmit={handleSubmit}>
                    <div className="space-y-6">
                        {relatedIncidentId && (
                            <div className="p-4 bg-blue-50 border-l-4 border-blue-500 text-blue-700 rounded-md">
                                <p className="font-semibold">此问题由事件 ${relatedIncidentId} 触发。</p>
                            </div>
                        )}
                        <FormInput 
                            label="问题标题" 
                            id="title" 
                            name="title" 
                            type="text" 
                            value={title}
                            onChange={(e) => setTitle(e.target.value)}
                            required 
                            placeholder="简要描述问题内容" 
                        />
                        <FormTextarea 
                            label="详细描述" 
                            id="description" 
                            name="description" 
                            rows={6} 
                            value={description}
                            onChange={(e) => setDescription(e.target.value)}
                            required 
                            placeholder="请提供问题的详细信息，包括影响范围、发生时间、已观察到的现象等..."
                        />
                        <div>
                            <label htmlFor="priority" className="block text-sm font-medium text-gray-700">优先级</label>
                            <select id="priority" name="priority" className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" defaultValue="中">
                                <option>高</option>
                                <option>中</option>
                                <option>低</option>
                            </select>
                        </div>
                        <FormTextarea label="根本原因分析 (RCA)" id="rootCause" name="rootCause" rows={4} placeholder="请详细说明问题的根本原因..." />
                        <FormTextarea label="临时解决方案" id="temporarySolution" name="temporarySolution" rows={3} placeholder="如果存在，请说明已采取的临时措施..." />
                        <div>
                            <label htmlFor="knownError" className="block text-sm font-medium text-gray-700">是否为已知错误</label>
                            <input type="checkbox" id="knownError" name="knownError" className="ml-2 h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500" />
                        </div>
                    </div>
                    <div className="mt-8 flex justify-end">
                        <button type="button" onClick={() => router.back()} className="bg-gray-200 text-gray-700 font-semibold py-2 px-4 rounded-lg hover:bg-gray-300 mr-4">
                            取消
                        </button>
                        <button type="submit" className="bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700">
                            创建问题
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
};

export default CreateProblemPage;