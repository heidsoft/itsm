'use client';

import React from 'react';
import { ChartPlaceholder } from '@/components/charts/ChartPlaceholder';

const ServiceCatalogUsagePage = () => {
  // Data aggregation logic simplified, using placeholder components

  // Color configuration removed, using placeholder components

  return (
    <div className='p-10 bg-gray-50 min-h-full'>
      <header className='mb-8'>
        <h2 className='text-4xl font-bold text-gray-800'>Service Catalog Usage Report</h2>
        <p className='text-gray-500 mt-1'>This report shows the usage of the service catalog.</p>
      </header>

      <div className='grid grid-cols-1 lg:grid-cols-2 gap-8'>
        <div className='bg-white p-6 rounded-lg shadow-md'>
          <h3 className='text-xl font-semibold text-gray-800 mb-4'>Requests by Service</h3>
          <ChartPlaceholder
            type='pie'
            title='Requests by Service Type Statistics'
            description='Shows the distribution of request counts by different service types'
            height={300}
          />
        </div>

        <div className='bg-white p-6 rounded-lg shadow-md'>
          <h3 className='text-xl font-semibold text-gray-800 mb-4'>Requests by Status</h3>
          <ChartPlaceholder
            type='bar'
            title='Requests by Status Statistics'
            description='Shows the distribution of request counts by different statuses'
            height={300}
          />
        </div>
      </div>
    </div>
  );
};

export default ServiceCatalogUsagePage;
