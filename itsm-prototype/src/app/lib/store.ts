import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { Tenant } from './api-config';
import { httpClient } from './http-client';

interface User {
  id: number;
  username: string;
  role: string;
  tenant_id?: number;
  email?: string;
  name?: string;
  department?: string;
}

interface AuthState {
  user: User | null;
  token: string | null;
  currentTenant: Tenant | null;
  isAuthenticated: boolean;
  login: (user: User, token: string, tenant?: Tenant) => void;
  logout: () => void;
  setCurrentTenant: (tenant: Tenant) => void;
  clearTenant: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      currentTenant: null,
      isAuthenticated: false,
      login: (user, token, tenant) => {
        set({ user, token, isAuthenticated: true, currentTenant: tenant });
        localStorage.setItem('auth_token', token);
        httpClient.setToken(token);
        if (tenant) {
          httpClient.setTenantId(tenant.id);
        }
      },
      logout: () => {
        set({ user: null, token: null, isAuthenticated: false, currentTenant: null });
        localStorage.removeItem('auth_token');
        localStorage.removeItem('current_tenant_id');
        httpClient.clearToken();
        httpClient.setTenantId(null);
      },
      setCurrentTenant: (tenant) => {
        set({ currentTenant: tenant });
        httpClient.setTenantId(tenant.id);
      },
      clearTenant: () => {
        set({ currentTenant: null });
        httpClient.setTenantId(null);
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({ 
        user: state.user, 
        token: state.token, 
        currentTenant: state.currentTenant 
      }),
    }
  )
);

// 租户管理状态
interface TenantState {
  tenants: Tenant[];
  loading: boolean;
  error: string | null;
  setTenants: (tenants: Tenant[]) => void;
  addTenant: (tenant: Tenant) => void;
  updateTenant: (id: number, tenant: Partial<Tenant>) => void;
  removeTenant: (id: number) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
}

export const useTenantStore = create<TenantState>((set) => ({
  tenants: [],
  loading: false,
  error: null,
  setTenants: (tenants) => set({ tenants }),
  addTenant: (tenant) => set((state) => ({ tenants: [...state.tenants, tenant] })),
  updateTenant: (id, updatedTenant) => set((state) => ({
    tenants: state.tenants.map((tenant) => 
      tenant.id === id ? { ...tenant, ...updatedTenant } : tenant
    )
  })),
  removeTenant: (id) => set((state) => ({
    tenants: state.tenants.filter((tenant) => tenant.id !== id)
  })),
  setLoading: (loading) => set({ loading }),
  setError: (error) => set({ error }),
}));