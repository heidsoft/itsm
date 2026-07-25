import { act } from '@testing-library/react';
import { useRecentVisitStore } from '../recent-visit-store';

describe('useRecentVisitStore', () => {
  beforeEach(() => {
    act(() => {
      useRecentVisitStore.setState({ visits: [], maxItems: 10 });
    });
  });

  describe('initial state', () => {
    it('starts with empty visits', () => {
      expect(useRecentVisitStore.getState().visits).toEqual([]);
    });
    it('has maxItems of 10', () => {
      expect(useRecentVisitStore.getState().maxItems).toBe(10);
    });
  });

  describe('addVisit', () => {
    it('adds a visit to the beginning', () => {
      act(() => {
        useRecentVisitStore.getState().addVisit({ title: 'Page A', path: '/a' });
      });
      const { visits } = useRecentVisitStore.getState();
      expect(visits).toHaveLength(1);
      expect(visits[0].title).toBe('Page A');
      expect(visits[0].path).toBe('/a');
      expect(visits[0].timestamp).toBeGreaterThan(0);
    });

    it('deduplicates by path and moves to front', () => {
      act(() => {
        useRecentVisitStore.getState().addVisit({ title: 'A', path: '/a' });
        useRecentVisitStore.getState().addVisit({ title: 'B', path: '/b' });
        useRecentVisitStore.getState().addVisit({ title: 'A updated', path: '/a' });
      });
      const { visits } = useRecentVisitStore.getState();
      expect(visits).toHaveLength(2);
      expect(visits[0].title).toBe('A updated');
      expect(visits[0].path).toBe('/a');
      expect(visits[1].path).toBe('/b');
    });

    it('respects maxItems limit', () => {
      act(() => {
        useRecentVisitStore.setState({ maxItems: 3 });
        for (let i = 0; i < 5; i++) {
          useRecentVisitStore.getState().addVisit({ title: `P${i}`, path: `/p${i}` });
        }
      });
      expect(useRecentVisitStore.getState().visits).toHaveLength(3);
    });
  });

  describe('removeVisit', () => {
    it('removes visit by path', () => {
      act(() => {
        useRecentVisitStore.getState().addVisit({ title: 'A', path: '/a' });
        useRecentVisitStore.getState().addVisit({ title: 'B', path: '/b' });
      });
      act(() => {
        useRecentVisitStore.getState().removeVisit('/a');
      });
      const { visits } = useRecentVisitStore.getState();
      expect(visits).toHaveLength(1);
      expect(visits[0].path).toBe('/b');
    });

    it('does nothing for non-existent path', () => {
      act(() => {
        useRecentVisitStore.getState().addVisit({ title: 'A', path: '/a' });
      });
      act(() => {
        useRecentVisitStore.getState().removeVisit('/nonexistent');
      });
      expect(useRecentVisitStore.getState().visits).toHaveLength(1);
    });
  });

  describe('clearVisits', () => {
    it('clears all visits', () => {
      act(() => {
        useRecentVisitStore.getState().addVisit({ title: 'A', path: '/a' });
        useRecentVisitStore.getState().addVisit({ title: 'B', path: '/b' });
      });
      act(() => {
        useRecentVisitStore.getState().clearVisits();
      });
      expect(useRecentVisitStore.getState().visits).toEqual([]);
    });
  });
});
