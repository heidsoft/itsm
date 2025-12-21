
'use client';

import React from 'react';

// 模拟数据库产品选项
const dbEngines = {
    'mysql_8_0': 'MySQL 8.0',
    'postgres_14': 'PostgreSQL 14',
    'sqlserver_2019_se': 'SQL Server 2019 SE',
};

const dbInstanceClasses = {
    'rds.mysql.s1.small': '通用型: 2核 4GB',
    'rds.mysql.s2.medium': '通用型: 4核 8GB',
    'rds.pg.m1.medium': '内存优化型: 4核 16GB',
};

export const ApplyRdsForm = () => {
    return (
        <div className="space-y-6">
            {/* 数据库引擎 */}
            <div>
                <label htmlFor="db-engine" className="block text-sm font-medium text-gray-700">数据库引擎</label>
                <select id="db-engine" name="db_engine" className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500">
                    {Object.entries(dbEngines).map(([id, name]) => (
                        <option key={id} value={id}>{name}</option>
                    ))}
                </select>
            </div>

            {/* 实例规格 */}
            <div>
                <label htmlFor="db-instance-class" className="block text-sm font-medium text-gray-700">实例规格</label>
                <select id="db-instance-class" name="db_instance_class" className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500">
                    {Object.entries(dbInstanceClasses).map(([id, name]) => (
                        <option key={id} value={id}>{name}</option>
                    ))}
                </select>
            </div>

            {/* 存储空间 */}
            <div>
                <label htmlFor="storage-size" className="block text-sm font-medium text-gray-700">存储空间 (GB)</label>
                <input type="number" id="storage-size" name="storage_size" defaultValue={50} min={20} max={1000} className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" />
            </div>

            {/* 数据库名称 */}
            <div>
                <label htmlFor="db-name" className="block text-sm font-medium text-gray-700">初始数据库名称</label>
                <input type="text" id="db-name" name="db_name" required className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" placeholder="例如：my_app_db" />
            </div>

            {/* 管理员账号 */}
            <div>
                <label htmlFor="master-username" className="block text-sm font-medium text-gray-700">管理员账号</label>
                <input type="text" id="master-username" name="master_username" required className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" placeholder="例如：db_admin" />
            </div>

            {/* 业务理由 */}
            <div>
                <label htmlFor="business-reason" className="block text-sm font-medium text-gray-700">业务理由</label>
                <textarea id="business-reason" name="business_reason" rows={4} className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" required placeholder="请说明此数据库的用途..."></textarea>
            </div>
        </div>
    );
};
