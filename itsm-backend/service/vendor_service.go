package service

import (
    "context"
    "errors"
    "itsm-backend/dto"
    "itsm-backend/ent"
    "itsm-backend/ent/vendor"

    "go.uber.org/zap"
)

type VendorService struct {
    client *ent.Client
    logger *zap.SugaredLogger
}

func NewVendorService(client *ent.Client, logger *zap.SugaredLogger) *VendorService {
    return &VendorService{client: client, logger: logger}
}

func (s *VendorService) CreateVendor(ctx context.Context, req *dto.CreateVendorRequest, tenantID int) (*dto.VendorResponse, error) {
    v, err := s.client.Vendor.Create().
        SetName(req.Name).
        SetCode(req.Code).
        SetVendorType(req.VendorType).
        SetContactName(req.ContactName).
        SetContactEmail(req.ContactEmail).
        SetContactPhone(req.ContactPhone).
        SetAddress(req.Address).
        SetWebsite(req.Website).
        SetTenantID(tenantID).
        Save(ctx)
    if err != nil {
        return nil, err
    }
    return &dto.VendorResponse{
        ID: v.ID, Name: v.Name, Code: v.Code, VendorType: v.VendorType,
        ContactName: v.ContactName, ContactEmail: v.ContactEmail,
        ContactPhone: v.ContactPhone, Address: v.Address, Website: v.Website,
        Rating: v.Rating, Status: v.Status, TenantID: v.TenantID,
        CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
    }, nil
}

func (s *VendorService) ListVendors(ctx context.Context, tenantID int, page, size int) ([]*dto.VendorResponse, int, error) {
    query := s.client.Vendor.Query().Where(vendor.TenantID(tenantID))
    total, _ := query.Count(ctx)
    vendors, _ := query.Offset((page-1)*size).Limit(size).All(ctx)
    result := make([]*dto.VendorResponse, len(vendors))
    for i, v := range vendors {
        result[i] = &dto.VendorResponse{
            ID: v.ID, Name: v.Name, Code: v.Code, VendorType: v.VendorType,
            ContactName: v.ContactName, ContactEmail: v.ContactEmail,
            ContactPhone: v.ContactPhone, Address: v.Address, Website: v.Website,
            Rating: v.Rating, Status: v.Status, TenantID: v.TenantID,
            CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
        }
    }
    return result, total, nil
}

func (s *VendorService) GetVendor(ctx context.Context, id int, tenantID int) (*dto.VendorResponse, error) {
    v, err := s.client.Vendor.Query().Where(vendor.ID(id), vendor.TenantID(tenantID)).First(ctx)
    if err != nil {
        return nil, errors.New("vendor not found")
    }
    return &dto.VendorResponse{
        ID: v.ID, Name: v.Name, Code: v.Code, VendorType: v.VendorType,
        ContactName: v.ContactName, ContactEmail: v.ContactEmail,
        ContactPhone: v.ContactPhone, Address: v.Address, Website: v.Website,
        Rating: v.Rating, Status: v.Status, TenantID: v.TenantID,
        CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
    }, nil
}

func (s *VendorService) DeleteVendor(ctx context.Context, id int, tenantID int) error {
    _, err := s.client.Vendor.Delete().Where(vendor.ID(id), vendor.TenantID(tenantID)).Exec(ctx)
    return err
}