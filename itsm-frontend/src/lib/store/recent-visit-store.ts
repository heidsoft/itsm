'use client';

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface RecentVisitItem {
  id: string;
  title: string;
  path: string;
  icon?: string;
  timestamp: number;
}

interface RecentVisitState {
  visits: RecentVisitItem[];
  maxItems: number;
  addVisit: (item: Omit<RecentVisitItem, 'timestamp' | 'id'>) => void;
  removeVisit: (path: string) => void;
  clearVisits: () => void;
}

export const useRecentVisitStore = create<RecentVisitState>()(
  persist(
    (set, get) => ({
      visits: [],
      maxItems: 10,

      addVisit: (item) => {
        const { visits, maxItems } = get();
        // 移除已存在的相同路径记录
        const filteredVisits = visits.filter(v => v.path !== item.path);
        // 添加新记录到开头
        const newVisits = [
          {
            ...item,
            id: `${item.path}-${Date.now()}`,
            timestamp: Date.now(),
          },
          ...filteredVisits,
        ].slice(0, maxItems);

        set({ visits: newVisits });
      },

      removeVisit: (path) => {
        set(state => ({
          visits: state.visits.filter(v => v.path !== path),
        }));
      },

      clearVisits: () => {
        set({ visits: [] });
      },
    }),
    {
      name: 'itsm-recent-visits',
    }
  )
);
