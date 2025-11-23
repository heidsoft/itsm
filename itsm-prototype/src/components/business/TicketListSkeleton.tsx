'use client';

import React from 'react';
import { Skeleton, Card, Row, Col } from 'antd';

const SkeletonRow: React.FC = () => (
  <div className='flex items-center p-4 border-b border-gray-100 bg-white'>
    <div className='w-12 flex justify-center'>
      <Skeleton.Avatar active size='small' shape='square' />
    </div>
    <div className='flex-1 flex items-center'>
      <Skeleton.Avatar active size='large' shape='square' />
      <div className='flex-1 ml-3'>
        <Skeleton.Input active style={{ width: '60%' }} />
        <Skeleton.Input active style={{ width: '40%', marginTop: '8px' }} />
      </div>
    </div>
    <div className='w-32 flex justify-center'>
      <Skeleton.Input active style={{ width: '80%' }} />
    </div>
    <div className='w-24 flex justify-center'>
      <Skeleton.Input active style={{ width: '80%' }} />
    </div>
    <div className='w-32 flex justify-center'>
      <Skeleton.Input active style={{ width: '80%' }} />
    </div>
    <div className='w-32 flex justify-center'>
      <Skeleton.Avatar active size='small' />
      <Skeleton.Input active style={{ width: '60%', marginLeft: '8px' }} />
    </div>
    <div className='w-32 flex justify-center'>
      <Skeleton.Input active style={{ width: '80%' }} />
    </div>
    <div className='w-48 flex justify-center'>
      <Skeleton.Button active size='small' />
      <Skeleton.Button active size='small' style={{ marginLeft: '8px' }} />
      <Skeleton.Button active size='small' style={{ marginLeft: '8px' }} />
    </div>
  </div>
);

export const TicketListSkeleton: React.FC<{ rows?: number }> = ({ rows = 5 }) => {
  return (
    <div className='bg-white rounded-lg border border-gray-100 shadow-sm overflow-hidden'>
      {Array.from({ length: rows }).map((_, index) => (
        <SkeletonRow key={index} />
      ))}
    </div>
  );
};
