
'use client';

import React from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, AlertTriangle, Search, GitMerge, Cpu, Server, Cloud, HardDrive, Network, Database, Zap } from 'lucide-react';

// 模拟CI详情数据
const mockCIDetail = {
    'CI-ECS-001': {
        name: 'Web服务器-生产环境',
        type: '云服务器',
        status: '运行中',
        business: '电商平台',
        owner: '运维部',
        description: '承载电商平台前端Web服务的阿里云ECS实例，位于华东1可用区J。',
        attributes: {
            '实例ID': 'i-bp1abcdefg',
            'IP地址': '192.168.1.100 (内网), 47.98.x.x (公网)',
            '操作系统': 'CentOS 7.9',
            'CPU': '8核',
            '内存': '16GB',
            '磁盘': '200GB SSD',
            '创建日期': '2024-01-01',
        },
        relatedTickets: [
            { id: 'INC-00125', type: '事件', title: 'Web服务器CPU使用率超过95%', status: '处理中' },
            { id: 'PRB-00003', type: '问题', title: '数据库连接池耗尽导致应用崩溃', status: '已知错误' },
            { id: 'CHG-00003', type: '变更', title: '紧急修复Web服务器安全漏洞', status: '实施中' },
        ],
        relationships: [
            { targetId: 'CI-APP-CRM', targetName: 'CRM应用系统', type: '承载' },
            { targetId: 'CI-RDS-001', targetName: 'CRM数据库-生产环境', type: '连接到' },
        ],
        // 新增影响关系
        impacts: [
            { id: 'CI-APP-CRM', name: 'CRM应用系统', type: '应用系统', impactLevel: '高' },
            { id: 'SLA-001', name: '生产CRM系统可用性', type: 'SLA', impactLevel: '高' },
        ],
        icon: Server,
    },
    'CI-RDS-001': {
        name: 'CRM数据库-生产环境',
        type: '云数据库',
        status: '运行中',
        business: '销售管理',
        owner: 'DBA团队',
        description: '电商平台CRM系统使用的MySQL数据库实例，高可用配置。',
        attributes: {
            '实例ID': 'rm-bp1abcdefg',
            '数据库类型': 'MySQL 8.0',
            '实例规格': '4核 16GB',
            '存储空间': '500GB SSD',
            '创建日期': '2023-10-01',
        },
        relatedTickets: [
            { id: 'INC-00122', type: '事件', title: '生产数据库主备同步延迟', status: '已解决' },
        ],
        relationships: [
            { targetId: 'CI-ECS-001', targetName: 'Web服务器-生产环境', type: '被连接' },
        ],
        // 新增影响关系
        impacts: [
            { id: 'CI-APP-CRM', name: 'CRM应用系统', type: '应用系统', impactLevel: '高' },
            { id: 'CI-ECS-001', name: 'Web服务器-生产环境', type: '云服务器', impactLevel: '中' },
            { id: 'SLA-001', name: '生产CRM系统可用性', type: 'SLA', impactLevel: '高' },
        ],
        icon: Database,
    },
    'CI-APP-CRM': {
        name: 'CRM应用系统',
        type: '应用系统',
        status: '运行中',
        business: '销售管理',
        owner: '开发部',
        description: '公司内部使用的客户关系管理系统。',
        attributes: {
            '版本': 'V3.2.1',
            '开发语言': 'Java',
            '部署环境': '生产',
        },
        relatedTickets: [
            { id: 'INC-00124', type: '事件', title: '用户报告无法访问CRM系统', status: '已分配' },
            { id: 'PRB-00001', type: '问题', title: 'CRM系统间歇性登录失败', status: '调查中' },
        ],
        relationships: [
            { targetId: 'CI-ECS-001', targetName: 'Web服务器-生产环境', type: '部署在' },
            { targetId: 'CI-RDS-001', targetName: 'CRM数据库-生产环境', type: '使用' },
        ],
        // 新增影响关系
        impacts: [
            { id: 'SLA-001', name: '生产CRM系统可用性', type: 'SLA', impactLevel: '高' },
            { id: 'REQ-00101', name: '用户无法登录CRM', type: '服务请求', impactLevel: '高' },
        ],
        icon: Cloud,
    },
};

