
'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { Database, Server, Cloud, HardDrive, Filter, PlusCircle } from 'lucide-react';

// 模拟CI数据
const mockCIs = [
    { id: 'CI-ECS-001', name: 'Web服务器-生产环境', type: '云服务器', status: '运行中', business: '电商平台', owner: '运维部', icon: Server },
    { id: 'CI-RDS-001', name: 'CRM数据库-生产环境', type: '云数据库', status: '运行中', business: '销售管理', owner: 'DBA团队', icon: Database },
    { id: 'CI-APP-CRM', name: 'CRM应用系统', type: '应用系统', status: '运行中', business: '销售管理', owner: '开发部', icon: Cloud },
    { id: 'CI-STORAGE-001', name: '对象存储-归档', type: '存储', status: '运行中', business: '数据归档', owner: '存储团队', icon: HardDrive },
];

const CIStatusBadge = ({ status }) => {
    const colors = {
        '运行中': 'bg-green-100 text-green-800',
        '已停止': 'bg-red-100 text-red-800',
        '维护中': 'bg-yellow-100 text-yellow-800',
    };
    return <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status]}`}>{status}</span>;
};

const CMDBListPage = () => {
    const [filter, setFilter] = useState('全部');

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8 flex justify-between items-center">
                <div>
                    <h2 className="text-4xl font-bold text-gray-800">配置管理数据库 (CMDB)</h2>
                    <p className="text-gray-500 mt-1">管理IT资产和配置项的详细信息及相互关系</p>
                </div>
                <Link href="/cmdb/new" passHref>
                    <button className="flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors">
                        <PlusCircle className="w-5 h-5 mr-2" />
                        新建配置项
                    </button>
                </Link>
            </header>

            {/* 筛选器 */}
            <div className="flex items-center mb-6 bg-white p-3 rounded-lg shadow-sm">
                <Filter className="w-5 h-5 text-gray-500 mr-3" />
                <span className="text-sm font-semibold mr-4">筛选:</span>
                <div className="flex space-x-2">
                    {['全部', '运行中', '已停止', '维护中'].map(f => (
                        <button key={f} onClick={() => setFilter(f)} className={`px-3 py-1 text-sm rounded-md ${filter === f ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-700'}`}>
                            {f}
                        </button>
                    ))}
                </div>
            </div>

            {/* CI列表表格 */}
            <div className="bg-white rounded-lg shadow-md overflow-hidden">
                <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                        <tr>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">CI ID</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">名称</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">类型</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">所属业务</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">负责人</th>
                        </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                        {mockCIs.map(ci => (
                            <tr key={ci.id} className="hover:bg-gray-50">
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <Link href={`/cmdb/${ci.id}`} className="text-blue-600 font-semibold hover:underline">
                                        {ci.id}
                                    </Link>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap max-w-sm truncate">{ci.name}</td>
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <div className="flex items-center">
                                        <ci.icon className="w-4 h-4 mr-2 text-gray-600" />
                                        {ci.type}
                                    </div>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap"><CIStatusBadge status={ci.status} /></td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{ci.business}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{ci.owner}</td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default CMDBListPage;
