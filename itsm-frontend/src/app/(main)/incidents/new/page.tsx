'use client';

import React, { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, Search, X } from 'lucide-react';
import { FormInput } from '@/components/forms/FormInput';
import { FormTextarea } from '@/components/forms/FormTextarea';
import { App } from 'antd';
import { incidentService, IncidentPriority, IncidentSource } from '@/lib/services/incident-service';
import { CMDBApi, ConfigurationItem } from '@/lib/api/cmdb-api';

interface ConfigItem extends Pick<ConfigurationItem, 'id' | 'name' | 'ci_type' | 'status'> {
  id: number;
}

export default function NewIncidentPage() {
  const router = useRouter();
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [selectedCIs, setSelectedCIs] = useState<ConfigItem[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [searchResults, setSearchResults] = useState<ConfigItem[]>([]);

  useEffect(() => {
    if (searchTerm.trim()) {
      const fetchCIs = async () => {
        try {
          const { default: CMDBApi } = await import('@/lib/api/cmdb-api');
          const results = await CMDBApi.searchCIs({ keyword: searchTerm });
          setSearchResults(results.items);
        } catch (error) {
          console.error('Failed to search CIs:', error);
          setSearchResults([]);
        }
      };
      
      const timeoutId = setTimeout(fetchCIs, 300);
      return () => clearTimeout(timeoutId);
    } else {
      setSearchResults([]);
    }
  }, [searchTerm]);

  const handleAddCI = (ci: ConfigItem) => {
    if (!selectedCIs.find(item => item.id === ci.id)) {
      setSelectedCIs([...selectedCIs, ci]);
    }
    setSearchTerm('');
  };

  const handleRemoveCI = (ciId: number) => {
    setSelectedCIs(selectedCIs.filter(ci => ci.id !== ciId));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    const formData = new FormData(e.target as HTMLFormElement);

    const data = {
      title: formData.get('title') as string,
      description: formData.get('description') as string,
      priority: formData.get('priority') as IncidentPriority,
      type: formData.get('type') as string,
      source: 'manual' as IncidentSource,
      affected_cis: selectedCIs.map(ci => ci.id),
    };

    try {
      await incidentService.createIncident(data);
      message.success('事件创建成功');
      router.push('/incidents');
    } catch (error) {
      console.error('Failed to create incident:', error);
      message.error('创建事件失败，请重试');
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
          返回
        </button>
        <h2 className='text-4xl font-bold text-gray-800'>创建新事件</h2>
        <p className='text-gray-500 mt-1'>手动记录一个新的IT事件</p>
      </header>

      <div className='bg-white p-8 rounded-lg shadow-md'>
        <form onSubmit={handleSubmit}>
          <div className='space-y-6'>
            <FormInput
              label='事件标题'
              id='title'
              name='title'
              type='text'
              required
              placeholder='简要描述该事件'
            />
            <FormTextarea
              label='详细描述'
              id='description'
              name='description'
              rows={6}
              required
              placeholder='请提供该事件的详细信息，包括影响范围、发生时间等...'
            />

            {/* Configuration Items Selection */}
            <div>
              <label className='block text-sm font-medium text-gray-700 mb-2'>
                受影响的配置项
              </label>

              <div className='flex items-center justify-between p-3 bg-blue-50 border border-blue-200 rounded-lg'>
                <div className='flex items-center space-x-2'>
                  <Search className='w-4 h-4 text-blue-600' />
                  <input
                    type='text'
                    placeholder='搜索配置项...'
                    value={searchTerm}
                    onChange={e => setSearchTerm(e.target.value)}
                    className='bg-transparent border-none outline-none flex-1 text-sm'
                  />
                </div>
              </div>

              {/* Search Results */}
              {searchResults.length > 0 && (
                <div className='mt-2 bg-white border border-gray-200 rounded-lg shadow-sm max-h-40 overflow-y-auto'>
                  {searchResults.map(ci => (
                    <div
                      key={ci.id}
                      onClick={() => handleAddCI(ci)}
                      className='p-3 hover:bg-gray-50 cursor-pointer border-b border-gray-100 last:border-b-0'
                    >
                      <div className='flex items-center justify-between'>
                        <div>
                          <div className='font-medium text-sm'>{ci.name}</div>
                          <div className='text-xs text-gray-500'>{ci.ci_type}</div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Selected CIs */}
              {selectedCIs.length > 0 && (
                <div className='mt-3'>
                  <div className='text-sm font-medium text-gray-700 mb-2'>已选择:</div>
                  <div className='flex flex-wrap gap-2'>
                    {selectedCIs.map(ci => (
                      <div
                        key={ci.id}
                        className='flex items-center bg-blue-100 text-blue-800 px-3 py-1 rounded-full text-sm'
                      >
                        <span>{ci.name}</span>
                        <button
                          type='button'
                          onClick={() => handleRemoveCI(ci.id)}
                          className='ml-2 text-blue-600 hover:text-blue-800'
                        >
                          <X className='w-3 h-3' />
                        </button>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {/* Priority Selection */}
            <div>
              <label className='block text-sm font-medium text-gray-700 mb-2'>优先级</label>
              <select
                name='priority'
                required
                className='w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent'
              >
                <option value=''>选择优先级</option>
                <option value='low'>低</option>
                <option value='medium'>中</option>
                <option value='high'>高</option>
                <option value='critical'>紧急</option>
              </select>
            </div>

            {/* Type Selection */}
            <div>
              <label className='block text-sm font-medium text-gray-700 mb-2'>事件类型</label>
              <select
                name='type'
                required
                className='w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent'
              >
                <option value=''>选择类型</option>
                <option value='hardware'>硬件</option>
                <option value='software'>软件</option>
                <option value='network'>网络</option>
                <option value='security'>安全</option>
                <option value='performance'>性能</option>
                <option value='other'>其他</option>
              </select>
            </div>
          </div>

          <div className='mt-8 flex justify-end space-x-4'>
            <button
              type='button'
              onClick={() => router.back()}
              className='px-6 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50'
            >
              取消
            </button>
            <button
              type='submit'
              className='px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700'
            >
              创建事件
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
