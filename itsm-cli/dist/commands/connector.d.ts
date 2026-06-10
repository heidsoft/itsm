import React from 'react';
export declare const ConnectorListCommand: React.FC;
export declare const ConnectorTestCommand: React.FC<{
    name: string;
}>;
export declare const ConnectorHealthCommand: React.FC;
export declare const ConnectorSendCommand: React.FC<{
    name: string;
    channel: string;
    message: string;
}>;
