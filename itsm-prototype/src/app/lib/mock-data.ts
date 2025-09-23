
// Mock incident data (converted from incidents/[incidentId]/page.tsx to array)
export const mockIncidentsData = [
    { id: 'INC-00125', title: 'Web server CPU usage exceeds 95% in Hangzhou Zone J', priority: 'High', status: 'In Progress', source: 'Alibaba Cloud Monitoring', sourceIcon: 'cpu', lastUpdate: '5 minutes ago', isMajorIncident: false, assignee: 'Zhang San', createdAt: '2025-06-28 10:15:23' },
    { id: 'INC-00124', title: 'Users report unable to access CRM system', priority: 'High', status: 'Assigned', source: 'Service Desk', sourceIcon: 'user', lastUpdate: '25 minutes ago', isMajorIncident: false, assignee: 'Wang Wu', createdAt: '2025-06-28 09:45:10' },
    { id: 'INC-00123', title: 'Suspicious SSH login attempts detected (47.98.x.x)', priority: 'Medium', status: 'In Progress', source: 'Security Center', sourceIcon: 'shield', lastUpdate: '1 hour ago', isMajorIncident: true, assignee: 'Zhao Liu', createdAt: '2025-06-28 08:30:00' },
    { id: 'INC-00122', title: 'Production database master-slave sync delay', priority: 'Medium', status: 'Resolved', source: 'Alibaba Cloud Monitoring', sourceIcon: 'cpu', lastUpdate: '3 hours ago', isMajorIncident: false, assignee: 'Qian Qi', createdAt: '2025-06-27 18:00:00' },
    { id: 'INC-00121', title: 'User password reset request failed', priority: 'Low', status: 'Closed', source: 'Service Desk', sourceIcon: 'user', lastUpdate: '1 day ago', isMajorIncident: false, assignee: 'Zhou Jiu', createdAt: '2025-06-27 10:00:00' },
];

// Mock problem data (converted from problems/[problemId]/page.tsx to array)
export const mockProblemsData = [
    { id: 'PRB-00001', title: 'CRM system intermittent login failures', status: 'Investigating', priority: 'High', assignee: 'Wang Wu', createdAt: '2025-06-20' },
    { id: 'PRB-00002', title: 'Some users unable to access VPN', status: 'Resolved', priority: 'Medium', assignee: 'Li Ming', createdAt: '2025-06-15' },
    { id: 'PRB-00003', title: 'Database connection pool exhaustion causing application crashes', status: 'Known Error', priority: 'High', assignee: 'Qian Qi', createdAt: '2025-06-10' },
    { id: 'PRB-00004', title: 'Email service sending delays', status: 'Closed', priority: 'Low', assignee: 'Zhao Liu', createdAt: '2025-06-05' },
];

// Mock change data (converted from changes/[changeId]/page.tsx to array)
export const mockChangesData = [
    { id: 'CHG-00001', title: 'CRM system database upgrade', type: 'Normal Change', status: 'Pending Approval', priority: 'High', assignee: 'Li Si', createdAt: '2025-06-25' },
    { id: 'CHG-00002', title: 'Add VPN gateway firewall rules', type: 'Standard Change', status: 'Approved', priority: 'Medium', assignee: 'Zhang San', createdAt: '2025-06-20' },
    { id: 'CHG-00003', title: 'Emergency fix for web server security vulnerability', type: 'Emergency Change', status: 'In Implementation', priority: 'Critical', assignee: 'Wang Wu', createdAt: '2025-06-18' },
    { id: 'CHG-00004', title: 'Email service migration to new platform', type: 'Normal Change', status: 'Completed', priority: 'Medium', assignee: 'Zhao Liu', createdAt: '2025-06-10' },
];

// Mock user request data (copied from my-requests/page.tsx)
export const mockRequestsData = [
    { id: 'REQ-00101', serviceName: 'Apply for Virtual Machine (VM)', status: 'In Progress', submittedAt: '2025-06-28 14:30', detailsLink: '/service-catalog/request/apply-vm' },
    { id: 'REQ-00100', serviceName: 'Reset Password', status: 'Completed', submittedAt: '2025-06-27 10:15', detailsLink: '/service-catalog/request/reset-password' },
    { id: 'REQ-00099', serviceName: 'Apply for Cloud Database (RDS)', status: 'Pending Approval', submittedAt: '2025-06-26 16:00', detailsLink: '/service-catalog/request/apply-rds' },
    { id: 'REQ-00098', serviceName: 'Object Storage Expansion', status: 'Rejected', submittedAt: '2025-06-25 09:00', detailsLink: '/service-catalog/request/expand-oss' },
    { id: 'REQ-00097', serviceName: 'Apply for Application Access Permission', status: 'Completed', submittedAt: '2025-06-24 11:45', detailsLink: '/service-catalog/request/apply-permission' },
];
