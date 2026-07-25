import { renderHook, act } from '@testing-library/react';
import { useTicketFilters } from '../useTicketFilters';

jest.mock('@/lib/services/ticket-service', () => ({
  TicketStatus: { OPEN: 'open', IN_PROGRESS: 'in_progress', RESOLVED: 'resolved', CLOSED: 'closed' },
  TicketPriority: { URGENT: 'urgent', HIGH: 'high', MEDIUM: 'medium', LOW: 'low' },
}));

describe('useTicketFilters', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should return default filter state', () => {
    const { result } = renderHook(() => useTicketFilters());

    expect(result.current.componentFilters).toEqual({
      status: 'all',
      priority: 'all',
      type: 'all',
      keyword: '',
      dateStart: '',
      dateEnd: '',
      sortBy: 'createdAt_desc',
    });
  });

  it('should compute empty domain filters from default state', () => {
    const { result } = renderHook(() => useTicketFilters());

    expect(result.current.domainFilters).toEqual({
      status: undefined,
      priority: undefined,
      type: undefined,
      keyword: undefined,
      dateRange: undefined,
    });
  });

  it('should update component filters', () => {
    const { result } = renderHook(() => useTicketFilters());

    act(() => {
      result.current.updateComponentFilters({ status: 'open', keyword: 'test' });
    });

    expect(result.current.componentFilters.status).toBe('open');
    expect(result.current.componentFilters.keyword).toBe('test');
  });

  it('should map component filters to domain filters', () => {
    const { result } = renderHook(() => useTicketFilters());

    act(() => {
      result.current.updateComponentFilters({
        status: 'open',
        priority: 'p1',
        keyword: 'search term',
        dateStart: '2024-01-01',
        dateEnd: '2024-01-31',
      });
    });

    expect(result.current.domainFilters.status).toBe('open');
    expect(result.current.domainFilters.priority).toBe('urgent');
    expect(result.current.domainFilters.keyword).toBe('search term');
    expect(result.current.domainFilters.dateRange).toEqual(['2024-01-01', '2024-01-31']);
  });

  it('should reset filters to default', () => {
    const { result } = renderHook(() => useTicketFilters());

    act(() => {
      result.current.updateComponentFilters({ status: 'open', keyword: 'test' });
    });

    act(() => {
      result.current.resetFilters();
    });

    expect(result.current.componentFilters.status).toBe('all');
    expect(result.current.componentFilters.keyword).toBe('');
  });

  it('should map domain filters to component filters', () => {
    const { result } = renderHook(() => useTicketFilters());

    const componentState = result.current.mapDomainToComponent({
      status: 'open' as any,
      priority: 'high' as any,
      keyword: 'test',
      dateRange: ['2024-01-01', '2024-01-31'],
    });

    expect(componentState.status).toBe('open');
    expect(componentState.priority).toBe('p2');
    expect(componentState.keyword).toBe('test');
    expect(componentState.dateStart).toBe('2024-01-01');
    expect(componentState.dateEnd).toBe('2024-01-31');
  });

  it('should update domain filters and convert to component state', () => {
    const { result } = renderHook(() => useTicketFilters());

    act(() => {
      result.current.updateDomainFilters({
        status: 'resolved' as any,
        keyword: 'domain filter',
      });
    });

    expect(result.current.componentFilters.status).toBe('resolved');
    expect(result.current.componentFilters.keyword).toBe('domain filter');
  });
});
