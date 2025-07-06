'use client';

import React, { useState } from 'react';
import Link from 'next/link'; // 导入Link组件
import { HardDrive, UserCog, ShieldCheck, Server, Search, Clock } from 'lucide-react';

// 更新：为每个服务项添加唯一的id
const serviceCatalogItems = [
    { 
        category: '云资源服务', 
        icon: HardDrive, 
        items: [
            { id: 'apply-vm', title: '申请虚拟机 (VM)', description: '根据业务需求申请标准配置的云服务器', sla: '2个工作日' },
            { id: 'apply-rds', title: '申请云数据库 (RDS)', description: '申请MySQL, PostgreSQL等托管数据库服务', sla: '4小时' },
            { id: 'expand-oss', title: '对象存储扩容', description: '为现有的OSS/S3存储桶增加容量', sla: '1小时' },
        ]
    },
    { 
        category: '账号与权限', 
        icon: UserCog, 
        items: [
            { id: 'reset-password', title: '重置密码', description: '重置您的域账户或应用账户密码', sla: '15分钟' },
            { id: 'apply-permission', title: '申请应用访问权限', description: '获取访问内部业务系统（如CRM, ERP）的权限', sla: '1工作日' },
            { id: 'create-account', title: '创建新员工账户', description: '为新入职员工开通标准IT账户和资源', sla: '1工作日' },
        ]
    },
    { 
        category: '安全服务', 
        icon: ShieldCheck, 
        items: [
            { id: 'report-security-incident', title: '报告安全事件', description: '报告潜在的病毒、钓鱼邮件或数据泄露风险', sla: '立即响应' },
            { id: 'change-firewall-policy', title: '申请防火墙策略变更', description: '为特定业务开放或限制网络端口访问', sla: '8小时' },
        ]
    },
];

// 更新：将按钮改为Link组件
const ServiceItemCard = ({ id, title, description, sla }) => (
    <div className="bg-white p-6 rounded-lg shadow-md hover:shadow-xl transition-all duration-300 flex flex-col">
        <h4 className="text-lg font-semibold text-gray-800">{title}</h4>
        <p className="text-gray-600 mt-2 flex-grow">{description}</p>
        <div className="flex items-center text-sm text-gray-500 mt-4 pt-4 border-t border-gray-100">
            <Clock className="w-4 h-4 mr-2" />
            <span>预计交付时间: {sla}</span>
        </div>
        <Link href={`/service-catalog/request/${id}`} passHref>
            <button className="mt-4 w-full bg-blue-600 text-white font-semibold py-2 rounded-lg hover:bg-blue-700 transition-colors">
                发起请求
            </button>
        </Link>
    </div>
);

const ServiceCatalogPage = () => {
    const [searchTerm, setSearchTerm] = useState('');

    const filteredItems = serviceCatalogItems.map(category => ({
        ...category,
        items: category.items.filter(item => 
            item.title.toLowerCase().includes(searchTerm.toLowerCase()) || 
            item.description.toLowerCase().includes(searchTerm.toLowerCase())
        )
    })).filter(category => category.items.length > 0);

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <h2 className="text-4xl font-bold text-gray-800">服务目录</h2>
                <p className="text-gray-500 mt-1">您的一站式IT服务请求中心</p>
            </header>

            <div className="mb-8">
                <div className="relative">
                    <Search className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
                    <input 
                        type="text"
                        placeholder="搜索服务，例如：虚拟机、重置密码..."
                        className="w-full pl-12 pr-4 py-3 border border-gray-300 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        onChange={(e) => setSearchTerm(e.target.value)}
                    />
                </div>
            </div>

            <div className="space-y-12">
                {filteredItems.length > 0 ? filteredItems.map(category => (
                    <section key={category.category}>
                        <div className="flex items-center mb-4">
                            <category.icon className="w-6 h-6 mr-3 text-blue-600" />
                            <h3 className="text-2xl font-semibold text-gray-700">{category.category}</h3>
                        </div>
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
                            {category.items.map(item => (
                                <ServiceItemCard key={item.id} {...item} />
                            ))}
                        </div>
                    </section>
                )) : (
                    <div className="text-center py-16">
                        <p className="text-gray-500 text-lg">未找到匹配的服务项。</p>
                    </div>
                )}
            </div>
        </div>
    );
};

export default ServiceCatalogPage;