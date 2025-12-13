'use client';

import React, { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Clock, ArrowLeft, Info, AlertCircle } from 'lucide-react';
import { ServiceCatalogApi } from '@/lib/api/service-catalog-api';
import { ServiceItem, ServiceStatus } from '@/types/service-catalog';

const ServiceRequestPage = () => {
  const params = useParams();
  const router = useRouter();
  const serviceId = parseInt(params.serviceId as string);

  const [catalog, setCatalog] = useState<ServiceItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reason, setReason] = useState('');

  // 获取服务目录详情
  useEffect(() => {
    const fetchCatalog = async () => {
      try {
        setLoading(true);
        const response = await ServiceCatalogApi.getServices({ page: 1, pageSize: 100 });
        const foundCatalog = response.services.find(c => Number(c.id) === serviceId);
        if (foundCatalog) setCatalog(foundCatalog);
        else setError('服务目录不存在');
      } catch (err) {
        setError('获取服务目录失败');
        console.error('Failed to fetch catalog:', err);
      } finally {
        setLoading(false);
      }
    };

    if (serviceId) {
      fetchCatalog();
    }
  }, [serviceId]);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    if (!catalog) return;

    try {
      setSubmitting(true);
      await ServiceCatalogApi.createServiceRequest({
        serviceId: String(catalog.id),
        formData: { reason: reason.trim() || '' },
      });

      alert(`服务请求 "${catalog.name}" 已成功提交！`);
      router.push('/my-requests');
    } catch (err) {
      console.error('Failed to create service request:', err);
      alert('提交服务请求失败，请稍后重试');
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className='p-10 bg-gray-50 min-h-full flex items-center justify-center'>
        <div className='text-center'>
          <div className='animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4'></div>
          <p className='text-gray-600'>加载服务信息中...</p>
        </div>
      </div>
    );
  }

  if (error || !catalog) {
    return (
      <div className='p-10 bg-gray-50 min-h-full flex items-center justify-center'>
        <div className='text-center'>
          <AlertCircle className='w-12 h-12 text-red-500 mx-auto mb-4' />
          <p className='text-red-600 mb-4'>{error || '服务不存在'}</p>
          <button
            onClick={() => router.push('/service-catalog')}
            className='bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700'
          >
            返回服务目录
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className='p-10 bg-gray-50 min-h-full'>
      <header className='mb-8'>
        <button
          onClick={() => router.back()}
          className='flex items-center text-blue-600 hover:underline mb-4'
        >
          <ArrowLeft className='w-5 h-5 mr-2' />
          返回服务目录
        </button>
        <h2 className='text-4xl font-bold text-gray-800'>服务请求：{catalog.name}</h2>
        <p className='text-gray-500 mt-1'>{catalog.shortDescription || ''}</p>
      </header>

      <div className='grid grid-cols-1 lg:grid-cols-3 gap-8'>
        <div className='lg:col-span-2 bg-white p-8 rounded-lg shadow-md'>
          <form onSubmit={handleSubmit}>
            <div className='space-y-6'>
              <div>
                <label htmlFor='reason' className='block text-sm font-medium text-gray-700 mb-2'>
                  申请理由 <span className='text-gray-400'>(可选)</span>
                </label>
                <textarea
                  id='reason'
                  value={reason}
                  onChange={e => setReason(e.target.value)}
                  rows={4}
                  className='w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500'
                  placeholder='请详细说明您申请此服务的业务背景和目的...'
                  maxLength={500}
                />
                <p className='text-sm text-gray-500 mt-1'>{reason.length}/500</p>
              </div>
            </div>

            <div className='mt-8 flex justify-end space-x-4'>
              <button
                type='button'
                onClick={() => router.back()}
                className='bg-gray-200 text-gray-700 font-semibold py-2 px-4 rounded-lg hover:bg-gray-300'
                disabled={submitting}
              >
                取消
              </button>
              <button
                type='submit'
                className='bg-blue-600 text-white font-semibold py-2 px-4 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed'
                disabled={submitting}
              >
                {submitting ? '提交中...' : '提交请求'}
              </button>
            </div>
          </form>
        </div>

        <div className='lg:col-span-1'>
          <div className='bg-blue-50 border-l-4 border-blue-500 p-6 rounded-r-lg'>
            <div className='flex items-center'>
              <Info className='w-6 h-6 text-blue-700 mr-3' />
              <h3 className='text-xl font-semibold text-blue-800'>服务摘要</h3>
            </div>
            <div className='mt-4 space-y-3 text-gray-700'>
              <div className='flex justify-between'>
                <span className='font-semibold'>服务类别:</span>
                <span>{catalog.category}</span>
              </div>
              <div className='flex justify-between items-center'>
                <span className='font-semibold'>预计交付时间:</span>
                <span className='flex items-center font-bold text-blue-700'>
                  <Clock className='w-4 h-4 mr-1.5' />
                  {catalog.availability?.responseTime ||
                    catalog.availability?.resolutionTime ||
                    '-'}
                </span>
              </div>
              <div className='flex justify-between'>
                <span className='font-semibold'>服务状态:</span>
                <span
                  className={`px-2 py-1 rounded text-xs font-medium ${
                    catalog.status === ServiceStatus.PUBLISHED
                      ? 'bg-green-100 text-green-800'
                      : 'bg-red-100 text-red-800'
                  }`}
                >
                  {catalog.status === ServiceStatus.PUBLISHED ? '可用' : '不可用'}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ServiceRequestPage;
