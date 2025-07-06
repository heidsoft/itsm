'use client';

import React, { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, Cpu, Link as LinkIcon, Search, BookOpen, PlusCircle, CheckCircle, PauseCircle, XCircle, Zap, Clock, MessageSquare, PlayCircle } from 'lucide-react';

// 模拟数据
const mockIncidentDetail = {
    'INC-00125': {
        title: '杭州可用区J的Web服务器CPU使用率超过95%',
        priority: '高', 
        status: '处理中',
        source: '阿里云监控',
        reporter: 'system',
        assignee: '张三',
        createdAt: '2025-06-28 10:15:23',
        confirmedAt: '2025-06-28 10:18:00',
        resolvedAt: null,
        isMajorIncident: false,
        description: '监控系统检测到实例 i-bp1abcdefg 的CPU使用率在过去15分钟内持续高于95%。可能原因：应用程序内存泄漏或流量突增。需要立即排查。',
        affectedCI: { name: 'i-bp1abcdefg', type: '云服务器', link: '/cmdb/ci/i-bp1abcdefg', id: 'CI-ECS-001' },
        logs: [
            '[10:15] 事件创建并自动分配给云资源团队。',
            '[10:18] 张三已确认接收事件。',
            '[10:25] 张三正在登录服务器排查进程... ',
            '[10:35] 发现某应用进程占用大量CPU，已尝试重启服务。',
            '[10:40] CPU使用率回落至正常水平，事件状态更新为已解决。',
        ],
        comments: [
            { author: '张三', timestamp: '2025-06-28 10:20', text: '已收到事件，正在排查中。' },
        ]
    },
    'INC-00124': {
        title: '用户报告无法访问CRM系统',
        priority: '高', 
        status: '已分配',
        source: '服务台',
        reporter: '李四',
        assignee: '王五',
        createdAt: '2025-06-28 09:45:10',
        confirmedAt: '2025-06-28 09:50:00',
        resolvedAt: null,
        isMajorIncident: false,
        description: '用户李四报告无法通过浏览器访问CRM系统，尝试清除缓存和更换浏览器无效。怀疑是网络或应用服务问题。',
        affectedCI: { name: 'CRM-APP-SERVER', type: '应用服务器', link: '/cmdb/ci/CRM-APP-SERVER', id: 'CI-APP-CRM' },
        logs: [
            '[09:45] 事件创建并分配给应用支持团队。',
            '[09:50] 王五已确认接收事件，开始排查。',
            '[10:05] 初步判断为CRM应用服务异常，正在尝试重启。',
        ],
        comments: []
    },
    'INC-00123': {
        title: '检测到可疑的SSH登录尝试 (47.98.x.x)',
        priority: '中', 
        status: '处理中',
        source: '安全中心',
        reporter: 'system',
        assignee: '赵六',
        createdAt: '2025-06-28 08:30:00',
        confirmedAt: '2025-06-28 08:35:00',
        resolvedAt: null,
        isMajorIncident: true, // 模拟一个重大事件
        description: '安全系统检测到来自未知IP地址 (47.98.x.x) 对生产环境服务器的多次SSH登录失败尝试。可能存在暴力破解风险。',
        affectedCI: { name: 'PROD-BASTION-HOST', type: '跳板机', link: '/cmdb/ci/PROD-BASTION-HOST', id: 'CI-PROD-BASTION-HOST' },
        logs: [
            '[08:30] 事件创建并自动分配给安全团队。',
            '[08:35] 赵六已确认接收事件，正在分析日志。',
            '[08:45] 已将可疑IP加入防火墙黑名单。',
        ],
        comments: []
    },
    'INC-00122': {
        title: '生产数据库主备同步延迟',
        priority: '中', 
        status: '已解决',
        source: '阿里云监控',
        reporter: 'system',
        assignee: '钱七',
        createdAt: '2025-06-27 18:00:00',
        confirmedAt: '2025-06-27 18:05:00',
        resolvedAt: '2025-06-27 18:45:00',
        isMajorIncident: false,
        description: '监控系统告警，生产数据库（RDS）主备同步延迟超过阈值（30秒）。可能影响数据一致性。',
        affectedCI: { name: 'RDS-PROD-DB', type: '云数据库', link: '/cmdb/ci/RDS-PROD-DB', id: 'CI-RDS-001' },
        logs: [
            '[18:00] 事件创建并自动分配给数据库团队。',
            '[18:05] 钱七已确认接收事件，开始排查。',
            '[18:30] 发现是由于某个大事务导致同步阻塞，已优化SQL并强制同步。',
            '[18:45] 同步延迟恢复正常，事件状态更新为已解决。',
        ],
        comments: []
    },
    'INC-00121': {
        title: '用户请求重置密码失败',
        priority: '低', 
        status: '已关闭',
        source: '服务台',
        reporter: '孙八',
        assignee: '周九',
        createdAt: '2025-06-27 10:00:00',
        confirmedAt: '2025-06-27 10:05:00',
        resolvedAt: '2025-06-27 10:20:00',
        isMajorIncident: false,
        description: '用户孙八通过自助服务门户尝试重置密码失败，请求服务台协助。',
        affectedCI: { name: 'AD-SERVER-01', type: '域控制器', link: '/cmdb/ci/AD-SERVER-01', id: 'CI-AD-SERVER-01' },
        logs: [
            '[10:00] 事件创建并分配给IT支持团队。',
            '[10:05] 周九已确认接收事件。',
            '[10:15] 经排查，用户输入旧密码错误次数过多导致账户锁定，已解锁并指导用户正确重置。',
            '[10:20] 事件状态更新为已关闭。',
        ],
        comments: []
    },
};

