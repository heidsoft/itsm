
// 模拟事件数据 (从 incidents/[incidentId]/page.tsx 复制并转换为数组)
export const mockIncidentsData = [
    { id: 'INC-00125', title: '杭州可用区J的Web服务器CPU使用率超过95%', priority: '高', status: '处理中', source: '阿里云监控', sourceIcon: Cpu, lastUpdate: '5分钟前', isMajorIncident: false, assignee: '张三', createdAt: '2025-06-28 10:15:23' },
    { id: 'INC-00124', title: '用户报告无法访问CRM系统', priority: '高', status: '已分配', source: '服务台', sourceIcon: User, lastUpdate: '25分钟前', isMajorIncident: false, assignee: '王五', createdAt: '2025-06-28 09:45:10' },
    { id: 'INC-00123', title: '检测到可疑的SSH登录尝试 (47.98.x.x)', priority: '中', status: '处理中', source: '安全中心', sourceIcon: Shield, lastUpdate: '1小时前', isMajorIncident: true, assignee: '赵六', createdAt: '2025-06-28 08:30:00' },
    { id: 'INC-00122', title: '生产数据库主备同步延迟', priority: '中', status: '已解决', source: '阿里云监控', sourceIcon: Cpu, lastUpdate: '3小时前', isMajorIncident: false, assignee: '钱七', createdAt: '2025-06-27 18:00:00' },
    { id: 'INC-00121', title: '用户请求重置密码失败', priority: '低', status: '已关闭', source: '服务台', sourceIcon: User, lastUpdate: '1天前', isMajorIncident: false, assignee: '周九', createdAt: '2025-06-27 10:00:00' },
];

// 模拟问题数据 (从 problems/[problemId]/page.tsx 复制并转换为数组)
export const mockProblemsData = [
    { id: 'PRB-00001', title: 'CRM系统间歇性登录失败', status: '调查中', priority: '高', assignee: '王五', createdAt: '2025-06-20' },
    { id: 'PRB-00002', title: '部分用户无法访问VPN', status: '已解决', priority: '中', assignee: '李明', createdAt: '2025-06-15' },
    { id: 'PRB-00003', title: '数据库连接池耗尽导致应用崩溃', status: '已知错误', priority: '高', assignee: '钱七', createdAt: '2025-06-10' },
    { id: 'PRB-00004', title: '邮件服务发送延迟', status: '已关闭', priority: '低', assignee: '赵六', createdAt: '2025-06-05' },
];

// 模拟变更数据 (从 changes/[changeId]/page.tsx 复制并转换为数组)
export const mockChangesData = [
    { id: 'CHG-00001', title: 'CRM系统数据库升级', type: '普通变更', status: '待审批', priority: '高', assignee: '李四', createdAt: '2025-06-25' },
    { id: 'CHG-00002', title: '新增VPN网关防火墙规则', type: '标准变更', status: '已批准', priority: '中', assignee: '张三', createdAt: '2025-06-20' },
    { id: 'CHG-00003', title: '紧急修复Web服务器安全漏洞', type: '紧急变更', status: '实施中', priority: '紧急', assignee: '王五', createdAt: '2025-06-18' },
    { id: 'CHG-00004', title: '邮件服务迁移至新平台', type: '普通变更', status: '已完成', priority: '中', assignee: '赵六', createdAt: '2025-06-10' },
];

// 模拟用户请求数据 (从 my-requests/page.tsx 复制)
export const mockRequestsData = [
    { id: 'REQ-00101', serviceName: '申请虚拟机 (VM)', status: '处理中', submittedAt: '2025-06-28 14:30', detailsLink: '/service-catalog/request/apply-vm' },
    { id: 'REQ-00100', serviceName: '重置密码', status: '已完成', submittedAt: '2025-06-27 10:15', detailsLink: '/service-catalog/request/reset-password' },
    { id: 'REQ-00099', serviceName: '申请云数据库 (RDS)', status: '待审批', submittedAt: '2025-06-26 16:00', detailsLink: '/service-catalog/request/apply-rds' },
    { id: 'REQ-00098', serviceName: '对象存储扩容', status: '已拒绝', submittedAt: '2025-06-25 09:00', detailsLink: '/service-catalog/request/expand-oss' },
    { id: 'REQ-00097', serviceName: '申请应用访问权限', status: '已完成', submittedAt: '2025-06-24 11:45', detailsLink: '/service-catalog/request/apply-permission' },
];
