import React from 'react';
import ServiceRequestList from '@/modules/service-request/components/ServiceRequestList';
import { Metadata } from 'next';

export const metadata: Metadata = {
    title: 'Service Requests | ITSM',
    description: 'Manage your service requests and approvals',
};

export default function ServiceRequestsPage() {
    return (
        <div className="p-6">
            <div className="mb-6">
                <h1 className="text-2xl font-bold text-gray-900">服务请求</h1>
                <p className="text-gray-500">查看我的请求状态及处理待办审批</p>
            </div>
            <ServiceRequestList />
        </div>
    );
}
