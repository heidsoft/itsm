'use client';

import { useState, useEffect } from 'react';
import { mockCIs, mockRelations } from '@/app/lib/cmdb-relations';

export const useCMDBData = () => {
  const [cis, setCis] = useState(mockCIs);
  const [relations] = useState(mockRelations);
  const [loading, setLoading] = useState(true);
  const [searchText, setSearchText] = useState('');
  const [filterType, setFilterType] = useState('');

  useEffect(() => {
    // Simulate data fetching
    setTimeout(() => {
      setLoading(false);
    }, 1000);
  }, []);

  const filteredCIs = cis.filter(ci => {
    const searchTextLower = searchText.toLowerCase();
    const nameMatch = ci.name.toLowerCase().includes(searchTextLower);
    const idMatch = ci.id.toLowerCase().includes(searchTextLower);
    const ipMatch = ci.ip.toLowerCase().includes(searchTextLower);
    const typeMatch = filterType ? ci.type === filterType : true;
    return (nameMatch || idMatch || ipMatch) && typeMatch;
  });

  const refresh = () => {
    setLoading(true);
    setTimeout(() => {
      setCis(mockCIs);
      setLoading(false);
    }, 1000);
  };

  return {
    cis: filteredCIs,
    relations,
    loading,
    setSearchText,
    setFilterType,
    refresh,
    setCis,
  };
};
