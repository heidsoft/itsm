
import React from 'react';
import { Sidebar } from '../components/Sidebar';

export default function ReportsLayout({ children }: { children: React.ReactNode }) {
    return (
        <div className="flex h-screen bg-gray-100 font-sans">
            <Sidebar />
            <main className="flex-1 overflow-y-auto">
                {children}
            </main>
        </div>
    );
}
