
'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { Search, AlertTriangle, User, PlusCircle, Filter } from 'lucide-react';

import { mockProblemsData } from '../lib/mock-data';

const PriorityBadge = ({ priority }) => {
    const colors = {
        '高': 'bg-red-100 text-red-800 border-red-300',
        '中': 'bg-yellow-100 text-yellow-800 border-yellow-300',
        '低': 'bg-blue-100 text-blue-800 border-blue-300',
    };
    return <span className={`px-2 py-1 text-xs font-semibold rounded-full border ${colors[priority]}`}>{priority}</span>;
};

const StatusBadge = ({ status }) => {
    const colors = {
        '调查中': 'bg-blue-100 text-blue-800',
        '已解决': 'bg-green-100 text-green-800',
        '已知错误': 'bg-purple-100 text-purple-800',
        '已关闭': 'bg-gray-200 text-gray-800',
    };
    return <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status]}`}>{status}</span>;
};

const ProblemListPage = () => {
    const [filter, setFilter] = useState('全部');

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8 flex justify-between items-center">
                <div>
                    <h2 className="text-4xl font-bold text-gray-800">问题管理</h2>
                    <p className="text-gray-500 mt-1">识别、分析和解决IT服务的根本原因</p>
                </div>
                <Link href="/problems/new" passHref>
                    <button className="flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors">
                        <PlusCircle className="w-5 h-5 mr-2" />
                        新建问题
                    </button>
                </Link>
            </header>

            {/* 筛选器 */}
            <div className="flex items-center mb-6 bg-white p-3 rounded-lg shadow-sm">
                <Filter className="w-5 h-5 text-gray-500 mr-3" />
                <span className="text-sm font-semibold mr-4">筛选:</span>
                <div className="flex space-x-2">
                    {['全部', '调查中', '已解决', '已知错误', '已关闭'].map(f => (
                        <button key={f} onClick={() => setFilter(f)} className={`px-3 py-1 text-sm rounded-md ${filter === f ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-700'}`}>
                            {f}
                        </button>
                    ))}
                </div>
            </div>

            {/* 问题列表表格 */}
            <div className="bg-white rounded-lg shadow-md overflow-hidden">
                <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                        <tr>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">问题ID</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">优先级</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">标题</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">负责人</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">创建时间</th>
                        </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                        {mockProblemsData.map(prob => (
                            <tr key={prob.id} className="hover:bg-gray-50">
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <Link href={`/problems/${prob.id}`} className="text-blue-600 font-semibold hover:underline">
                                        {prob.id}
                                    </Link>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap"><PriorityBadge priority={prob.priority} /></td>
                                <td className="px-6 py-4 whitespace-nowrap max-w-sm truncate">{prob.title}</td>
                                <td className="px-6 py-4 whitespace-nowrap"><StatusBadge status={prob.status} /></td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{prob.assignee}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{prob.createdAt}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default ProblemListPage;
