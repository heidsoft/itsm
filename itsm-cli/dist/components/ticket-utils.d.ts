import React from 'react';
export declare function StatusBadge({ status }: {
    status: string;
}): React.JSX.Element;
export declare function PriorityBadge({ priority }: {
    priority: string;
}): React.JSX.Element;
export declare function TicketRow({ ticket }: {
    ticket: {
        ticketNumber: string;
        title: string;
        status: string;
        priority: string;
        assigneeName?: string;
    };
}): React.JSX.Element;
