
'use client';

import {Zap, ArrowLeft, Network, Server, Database, Cloud } from 'lucide-react';
import React from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';

// Mock CI detail data
const mockCIDetail = {
    'CI-ECS-001': {
        name: 'Web Server - Production Environment',
        type: 'Cloud Server',
        status: 'Running',
        business: 'E-commerce Platform',
        owner: 'Operations Team',
        description: 'Alibaba Cloud ECS instance hosting the e-commerce platform frontend web services, located in East China 1 Availability Zone J.',
        attributes: {
            'Instance ID': 'i-bp1abcdefg',
            'IP Address': '192.168.1.100 (Private), 47.98.x.x (Public)',
            'Operating System': 'CentOS 7.9',
            'CPU': '8 Cores',
            'Memory': '16GB',
            'Disk': '200GB SSD',
            'Creation Date': '2024-01-01',
        },
        relatedTickets: [
            { id: 'INC-00125', type: 'Incident', title: 'Web server CPU usage exceeds 95%', status: 'In Progress' },
            { id: 'PRB-00003', type: 'Problem', title: 'Database connection pool exhaustion causing application crash', status: 'Known Error' },
            { id: 'CHG-00003', type: 'Change', title: 'Emergency fix for web server security vulnerability', status: 'Implementing' },
        ],
        relationships: [
            { targetId: 'CI-APP-CRM', targetName: 'CRM Application System', type: 'Hosts' },
            { targetId: 'CI-RDS-001', targetName: 'CRM Database - Production Environment', type: 'Connects to' },
        ],
        impacts: [
            { id: 'CI-APP-CRM', name: 'CRM Application System', type: 'Application System', impactLevel: 'High' },
            { id: 'SLA-001', name: 'Production CRM System Availability', type: 'SLA', impactLevel: 'High' },
        ],
        icon: Server,
    },
    'CI-RDS-001': {
        name: 'CRM Database - Production Environment',
        type: 'Cloud Database',
        status: 'Running',
        business: 'Sales Management',
        owner: 'DBA Team',
        description: 'MySQL database instance used by the e-commerce platform CRM system, configured for high availability.',
        attributes: {
            'Instance ID': 'rm-bp1abcdefg',
            'Database Type': 'MySQL 8.0',
            'Instance Specification': '4 Cores 16GB',
            'Storage Space': '500GB SSD',
            'Creation Date': '2023-10-01',
        },
        relatedTickets: [
            { id: 'INC-00122', type: 'Incident', title: 'Production database master-slave synchronization delay', status: 'Resolved' },
        ],
        relationships: [
            { targetId: 'CI-ECS-001', targetName: 'Web Server - Production Environment', type: 'Connected by' },
        ],
        impacts: [
            { id: 'CI-APP-CRM', name: 'CRM Application System', type: 'Application System', impactLevel: 'High' },
            { id: 'CI-ECS-001', name: 'Web Server - Production Environment', type: 'Cloud Server', impactLevel: 'Medium' },
            { id: 'SLA-001', name: 'Production CRM System Availability', type: 'SLA', impactLevel: 'High' },
        ],
        icon: Database,
    },
    'CI-APP-CRM': {
        name: 'CRM Application System',
        type: 'Application System',
        status: 'Running',
        business: 'Sales Management',
        owner: 'Development Team',
        description: 'Customer relationship management system used internally by the company.',
        attributes: {
            'Version': 'V3.2.1',
            'Development Language': 'Java',
            'Deployment Environment': 'Production',
        },
        relatedTickets: [
            { id: 'INC-00124', type: 'Incident', title: 'Users report unable to access CRM system', status: 'Assigned' },
            { id: 'PRB-00001', type: 'Problem', title: 'CRM system intermittent login failures', status: 'Under Investigation' },
        ],
        relationships: [
            { targetId: 'CI-ECS-001', targetName: 'Web Server - Production Environment', type: 'Deployed on' },
            { targetId: 'CI-RDS-001', targetName: 'CRM Database - Production Environment', type: 'Uses' },
        ],
        impacts: [
            { id: 'SLA-001', name: 'Production CRM System Availability', type: 'SLA', impactLevel: 'High' },
            { id: 'REQ-00101', name: 'User unable to login to CRM', type: 'Service Request', impactLevel: 'High' },
        ],
        icon: Cloud,
    },
};