const CIDetailPage = () => {
    const params = useParams();
    const router = useRouter();
    const ciId = params.ciId as string;
    const ci = mockCIDetail[ciId];

    if (!ci) {
        return <div className="p-10">配置项不存在或加载失败。</div>;
    }

    const Icon = ci.icon;

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <button onClick={() => router.back()} className="flex items-center text-blue-600 hover:underline mb-4">
                    <ArrowLeft className="w-5 h-5 mr-2" />
                    返回CMDB列表
                </button>
                <h2 className="text-4xl font-bold text-gray-800">配置项详情：{ci.name}</h2>
                <p className="text-gray-500 mt-1">CI ID: {ciId}</p>
            </header>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                {/* 左侧：CI属性和描述 */}
                <div className="lg:col-span-2 bg-white p-8 rounded-lg shadow-md">
                    <div className="flex items-center mb-4">
                        <Icon className="w-8 h-8 mr-3 text-blue-600" />
                        <h3 className="text-2xl font-semibold text-gray-700">基本信息</h3>
                    </div>
                    <p className="text-gray-600 mb-8">{ci.description}</p>

                    <h3 className="text-xl font-semibold text-gray-700 mb-4">属性</h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
                        {Object.entries(ci.attributes).map(([key, value]) => (
                            <div key={key} className="flex justify-between items-center border-b border-gray-100 py-2">
                                <span className="font-medium text-gray-600">{key}:</span>
                                <span className="text-gray-800">{value}</span>
                            </div>
                        ))}
                    </div>

                    <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                        <Network className="w-5 h-5 mr-2" /> 关系图 (模拟)
                    </h3>
                    <div className="bg-gray-50 p-6 rounded-lg border border-gray-200">
                        <p className="text-gray-600">此区域将展示配置项之间的关系图，例如：</p>
                        <ul className="list-disc list-inside text-gray-700 mt-2">
                            {ci.relationships.map((rel, i) => (
                                <li key={i}>
                                    {ci.name} <span className="font-semibold text-blue-600">{rel.type}</span> <Link href={`/cmdb/${rel.targetId}`} className="text-blue-600 hover:underline">{rel.targetName}</Link>
                                </li>
                            ))}
                        </ul>
                        <p className="text-sm text-gray-500 mt-4">（在实际产品中，这里会是一个交互式的关系拓扑图）</p>
                    </div>
                </div>

                {/* 右侧：元数据和关联工单 */}
                <div className="space-y-8">
                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4">基本信息</h3>
                        <div className="space-y-3 text-sm">
                            <div className="flex justify-between"><span>类型:</span><span className="font-semibold">{ci.type}</span></div>
                            <div className="flex justify-between"><span>状态:</span><span className="font-semibold">{ci.status}</span></div>
                            <div className="flex justify-between"><span>所属业务:</span><span className="font-semibold">{ci.business}</span></div>
                            <div className="flex justify-between"><span>负责人:</span><span className="font-semibold">{ci.owner}</span></div>
                        </div>
                    </div>

                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <AlertTriangle className="w-5 h-5 mr-2" /> 关联工单
                        </h3>
                        <div className="space-y-3">
                            {ci.relatedTickets.length > 0 ? ci.relatedTickets.map(ticket => (
                                <Link 
                                    key={ticket.id} 
                                    href={`/${ticket.type === '事件' ? 'incidents' : ticket.type === '问题' ? 'problems' : 'changes'}/${ticket.id}`} 
                                    className="block p-3 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors"
                                >
                                    <p className="font-semibold text-blue-700">{ticket.id} ({ticket.type})</p>
                                    <p className="text-sm text-gray-600">{ticket.title}</p>
                                    <span className="text-xs text-gray-500">状态: {ticket.status}</span>
                                </Link>
                            )) : (
                                <p className="text-sm text-gray-500">无关联工单。</p>
                            )}
                        </div>
                    </div>

                    {/* 新增：影响分析区域 */}
                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <Zap className="w-5 h-5 mr-2" /> 影响分析
                        </h3>
                        <div className="space-y-3">
                            {ci.impacts.length > 0 ? ci.impacts.map(impact => (
                                <Link 
                                    key={impact.id} 
                                    href={`/${impact.type === 'SLA' ? 'sla' : 'cmdb'}/${impact.id}`} 
                                    className="block p-3 bg-red-50 rounded-lg hover:bg-red-100 transition-colors"
                                >
                                    <p className="font-semibold text-red-700">{impact.name} ({impact.type})</p>
                                    <span className="text-xs text-gray-500">影响级别: {impact.impactLevel}</span>
                                </Link>
                            )) : (
                                <p className="text-sm text-gray-500">无直接影响。</p>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default CIDetailPage;
