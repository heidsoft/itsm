'use client';

import { useState, useEffect } from 'react';

export const useAdminData = () => {
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Simulate data fetching
    setTimeout(() => {
      setLoading(false);
    }, 1000);
  }, []);

  return { loading };
};
