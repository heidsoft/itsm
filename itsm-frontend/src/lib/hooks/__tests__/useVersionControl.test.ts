import { renderHook, act, waitFor } from '@testing-library/react';
import {
  useVersionControl,
  useVersionTracker,
  VersionConflictError,
  isVersionConflictError,
  parseVersionConflictError,
} from '../useVersionControl';

// Mock antd
jest.mock('antd', () => ({
  Modal: {
    confirm: jest.fn(({ onOk, onCancel }) => {
      // Default: resolve with refresh (onOk)
      onOk?.();
    }),
  },
  message: {
    success: jest.fn(),
    error: jest.fn(),
    info: jest.fn(),
  },
}));

// Mock lucide-react
jest.mock('lucide-react', () => ({
  RefreshCw: () => null,
  AlertCircle: () => null,
}));

describe('useVersionControl', () => {
  describe('VersionConflictError', () => {
    it('should create a version conflict error', () => {
      const error = new VersionConflictError(1, 3, { id: 1 });
      expect(error.name).toBe('VersionConflictError');
      expect(error.currentVersion).toBe(1);
      expect(error.serverVersion).toBe(3);
      expect(error.serverData).toEqual({ id: 1 });
      expect(error.message).toContain('版本冲突');
    });
  });

  describe('isVersionConflictError', () => {
    it('should return true for VersionConflictError instance', () => {
      const error = new VersionConflictError(1, 2);
      expect(isVersionConflictError(error)).toBe(true);
    });

    it('should return true for 409 status error', () => {
      expect(isVersionConflictError({ status: 409 })).toBe(true);
      expect(isVersionConflictError({ code: 409 })).toBe(true);
      expect(isVersionConflictError({ statusCode: 409 })).toBe(true);
    });

    it('should return false for non-conflict errors', () => {
      expect(isVersionConflictError(new Error('generic'))).toBe(false);
      expect(isVersionConflictError({ status: 500 })).toBe(false);
      expect(isVersionConflictError(null)).toBe(false);
      expect(isVersionConflictError(undefined)).toBe(false);
      expect(isVersionConflictError('string error')).toBe(false);
    });
  });

  describe('parseVersionConflictError', () => {
    it('should parse VersionConflictError instance', () => {
      const error = new VersionConflictError(1, 5, { name: 'data' });
      const parsed = parseVersionConflictError(error);
      expect(parsed).toBe(error);
    });

    it('should parse 409 status error with data', () => {
      const error = {
        status: 409,
        data: { version: 5, serverData: { name: 'server' } },
      };
      const parsed = parseVersionConflictError(error);
      expect(parsed).toBeInstanceOf(VersionConflictError);
      expect(parsed?.serverVersion).toBe(5);
    });

    it('should parse 409 with currentVersion field', () => {
      const error = {
        statusCode: 409,
        data: { currentVersion: 7 },
      };
      const parsed = parseVersionConflictError(error);
      expect(parsed?.serverVersion).toBe(7);
    });

    it('should return null for non-conflict errors', () => {
      expect(parseVersionConflictError(new Error('generic'))).toBeNull();
      expect(parseVersionConflictError({ status: 500 })).toBeNull();
      expect(parseVersionConflictError(null)).toBeNull();
    });
  });

  describe('useVersionControl hook', () => {
    const mockUpdateData = jest.fn();
    const mockFetchLatest = jest.fn();

    const defaultConfig = {
      initialData: { id: 1, name: 'Test', version: 1 } as any,
      updateData: mockUpdateData,
      fetchLatest: mockFetchLatest,
      dataName: '工单',
    };

    beforeEach(() => {
      jest.clearAllMocks();
    });

    it('should initialize with correct state', () => {
      const { result } = renderHook(() => useVersionControl(defaultConfig));

      expect(result.current.data).toEqual({ id: 1, name: 'Test', version: 1 });
      expect(result.current.version).toBe(1);
      expect(result.current.isUpdating).toBe(false);
      expect(result.current.hasConflict).toBe(false);
      expect(result.current.conflictInfo).toBeNull();
      expect(result.current.isLoading).toBe(false);
    });

    it('should initialize with null data', () => {
      const { result } = renderHook(() =>
        useVersionControl({ ...defaultConfig, initialData: null })
      );

      expect(result.current.data).toBeNull();
      expect(result.current.version).toBe(0);
    });

    it('should update data successfully', async () => {
      mockUpdateData.mockResolvedValue({ id: 1, name: 'Updated', version: 2 });

      const { result } = renderHook(() => useVersionControl(defaultConfig));

      await act(async () => {
        const updated = await result.current.update({ name: 'Updated' });
        expect(updated).toEqual({ id: 1, name: 'Updated', version: 2 });
      });

      expect(result.current.data).toEqual({ id: 1, name: 'Updated', version: 2 });
      expect(result.current.version).toBe(2);
      expect(result.current.isUpdating).toBe(false);
    });

    it('should throw when no data to update', async () => {
      const { result } = renderHook(() =>
        useVersionControl({ ...defaultConfig, initialData: null })
      );

      await expect(
        act(async () => {
          await result.current.update({ name: 'Fail' } as any);
        })
      ).rejects.toThrow('No data to update');
    });

    it('should refresh data', async () => {
      mockFetchLatest.mockResolvedValue({ id: 1, name: 'Fresh', version: 5 } as any);

      const { result } = renderHook(() => useVersionControl(defaultConfig));

      await act(async () => {
        const refreshed = await result.current.refresh();
        expect(refreshed).toEqual({ id: 1, name: 'Fresh', version: 5 });
      });

      expect(result.current.data).toEqual({ id: 1, name: 'Fresh', version: 5 });
      expect(result.current.version).toBe(5);
    });

    it('should throw when refreshing without fetchLatest', async () => {
      const { result } = renderHook(() =>
        useVersionControl({ ...defaultConfig, fetchLatest: undefined })
      );

      await expect(
        act(async () => {
          await result.current.refresh();
        })
      ).rejects.toThrow('fetchLatest not provided');
    });

    it('should force overwrite data', async () => {
      mockUpdateData.mockResolvedValue({ id: 1, name: 'Forced', version: 10 } as any);

      const { result } = renderHook(() => useVersionControl(defaultConfig));

      await act(async () => {
        const forced = await result.current.forceOverwrite({ name: 'Forced' });
        expect(forced).toEqual({ id: 1, name: 'Forced', version: 10 });
      });

      expect(result.current.data).toEqual({ id: 1, name: 'Forced', version: 10 });
      expect(result.current.hasConflict).toBe(false);
    });

    it('should throw when force overwrite with no data', async () => {
      const { result } = renderHook(() =>
        useVersionControl({ ...defaultConfig, initialData: null })
      );

      await expect(
        act(async () => {
          await result.current.forceOverwrite({ name: 'Fail' } as any);
        })
      ).rejects.toThrow('No data to update');
    });

    it('should clear conflict state', () => {
      const { result } = renderHook(() => useVersionControl(defaultConfig));

      act(() => {
        result.current.clearConflict();
      });

      expect(result.current.hasConflict).toBe(false);
      expect(result.current.conflictInfo).toBeNull();
    });

    it('should discard local changes', async () => {
      mockFetchLatest.mockResolvedValue({ id: 1, name: 'Latest', version: 3 } as any);

      const { result } = renderHook(() => useVersionControl(defaultConfig));

      await act(async () => {
        await result.current.discardLocalChanges();
      });

      expect(result.current.data).toEqual({ id: 1, name: 'Latest', version: 3 });
    });

    it('should handle non-conflict update errors', async () => {
      mockUpdateData.mockRejectedValue(new Error('Server error'));

      const { result } = renderHook(() => useVersionControl(defaultConfig));

      await expect(
        act(async () => {
          await result.current.update({ name: 'Fail' });
        })
      ).rejects.toThrow('Server error');

      expect(result.current.isUpdating).toBe(false);
    });

    it('should update data when initialData changes', () => {
      const { result, rerender } = renderHook(
        (props) => useVersionControl(props),
        { initialProps: defaultConfig }
      );

      expect(result.current.version).toBe(1);

      rerender({
        ...defaultConfig,
        initialData: { id: 1, name: 'Changed', version: 3 } as any,
      });

      expect(result.current.data).toEqual({ id: 1, name: 'Changed', version: 3 });
      expect(result.current.version).toBe(3);
    });
  });

  describe('useVersionTracker', () => {
    it('should initialize with default version', () => {
      const { result } = renderHook(() => useVersionTracker());

      expect(result.current.version).toBe(0);
      expect(result.current.baseVersion).toBe(0);
      expect(result.current.hasChanges).toBe(false);
    });

    it('should initialize with custom version', () => {
      const { result } = renderHook(() => useVersionTracker(5));

      expect(result.current.version).toBe(5);
      expect(result.current.baseVersion).toBe(5);
      expect(result.current.hasChanges).toBe(false);
    });

    it('should track version changes', () => {
      const { result } = renderHook(() => useVersionTracker(1));

      act(() => {
        result.current.updateVersion(2);
      });

      expect(result.current.version).toBe(2);
      expect(result.current.hasChanges).toBe(true);
    });

    it('should not show changes when version matches base', () => {
      const { result } = renderHook(() => useVersionTracker(1));

      act(() => {
        result.current.updateVersion(1);
      });

      expect(result.current.hasChanges).toBe(false);
    });

    it('should mark as saved', () => {
      const { result } = renderHook(() => useVersionTracker(1));

      act(() => {
        result.current.updateVersion(3);
      });

      expect(result.current.hasChanges).toBe(true);

      act(() => {
        result.current.markSaved();
      });

      expect(result.current.version).toBe(3);
      expect(result.current.baseVersion).toBe(3);
      expect(result.current.hasChanges).toBe(false);
    });

    it('should mark saved with explicit version', () => {
      const { result } = renderHook(() => useVersionTracker(1));

      act(() => {
        result.current.markSaved(10);
      });

      expect(result.current.version).toBe(10);
      expect(result.current.baseVersion).toBe(10);
      expect(result.current.hasChanges).toBe(false);
    });

    it('should reset to new version', () => {
      const { result } = renderHook(() => useVersionTracker(5));

      act(() => {
        result.current.updateVersion(10);
      });

      act(() => {
        result.current.reset(0);
      });

      expect(result.current.version).toBe(0);
      expect(result.current.baseVersion).toBe(0);
      expect(result.current.hasChanges).toBe(false);
    });

    it('should check stale correctly', () => {
      const { result } = renderHook(() => useVersionTracker(3));

      expect(result.current.checkStale(5)).toBe(true);
      expect(result.current.checkStale(3)).toBe(false);
      expect(result.current.checkStale(1)).toBe(false);
    });
  });
});
