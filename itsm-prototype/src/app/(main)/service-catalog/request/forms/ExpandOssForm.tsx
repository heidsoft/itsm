
'use client';

import React from 'react';

// 模拟用户拥有的存储桶列表 (真实场景应从API获取)
const userBuckets = [
    'app-prod-data-bucket',
    'user-uploads-backup',
    'logs-archive-2025',
];

export const ExpandOssForm = () => {
    return (
        <div className="space-y-6">
            {/* 选择存储桶 */}
            <div>
                <label htmlFor="bucket-name" className="block text-sm font-medium text-gray-700">选择要扩容的存储桶</label>
                <select id="bucket-name" name="bucket_name" className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500">
                    {userBuckets.map(bucket => (
                        <option key={bucket} value={bucket}>{bucket}</option>
                    ))}
                </select>
            </div>

            {/* 扩容容量 */}
            <div>
                <label htmlFor="additional-capacity" className="block text-sm font-medium text-gray-700">需要增加的容量 (GB)</label>
                <input 
                    type="number" 
                    id="additional-capacity" 
                    name="additional_capacity_gb" 
                    defaultValue={100} 
                    min={10} 
                    max={5000} 
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" 
                    required 
                />
            </div>

            {/* 业务理由 */}
            <div>
                <label htmlFor="business-reason" className="block text-sm font-medium text-gray-700">扩容理由</label>
                <textarea 
                    id="business-reason" 
                    name="business_reason" 
                    rows={4} 
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500" 
                    required 
                    placeholder="请说明扩容的原因，例如：业务数据量增长预期..."
                ></textarea>
            </div>
        </div>
    );
};
