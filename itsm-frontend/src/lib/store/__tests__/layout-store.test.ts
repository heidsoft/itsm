/**
 * Real tests for the Zustand layout store in
 * `src/lib/store/layout-store.ts`.
 *
 * Replaces the previous 312-line `layout-store.test.ts`, which had two
 * completely separate test suites glued together:
 *   - Lines 1-149: hardcoded local-variable no-ops (`expect(false).toBe(false)`)
 *     that did not import or exercise the store at all.
 *   - Lines 150-211: a second copy of the same no-ops.
 *   - Lines 213-312: a small TDD-style suite that DID exercise the store,
 *     but used `jest.resetModules()` and dynamic imports to work around
 *     Zustand's module-level singleton — a smell that hides real bugs.
 *
 * The store only exposes `collapsed`, `toggleCollapsed`, and `setCollapsed`.
 * We test those three things directly.
 */

import { act } from '@testing-library/react';
import { useLayoutStore } from '../layout-store';

describe('useLayoutStore', () => {
  beforeEach(() => {
    // Reset to the documented initial state between tests so we never
    // depend on test ordering.
    useLayoutStore.setState({ collapsed: false });
  });

  describe('initial state', () => {
    it('exposes collapsed: false on first read', () => {
      expect(useLayoutStore.getState().collapsed).toBe(false);
    });
  });

  describe('toggleCollapsed', () => {
    it('flips false → true', () => {
      act(() => {
        useLayoutStore.getState().toggleCollapsed();
      });
      expect(useLayoutStore.getState().collapsed).toBe(true);
    });

    it('flips true → false', () => {
      act(() => {
        useLayoutStore.setState({ collapsed: true });
      });
      act(() => {
        useLayoutStore.getState().toggleCollapsed();
      });
      expect(useLayoutStore.getState().collapsed).toBe(false);
    });

    it('returns to the same value after two toggles', () => {
      const before = useLayoutStore.getState().collapsed;
      act(() => {
        useLayoutStore.getState().toggleCollapsed();
      });
      act(() => {
        useLayoutStore.getState().toggleCollapsed();
      });
      expect(useLayoutStore.getState().collapsed).toBe(before);
    });
  });

  describe('setCollapsed', () => {
    it('sets collapsed to true', () => {
      act(() => {
        useLayoutStore.getState().setCollapsed(true);
      });
      expect(useLayoutStore.getState().collapsed).toBe(true);
    });

    it('sets collapsed to false after it was true', () => {
      act(() => {
        useLayoutStore.getState().setCollapsed(true);
      });
      act(() => {
        useLayoutStore.getState().setCollapsed(false);
      });
      expect(useLayoutStore.getState().collapsed).toBe(false);
    });
  });

  describe('consumer synchronization (Zustand guarantees)', () => {
    it('mutations are visible to subsequent getState() calls', () => {
      // Zustand's `getState()` returns a snapshot of the current state.
      // Two calls in a row will return equal-but-not-identical objects
      // when the store has been mutated in between, and identical
      // references when nothing has changed. This contract is what makes
      // multiple React components share the same value.
      const before = useLayoutStore.getState().collapsed;

      act(() => {
        useLayoutStore.getState().setCollapsed(true);
      });

      const after = useLayoutStore.getState().collapsed;
      expect(before).toBe(false);
      expect(after).toBe(true);
    });
  });
});