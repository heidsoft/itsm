package service

import (
	"context"
	"errors"
	"fmt"

	"itsm-backend/dto"
	"itsm-backend/ent"
	"itsm-backend/ent/vendor"

	"go.uber.org/zap"
)

// 供应商域的失败分类。改动前 ListVendors 用 `total, _ :=` / `vendors, _ :=` 吞掉
// Count 和 All 的错误后返回空列表，GetVendor 把所有错误（含数据库故障）压成
// "vendor not found"，DeleteVendor 对不存在的记录返回 nil —— 三种情况都把后端故障
// 或跨租户访问伪装成 200 成功。
var (
	// ErrVendorNotFound 表示本租户内不存在该供应商。查询都带租户谓词，
	// 所以跨租户访问与真实不存在同样映射为 not found（fail closed，不泄漏存在性）。
	ErrVendorNotFound = errors.New("vendor not found")
	// ErrVendorCodeExists 表示供应商编码已被占用。vendors.code 是数据库唯一索引，
	// 冲突由约束报错而不是前置查询产生。
	ErrVendorCodeExists = errors.New("vendor code already exists")
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
		// code 的唯一索引是全局的（不按 tenant 组合），所以其他租户占用的编码
		// 同样表现为约束冲突；租户维度的唯一键收敛属于 E5 vendors 归属裁决范围。
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("%w: code=%s tenant_id=%d: %w", ErrVendorCodeExists, req.Code, tenantID, err)
		}
		return nil, fmt.Errorf("create vendor: %w", err)
	}
	return toVendorResponse(v), nil
}

func (s *VendorService) ListVendors(ctx context.Context, tenantID int, page, size int) ([]*dto.VendorResponse, int, error) {
	query := s.client.Vendor.Query().Where(vendor.TenantID(tenantID))
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count vendors: %w", err)
	}
	vendors, err := query.Offset((page - 1) * size).Limit(size).All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list vendors: %w", err)
	}
	result := make([]*dto.VendorResponse, len(vendors))
	for i, v := range vendors {
		result[i] = toVendorResponse(v)
	}
	return result, total, nil
}

func (s *VendorService) GetVendor(ctx context.Context, id int, tenantID int) (*dto.VendorResponse, error) {
	v, err := s.client.Vendor.Query().Where(vendor.ID(id), vendor.TenantID(tenantID)).First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fmt.Errorf("%w: id=%d tenant_id=%d", ErrVendorNotFound, id, tenantID)
		}
		return nil, fmt.Errorf("get vendor: %w", err)
	}
	return toVendorResponse(v), nil
}

func (s *VendorService) DeleteVendor(ctx context.Context, id int, tenantID int) error {
	deleted, err := s.client.Vendor.Delete().Where(vendor.ID(id), vendor.TenantID(tenantID)).Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete vendor: %w", err)
	}
	// 条件删除影响 0 行意味着记录不属于本租户（或已被删除），不能报成功。
	if deleted == 0 {
		return fmt.Errorf("%w: id=%d tenant_id=%d", ErrVendorNotFound, id, tenantID)
	}
	return nil
}

func toVendorResponse(v *ent.Vendor) *dto.VendorResponse {
	return &dto.VendorResponse{
		ID: v.ID, Name: v.Name, Code: v.Code, VendorType: v.VendorType,
		ContactName: v.ContactName, ContactEmail: v.ContactEmail,
		ContactPhone: v.ContactPhone, Address: v.Address, Website: v.Website,
		Rating: v.Rating, Status: v.Status, TenantID: v.TenantID,
		CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt,
	}
}
