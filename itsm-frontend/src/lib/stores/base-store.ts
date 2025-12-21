import { create } from 'zustand';

/**
 * 基础状态接口
 */
export interface BaseState {
  loading: boolean;
  error: string | null;
  initialized: boolean;
}

/**
 * 分页状态接口
 */
export interface PaginationState {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
}

/**
 * 列表状态接口
 */
export interface ListState<T> extends BaseState, PaginationState {
  items: T[];
  selectedItems: T[];
  filters: Record<string, unknown>;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

/**
 * 详情状态接口
 */
export interface DetailState<T> extends BaseState {
  item: T | null;
}

/**
 * 基础操作接口
 */
export interface BaseActions {
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  clearError: () => void;
  setInitialized: (initialized: boolean) => void;
  reset: () => void;
}

/**
 * 分页操作接口
 */
export interface PaginationActions {
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
  setPagination: (pagination: Partial<PaginationState>) => void;
  nextPage: () => void;
  prevPage: () => void;
  goToPage: (page: number) => void;
}

/**
 * 列表操作接口
 */
export interface ListActions<T> extends BaseActions, PaginationActions {
  setItems: (items: T[]) => void;
  addItem: (item: T) => void;
  updateItem: (id: string | number, updates: Partial<T>) => void;
  removeItem: (id: string | number) => void;
  setSelectedItems: (selectedItems: T[]) => void;
  selectItem: (item: T) => void;
  deselectItem: (item: T) => void;
  selectAll: () => void;
  deselectAll: () => void;
  setFilters: (filters: Record<string, unknown>) => void;
  updateFilter: (key: string, value: unknown) => void;
  clearFilters: () => void;
  setSorting: (sortBy: string, sortOrder: 'asc' | 'desc') => void;
  clearSorting: () => void;
}

/**
 * 详情操作接口
 */
export interface DetailActions<T> extends BaseActions {
  setItem: (item: T | null) => void;
  updateItem: (updates: Partial<T>) => void;
}

/**
 * 创建基础状态
 */
export const createBaseState = (): BaseState => ({
  loading: false,
  error: null,
  initialized: false,
});

/**
 * 创建分页状态
 */
export const createPaginationState = (): PaginationState => ({
  page: 1,
  pageSize: 20,
  total: 0,
  totalPages: 0,
});

/**
 * 创建列表状态
 */
export const createListState = <T>(): ListState<T> => ({
  ...createBaseState(),
  ...createPaginationState(),
  items: [],
  selectedItems: [],
  filters: {},
  sortBy: undefined,
  sortOrder: undefined,
});

/**
 * 创建详情状态
 */
export const createDetailState = <T>(): DetailState<T> => ({
  ...createBaseState(),
  item: null,
});

/**
 * 创建基础操作
 */
export const createBaseActions = <T extends BaseState>(
  set: (partial: Partial<T>) => void
): BaseActions => ({
  setLoading: (loading: boolean) => set({ loading } as Partial<T>),
  setError: (error: string | null) => set({ error } as Partial<T>),
  clearError: () => set({ error: null } as Partial<T>),
  setInitialized: (initialized: boolean) => set({ initialized } as Partial<T>),
  reset: () => set(createBaseState() as Partial<T>),
});

/**
 * 创建分页操作
 */
export const createPaginationActions = <T extends PaginationState>(
  set: (partial: Partial<T>) => void,
  get: () => T
): PaginationActions => ({
  setPage: (page: number) => set({ page } as Partial<T>),
  setPageSize: (pageSize: number) => set({ pageSize, page: 1 } as Partial<T>),
  setPagination: (pagination: Partial<PaginationState>) => set(pagination as Partial<T>),
  nextPage: () => {
    const state = get();
    if (state.page < state.totalPages) {
      set({ page: state.page + 1 } as Partial<T>);
    }
  },
  prevPage: () => {
    const state = get();
    if (state.page > 1) {
      set({ page: state.page - 1 } as Partial<T>);
    }
  },
  goToPage: (page: number) => {
    const state = get();
    if (page >= 1 && page <= state.totalPages) {
      set({ page } as Partial<T>);
    }
  },
});

/**
 * 创建列表操作
 */
export const createListActions = <T extends ListState<Item>, Item>(
  set: (partial: Partial<T>) => void,
  get: () => T,
  getItemId: (item: Item) => string | number
): ListActions<Item> => {
  const baseActions = createBaseActions(set);
  const paginationActions = createPaginationActions(set, get);

  return {
    ...baseActions,
    ...paginationActions,
    setItems: (items: Item[]) => set({ items } as Partial<T>),
    addItem: (item: Item) => {
      const state = get();
      set({ items: [...state.items, item] } as Partial<T>);
    },
    updateItem: (id: string | number, updates: Partial<Item>) => {
      const state = get();
      const items = state.items.map(item =>
        getItemId(item) === id ? { ...item, ...updates } : item
      );
      set({ items } as Partial<T>);
    },
    removeItem: (id: string | number) => {
      const state = get();
      const items = state.items.filter(item => getItemId(item) !== id);
      const selectedItems = state.selectedItems.filter(item => getItemId(item) !== id);
      set({ items, selectedItems } as Partial<T>);
    },
    setSelectedItems: (selectedItems: Item[]) => set({ selectedItems } as Partial<T>),
    selectItem: (item: Item) => {
      const state = get();
      const isSelected = state.selectedItems.some(selected => getItemId(selected) === getItemId(item));
      if (!isSelected) {
        set({ selectedItems: [...state.selectedItems, item] } as Partial<T>);
      }
    },
    deselectItem: (item: Item) => {
      const state = get();
      const selectedItems = state.selectedItems.filter(selected => getItemId(selected) !== getItemId(item));
      set({ selectedItems } as Partial<T>);
    },
    selectAll: () => {
      const state = get();
      set({ selectedItems: [...state.items] } as Partial<T>);
    },
    deselectAll: () => set({ selectedItems: [] as Item[] } as Partial<T>),
    setFilters: (filters: Record<string, unknown>) => set({ filters, page: 1 } as Partial<T>),
    updateFilter: (key: string, value: unknown) => {
      const state = get();
      set({ 
        filters: { ...state.filters, [key]: value },
        page: 1 
      } as Partial<T>);
    },
    clearFilters: () => set({ filters: {}, page: 1 } as Partial<T>),
    setSorting: (sortBy: string, sortOrder: 'asc' | 'desc') => 
      set({ sortBy, sortOrder } as Partial<T>),
    clearSorting: () => set({ sortBy: undefined, sortOrder: undefined } as Partial<T>),
  };
};

/**
 * 创建详情操作
 */
export const createDetailActions = <T extends DetailState<Item>, Item>(
  set: (partial: Partial<T>) => void
): DetailActions<Item> => {
  const baseActions = createBaseActions(set);

  return {
    ...baseActions,
    setItem: (item: Item | null) => set({ item } as Partial<T>),
    updateItem: (updates: Partial<Item>) => {
      set((state: T) => ({
        item: state.item ? { ...state.item, ...updates } : null
      } as Partial<T>));
    },
  };
};

/**
 * 创建列表Store工厂函数
 */
export const createListStore = <Item>(
  getItemId: (item: Item) => string | number,
  initialState?: Partial<ListState<Item>>
) => {
  return create<ListState<Item> & ListActions<Item>>((set, get) => {
    const actions = createListActions(set, get, getItemId);
    
    return {
      ...createListState<Item>(),
      ...initialState,
      ...actions,
    };
  });
};

/**
 * 创建详情Store工厂函数
 */
export const createDetailStore = <Item>(
  initialState?: Partial<DetailState<Item>>
) => {
  return create<DetailState<Item> & DetailActions<Item>>((set) => {
    const actions = createDetailActions(set);
    
    return {
      ...createDetailState<Item>(),
      ...initialState,
      ...actions,
    };
  });
};