// 模拟CMDB数据，用于更新CI状态
const mockCIData = {
    'CI-ECS-001': { status: '运行中' },
    'CI-RDS-001': { status: '运行中' },
    'CI-APP-CRM': { status: '运行中' },
    'CI-PROD-BASTION-HOST': { status: '运行中' },
    'CI-AD-SERVER-01': { status: '运行中' },
};

// 辅助函数：计算时间差（分钟）
const calculateTimeDiffInMinutes = (start, end) => {
    if (!start || !end) return 'N/A';
    const startDate = new Date(start);
    const endDate = new Date(end);
    const diff = Math.abs(endDate.getTime() - startDate.getTime());
    return `${Math.round(diff / (1000 * 60))} 分钟`;
};

const KnowledgeBaseSuggestion = ({ title, similarity }) => (
    <div className="p-3 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors flex justify-between items-center">
        <div>
            <p className="font-semibold text-gray-800">{title}</p>
            <span className="text-sm text-green-600">相似度: {similarity}</span>
        </div>
        <button className="text-sm text-blue-600 hover:underline">查看</button>
    </div>
);

const IncidentDetailPage = () => {
    const params = useParams();
    const router = useRouter();
    const incidentId = params.incidentId as string;
    const [incident, setIncident] = useState(mockIncidentDetail[incidentId]); // 使用useState来管理事件状态
    const [newComment, setNewComment] = useState('');

    const updateIncidentStatus = (newStatus, logMessage, updateTimes = {}) => {
        const updatedIncident = {
            ...incident,
            status: newStatus,
            logs: [...incident.logs, `[${new Date().toLocaleString()}] ${logMessage}`],
            ...updateTimes
        };
        setIncident(updatedIncident);
        alert(`事件状态已更新为: ${newStatus}`);
        // 在真实应用中，这里会调用API更新后端数据

        // 模拟CMDB CI状态联动
        if (newStatus === '已解决' && incident.affectedCI && mockCIData[incident.affectedCI.id]) {
            mockCIData[incident.affectedCI.id].status = '运行中'; // 假设解决事件后CI恢复正常
            alert(`关联CI ${incident.affectedCI.name} 的状态已更新为：运行中`);
            console.log(`CMDB CI ${incident.affectedCI.id} 状态更新为: 运行中`);
        }
    };

    const handleResolveIncident = () => {
        if (incident.status === '已解决' || incident.status === '已关闭') {
            alert('事件已解决或已关闭，无需重复操作。');
            return;
        }
        const resolutionNotes = prompt('请输入解决方案描述：');
        if (resolutionNotes) {
            updateIncidentStatus('已解决', `事件已解决。解决方案：${resolutionNotes}`, { resolvedAt: new Date().toLocaleString() });
        }
    };

    const handleCloseIncident = () => {
        if (incident.status === '已关闭') {
            alert('事件已关闭，无需重复操作。');
            return;
        }
        if (incident.status !== '已解决') {
            const confirmClose = confirm('事件尚未解决，确定要直接关闭吗？');
            if (!confirmClose) return;
        }
        updateIncidentStatus('已关闭', '事件已关闭。');
    };

    const handleSuspendIncident = () => {
        if (incident.status === '挂起') {
            alert('事件已处于挂起状态。');
            return;
        }
        const suspendReason = prompt('请输入挂起原因：');
        if (suspendReason) {
            updateIncidentStatus('挂起', `事件已挂起。原因：${suspendReason}`);
        }
    };

    const handleResumeIncident = () => {
        if (incident.status !== '挂起') {
            alert('事件未处于挂起状态。');
            return;
        }
        updateIncidentStatus('处理中', '事件已恢复处理。');
    };

    const handleMarkAsMajorIncident = () => {
        if (incident.isMajorIncident) {
            alert('此事件已是重大事件。');
            return;
        }
        const confirmMajor = confirm('确定要将此事件标记为重大事件吗？这将触发重大事件响应流程。');
        if (confirmMajor) {
            setIncident({ ...incident, isMajorIncident: true });
            alert('事件已标记为重大事件！重大事件响应团队已被通知。');
            // 实际应用中，这里会触发更复杂的MIM流程，如创建会议、通知高层等
        }
    };

    const handleAddComment = () => {
        if (newComment.trim()) {
            const updatedIncident = {
                ...incident,
                comments: [...incident.comments, { author: '当前用户', timestamp: new Date().toLocaleString(), text: newComment.trim() }]
            };
            setIncident(updatedIncident);
            setNewComment('');
            // 在真实应用中，这里会调用API保存评论
        }
    };

    if (!incident) {
        return <div className="p-10">事件不存在或加载失败。</div>;
    }

    const mtta = calculateTimeDiffInMinutes(incident.createdAt, incident.confirmedAt);
    const mttr = calculateTimeDiffInMinutes(incident.confirmedAt, incident.resolvedAt);

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <button onClick={() => router.back()} className="flex items-center text-blue-600 hover:underline mb-4">
                    <ArrowLeft className="w-5 h-5 mr-2" />
                    返回事件列表
                </button>
                <div className="flex justify-between items-start">
                    <div>
                        <h2 className="text-4xl font-bold text-gray-800 flex items-center">
                            {incident.title}
                            {incident.isMajorIncident && (
                                <span className="ml-4 px-3 py-1 text-sm font-bold rounded-full bg-red-600 text-white flex items-center animate-pulse">
                                    <Zap className="w-4 h-4 mr-2" /> 重大事件
                                </span>
                            )}
                        </h2>
                        <p className="text-gray-500 mt-1">事件ID: {incidentId}</p>
                    </div>
                    <div className="flex space-x-4">
                        {!incident.isMajorIncident && (
                            <button 
                                onClick={handleMarkAsMajorIncident}
                                className="flex items-center bg-red-500 text-white font-semibold py-2 px-4 rounded-lg hover:bg-red-600 transition-colors"
                            >
                                <Zap className="w-5 h-5 mr-2" />
                                标记为重大事件
                            </button>
                        )}
                        <Link 
                            href={`/problems/new?fromIncidentId=${incidentId}&incidentTitle=${encodeURIComponent(incident.title)}&incidentDescription=${encodeURIComponent(incident.description)}`}
                            passHref
                        >
                            <button 
                                className="flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors"
                            >
                                <PlusCircle className="w-5 h-5 mr-2" />
                                创建问题
                            </button>
                        </Link>
                        {incident.status === '挂起' ? (
                            <button 
                                onClick={handleResumeIncident}
                                className="flex items-center bg-purple-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-purple-700 transition-colors"
                            >
                                <PlayCircle className="w-5 h-5 mr-2" />
                                恢复处理
                            </button>
                        ) : (
                            <button 
                                onClick={handleSuspendIncident}
                                className="flex items-center bg-yellow-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-yellow-700 transition-colors"
                            >
                                <PauseCircle className="w-5 h-5 mr-2" />
                                挂起事件
                            </button>
                        )}
                        {incident.status !== '已解决' && incident.status !== '已关闭' && (
                            <button 
                                onClick={handleResolveIncident}
                                className="flex items-center bg-green-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-green-700 transition-colors"
                            >
                                <CheckCircle className="w-5 h-5 mr-2" />
                                解决事件
                            </button>
                        )}
                        {incident.status === '已解决' && incident.status !== '已关闭' && (
                            <button 
                                onClick={handleCloseIncident}
                                className="flex items-center bg-gray-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-gray-700 transition-colors"
                            >
                                <XCircle className="w-5 h-5 mr-2" />
                                关闭事件
                            </button>
                        )}
                    </div>
                </div>
            </header>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                {/* 左侧：事件详情和处理日志 */}
                <div className="lg:col-span-2 bg-white p-8 rounded-lg shadow-md">
                    <h3 className="text-xl font-semibold text-gray-700 mb-4">事件描述</h3>
                    <p className="text-gray-600 mb-8">{incident.description}</p>
                    
                    <h3 className="text-xl font-semibold text-gray-700 mb-4">处理日志</h3>
                    <div className="space-y-4">
                        {incident.logs.map((log, i) => (
                            <p key={i} className="text-sm text-gray-500 font-mono">{log}</p>
                        ))}
                    </div>

                    {/* 评论/备注区域 */}
                    <div className="mt-8 pt-8 border-t border-gray-200">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <MessageSquare className="w-5 h-5 mr-2" /> 内部评论/备注
                        </h3>
                        <div className="space-y-4 mb-4">
                            {incident.comments.length > 0 ? incident.comments.map((comment, i) => (
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
                        <h3 className="text-xl font-semibold text-gray-700 mb-4">事件信息</h3>
                        <div className="space-y-3 text-sm">
                            <div className="flex justify-between"><span>状态:</span><span className="font-semibold">{incident.status}</span></div>
                            <div className="flex justify-between"><span>优先级:</span><span className="font-semibold text-red-600">{incident.priority}</span></div>
                            <div className="flex justify-between"><span>负责人:</span><span className="font-semibold">{incident.assignee}</span></div>
                            <div className="flex justify-between"><span>来源:</span><span className="font-semibold">{incident.source}</span></div>
                            <div className="flex justify-between"><span>创建时间:</span><span className="font-semibold">{incident.createdAt}</span></div>
                            <div className="flex justify-between"><span>平均确认时间 (MTTA):</span><span className="font-semibold">{mtta}</span></div>
                            <div className="flex justify-between"><span>平均恢复时间 (MTTR):</span><span className="font-semibold">{mttr}</span></div>
                        </div>
                    </div>

                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <LinkIcon className="w-5 h-5 mr-2" /> 关联的配置项 (CI)
                        </h3>
                        <div className="flex items-center p-3 bg-blue-50 rounded-lg">
                            <Cpu className="w-6 h-6 text-blue-600 mr-3" />
                            <div>
                                <Link href={incident.affectedCI.link} className="font-semibold text-blue-700 hover:underline">
                                    {incident.affectedCI.name}
                                </Link>
                                <p className="text-sm text-gray-600">{incident.affectedCI.type}</p>
                            </div>
                        </div>
                    </div>

                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <BookOpen className="w-5 h-5 mr-2" /> 知识库助手
                        </h3>
                        <div className="space-y-3">
                            <KnowledgeBaseSuggestion title="如何处理Web服务器高CPU占用率" similarity="92%" />
                            <KnowledgeBaseSuggestion title="Linux性能瓶颈分析指南" similarity="85%" />
                        
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default IncidentDetailPage;