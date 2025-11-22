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

// 模拟改进计划数据
const mockImprovements = [
  {
    id: 'IMP-001',
    title: '优化云资源申请流程',
    status: '进行中',
    owner: '产品部',
    target: '将云资源交付时间缩短20%',
    createdAt: '2025-05-01',
  },
  {
    id: 'IMP-002',
    title: '提升服务台首次解决率',
    status: '待评估',
    owner: '服务台经理',
    target: '将首次解决率提升至85%',
    createdAt: '2025-04-10',
  },
  {
    id: 'IMP-003',
    title: '自动化安全漏洞扫描与修复',
    status: '已完成',
    owner: '安全团队',
    target: '自动化覆盖率达到90%',
    createdAt: '2025-03-01',
  },
];

const ImprovementStatusBadge = ({ status }) => {
  const colors = {
    进行中: 'bg-blue-100 text-blue-800',
    待评估: 'bg-yellow-100 text-yellow-800',
    已完成: 'bg-green-100 text-green-800',
  };
  return (
    <span className={`px-2 py-1 text-xs font-medium rounded-full ${colors[status]}`}>{status}</span>
  );
};

const ImprovementListPage = () => {
  const [filter, setFilter] = useState('全部');

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
            {mockImprovements.map(imp => (
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
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default ImprovementListPage;
