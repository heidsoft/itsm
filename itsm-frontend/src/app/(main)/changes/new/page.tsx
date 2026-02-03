'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft } from 'lucide-react';
import { FormInput } from '@/components/forms/FormInput';
import { FormTextarea } from '@/components/forms/FormTextarea';
import { App } from 'antd';

const CreateChangePage = () => {
  const router = useRouter();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setLoading(true);
    const formData = new FormData(e.currentTarget);
    const data = Object.fromEntries(formData.entries());
    console.log('新建变更数据:', JSON.stringify(data, null, 2));

    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 1000));
      message.success('变更请求已成功提交！');
      router.push('/changes');
    } catch (error) {
      message.error('提交失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className='p-10 bg-gray-50 min-h-full'>
      <header className='mb-8'>
        <button
          onClick={() => router.back()}
          className='flex items-center text-blue-600 hover:underline mb-4'
        >
          <ArrowLeft className='w-5 h-5 mr-2' />
          返回变更列表
        </button>
        <h2 className='text-4xl font-bold text-gray-800'>新建变更请求</h2>
        <p className='text-gray-500 mt-1'>提交新的IT基础设施或服务变更请求</p>
      </header>

      <div className='bg-white p-8 rounded-lg shadow-md'>
        <form onSubmit={handleSubmit}>
          <div className='space-y-6'>
            <FormInput
              label='变更标题'
              id='title'
              name='title'
              type='text'
              required
              placeholder='简要描述变更内容'
            />
            <FormTextarea
              label='详细描述'
              id='description'
              name='description'
              rows={4}
              required
              placeholder='请详细说明变更的目的、范围和内容...'
            />
            <FormTextarea
              label='变更理由'
              id='justification'
              name='justification'
              rows={3}
              required
              placeholder='为什么需要进行此变更？（例如：解决问题PRB-XXXXX，满足新业务需求等）'
            />
            <div>
              <label htmlFor='type' className='block text-sm font-medium text-gray-700'>
                变更类型
              </label>
              <select
                id='type'
                name='type'
                className='mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500'
                defaultValue='普通变更'
              >
                <option>普通变更</option>
                <option>标准变更</option>
                <option>紧急变更</option>
              </select>
            </div>
            <div>
              <label htmlFor='priority' className='block text-sm font-medium text-gray-700'>
                优先级
              </label>
              <select
                id='priority'
                name='priority'
                className='mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500'
                defaultValue='中'
              >
                <option>紧急</option>
                <option>高</option>
                <option>中</option>
                <option>低</option>
              </select>
            </div>
            <div className='grid grid-cols-1 md:grid-cols-2 gap-6'>
              <FormInput
                label='计划开始时间'
                id='plannedStartDate'
                name='plannedStartDate'
                type='datetime-local'
              />
              <FormInput
                label='计划结束时间'
                id='plannedEndDate'
                name='plannedEndDate'
                type='datetime-local'
              />
            </div>
            <div>
              <label htmlFor='impactScope' className='block text-sm font-medium text-gray-700'>
                影响范围
              </label>
              <select
                id='impactScope'
                name='impactScope'
                className='mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500'
                defaultValue='中'
              >
                <option>高 (影响核心业务)</option>
                <option>中 (影响部分业务或用户)</option>
                <option>低 (影响较小或无影响)</option>
              </select>
            </div>
            <div>
              <label htmlFor='riskLevel' className='block text-sm font-medium text-gray-700'>
                风险等级
              </label>
              <select
                id='riskLevel'
                name='riskLevel'
                className='mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500'
                defaultValue='中'
              >
                <option>高</option>
                <option>中</option>
                <option>低</option>
              </select>
            </div>
            <FormInput
              label='受影响的配置项 (CI ID, 多个用逗号分隔)'
              id='affectedCIs'
              name='affectedCIs'
              type='text'
              placeholder='例如：CI-ECS-001, CI-APP-CRM'
            />
            <FormTextarea
              label='实施计划'
              id='implementationPlan'
              name='implementationPlan'
              rows={5}
              required
              placeholder='详细描述变更的实施步骤...'
            />
            <FormTextarea
              label='回滚计划'
              id='rollbackPlan'
              name='rollbackPlan'
              rows={5}
              required
              placeholder='详细描述如果变更失败如何回滚...'
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
              提交变更请求
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default CreateChangePage;
