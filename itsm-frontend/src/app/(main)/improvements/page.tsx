'use client';

import {
  TrendingUp,
  Target,
  CheckCircle,
  AlertCircle,
  Users,
  Clock,
  PlusCircle,
  Filter,
} from 'lucide-react';

import React, { useState } from 'react';
import Link from 'next/link';

type ImprovementStatus = '进行中' | '待评估' | '已完成';

interface Improvement {
  id: string;
  title: string;
  status: ImprovementStatus;
  owner: string;
  target: string;
  createdAt: string;
}

const ImprovementStatusBadge = ({ status }: { status: ImprovementStatus }) => {
  const colors: Record<ImprovementStatus, string> = {
    进行中: 'bg-blue-100 text-blue-800',
    待评估: 'bg-yellow-100 text-yellow-800',
    已完成: 'bg-green-100 text-green-800',
  };
  return (
    <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status] || ''}`}>
      {status}
    </span>
  );
};

const ImprovementListPage = () => {
  const [improvements, setImprovements] = useState<Improvement[]>([]);
  const [filter, setFilter] = useState('全部');

  // 加载改进计划
  React.useEffect(() => {
    loadImprovements();
  }, []);

  const loadImprovements = async () => {
    try {
      const { TicketApi } = await import('@/lib/api/ticket-api');
      // 假设改进计划是一种特殊类型的工单
      const response = await TicketApi.getTickets({ type: 'improvement', page: 1, size: 100 });
      
      const mappedImprovements: Improvement[] = response.tickets.map((ticket: any) => ({
        id: ticket.ticketNumber || `IMP-${ticket.id}`,
        title: ticket.title,
        status: mapStatus(ticket.status),
        owner: ticket.assignee?.name || '未分配',
        target: ticket.description || '无目标描述',
        createdAt: new Date(ticket.createdAt).toLocaleDateString()
      }));
      
      setImprovements(mappedImprovements);
    } catch (error) {
      console.error('加载改进计划失败:', error);
      // 失败时保持空列表，不显示mock数据
    }
  };

  const mapStatus = (ticketStatus: string): ImprovementStatus => {
    switch (ticketStatus) {
      case 'open': return '待评估';
      case 'in_progress': return '进行中';
      case 'resolved':
      case 'closed': return '已完成';
      default: return '待评估';
    }
  };

  const filteredImprovements = improvements.filter(imp => {
    if (filter === '全部') return true;
    return imp.status === filter;
  });

  return (
    <div className='p-10 bg-gray-50 min-h-full'>
      <header className='mb-8 flex justify-between items-center'>
        <div>
          <h2 className='text-4xl font-bold text-gray-800'>持续改进</h2>
          <p className='text-gray-500 mt-1'>识别、规划和实施IT服务和流程的改进</p>
        </div>
        <Link href='/improvements/new' passHref>
          <button className='flex items-center bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 transition-colors'>
            <PlusCircle className='w-5 h-5 mr-2' />
            新建改进计划
          </button>
        </Link>
      </header>

      {/* 筛选器 */}
      <div className='flex items-center mb-6 bg-white p-3 rounded-lg shadow-sm'>
        <Filter className='w-5 h-5 text-gray-500 mr-3' />
        <span className='text-sm font-semibold mr-4'>筛选:</span>
        <div className='flex space-x-2'>
          {['全部', '进行中', '待评估', '已完成'].map(f => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className={`px-3 py-1 text-sm rounded-md ${
                filter === f ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-700'
              }`}
            >
              {f}
            </button>
          ))}
        </div>
      </div>

      {/* 改进计划列表表格 */}
      <div className='bg-white rounded-lg shadow-md overflow-hidden'>
        <table className='min-w-full divide-y divide-gray-200'>
          <thead className='bg-gray-50'>
            <tr>
              <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                计划ID
              </th>
              <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                标题
              </th>
              <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                状态
              </th>
              <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                负责人
              </th>
              <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                目标
              </th>
              <th className='px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider'>
                创建时间
              </th>
            </tr>
          </thead>
          <tbody className='bg-white divide-y divide-gray-200'>
            {filteredImprovements.length > 0 ? (
              filteredImprovements.map(imp => (
                <tr key={imp.id} className='hover:bg-gray-50'>
                  <td className='px-6 py-4 whitespace-nowrap'>
                    <Link
                      href={`/improvements/${imp.id}`}
                      className='text-blue-600 font-semibold hover:underline'
                    >
                      {imp.id}
                    </Link>
                  </td>
                  <td className='px-6 py-4 whitespace-nowrap max-w-sm truncate'>{imp.title}</td>
                  <td className='px-6 py-4 whitespace-nowrap'>
                    <ImprovementStatusBadge status={imp.status} />
                  </td>
                  <td className='px-6 py-4 whitespace-nowrap text-sm text-gray-700'>{imp.owner}</td>
                  <td className='px-6 py-4 whitespace-nowrap max-w-sm truncate text-sm text-gray-700'>
                    {imp.target}
                  </td>
                  <td className='px-6 py-4 whitespace-nowrap text-sm text-gray-500'>
                    {imp.createdAt}
                  </td>
                </tr>
              ))
            ) : (
              <tr>
                <td colSpan={6} className="px-6 py-4 text-center text-gray-500">
                  暂无改进计划
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default ImprovementListPage;
