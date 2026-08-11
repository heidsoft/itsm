'use client';

import { useQuery } from '@tanstack/react-query';
import {
  capabilityAllows,
  getCapabilities,
  type ProductCapabilityState,
} from '@/lib/api/capability-api';

export const useCapabilities = () => {
  const query = useQuery({
    queryKey: ['product-capabilities'],
    queryFn: getCapabilities,
    staleTime: 60_000,
    retry: 1,
  });
  const capabilities = query.data?.items || [];

  return {
    ...query,
    capabilities,
    allows: (key: string, action = 'read') => capabilityAllows(capabilities, key, action),
    find: (key: string): ProductCapabilityState | undefined =>
      capabilities.find(item => item.key === key),
  };
};
