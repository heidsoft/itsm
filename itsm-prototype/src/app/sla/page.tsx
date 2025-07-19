'use client';

import { Filter, PlusCircle } from 'lucide-react';

import React, { useState } from 'react';
import Link from 'next/link';
import  from 'lucide-react';

// 模拟SLA数据
const mockSLAs = [
    { id: 'SLA-001', name: '生产CRM系统可用性', service: 'CRM系统', target: '99.9% 可用性', actual: '99.85%', status: '轻微违约', lastReview: '2025-06-01' },
    { id: 'SLA-002', name: '内部IT服务台响应时间', service: 'IT服务台', target: '80% 15分钟内响应', actual: '85%', status: '达标', lastReview: '2025-06-15' },
    { id: 'SLA-003', name: '云资源申请交付时间', service: '云资源服务', target: '90% 2工作日内交付', actual: '88%', status: '违约', lastReview: '2025-06-20' },
    { id: 'SLA-004', name: '邮件服务可用性', service: '邮件服务', target: '99.99% 可用性', actual: '99.99%', status: '达标', lastReview: '2025-06-05' },
];

const SLAStatusBadge = ({ status }) => {
    const colors = {
        '达标': 'bg-green-100 text-green-800',
        '轻微违约': 'bg-yellow-100 text-yellow-800',
        '违约': 'bg-red-100 text-red-800',
    };
    return <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status]}`}>{status}</span>;
};

const SLAListPage = () => {
    const [filter, setFilter] = useState('全部');

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8 flex justify-between items-center">
                <div>
                    <h2 className="text-4xl font-bold text-gray-800">服务级别管理</h2>
                    <p className="text-gray-500 mt-1">定义、监控和管理IT服务的性能和质量</p>
                </div>
                <Link href="/sla/new" passHref>
                    <button className="flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors">
                        <PlusCircle className="w-5 h-5 mr-2" />
                        新建SLA
                    </button>
                </Link>
            </header>

            {/* 筛选器 */}
            <div className="flex items-center mb-6 bg-white p-3 rounded-lg shadow-sm">
                <Filter className="w-5 h-5 text-gray-500 mr-3" />
                <span className="text-sm font-semibold mr-4">筛选:</span>
                <div className="flex space-x-2">
                    {['全部', '达标', '轻微违约', '违约'].map(f => (
                        <button key={f} onClick={() => setFilter(f)} className={`px-3 py-1 text-sm rounded-md ${filter === f ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-700'}`}>
                            {f}
                        </button>
                    ))}
                </div>
            </div>

            {/* SLA列表表格 */}
            <div className="bg-white rounded-lg shadow-md overflow-hidden">
                <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                        <tr>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">SLA ID</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">SLA 名称</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">服务对象</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">目标</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">实际达成</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">最后评审</th>
                        </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                        {mockSLAs.map(sla => (
                            <tr key={sla.id} className="hover:bg-gray-50">
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <Link href={`/sla/${sla.id}`} className="text-blue-600 font-semibold hover:underline">
                                        {sla.id}
                                    </Link>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap max-w-sm truncate">{sla.name}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{sla.service}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{sla.target}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{sla.actual}</td>
                                <td className="px-6 py-4 whitespace-nowrap"><SLAStatusBadge status={sla.status} /></td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{sla.lastReview}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default SLAListPage;