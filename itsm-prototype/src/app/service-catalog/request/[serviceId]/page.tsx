

'use client';

import React from 'react';
import { useParams, useRouter } from 'next/navigation';
import { HardDrive, UserCog, ShieldCheck, Clock, ArrowLeft, Info } from 'lucide-react';
import { ApplyVmForm } from '../forms/ApplyVmForm';
import { ApplyRdsForm } from '../forms/ApplyRdsForm'; // 导入RDS表单
import { ExpandOssForm } from '../forms/ExpandOssForm'; // 导入OSS表单

// --- 数据 ---
const allServiceItems = {
    'apply-vm': { title: '申请虚拟机 (VM)', description: '根据业务需求申请标准配置的云服务器', sla: '2个工作日', category: '云资源服务', icon: HardDrive, form: 'ApplyVmForm' },
    'apply-rds': { title: '申请云数据库 (RDS)', description: '申请MySQL, PostgreSQL等托管数据库服务', sla: '4小时', category: '云资源服务', icon: HardDrive, form: 'ApplyRdsForm' },
    'expand-oss': { title: '对象存储扩容', description: '为现有的OSS/S3存储桶增加容量', sla: '1小时', category: '云资源服务', icon: HardDrive, form: 'ExpandOssForm' },
    'reset-password': { title: '重置密码', description: '重置您的域账户或应用账户密码', sla: '15分钟', category: '账号与权限', icon: UserCog },
    'apply-permission': { title: '申请应用访问权限', description: '获取访问内部业务系统（如CRM, ERP）的权限', sla: '1工作日', category: '账号与权限', icon: UserCog },
    'create-account': { title: '创建新员工账户', description: '为新入职员工开通标准IT账户和资源', sla: '1工作日', category: '账号与权限', icon: UserCog },
    'report-security-incident': { title: '报告安全事件', description: '报告潜在的病毒、钓鱼邮件或数据泄露风险', sla: '立即响应', category: '安全服务', icon: ShieldCheck },
    'change-firewall-policy': { title: '申请防火墙策略变更', description: '为特定业务开放或限制网络端口访问', sla: '8小时', category: '安全服务', icon: ShieldCheck },
};

// 通用表单组件
const GenericForm = () => (
    <div className="space-y-6">
        <div>
            <label htmlFor="business-reason" className="block text-sm font-medium text-gray-700">业务理由</label>
            <textarea id="business-reason" name="business-reason" rows={4} className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" required placeholder="请详细说明您申请此服务的业务背景和目的..."></textarea>
        </div>
        <div>
            <label htmlFor="urgency" className="block text-sm font-medium text-gray-700">紧急程度</label>
            <select id="urgency" name="urgency" className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" defaultValue="常规">
                <option>高</option>
                <option>中</option>
                <option>低</option>
            </select>
        </div>
    </div>
);

const ServiceRequestPage = () => {
    const params = useParams();
    const router = useRouter();
    const serviceId = params.serviceId as string;
    const service = allServiceItems[serviceId];

    const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
        e.preventDefault();
        const formData = new FormData(e.currentTarget);
        const data = Object.fromEntries(formData.entries());
        console.log('提交的表单数据:', JSON.stringify(data, null, 2));
        alert(`服务请求 "${service.title}" 已提交！\n数据已在控制台打印，可直接用于后续的流程引擎和IaC。`);
        router.push('/service-catalog');
    };

    const renderForm = () => {
        switch (service?.form) {
            case 'ApplyVmForm':
                return <ApplyVmForm />;
            case 'ApplyRdsForm':
                return <ApplyRdsForm />;
            case 'ExpandOssForm':
                return <ExpandOssForm />;
            default:
                return <GenericForm />;
        }
    };

    if (!service) {
        return <div className="p-10">服务不存在或加载失败。</div>;
    }

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <button onClick={() => router.back()} className="flex items-center text-blue-600 hover:underline mb-4">
                    <ArrowLeft className="w-5 h-5 mr-2" />
                    返回服务目录
                </button>
                <h2 className="text-4xl font-bold text-gray-800">服务请求：{service.title}</h2>
                <p className="text-gray-500 mt-1">{service.description}</p>
            </header>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                <div className="lg:col-span-2 bg-white p-8 rounded-lg shadow-md">
                    <form onSubmit={handleSubmit}>
                        {renderForm()}
                        <div className="mt-8 flex justify-end">
                            <button type="button" onClick={() => router.back()} className="bg-gray-200 text-gray-700 font-semibold py-2 px-4 rounded-lg hover:bg-gray-300 mr-4">
                                取消
                            </button>
                            <button type="submit" className="bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700">
                                提交请求
                            </button>
                        </div>
                    </form>
                </div>

                <div className="lg:col-span-1">
                    <div className="bg-blue-50 border-l-4 border-blue-500 p-6 rounded-r-lg">
                        <div className="flex items-center">
                            <Info className="w-6 h-6 text-blue-700 mr-3" />
                            <h3 className="text-xl font-semibold text-blue-800">服务摘要</h3>
                        </div>
                        <div className="mt-4 space-y-3 text-gray-700">
                            <div className="flex justify-between">
                                <span className="font-semibold">服务类别:</span>
                                <span>{service.category}</span>
                            </div>
                            <div className="flex justify-between items-center">
                                <span className="font-semibold">预计交付时间:</span>
                                <span className="flex items-center font-bold text-blue-700">
                                    <Clock className="w-4 h-4 mr-1.5" />
                                    {service.sla}
                                </span>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default ServiceRequestPage;
