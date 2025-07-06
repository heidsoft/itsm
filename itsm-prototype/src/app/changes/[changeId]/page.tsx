'use client';

import React, { useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import { ArrowLeft, AlertTriangle, Search, CheckCircle, XCircle, GitMerge, PlayCircle, Flag, ClipboardCheck, MessageSquare } from 'lucide-react';

// 模拟变更详情数据
const mockChangeDetail = {
    'CHG-00001': {
        title: 'CRM系统数据库升级',
        description: '将生产环境CRM系统的MySQL数据库从5.7升级到8.0，以提升性能和安全性。预计停机时间2小时。',
        justification: '解决PRB-00001（CRM系统间歇性登录失败）的根本原因，提升系统稳定性。',
        type: '普通变更',
        status: '待审批',
        priority: '高',
        assignee: '李四',
        createdAt: '2025-06-25 10:00:00',
        plannedStartDate: '2025-07-10 02:00:00',
        plannedEndDate: '2025-07-10 04:00:00',
        impactScope: '高 (影响核心业务)',
        riskLevel: '中',
        implementationPlan: '1. 备份数据库；2. 升级MySQL版本；3. 数据迁移和验证；4. 应用连接测试。',
        rollbackPlan: '如果升级失败，回滚到备份的5.7数据库。',
        approvals: [
            { role: '业务负责人', status: '待审批', approver: null, comment: null },
            { role: '技术负责人', status: '待审批', approver: null, comment: null },
        ],
        relatedTickets: [
            { id: 'PRB-00001', type: '问题', title: 'CRM系统间歇性登录失败', status: '调查中' },
        ],
        affectedCIs: [
            { id: 'CI-RDS-001', name: 'CRM数据库-生产环境', type: '云数据库' },
            { id: 'CI-APP-CRM', name: 'CRM应用系统', type: '应用系统' },
        ],
        logs: [
            '[2025-06-25 10:05] 变更请求创建并提交审批。',
        ],
        comments: [
            { author: '李四', timestamp: '2025-06-25 10:10', text: '已提交变更请求，等待业务和技术负责人审批。' },
        ]
    },
    'CHG-00002': {
        title: '新增VPN网关防火墙规则',
        description: '为新上线的远程办公应用，在VPN网关防火墙上开放特定端口和IP范围。',
        justification: '支持新远程办公应用上线，确保用户正常访问。',
        type: '标准变更',
        status: '已批准',
        priority: '中',
        assignee: '张三',
        createdAt: '2025-06-20 14:00:00',
        plannedStartDate: '2025-06-21 09:00:00',
        plannedEndDate: '2025-06-21 09:30:00',
        impactScope: '低',
        riskLevel: '低',
        implementationPlan: '1. 登录防火墙；2. 添加新规则；3. 测试连通性。',
        rollbackPlan: '删除新增规则。',
        approvals: [
            { role: '网络负责人', status: '已批准', approver: '王小明', comment: '规则已审核，符合安全策略。' },
        ],
        relatedTickets: [
            { id: 'INC-00110', type: '事件', title: 'VPN连接失败告警', status: '已解决' },
        ],
        affectedCIs: [
            { id: 'CI-VPN-GW-001', name: 'VPN网关', type: '网络设备' },
        ],
        logs: [
            '[2025-06-20 14:05] 变更请求创建并提交审批。',
            '[2025-06-20 15:00] 网络负责人批准。',
            '[2025-06-21 09:00] 变更开始实施。',
            '[2025-06-21 09:20] 变更实施完成并验证。',
        ],
        comments: []
    },
    'CHG-00003': {
        title: '紧急修复Web服务器安全漏洞',
        description: '针对发现的Apache Log4j漏洞进行紧急补丁修复，以防止潜在的安全攻击。',
        justification: '响应紧急安全漏洞，避免潜在的数据泄露和系统入侵。',
        type: '紧急变更',
        status: '实施中',
        priority: '紧急',
        assignee: '王五',
        createdAt: '2025-06-18 20:00:00',
        plannedStartDate: '2025-06-18 20:30:00',
        plannedEndDate: '2025-06-18 21:00:00',
        impactScope: '高',
        riskLevel: '高',
        implementationPlan: '1. 下载补丁；2. 应用补丁；3. 重启服务；4. 验证。',
        rollbackPlan: '卸载补丁，恢复旧版本。',
        approvals: [
            { role: '安全负责人', status: '已批准', approver: '陈大明', comment: '紧急变更，已批准。' },
        ],
        relatedTickets: [
            { id: 'INC-00123', type: '事件', title: '检测到可疑的SSH登录尝试', status: '处理中' },
        ],
        affectedCIs: [
            { id: 'CI-ECS-001', name: 'Web服务器-生产环境', type: '云服务器' },
        ],
        logs: [
            '[2025-06-18 20:05] 紧急变更请求创建并自动批准。',
            '[2025-06-18 20:30] 变更开始实施。',
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

const ChangeDetailPage = () => {
    const params = useParams();
    const router = useRouter();
    const changeId = params.changeId as string;
    const [change, setChange] = useState(mockChangeDetail[changeId]); // 使用useState来管理变更状态
    const [newComment, setNewComment] = useState('');

    const updateChangeStatus = (newStatus, logMessage) => {
        const updatedChange = {
            ...change,
            status: newStatus,
            logs: [...change.logs, `[${new Date().toLocaleString()}] ${logMessage}`]
        };
        setChange(updatedChange);
        alert(`变更状态已更新为: ${newStatus}`);
        // 在真实应用中，这里会调用API更新后端数据
    };

    const handleApprove = (index) => {
        const updatedApprovals = [...change.approvals];
        updatedApprovals[index] = { ...updatedApprovals[index], status: '已批准', approver: '当前用户', comment: '已批准' };
        setChange({ ...change, approvals: updatedApprovals });
        alert(`审批人 ${updatedApprovals[index].role} 已批准变更 ${changeId}！`);
        // 检查是否所有审批都已完成，如果是，则将变更状态更新为“已批准”
        if (updatedApprovals.every(app => app.status === '已批准')) {
            updateChangeStatus('已批准', '所有审批已完成，变更已批准。');
        }
    };

    const handleReject = (index) => {
        const comment = prompt('请输入拒绝理由：');
        if (comment) {
            const updatedApprovals = [...change.approvals];
            updatedApprovals[index] = { ...updatedApprovals[index], status: '已拒绝', approver: '当前用户', comment: comment };
            setChange({ ...change, approvals: updatedApprovals });
            alert(`审批人 ${updatedApprovals[index].role} 已拒绝变更 ${changeId}！`);
            updateChangeStatus('已拒绝', `变更被拒绝。理由：${comment}`);
            // 实际应用中会更新后端状态
        }
    };

    const handleStartImplementation = () => {
        if (change.status !== '已批准') {
            alert('变更未处于已批准状态，无法开始实施。');
            return;
        }
        updateChangeStatus('实施中', '变更实施已开始。');
    };

    const handleCompleteImplementation = () => {
        if (change.status !== '实施中') {
            alert('变更未处于实施中状态，无法完成实施。');
            return;
        }
        const implementationNotes = prompt('请输入实施结果描述：');
        if (implementationNotes) {
            updateChangeStatus('已完成', `变更实施已完成。结果：${implementationNotes}`);
        }
    };

    const handleCompleteReview = () => {
        if (change.status !== '已完成') {
            alert('变更未处于已完成状态，无法进行评审。');
            return;
        }
        const reviewNotes = prompt('请输入评审结果描述：');
        if (reviewNotes) {
            updateChangeStatus('已评审', `变更评审已完成。结果：${reviewNotes}`);
        }
    };

    const handleAddComment = () => {
        if (newComment.trim()) {
            const updatedChange = {
                ...change,
                comments: [...change.comments, { author: '当前用户', timestamp: new Date().toLocaleString(), text: newComment.trim() }]
            };
            setChange(updatedChange);
            setNewComment('');
            // 在真实应用中，这里会调用API保存评论
        }
    };

    if (!change) {
        return <div className="p-10">变更不存在或加载失败。</div>;
    }

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <button onClick={() => router.back()} className="flex items-center text-blue-600 hover:underline mb-4">
                    <ArrowLeft className="w-5 h-5 mr-2" />
                    返回变更列表
                </button>
                <div className="flex justify-between items-start">
                    <div>
                        <h2 className="text-4xl font-bold text-gray-800">变更：{change.title}</h2>
                        <p className="text-gray-500 mt-1">变更ID: {changeId}</p>
                    </div>
                    <div className="flex space-x-4">
                        {change.status === '已批准' && (
                            <button 
                                onClick={handleStartImplementation}
                                className="flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors"
                            >
                                <PlayCircle className="w-5 h-5 mr-2" />
                                开始实施
                            </button>
                        )}
                        {change.status === '实施中' && (
                            <button 
                                onClick={handleCompleteImplementation}
                                className="flex items-center bg-green-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-green-700 transition-colors"
                            >
                                <Flag className="w-5 h-5 mr-2" />
                                完成实施
                            </button>
                        )}
                        {change.status === '已完成' && (
                            <button 
                                onClick={handleCompleteReview}
                                className="flex items-center bg-purple-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-purple-700 transition-colors"
                            >
                                <ClipboardCheck className="w-5 h-5 mr-2" />
                                完成评审
                            </button>
                        )}
                    </div>
                </div>
            </header>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                {/* 左侧：变更详情 */}
                <div className="lg:col-span-2 bg-white p-8 rounded-lg shadow-md">
                    <h3 className="text-xl font-semibold text-gray-700 mb-4">变更描述</h3>
                    <p className="text-gray-600 mb-8">{change.description}</p>
                    
                    <h3 className="text-xl font-semibold text-gray-700 mb-4">变更理由</h3>
                    <p className="text-gray-600 mb-8">{change.justification}</p>

                    <h3 className="text-xl font-semibold text-gray-700 mb-4">变更类型</h3>
                    <p className="text-gray-600 mb-8"><span className="font-semibold">{change.type}</span></p>

                    <h3 className="text-xl font-semibold text-gray-700 mb-4">影响与风险</h3>
                    <p className="text-gray-600 mb-2"><span className="font-semibold">影响范围:</span> {change.impactScope}</p>
                    <p className="text-gray-600 mb-8"><span className="font-semibold">风险等级:</span> {change.riskLevel}</p>

                    <h3 className="text-xl font-semibold text-gray-700 mb-4">实施计划</h3>
                    <p className="text-gray-600 mb-8 whitespace-pre-wrap">{change.implementationPlan}</p>

                    <h3 className="text-xl font-semibold text-gray-700 mb-4">回滚计划</h3>
                    <p className="text-gray-600 mb-8 whitespace-pre-wrap">{change.rollbackPlan}</p>

                    <h3 className="text-xl font-semibold text-gray-700 mb-4">变更日志</h3>
                    <div className="space-y-4">
                        {change.logs.map((log, i) => (
                            <p key={i} className="text-sm text-gray-500 font-mono">{log}</p>
                        ))}
                    </div>

                    {/* 评论/备注区域 */}
                    <div className="mt-8 pt-8 border-t border-gray-200">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <MessageSquare className="w-5 h-5 mr-2" /> 内部评论/备注
                        </h3>
                        <div className="space-y-4 mb-4">
                            {change.comments.length > 0 ? change.comments.map((comment, i) => (
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
                        <h3 className="text-xl font-semibold text-gray-700 mb-4">变更信息</h3>
                        <div className="space-y-3 text-sm">
                            <div className="flex justify-between"><span>状态:</span><span className="font-semibold">{change.status}</span></div>
                            <div className="flex justify-between"><span>优先级:</span><span className="font-semibold">{change.priority}</span></div>
                            <div className="flex justify-between"><span>负责人:</span><span className="font-semibold">{change.assignee}</span></div>
                            <div className="flex justify-between"><span>创建时间:</span><span className="font-semibold">{change.createdAt}</span></div>
                            <div className="flex justify-between"><span>计划开始:</span><span className="font-semibold">{change.plannedStartDate}</span></div>
                            <div className="flex justify-between"><span>计划结束:</span><span className="font-semibold">{change.plannedEndDate}</span></div>
                        </div>
                    </div>

                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4">审批流程</h3>
                        <div className="space-y-3">
                            {change.approvals.map((approval, i) => (
                                <div key={i} className="p-3 bg-gray-50 rounded-lg">
                                    <div className="flex justify-between items-center mb-1">
                                        <span className="font-semibold text-gray-700">{approval.role}</span>
                                        <ApprovalStatusBadge status={approval.status} />
                                    </div>
                                    {approval.approver && <p className="text-sm text-gray-600">审批人: {approval.approver}</p>}
                                    {approval.comment && <p className="text-sm text-gray-500 italic">"{approval.comment}"</p>}
                                    {approval.status === '待审批' && (
                                        <div className="mt-2 flex space-x-2">
                                            <button 
                                                onClick={() => handleApprove(i)}
                                                className="flex-1 bg-green-500 text-white text-xs py-1 rounded hover:bg-green-600"
                                            >批准</button>
                                            <button 
                                                onClick={() => handleReject(i)}
                                                className="flex-1 bg-red-500 text-white text-xs py-1 rounded hover:bg-red-600"
                                            >拒绝</button>
                                        </div>
                                    )}
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <GitMerge className="w-5 h-5 mr-2" /> 关联工单
                        </h3>
                        <div className="space-y-3">
                            {change.relatedTickets.length > 0 ? change.relatedTickets.map(ticket => (
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

                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <AlertTriangle className="w-5 h-5 mr-2" /> 受影响的配置项
                        </h3>
                        <div className="space-y-3">
                            {change.affectedCIs.length > 0 ? change.affectedCIs.map(ci => (
                                <Link 
                                    key={ci.id} 
                                    href={`/cmdb/${ci.id}`} 
                                    className="block p-3 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors"
                                >
                                    <p className="font-semibold text-blue-700">{ci.name} ({ci.type})</p>
                                    <span className="text-xs text-gray-500">CI ID: {ci.id}</span>
                                </Link>
                            )) : (
                                <p className="text-sm text-gray-500">无受影响的配置项。</p>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default ChangeDetailPage;