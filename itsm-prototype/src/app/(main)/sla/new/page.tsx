'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { FormInput } from '@/components/forms/FormInput';
import { FormTextarea } from '@/components/forms/FormTextarea';

const CreateSLAPage = () => {
  const router = useRouter();

  const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const formData = new FormData(e.currentTarget);
    const data = Object.fromEntries(formData.entries());
    console.log('新建SLA数据:', JSON.stringify(data, null, 2));
    alert('SLA已成功创建！');
    router.push('/sla'); // 提交后返回SLA列表
  };

  return (
    <div className='p-10 bg-gray-50 min-h-full'>
      <header className='mb-8'>
        <button
          onClick={() => router.back()}
          className='flex items-center text-blue-600 hover:underline mb-4'
        >
          <ArrowLeft className='w-5 h-5 mr-2' />
          返回SLA列表
        </button>
        <h2 className='text-4xl font-bold text-gray-800'>新建服务级别协议 (SLA)</h2>
        <p className='text-gray-500 mt-1'>定义服务提供商和客户之间的服务水平期望</p>
      </header>

      <div className='bg-white p-8 rounded-lg shadow-md'>
        <form onSubmit={handleSubmit}>
          <div className='space-y-6'>
            <FormInput
              label='SLA 名称'
              id='slaName'
              name='slaName'
              type='text'
              required
              placeholder='例如：生产CRM系统可用性SLA'
            />
            <FormTextarea
              label='服务描述 (协议、参与方、提供的服务摘要)'
              id='serviceDescription'
              name='serviceDescription'
              rows={4}
              required
              placeholder='请详细描述此SLA涵盖的服务范围、涉及的客户和提供商...'
            />

            <h3 className='text-xl font-semibold text-gray-700 mt-8 mb-4'>服务质量与响应性目标</h3>
            <FormInput
              label='服务质量目标 (例如：99.9% 可用性)'
              id='serviceQualityTarget'
              name='serviceQualityTarget'
              type='text'
              placeholder='例如：99.9% 可用性'
            />
            <FormInput
              label='响应时间目标 (例如：高优先级事件15分钟内响应)'
              id='responseTimeTarget'
              name='responseTimeTarget'
              type='text'
              placeholder='例如：高优先级事件15分钟内响应'
            />
            <FormInput
              label='解决时间目标 (例如：高优先级事件4小时内解决)'
              id='resolutionTimeTarget'
              name='resolutionTimeTarget'
              type='text'
              placeholder='例如：高优先级事件4小时内解决'
            />

            <h3 className='text-xl font-semibold text-gray-700 mt-8 mb-4'>绩效衡量与管理</h3>
            <FormTextarea
              label='绩效衡量 (需要衡量的指标列表，如何衡量)'
              id='performanceMeasures'
              name='performanceMeasures'
              rows={4}
              placeholder='例如：可用性通过监控系统自动采集，响应时间通过工单系统记录...'
            />

            <h3 className='text-xl font-semibold text-gray-700 mt-8 mb-4'>附加条款 (可选)</h3>
            <FormTextarea
              label='未能满足约定条款的惩罚'
              id='penalties'
              name='penalties'
              rows={3}
              placeholder='例如：若月度可用性低于99.5%，将对客户进行服务费减免...'
            />
            <FormTextarea
              label='取消条件'
              id='cancellationConditions'
              name='cancellationConditions'
              rows={3}
              placeholder='例如：若客户连续三个月未支付服务费用，SLA自动失效...'
            />
          </div>
          <div className='mt-8 flex justify-end'>
            <button
              type='button'
              onClick={() => router.back()}
              className='bg-gray-200 text-gray-700 font-semibold py-2 px-4 rounded-lg hover:bg-gray-300 mr-4'
            >
              取消
            </button>
            <button
              type='submit'
              className='bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700'
            >
              创建SLA
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateSLAPage;
