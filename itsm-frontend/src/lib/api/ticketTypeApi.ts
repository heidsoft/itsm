import { httpClient } from './http-client';
import type { CreateTicketTypeRequest, TicketTypeDefinition, TicketTypeListResponse, TicketTypePresetDefinition, UpdateTicketTypeRequest } from '@/types/ticket-type';

export class TicketTypeApi {
  static list(params?: { status?: string; keyword?: string; includeArchived?: boolean; page?: number; pageSize?: number }) { return httpClient.get<TicketTypeListResponse>('/api/v1/ticket-types', params); }
  static get(id: number) { return httpClient.get<TicketTypeDefinition>(`/api/v1/ticket-types/${id}`); }
  static create(data: CreateTicketTypeRequest) { return httpClient.post<TicketTypeDefinition>('/api/v1/ticket-types', data); }
  static update(id: number, data: UpdateTicketTypeRequest) { return httpClient.put<TicketTypeDefinition>(`/api/v1/ticket-types/${id}`, data); }
  static setEnabled(id: number, enabled: boolean) {
    if (enabled) {
      return httpClient.post<TicketTypeDefinition>(`/api/v1/ticket-types/${id}/enable`, {});
    }
    return httpClient.post<TicketTypeDefinition>(`/api/v1/ticket-types/${id}/disable`, {});
  }
  static restore(id: number) { return httpClient.post<TicketTypeDefinition>(`/api/v1/ticket-types/${id}/restore`, {}); }
  static clone(id: number, code: string, name: string) { return httpClient.post<TicketTypeDefinition>(`/api/v1/ticket-types/${id}/clone`, { code, name }); }
  static listPresets() { return httpClient.get<TicketTypePresetDefinition[]>('/api/v1/ticket-type-presets'); }
  static installPreset(id: string, data: { code?: string; name?: string } = {}) { return httpClient.post<TicketTypeDefinition>(`/api/v1/ticket-type-presets/${id}/install`, data); }
}
