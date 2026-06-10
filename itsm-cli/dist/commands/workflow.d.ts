import React from 'react';
export declare const WorkflowInstancesCommand: React.FC<{
    page?: number;
}>;
export declare const WorkflowTasksCommand: React.FC;
export declare const WorkflowCompleteCommand: React.FC<{
    id: string;
    outcome: string;
    comment?: string;
}>;
export declare const ApprovalListCommand: React.FC<{
    status?: string;
}>;
export declare const ApprovalActionCommand: React.FC<{
    id: string;
    action: 'approve' | 'reject';
    comment?: string;
}>;