const CIDetailPage = () => {
    const params = useParams();
    const router = useRouter();
    const ciId = params.ciId as string;
    const ci = mockCIDetail[ciId as keyof typeof mockCIDetail];

    if (!ci) {
        return <div className="p-10">Configuration item not found or failed to load.</div>;
    }

    const Icon = ci.icon;

    return (
        <div className="p-10 bg-gray-50 min-h-full">
            <header className="mb-8">
                <button onClick={() => router.back()} className="flex items-center text-blue-600 hover:underline mb-4">
                    <ArrowLeft className="w-5 h-5 mr-2" />
                    Back to CMDB List
                </button>
                <h2 className="text-4xl font-bold text-gray-800">Configuration Item Details: {ci.name}</h2>
                <p className="text-gray-500 mt-1">CI ID: {ciId}</p>
            </header>

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                {/* Left: CI attributes and description */}
                <div className="lg:col-span-2 bg-white p-8 rounded-lg shadow-md">
                    <div className="flex items-center mb-4">
                        <Icon className="w-8 h-8 mr-3 text-blue-600" />
                        <h3 className="text-2xl font-semibold text-gray-700">Basic Information</h3>
                    </div>
                    <p className="text-gray-600 mb-8">{ci.description}</p>

                    <h3 className="text-xl font-semibold text-gray-700 mb-4">Attributes</h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
                        {Object.entries(ci.attributes).map(([key, value]) => (
                            <div key={key} className="flex justify-between items-center border-b border-gray-100 py-2">
                                <span className="font-medium text-gray-600">{key}:</span>
                                <span className="text-gray-800">{value}</span>
                            </div>
                        ))}
                    </div>

                    <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                        <Network className="w-5 h-5 mr-2" /> Relationship Diagram (Mock)
                    </h3>
                    <div className="bg-gray-50 p-6 rounded-lg border border-gray-200">
                        <p className="text-gray-600">This area will display the relationship diagram between configuration items, for example:</p>
                        <ul className="list-disc list-inside text-gray-700 mt-2">
                            {ci.relationships.map((rel, i) => (
                                <li key={i}>
                                    {ci.name} <span className="font-semibold text-blue-600">{rel.type}</span> <Link href={`/cmdb/${rel.targetId}`} className="text-blue-600 hover:underline">{rel.targetName}</Link>
                                </li>
                            ))}
                        </ul>
                        <p className="text-sm text-gray-500 mt-4">(In the actual product, this would be an interactive relationship topology diagram)</p>
                    </div>
                </div>

                {/* Right: metadata and related tickets */}
                <div className="space-y-8">
                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4">Basic Information</h3>
                        <div className="space-y-3 text-sm">
                            <div className="flex justify-between"><span>Type:</span><span className="font-semibold">{ci.type}</span></div>
                            <div className="flex justify-between"><span>Status:</span><span className="font-semibold">{ci.status}</span></div>
                            <div className="flex justify-between"><span>Business:</span><span className="font-semibold">{ci.business}</span></div>
                            <div className="flex justify-between"><span>Owner:</span><span className="font-semibold">{ci.owner}</span></div>
                        </div>
                    </div>

                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <AlertTriangle className="w-5 h-5 mr-2" /> Related Tickets
                        </h3>
                        <div className="space-y-3">
                            {ci.relatedTickets.length > 0 ? ci.relatedTickets.map(ticket => (
                                <Link 
                                    key={ticket.id} 
                                    href={`/${ticket.type === 'Incident' ? 'incidents' : ticket.type === 'Problem' ? 'problems' : 'changes'}/${ticket.id}`} 
                                    className="block p-3 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors"
                                >
                                    <p className="font-semibold text-blue-700">{ticket.id} ({ticket.type})</p>
                                    <p className="text-sm text-gray-600">{ticket.title}</p>
                                    <span className="text-xs text-gray-500">Status: {ticket.status}</span>
                                </Link>
                            )) : (
                                <p className="text-sm text-gray-500">No related tickets.</p>
                            )}
                        </div>
                    </div>

                    {/* Impact Analysis */}
                    <div className="bg-white p-6 rounded-lg shadow-md">
                        <h3 className="text-xl font-semibold text-gray-700 mb-4 flex items-center">
                            <Zap className="w-5 h-5 mr-2" /> Impact Analysis
                        </h3>
                        <div className="space-y-3">
                            {ci.impacts.length > 0 ? ci.impacts.map(impact => (
                                <Link 
                                    key={impact.id} 
                                    href={`/${impact.type === 'SLA' ? 'sla' : 'cmdb'}/${impact.id}`} 
                                    className="block p-3 bg-red-50 rounded-lg hover:bg-red-100 transition-colors"
                                >
                                    <p className="font-semibold text-red-700">{impact.name} ({impact.type})</p>
                                    <span className="text-xs text-gray-500">Impact Level: {impact.impactLevel}</span>
                                </Link>
                            )) : (
                                <p className="text-sm text-gray-500">No direct impact.</p>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default CIDetailPage;
