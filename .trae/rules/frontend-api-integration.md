---
alwaysApply: false
description: Frontend API Integration Guidelines - No Mock Data
---

# Frontend API Integration Rules

## Core Principle

**Strictly prohibit using mock data in the frontend code. All data must be fetched from the real backend API.**

## API Client Structure

### Location
- API clients: `src/lib/api/*.ts`
- Store (Zustand): `src/lib/store/*.ts`
- Types: `src/types/*.ts`

### Example API Client

```typescript
// src/lib/api/ticket-api.ts
import { apiClient } from './client';

export const TicketAPI = {
  list: (params?: TicketQueryParams) =>
    apiClient.get<TicketListResponse>('/tickets', { params }),

  get: (id: number) =>
    apiClient.get<TicketResponse>(`/tickets/${id}`),

  create: (data: CreateTicketRequest) =>
    apiClient.post<TicketResponse>('/tickets', data),

  update: (id: number, data: UpdateTicketRequest) =>
    apiClient.put<TicketResponse>(`/tickets/${id}`, data),

  delete: (id: number) =>
    apiClient.delete(`/tickets/${id}`),
};
```

### Base Client

```typescript
// src/lib/api/client.ts
import axios from 'axios';

const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL + '/api/v1',
  withCredentials: true,
});

apiClient.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export { apiClient };
```

## State Management

### Zustand Store Pattern

```typescript
// src/lib/store/ticket-store.ts
import { create } from 'zustand';
import { TicketAPI } from '../api/ticket-api';

interface TicketState {
  tickets: Ticket[];
  loading: boolean;
  error: string | null;
  fetchTickets: (params?: TicketQueryParams) => Promise<void>;
}

export const useTicketStore = create<TicketState>((set) => ({
  tickets: [],
  loading: false,
  error: null,

  fetchTickets: async (params) => {
    set({ loading: true, error: null });
    try {
      const response = await TicketAPI.list(params);
      set({ tickets: response.data.items, loading: false });
    } catch (error) {
      set({ error: error.message, loading: false });
    }
  },
}));
```

## Error Handling

### API Error States

```typescript
// ❌ Wrong - mock data fallback
try {
  const data = await api.getData();
  setData(data);
} catch {
  setData(MOCK_DATA); // Never do this
}

// ✅ Correct - proper error handling
const { data, error } = useQuery({
  queryKey: ['tickets'],
  queryFn: () => TicketAPI.list(),
  retry: false,
  errorElement: <ErrorState message={error?.message} />,
});
```

### Loading States

```typescript
// Component usage
function TicketList() {
  const { tickets, loading, error } = useTicketStore();

  if (loading) return <Skeleton />;
  if (error) return <ErrorState message={error} />;
  if (!tickets.length) return <EmptyState />;

  return <TicketTable tickets={tickets} />;
}
```

## Backend API Alignment (404 Handling)

If a frontend API call returns **404 Not Found**:

1. **Verify Frontend Request**: Check URL path, HTTP method, and parameters
2. **Verify Backend Implementation**:
   - Check `router/router.go` to ensure route exists and is registered
   - Verify controller and handler functions are linked
3. **Action**:
   - If backend is missing the endpoint, **implement the endpoint in the backend**
   - If path is mismatched, correct the frontend path or alias the backend route
   - **Never revert frontend to use mock data**

## Module Coverage

| Module | API File | Store | Status |
|--------|----------|-------|--------|
| Tickets | ticket-api.ts | ticket-store.ts | ✅ |
| Incidents | incident-api.ts | incident-store.ts | ✅ |
| Changes | change-api.ts | change-store.ts | ✅ |
| Knowledge | knowledge-api.ts | - | ⚠️ |
| Service Catalog | catalog-api.ts | catalog-store.ts | ✅ |
| SLA | sla-api.ts | - | ⚠️ |
| Workflow | workflow-api.ts | - | ⚠️ |
| CMDB | cmdb-api.ts | cmdb-store.ts | ✅ |
| MSP | msp-api.ts | msp-store.ts | ✅ |

## Field Naming

### Frontend (camelCase) ↔ Backend (snake_case)

| Frontend | Backend | Example |
|----------|---------|---------|
| ticketNumber | ticket_number | CHG-2026-001 |
| assigneeId | assignee_id | 42 |
| createdAt | created_at | 2026-05-17T10:00:00Z |

### Use TypeScript Types

```typescript
// types/ticket.ts
export interface Ticket {
  id: number;
  ticketNumber: string;
  title: string;
  status: TicketStatus;
  priority: TicketPriority;
  assigneeId?: number;
  tenantId: number;
  createdAt: string;
  updatedAt: string;
}

// Use in API calls
const tickets = await TicketAPI.list({ status: 'open', page: 1 });
```

## Checklist

- [ ] Remove any `mock-data.ts` files
- [ ] Remove `mockData`, `DEMO_DATA` from components
- [ ] Remove catch blocks that load mock data
- [ ] Verify `useQuery` / `useSWR` calls real API
- [ ] Handle 404 by implementing backend, not mocking
- [ ] Use Zustand stores for global state
- [ ] Use local state for component-specific data
- [ ] Apply to all modules: Tickets, SLA, Workflow, Analytics, Users, Catalog