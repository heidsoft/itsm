jest.mock('@/lib/env', () => ({
  logger: { debug: jest.fn(), error: jest.fn() },
}));

import {
  saveFilters,
  restoreFilters,
  clearFilters,
  clearAllFilters,
  getSavedFilterPages,
  getDefaultFilters,
  DEFAULT_FILTERS,
} from '../filter-persistence';

describe('filter-persistence', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  describe('saveFilters / restoreFilters', () => {
    it('persists and restores filters', () => {
      saveFilters('tickets', { status: 'open', priority: 'high' });
      const restored = restoreFilters('tickets');
      expect(restored).toEqual({ status: 'open', priority: 'high' });
    });

    it('returns defaults when nothing saved', () => {
      const result = restoreFilters('tickets', { status: 'all' });
      expect(result).toEqual({ status: 'all' });
    });

    it('merges saved data with defaults', () => {
      saveFilters('tickets', { priority: 'low' });
      const result = restoreFilters('tickets', { status: 'all', priority: 'all' });
      expect(result).toEqual({ status: 'all', priority: 'low' });
    });
  });

  describe('clearFilters', () => {
    it('removes specific page filters', () => {
      saveFilters('tickets', { x: 1 });
      saveFilters('incidents', { y: 2 });
      clearFilters('tickets');
      expect(restoreFilters('tickets')).toEqual({});
      expect(restoreFilters('incidents')).toEqual({ y: 2 });
    });
  });

  describe('clearAllFilters', () => {
    it('removes all itsm_filter_ keys', () => {
      saveFilters('a', { x: 1 });
      saveFilters('b', { y: 2 });
      localStorage.setItem('other_key', 'keep');
      clearAllFilters();
      expect(restoreFilters('a')).toEqual({});
      expect(restoreFilters('b')).toEqual({});
      expect(localStorage.getItem('other_key')).toBe('keep');
    });
  });

  describe('getSavedFilterPages', () => {
    it('lists saved page keys', () => {
      saveFilters('tickets', { a: 1 });
      saveFilters('incidents', { b: 2 });
      const pages = getSavedFilterPages();
      expect(pages).toContain('tickets');
      expect(pages).toContain('incidents');
    });
  });

  describe('getDefaultFilters', () => {
    it('returns known page defaults', () => {
      expect(getDefaultFilters('incidents')).toEqual(DEFAULT_FILTERS.incidents);
    });

    it('returns {} for unknown page', () => {
      expect(getDefaultFilters('unknown_page')).toEqual({});
    });
  });
});
