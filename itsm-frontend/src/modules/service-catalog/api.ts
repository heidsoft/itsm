/**
 * Service Catalog API Module
 */

import { httpClient } from '@/lib/api/http-client';
import { ServiceCategory, ServiceStatus } from './constants';
import type {
    ServiceItem,
    CreateServiceItemRequest,
    UpdateServiceItemRequest,
    ServiceQuery,
} from './types';

// Backward compatibility or shared types if needed from global types
// import { ... } from '@/types/service-catalog';

export class ServiceCatalogApi {
    // ==================== Adpaters ====================

    private static toBackendStatus(status?: unknown): 'enabled' | 'disabled' | undefined {
        if (!status) return undefined;
        const s = String(status);
        if (s === ServiceStatus.PUBLISHED || s === 'enabled') return 'enabled';
        if (s === ServiceStatus.DRAFT || s === ServiceStatus.RETIRED || s === 'disabled') return 'disabled';
        return 'disabled';
    }

    private static toFrontendStatus(status?: unknown): ServiceStatus {
        const s = String(status || '');
        return (s === 'enabled' ? ServiceStatus.PUBLISHED : ServiceStatus.RETIRED);
    }

    private static toServiceItem(raw: any): ServiceItem {
        return {
            id: String(raw?.id),
            name: String(raw?.name || ''),
            category: (raw?.category as ServiceCategory) || ServiceCategory.IT_SERVICE,
            status: ServiceCatalogApi.toFrontendStatus(raw?.status),
            shortDescription: String(raw?.description || ''),
            fullDescription: String(raw?.description || ''),
            ciTypeId: typeof raw?.ci_type_id === 'number' ? raw.ci_type_id : undefined,
            cloudServiceId: typeof raw?.cloud_service_id === 'number' ? raw.cloud_service_id : undefined,
            tags: [],
            requiresApproval: true,
            createdBy: 0,
            createdByName: '',
            createdAt: raw?.created_at ? new Date(raw.created_at) : new Date(),
            updatedAt: raw?.updated_at ? new Date(raw.updated_at) : new Date(),
            availability: {
                responseTime: raw?.delivery_time ? Number(raw.delivery_time) : undefined,
            },
        };
    }

    // ==================== Service Items ====================

    static async getServices(
        query?: ServiceQuery
    ): Promise<{
        services: ServiceItem[];
        total: number;
    }> {
        const page = query?.page ?? 1;
        const size = query?.pageSize ?? 10;
        const category = query?.category ? String(query.category) : undefined;
        const status = ServiceCatalogApi.toBackendStatus(query?.status);

        const resp = await httpClient.get<{
            catalogs: any[];
            total: number;
            page: number;
            size: number;
        }>('/api/v1/service-catalogs', {
            page,
            size,
            ...(category ? { category } : {}),
            ...(status ? { status } : {}),
        });

        let services = (resp.catalogs || []).map(ServiceCatalogApi.toServiceItem);

        // Client-side search fallback if backend doesn't support search yet
        if (query?.search) {
            const q = query.search.toLowerCase();
            services = services.filter(s => (s.name || '').toLowerCase().includes(q) || (s.shortDescription || '').toLowerCase().includes(q));
        }

        return { services, total: resp.total || 0 };
    }

    static async getService(id: string): Promise<ServiceItem> {
        const resp = await httpClient.get<any>(`/api/v1/service-catalogs/${id}`);
        return ServiceCatalogApi.toServiceItem(resp);
    }

    static async createService(
        request: CreateServiceItemRequest
    ): Promise<ServiceItem> {
        const payload = {
            name: request.name,
            category: String(request.category),
            description: request.shortDescription || request.fullDescription || '',
            ci_type_id: request.ciTypeId,
            cloud_service_id: request.cloudServiceId,
            delivery_time: String(request.availability?.responseTime ?? request.availability?.resolutionTime ?? 1),
            status: ServiceCatalogApi.toBackendStatus(request.status) || 'enabled',
        };
        const resp = await httpClient.post<any>('/api/v1/service-catalogs', payload);
        return ServiceCatalogApi.toServiceItem(resp);
    }

    static async updateService(
        id: string,
        request: UpdateServiceItemRequest
    ): Promise<ServiceItem> {
        const payload: Record<string, unknown> = {};
        if (request.name !== undefined) payload.name = request.name;
        if (request.category !== undefined) payload.category = String(request.category);
        if (request.shortDescription !== undefined || request.fullDescription !== undefined) {
            payload.description = request.shortDescription || request.fullDescription || '';
        }
        if (request.availability?.responseTime !== undefined) {
            payload.delivery_time = String(request.availability.responseTime);
        }
        if (request.status !== undefined) {
            payload.status = ServiceCatalogApi.toBackendStatus(request.status);
        }
        if (request.ciTypeId !== undefined) payload.ci_type_id = request.ciTypeId;
        if (request.cloudServiceId !== undefined) payload.cloud_service_id = request.cloudServiceId;

        const resp = await httpClient.put<any>(`/api/v1/service-catalogs/${id}`, payload);
        return ServiceCatalogApi.toServiceItem(resp);
    }

    static async deleteService(id: string): Promise<void> {
        return httpClient.delete(`/api/v1/service-catalogs/${id}`);
    }
}
