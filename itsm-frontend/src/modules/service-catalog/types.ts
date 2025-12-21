import { ServiceCategory, ServiceStatus, ServiceRequestStatus } from './constants';

export interface ServiceItem {
    id: string;
    name: string;
    category: ServiceCategory;
    status: ServiceStatus;
    shortDescription: string;
    fullDescription?: string;

    // Visuals
    icon?: string;
    banner?: string;
    screenshots?: string[];

    // Details
    provider?: string;
    owner?: number;
    ownerName?: string;
    supportTeam?: string;

    // Pricing
    pricing?: {
        type: 'free' | 'fixed' | 'variable';
        amount?: number;
        currency?: string;
        billingCycle?: 'one_time' | 'monthly' | 'yearly';
    };

    // Availability
    availability?: {
        businessHours?: string;
        supportLevel?: 'basic' | 'standard' | 'premium';
        responseTime?: number;        // Hours
        resolutionTime?: number;      // Hours
    };

    // Relations
    relatedCIs?: string[];
    requiredServices?: string[];
    documentation?: string[];

    // Form Configuration
    requestForm?: ServiceRequestForm;

    // Stats
    viewCount?: number;
    requestCount?: number;
    rating?: number;

    // Tags
    tags: string[];
    searchKeywords?: string[];

    // Workflow
    requiresApproval: boolean;
    approvalWorkflow?: string;

    // Metadata
    createdBy: number;
    createdByName: string;
    updatedBy?: number;
    updatedByName?: string;
    createdAt: Date;
    updatedAt: Date;
    publishedAt?: Date;
}

export interface ServiceRequestForm {
    fields: ServiceFormField[];
    layout?: 'single' | 'two_column';
    submitButtonText?: string;
    successMessage?: string;
}

export interface ServiceFormField {
    id: string;
    name: string;
    label: string;
    type: 'text' | 'textarea' | 'number' | 'date' | 'select' | 'multiselect' | 'checkbox' | 'radio' | 'file';
    required: boolean;
    placeholder?: string;
    defaultValue?: any;
    options?: Array<{
        label: string;
        value: string;
    }>;
    validation?: {
        min?: number;
        max?: number;
        pattern?: string;
        message?: string;
    };
    helpText?: string;
    dependsOn?: {
        field: string;
        value: any;
    };
}

export interface CreateServiceItemRequest {
    name: string;
    category: ServiceCategory;
    shortDescription: string;
    fullDescription?: string;
    icon?: string;
    provider?: string;
    owner?: number;
    supportTeam?: string;
    pricing?: ServiceItem['pricing'];
    availability?: ServiceItem['availability'];
    requestForm?: ServiceRequestForm;
    tags?: string[];
    requiresApproval?: boolean;
    approvalWorkflow?: string;
    status?: ServiceStatus;
}

export type UpdateServiceItemRequest = Partial<CreateServiceItemRequest>;

export interface ServiceQuery {
    category?: ServiceCategory;
    status?: ServiceStatus;
    search?: string;
    tags?: string[];
    minRating?: number;
    page?: number;
    pageSize?: number;
    sortBy?: 'name' | 'rating' | 'popularity' | 'newest';
    sortOrder?: 'asc' | 'desc';
}
