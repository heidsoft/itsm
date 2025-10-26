'use client';

import React from 'react';
import { ChartPlaceholder } from '@/components/charts/ChartPlaceholder';

const IncidentTrendsPage = () => {
  // Data aggregation logic simplified, using placeholder components

  return (
    <div className='p-10 bg-gray-50 min-h-full'>
      <header className='mb-8'>
        <h2 className='text-4xl font-bold text-gray-800'>Incident Trends Report</h2>
        <p className='text-gray-500 mt-1'>
          This report shows the trend of incidents over time and by priority.
        </p>
      </header>

      <div className='grid grid-cols-1 lg:grid-cols-2 gap-8'>
        <div className='bg-white p-6 rounded-lg shadow-md'>
          <h3 className='text-xl font-semibold text-gray-800 mb-4'>Incidents per Day</h3>
          <ChartPlaceholder
            type='line'
            title='Daily Incident Count Trend'
            description='Shows the trend of daily incident counts'
            height={300}
          />
        </div>

        <div className='bg-white p-6 rounded-lg shadow-md'>
          <h3 className='text-xl font-semibold text-gray-800 mb-4'>Incidents by Priority</h3>
          <ChartPlaceholder
            type='bar'
            title='Incidents by Priority Statistics'
            description='Shows the distribution of incident counts by different priorities'
            height={300}
          />
        </div>
      </div>
    </div>
  );
};

export default IncidentTrendsPage;
