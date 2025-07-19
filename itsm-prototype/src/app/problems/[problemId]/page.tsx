'use client';

import { GitMerge, BookOpen, AlertTriangle, MessageSquare, ArrowLeft } from 'lucide-react';

import React, { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
// 模拟问题详情数据
const mockProblemDetail = {
    'PRB-00001': {
        title: 'CRM系统间歇性登录失败',
        description: '用户报告CRM系统在高峰期出现间歇性登录失败，表现为登录页面卡顿或直接报错。初步怀疑是认证服务或数据库连接问题。',
        status: '调查中',
        priority: '高',
        assignee: '王五',
        createdAt: '2025-06-20 14:30:00',
        rootCause: '待定',
        temporarySolution: '无',
        knownError: false,
        relatedIncidents: [
            { id: 'INC-00124', title: '用户报告无法访问CRM系统', status: '已解决' },
            { id: 'INC-00118', title: 'CRM登录超时告警', status: '已解决' },
        ],
        logs: [
            '[2025-06-20 14:35] 问题创建，并分配给应用支持团队。',
            '[2025-06-20 15:00] 王五开始调查，收集CRM应用日志和认证服务日志。',
            '[2025-06-21 09:00] 初步分析日志，发现认证服务在高峰期有大量慢查询。',
        ],
        comments: [
            { author: '王五', timestamp: '2025-06-20 14:40', text: '已收到问题，正在收集相关日志。' },
        ]
    },
    'PRB-00002': {
        title: '部分用户无法访问VPN',
        description: '多名远程办公用户报告无法连接公司VPN，或连接后无法访问内部资源。怀疑是VPN网关或认证服务配置问题。',
        status: '已解决',
        priority: '中',
        assignee: '李明',
        createdAt: '2025-06-15 10:00:00',
        rootCause: 'VPN网关配置错误',
        temporarySolution: '无',
        knownError: false,
        relatedIncidents: [
            { id: 'INC-00110', title: 'VPN连接失败告警', status: '已解决' },
            { id: 'INC-00105', title: '用户报告无法访问内部网络', status: '已解决' },
        ],
        logs: [
            '[2025-06-15 10:05] 问题创建，并分配给网络团队。',
            '[2025-06-15 11:30] 李明排查发现VPN网关路由配置有误。',
            '[2025-06-15 12:00] 已修复配置，VPN服务恢复正常。',
            '[2025-06-15 12:15] 问题状态更新为已解决。',
        ],
        comments: []
    },
    'PRB-00003': {
        title: '数据库连接池耗尽导致应用崩溃',
        description: '生产环境应用频繁出现数据库连接池耗尽错误，导致应用服务中断。怀疑是应用代码未正确释放连接或数据库连接数设置不足。',
        status: '已知错误',
        priority: '高',
        assignee: '钱七',
        createdAt: '2025-06-10 09:00:00',
        rootCause: '应用代码未正确释放数据库连接',
        temporarySolution: '定时重启应用服务',
        knownError: true,
        relatedIncidents: [
            { id: 'INC-00100', title: '应用服务崩溃告警', status: '已解决' },
            { id: 'INC-00095', title: '数据库连接数超限告警', status: '已解决' },
        ],
        logs: [
            '[2025-06-10 09:05] 问题创建，并分配给开发团队。',
            '[2025-06-10 10:30] 钱七开始代码审查和数据库监控。',
            '[2025-06-11 14:00] 确认是应用代码逻辑问题，需要发布新版本修复。已记录为已知错误，并提供临时解决方案。',
        ],
        comments: []
    },
};

const ApprovalStatusBadge = ({ status }) => {
    const colors = {
        '待审批': 'bg-yellow-100 text-yellow-800',
        '已批准': 'bg-green-100 text-green-800',
        '已拒绝': 'bg-red-100 text-red-800',
    };
    return <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status]}`}>{status}</span>;
};

const ProblemDetailPage = () => {
    const params = useParams();
    const router = useRouter();
    const problemId = params.problemId as string;
    const [problem, setProblem] = useState(mockProblemDetail[problemId]);
    const [newComment, setNewComment] = useState('');

    const handleCreateChange = () => {
        alert(`已从问题 ${problemId} 创建变更请求！\n（此为模拟操作，实际会跳转到变更创建页面）`);
        // 实际应用中，这里会跳转到变更创建页面，并预填充问题信息
        // router.push(`/changes/create?fromProblem=${problemId}`);
    };

    const handlePublishToKB = () => {
        // 实际应用中，这里会跳转到知识库文章编辑页面，并预填充问题解决方案
        router.push(`/knowledge-base/new?fromProblemId=${problemId}&problemTitle=${encodeURIComponent(problem.title)}&problemSolution=${encodeURIComponent(problem.rootCause + (problem.temporarySolution ? '\n临时解决方案：' + problem.temporarySolution : ''))}`);
    };

    const handleAddComment = () => {
        if (newComment.trim()) {
            const updatedProblem = {
                ...problem,
                comments: [...problem.comments, { author: '当前用户', timestamp: new Date().toLocaleString(), text: newComment.trim() }]
            };
            setProblem(updatedProblem);
            setNewComment('');
            // 在真实应用中，这里会调用API保存评论
        }
    };

    if (!problem) {
        return <div className="p-10">问题不存在或加载失败。</div>;
    }

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <button onClick={() => router.back()} className="flex items-center text-blue-600 hover:underline mb-4">
                    <ArrowLeft className="w-5 h-5 mr-2" />
                    返回问题列表
                </button>
                <div className="flex justify-between items-start">
                    <div>
                        <h2 className="text-4xl font-bold text-gray-800">问题：{problem.title}</h2>
                        <p className="text-gray-500 mt-1">问题ID: {problemId}</p>
                    </div>
                    <div className="flex space-x-4">
                        <button 
                            onClick={handleCreateChange}
                            className="flex items-center bg-purple-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-purple-700 transition-colors"
                        >
                            <GitMerge className="w-5 h-5 mr-2" />
                            创建变更
                        </button>
                        <button 
                            onClick={handlePublishToKB}
                            className="flex items-center bg-green-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-green-700 transition-colors"
                        >
                            <BookOpen className="w-5 h-5 mr-2" />
                            发布到知识库
                        </button>
                    </div>
                </div>
            </header>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                {/* 左侧：问题详情和处理日志 */}
                <div className="lg:col-span-2 bg-white p-8 rounded-lg shadow-md">
                    <h3 className="text-xl font-semibold text-gray-700 mb-4">问题描述</h3>
                    <p className="text-gray-600 mb-8">{problem.description}</p>
                    
                    <h3 className="text-xl font-semibold text-gray-700 mb-4">根本原因分析 (RCA)</h3>
                    <p className="text-gray-600 mb-8">{problem.rootCause}</p>

                    {problem.temporarySolution && (
                        <div className="mb-8">
                            <h3 className="text-xl font-semibold text-gray-700 mb-4">临时解决方案</h3>
                            <p className="text-gray-600">{problem.temporarySolution}</p>
                        </div>
                    )}

                    <h3 className="text-xl font-semibold text-gray-700 mb-4">处理日志</h3>
                    <div className="space-y-4">
                        {problem.logs.map((log, i) => (
                            <p key={i} className="text-sm text-gray-500 font-mono">{log}</p>
                        ))}
                    </div>

                    {/* 评论/备注区域 */}
                    <div className="mt-8 pt-8 border-t border-gray-200">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <MessageSquare className="w-5 h-5 mr-2" /> 内部评论/备注
                        </h3>
                        <div className="space-y-4 mb-4">
                            {problem.comments.length > 0 ? problem.comments.map((comment, i) => (
                                <div key={i} className="bg-gray-50 p-3 rounded-lg">
                                    <p className="text-sm font-semibold text-gray-800">{comment.author} <span className="text-gray-500 font-normal">于 {comment.timestamp}</span></p>
                                    <p className="text-gray-700 mt-1">{comment.text}</p>
                                </div>
                            )) : (
                                <p className="text-sm text-gray-500">暂无评论。</p>
                            )}
                        </div>
                        <textarea 
                            className="w-full p-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                            rows={3}
                            placeholder="添加内部评论或备注..."
                            value={newComment}
                            onChange={(e) => setNewComment(e.target.value)}
                        ></textarea>
                        <button 
                            onClick={handleAddComment}
                            className="mt-2 bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors"
                        >
                            添加评论
                        </button>
                    </div>
                </div>

                {/* 右侧：元数据和联动功能 */}
                <div className="space-y-8">
                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4">问题信息</h3>
                        <div className="space-y-3 text-sm">
                            <div className="flex justify-between"><span>状态:</span><span className="font-semibold">{problem.status}</span></div>
                            <div className="flex justify-between"><span>优先级:</span><span className="font-semibold text-red-600">{problem.priority}</span></div>
                            <div className="flex justify-between"><span>负责人:</span><span className="font-semibold">{problem.assignee}</span></div>
                            <div className="flex justify-between"><span>创建时间:</span><span className="font-semibold">{problem.createdAt}</span></div>
                            <div className="flex justify-between"><span>已知错误:</span><span className="font-semibold">{problem.knownError ? '是' : '否'}</span></div>
                        </div>
                    </div>

                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <AlertTriangle className="w-5 h-5 mr-2" /> 关联事件
                        </h3>
                        <div className="space-y-3">
                            {problem.relatedIncidents.map(inc => (
                                <Link key={inc.id} href={`/incidents/${inc.id}`} className="block p-3 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors">
                                    <p className="font-semibold text-blue-700">{inc.id} ({inc.type})</p>
                                    <p className="text-sm text-gray-600">{inc.title}</p>
                                    <span className="text-xs text-gray-500">状态: {inc.status}</span>
                                </Link>
                            ))}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default ProblemDetailPage;