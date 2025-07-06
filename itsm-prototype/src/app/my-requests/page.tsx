
'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { ListFilter, Clock, CheckCircle, XCircle, Hourglass } from 'lucide-react';

import { mockRequestsData } from '../lib/mock-data';

const RequestStatusBadge = ({ status }) => {
    const colors = {
        '处理中': 'bg-blue-100 text-blue-800',
        '已完成': 'bg-green-100 text-green-800',
        '待审批': 'bg-yellow-100 text-yellow-800',
        '已拒绝': 'bg-red-100 text-red-800',
    };
    const Icon = {
        '处理中': Hourglass,
        '已完成': CheckCircle,
        '待审批': Hourglass,
        '已拒绝': XCircle,
    }[status];

    return (
        <span className={`px-2 py-1 text-xs font-medium rounded-full flex items-center ${colors[status]}`}>
            {Icon && <Icon className="w-3 h-3 mr-1" />}
            {status}
        </span>
    );
};

const MyRequestsPage = () => {
    const [filter, setFilter] = useState('全部');

    const filteredRequests = mockRequestsData.filter(req => 
        filter === '全部' || req.status === filter
    );

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <h2 className="text-4xl font-bold text-gray-800">我的请求</h2>
                <p className="text-gray-500 mt-1">查看和跟踪您提交的所有服务请求</p>
            </header>

            {/* 筛选器 */}
            <div className="flex items-center mb-6 bg-white p-3 rounded-lg shadow-sm">
                <ListFilter className="w-5 h-5 text-gray-500 mr-3" />
                <span className="text-sm font-semibold mr-4">筛选:</span>
                <div className="flex space-x-2">
                    {['全部', '处理中', '已完成', '待审批', '已拒绝'].map(f => (
                        <button key={f} onClick={() => setFilter(f)} className={`px-3 py-1 text-sm rounded-md ${filter === f ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-700'}`}>
                            {f}
                        </button>
                    ))}
                </div>
            </div>

            {/* 请求列表表格 */}
            <div className="bg-white rounded-lg shadow-md overflow-hidden">
                <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                        <tr>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">请求ID</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">服务名称</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">提交时间</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
                        </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                        {filteredRequests.length > 0 ? filteredRequests.map(req => (
                            <tr key={req.id} className="hover:bg-gray-50">
                                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{req.id}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{req.serviceName}</td>
                                <td className="px-6 py-4 whitespace-nowrap"><RequestStatusBadge status={req.status} /></td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{req.submittedAt}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                                    <Link href={req.detailsLink} className="text-blue-600 hover:text-blue-900">查看详情</Link>
                                </td>
                            </tr>
                        )) : (
                            <tr>
                                <td colSpan={5} className="px-6 py-4 whitespace-nowrap text-center text-sm text-gray-500">
                                    没有找到您的请求。
                                </td>
                            </tr>
                        )}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default MyRequestsPage;
