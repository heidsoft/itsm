'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { ListFilter, PlusCircle, Tag, AlertTriangle, Search, GitMerge, BookOpen, User, Cpu, Shield } from 'lucide-react';

import { mockChangesData, mockIncidentsData, mockProblemsData, mockRequestsData } from '../lib/mock-data';

// 辅助函数：获取优先级颜色和图标
const getPriorityInfo = (priority) => {
    const colors = {
        '紧急': 'bg-red-100 text-red-800 border-red-300',
        '高': 'bg-orange-100 text-orange-800 border-orange-300',
        '中': 'bg-yellow-100 text-yellow-800 border-yellow-300',
        '低': 'bg-blue-100 text-blue-800 border-blue-300',
    };
    const icons = {
        '紧急': AlertTriangle,
        '高': AlertTriangle,
        '中': Tag,
        '低': Tag,
    };
    return { color: colors[priority] || colors['中'], Icon: icons[priority] || Tag };
};

// 辅助函数：获取状态颜色
const getStatusColor = (status) => {
    const colors = {
        '处理中': 'bg-blue-100 text-blue-800',
        '已分配': 'bg-purple-100 text-purple-800',
        '已解决': 'bg-green-100 text-green-800',
        '已关闭': 'bg-gray-200 text-gray-800',
        '待审批': 'bg-yellow-100 text-yellow-800',
        '已批准': 'bg-green-100 text-green-800',
        '实施中': 'bg-blue-100 text-blue-800',
        '已完成': 'bg-gray-200 text-gray-800',
        '已拒绝': 'bg-red-100 text-red-800',
        '调查中': 'bg-blue-100 text-blue-800',
        '已知错误': 'bg-purple-100 text-purple-800',
    };
    return colors[status] || 'bg-gray-100 text-gray-800';
};

