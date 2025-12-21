export enum ServiceCategory {
    IT_SERVICE = 'it_service',
    BUSINESS_SERVICE = 'business_service',
    TECHNICAL_SERVICE = 'technical_service',
    SUPPORT_SERVICE = 'support_service',
}

export enum ServiceStatus {
    DRAFT = 'draft',
    PUBLISHED = 'published',
    RETIRED = 'retired',
    // Backend legacy alignment
    ENABLED = 'enabled',
    DISABLED = 'disabled',
}

export enum ServiceRequestStatus {
    DRAFT = 'draft',
    SUBMITTED = 'submitted',
    PENDING_APPROVAL = 'pending_approval',
    APPROVED = 'approved',
    REJECTED = 'rejected',
    IN_PROGRESS = 'in_progress',
    COMPLETED = 'completed',
    CANCELLED = 'cancelled',
}
