'use client';

import { useState, useEffect } from 'react';

interface SatisfactionMetrics {
  overall: number;
  responseTime: number;
  resolutionQuality: number;
  communication: number;
  totalResponses: number;
}

export const useSatisfactionData = () => {
  const [metrics, setMetrics] = useState<SatisfactionMetrics | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Simulate data fetching
    setTimeout(() => {
      setMetrics({
        overall: 4.2,
        responseTime: 4.0,
        resolutionQuality: 4.3,
        communication: 4.1,
        totalResponses: 1250,
      });
      setLoading(false);
    }, 1000);
  }, []);

  return { metrics, loading };
};