const AllTicketsPage = () => {
    const [filterType, setFilterType] = useState('全部');
    const [filterStatus, setFilterStatus] = useState('全部');
    const [filterPriority, setFilterPriority] = useState('全部');

    // 聚合所有工单数据
    const allTickets = [
        ...mockIncidentsData.map(inc => ({ ...inc, type: 'Incident', priority: inc.priority, assignee: inc.assignee || '未分配', lastUpdate: inc.lastUpdate || inc.createdAt })),
        ...mockProblemsData.map(prob => ({ ...prob, type: 'Problem', priority: prob.priority, assignee: prob.assignee || '未分配', lastUpdate: prob.createdAt })),
        ...mockChangesData.map(change => ({ ...change, type: 'Change', priority: change.priority, assignee: change.assignee || '未分配', lastUpdate: change.createdAt })),
        ...mockRequestsData.map(req => ({ id: req.id, title: req.serviceName, type: 'Service Request', status: req.status, priority: '中', assignee: '用户', createdAt: req.submittedAt, lastUpdate: req.submittedAt })),
    ];

    const filteredTickets = allTickets.filter(ticket => {
        const matchesType = filterType === '全部' || ticket.type === filterType;
        const matchesStatus = filterStatus === '全部' || ticket.status === filterStatus;
        const matchesPriority = filterPriority === '全部' || ticket.priority === filterPriority;
        return matchesType && matchesStatus && matchesPriority;
    }).sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()); // 按创建时间倒序

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8 flex justify-between items-center">
                <div>
                    <h2 className="text-4xl font-bold text-gray-800">所有工单</h2>
                    <p className="text-gray-500 mt-1">统一管理所有IT事件、问题、变更和服务请求</p>
                </div>
                <div className="relative group">
                    <button className="flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors">
                        <PlusCircle className="w-5 h-5 mr-2" />
                        新建工单
                    </button>
                    <div className="absolute right-0 mt-2 w-48 bg-white rounded-md shadow-lg py-1 z-10 opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200">
                        <Link href="/incidents/new" className="block px-4 py-2 text-gray-800 hover:bg-gray-100">新建事件</Link>
                        <Link href="/problems/new" className="block px-4 py-2 text-gray-800 hover:bg-gray-100">新建问题</Link>
                        <Link href="/changes/new" className="block px-4 py-2 text-gray-800 hover:bg-gray-100">新建变更</Link>
                        {/* 服务请求通过服务目录发起，这里不直接提供入口 */}
                    </div>
                </div>
            </header>

            {/* 筛选器 */}
            <div className="flex flex-wrap items-center mb-6 bg-white p-3 rounded-lg shadow-sm gap-4">
                <ListFilter className="w-5 h-5 text-gray-500 mr-1" />
                <span className="text-sm font-semibold">筛选:</span>
                
                <select 
                    className="px-3 py-1 text-sm rounded-md border border-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    onChange={(e) => setFilterType(e.target.value)}
                    value={filterType}
                >
                    <option value="全部">所有类型</option>
                    <option value="Incident">事件</option>
                    <option value="Problem">问题</option>
                    <option value="Change">变更</option>
                    <option value="Service Request">服务请求</option>
                </select>

                <select 
                    className="px-3 py-1 text-sm rounded-md border border-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    onChange={(e) => setFilterStatus(e.target.value)}
                    value={filterStatus}
                >
                    <option value="全部">所有状态</option>
                    <option value="处理中">处理中</option>
                    <option value="已分配">已分配</option>
                    <option value="已解决">已解决</option>
                    <option value="已关闭">已关闭</option>
                    <option value="待审批">待审批</option>
                    <option value="已批准">已批准</option>
                    <option value="实施中">实施中</option>
                    <option value="已完成">已完成</option>
                    <option value="已拒绝">已拒绝</option>
                    <option value="调查中">调查中</option>
                    <option value="已知错误">已知错误</option>
                </select>

                <select 
                    className="px-3 py-1 text-sm rounded-md border border-gray-300 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    onChange={(e) => setFilterPriority(e.target.value)}
                    value={filterPriority}
                >
                    <option value="全部">所有优先级</option>
                    <option value="紧急">紧急</option>
                    <option value="高">高</option>
                    <option value="中">中</option>
                    <option value="低">低</option>
                </select>
            </div>

            {/* 工单列表表格 */}
            <div className="bg-white rounded-lg shadow-md overflow-hidden">
                <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                        <tr>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">工单ID</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">类型</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">标题</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">优先级</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">负责人</th>
                            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">最后更新</th>
                        </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                        {filteredTickets.length > 0 ? filteredTickets.map(ticket => (
                            <tr key={ticket.id} className="hover:bg-gray-50">
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <Link href={`/${ticket.type === 'Incident' ? 'incidents' : ticket.type === 'Problem' ? 'problems' : ticket.type === 'Change' ? 'changes' : 'my-requests'}/${ticket.id}`} className="text-blue-600 font-semibold hover:underline">
                                        {ticket.id}
                                    </Link>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{ticket.type}</td>
                                <td className="px-6 py-4 whitespace-nowrap max-w-sm truncate">{ticket.title}</td>
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <span className={`px-2 py-1 text-xs font-medium rounded-full ${getStatusColor(ticket.status)}`}>{ticket.status}</span>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap">
                                    <span className={`px-2 py-1 text-xs font-semibold rounded-full border ${getPriorityInfo(ticket.priority).color}`}>{ticket.priority}</span>
                                </td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-700">{ticket.assignee}</td>
                                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{ticket.lastUpdate || ticket.createdAt}</td>
                            </tr>
                        )) : (
                            <tr>
                                <td colSpan={7} className="px-6 py-4 whitespace-nowrap text-center text-sm text-gray-500">
                                    没有找到匹配的工单。
                                </td>
                            </tr>
                        )}
                    </tbody>
                </table>
            </div>
        </div>
    );
};

export default AllTicketsPage;