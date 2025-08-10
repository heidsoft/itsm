'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Search, X } from 'lucide-react';
import { FormInput } from '../../components/FormInput';
import { FormTextarea } from '../../components/FormTextarea';
import { IncidentAPI } from '../../lib/incident-api';

interface ConfigurationItem {
  id: number;
  name: string;
  type: string;
  status: string;
  description?: string;
}

const CreateIncidentPage = () => {
    const router = useRouter();
    const [configurationItems, setConfigurationItems] = useState<ConfigurationItem[]>([]);
    const [selectedCI, setSelectedCI] = useState<ConfigurationItem | null>(null);
    const [searchTerm, setSearchTerm] = useState('');
    const [isSearching, setIsSearching] = useState(false);
    const [showCISelector, setShowCISelector] = useState(false);

    // 搜索配置项
    const searchConfigurationItems = async (search: string) => {
        if (!search.trim()) {
            setConfigurationItems([]);
            return;
        }
        
        setIsSearching(true);
        try {
            const items = await IncidentAPI.getConfigurationItems(search);
            setConfigurationItems(items);
        } catch (error) {
            console.error('搜索配置项失败:', error);
            setConfigurationItems([]);
        } finally {
            setIsSearching(false);
        }
    };

    // 选择配置项
    const selectConfigurationItem = (ci: ConfigurationItem) => {
        setSelectedCI(ci);
        setShowCISelector(false);
        setSearchTerm('');
        setConfigurationItems([]);
    };

    // 移除已选择的配置项
    const removeSelectedCI = () => {
        setSelectedCI(null);
    };

    // 处理搜索输入变化
    useEffect(() => {
        const timer = setTimeout(() => {
            searchConfigurationItems(searchTerm);
        }, 300);
        
        return () => clearTimeout(timer);
    }, [searchTerm]);

    const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        const formData = new FormData(e.currentTarget);
        const data = Object.fromEntries(formData.entries());
        
        // 添加配置项ID
        if (selectedCI) {
            data.configuration_item_id = selectedCI.id.toString();
        }
        
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
                        
                        {/* 配置项关联 */}
                        <div>
                            <label className="block text-sm font-medium text-gray-700 mb-2">
                                关联配置项
                            </label>
                            {selectedCI ? (
                                <div className="flex items-center justify-between p-3 bg-blue-50 border border-blue-200 rounded-lg">
                                    <div>
                                        <div className="font-medium text-blue-900">{selectedCI.name}</div>
                                        <div className="text-sm text-blue-700">
                                            {selectedCI.type} • {selectedCI.status}
                                        </div>
                                        {selectedCI.description && (
                                            <div className="text-xs text-blue-600 mt-1">{selectedCI.description}</div>
                                        )}
                                    </div>
                                    <button
                                        type="button"
                                        onClick={removeSelectedCI}
                                        className="text-blue-500 hover:text-blue-700"
                                    >
                                        <X className="w-4 h-4" />
                                    </button>
                                </div>
                            ) : (
                                <div className="relative">
                                    <div className="flex">
                                        <input
                                            type="text"
                                            placeholder="搜索配置项..."
                                            value={searchTerm}
                                            onChange={(e) => setSearchTerm(e.target.value)}
                                            onFocus={() => setShowCISelector(true)}
                                            className="flex-1 rounded-l-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                                        />
                                        <button
                                            type="button"
                                            onClick={() => setShowCISelector(!showCISelector)}
                                            className="px-3 py-2 bg-gray-100 border border-l-0 border-gray-300 rounded-r-md hover:bg-gray-200"
                                        >
                                            <Search className="w-4 h-4" />
                                        </button>
                                    </div>
                                    
                                    {/* 配置项选择器 */}
                                    {showCISelector && (searchTerm || configurationItems.length > 0) && (
                                        <div className="absolute z-10 w-full mt-1 bg-white border border-gray-300 rounded-md shadow-lg max-h-60 overflow-auto">
                                            {isSearching ? (
                                                <div className="p-3 text-center text-gray-500">搜索中...</div>
                                            ) : configurationItems.length > 0 ? (
                                                <div>
                                                    {configurationItems.map((ci) => (
                                                        <button
                                                            key={ci.id}
                                                            type="button"
                                                            onClick={() => selectConfigurationItem(ci)}
                                                            className="w-full text-left p-3 hover:bg-gray-50 border-b border-gray-100 last:border-b-0"
                                                        >
                                                            <div className="font-medium text-gray-900">{ci.name}</div>
                                                            <div className="text-sm text-gray-600">
                                                                {ci.type} • {ci.status}
                                                            </div>
                                                            {ci.description && (
                                                                <div className="text-xs text-gray-500 mt-1 truncate">
                                                                    {ci.description}
                                                                </div>
                                                            )}
                                                        </button>
                                                    ))}
                                                </div>
                                            ) : searchTerm && !isSearching ? (
                                                <div className="p-3 text-center text-gray-500">未找到匹配的配置项</div>
                                            ) : null}
                                        </div>
                                    )}
                                </div>
                            )}
                        </div>

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