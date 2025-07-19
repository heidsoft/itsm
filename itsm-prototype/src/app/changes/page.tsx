
'use client';

import { Filter, PlusCircle } from 'lucide-react';

import React, { useState } from 'react';
import Link from 'next/link';
import  from 'lucide-react';

import { mockChangesData } from '../lib/mock-data';

const ChangeTypeBadge = ({ type }) => {
    const colors = {
        '普通变更': 'bg-blue-100 text-blue-800',
        '标准变更': 'bg-green-100 text-green-800',
        '紧急变更': 'bg-red-100 text-red-800',
    };
    return <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[type]}`}>{type}</span>;
};

const ChangeStatusBadge = ({ status }) => {
    const colors = {
        '待审批': 'bg-yellow-100 text-yellow-800',
        '已批准': 'bg-green-100 text-green-800',
        '实施中': 'bg-blue-100 text-blue-800',
        '已完成': 'bg-gray-200 text-gray-800',
        '已拒绝': 'bg-red-100 text-red-800',
    };
    return <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status]}`}>{status}</span>;
};

const ChangePriorityBadge = ({ priority }) => {
    const colors = {
        '紧急': 'bg-red-100 text-red-800 border-red-300',
        '高': 'bg-orange-100 text-orange-800 border-orange-300',
        '中': 'bg-yellow-100 text-yellow-800 border-yellow-300',
        '低': 'bg-blue-100 text-blue-800 border-blue-300',
    };
    return <span className={`px-2 py-1 text-xs font-semibold rounded-full border ${colors[priority]}`}>{priority}</span>;
};

const ChangeListPage = () => {
    const [filter, setFilter] = useState('全部');

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8 flex justify-between items-center">
                <div>
                    <h2 className="text-4xl font-bold text-gray-800">变更管理</h2>
                    <p className="text-gray-500 mt-1">规划、实施和控制IT基础设施和服务的变更</p>
                </div>
                <Link href="/changes/new" passHref>
                    <button className="flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors">
                        <PlusCircle className="w-5 h-5 mr-2" />
                        新建变更
                    </button>
                </Link>
            </header>

            {/* 筛选器 */}
            <div className="flex items-center mb-6 bg-white p-3 rounded-lg shadow-sm">
                <Filter className="w-5 h-5 text-gray-500 mr-3" />
                <span className="text-sm font-semibold mr-4">筛选:</span>
                <div className="flex space-x-2">
                    {['全部', '待审批', '已批准', '实施中', '已完成', '已拒绝'].map(f => (
                        <button key={f} onClick={() => setFilter(f)} className={`px-3 py-1 text-sm rounded-md ${filter === f ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-700'}`}>
                            {f}
                        </button>
                    ))}
                </div>
            </div>

            {/* 变更列表表格 */}
            <div className="bg-white rounded-lg shadow-md overflow-hidden">
                <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                        <tr>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">变更ID</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">类型</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">标题</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">优先级</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">负责人</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">创建时间</th>
                        </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                        {mockChangesData.map(change => (
                            <tr key={change.id} className="hover:bg-gray-50">
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <Link href={`/changes/${change.id}`} className="text-blue-600 font-semibold hover:underline">
                                        {change.id}
                                    </Link>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap"><ChangeTypeBadge type={change.type} /></td>
                                <td className="px-6 py-4 whitespace-nowrap max-w-sm truncate">{change.title}</td>
                                <td className="px-6 py-4 whitespace-nowrap"><ChangeStatusBadge status={change.status} /></td>
                                <td className="px-6 py-4 whitespace-nowrap"><ChangePriorityBadge priority={change.priority} /></td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{change.assignee}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{change.createdAt}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default ChangeListPage;
