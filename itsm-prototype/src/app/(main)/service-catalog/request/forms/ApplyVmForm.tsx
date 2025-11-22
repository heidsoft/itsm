
'use client';

import React, { useState, useEffect } from 'react';

// 模拟地域和可用区数据 (真实场景应从API获取)
const regions = {
    'cn-hangzhou': { name: '华东1 (杭州)', zones: ['cn-hangzhou-h', 'cn-hangzhou-i', 'cn-hangzhou-j'] },
    'cn-beijing': { name: '华北2 (北京)', zones: ['cn-beijing-a', 'cn-beijing-b', 'cn-beijing-c'] },
    'us-west-1': { name: '美国西部1 (硅谷)', zones: ['us-west-1a', 'us-west-1b'] },
};

// 模拟实例规格
const instanceTypes = {
    'ecs.g6.large': '通用型 g6, 2 vCPU, 8 GiB',
    'ecs.c6.large': '计算型 c6, 2 vCPU, 4 GiB',
    'ecs.r6.large': '内存型 r6, 2 vCPU, 16 GiB',
};

// 模拟镜像
const images = {
    'centos_7_9': 'CentOS 7.9 64位',
    'ubuntu_22_04': 'Ubuntu 22.04 LTS 64位',
    'windows_2019': 'Windows Server 2019 数据中心版 64位',
};

export const ApplyVmForm = ({ onSubmit }) => {
    const [selectedRegion, setSelectedRegion] = useState('cn-hangzhou');
    const [availableZones, setAvailableZones] = useState([]);

    useEffect(() => {
        setAvailableZones(regions[selectedRegion]?.zones || []);
    }, [selectedRegion]);

    return (
        <div className="space-y-6">
            {/* 地域选择 */}
            <div>
                <label htmlFor="region" className="block text-sm font-medium text-gray-700">地域</label>
                <select 
                    id="region" 
                    name="region"
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                    onChange={(e) => setSelectedRegion(e.target.value)}
                    value={selectedRegion}
                >
                    {Object.entries(regions).map(([id, { name }]) => (
                        <option key={id} value={id}>{name}</option>
                    ))}
                </select>
            </div>

            {/* 可用区选择 */}
            <div>
                <label htmlFor="zone" className="block text-sm font-medium text-gray-700">可用区</label>
                <select id="zone" name="zone" className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500">
                    {availableZones.map(zone => (
                        <option key={zone} value={zone}>{zone}</option>
                    ))}
                </select>
            </div>

            {/* 实例规格 */}
            <div>
                <label htmlFor="instance-type" className="block text-sm font-medium text-gray-700">实例规格</label>
                <select id="instance-type" name="instance-type" className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500">
                     {Object.entries(instanceTypes).map(([id, name]) => (
                        <option key={id} value={id}>{name}</option>
                    ))}
                </select>
            </div>

            {/* 操作系统镜像 */}
            <div>
                <label htmlFor="image" className="block text-sm font-medium text-gray-700">操作系统镜像</label>
                <select id="image" name="image" className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500">
                     {Object.entries(images).map(([id, name]) => (
                        <option key={id} value={id}>{name}</option>
                    ))}
                </select>
            </div>

             {/* 业务理由 */}
            <div>
                <label htmlFor="business-reason" className="block text-sm font-medium text-gray-700">业务理由</label>
                <textarea id="business-reason" name="business-reason" rows={4} className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" required placeholder="请详细说明您申请此服务的业务背景和目的..."></textarea>
            </div>
        </div>
    );
};
