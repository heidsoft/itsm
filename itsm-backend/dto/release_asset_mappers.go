package dto

import (
	"time"

	"itsm-backend/ent"
)

// intToPtr converts int to *int
func intToPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// timeToPtr converts time.Time to *time.Time
func timeToPtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// float64ToPtr converts float64 to *float64
func float64ToPtr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

// ===================================
// Release Mappers
// ===================================

// ToReleaseResponse converts an ent.Release to ReleaseResponse
func ToReleaseResponse(release *ent.Release) *ReleaseResponse {
	if release == nil {
		return nil
	}
	return &ReleaseResponse{
		ID:                 release.ID,
		ReleaseNumber:      release.ReleaseNumber,
		Title:              release.Title,
		Description:        release.Description,
		Type:               release.Type,
		Status:             release.Status,
		Severity:           release.Severity,
		Environment:        release.Environment,
		ChangeID:           release.ChangeID,
		OwnerID:            release.OwnerID,
		CreatedBy:          release.CreatedBy,
		TenantID:           release.TenantID,
		PlannedReleaseDate: timeToPtr(release.PlannedReleaseDate),
		ActualReleaseDate:  timeToPtr(release.ActualReleaseDate),
		PlannedStartDate:   timeToPtr(release.PlannedStartDate),
		PlannedEndDate:     timeToPtr(release.PlannedEndDate),
		ReleaseNotes:       release.ReleaseNotes,
		RollbackProcedure:  release.RollbackProcedure,
		ValidationCriteria: release.ValidationCriteria,
		AffectedSystems:    release.AffectedSystems,
		AffectedComponents: release.AffectedComponents,
		DeploymentSteps:    release.DeploymentSteps,
		Tags:               release.Tags,
		IsEmergency:        release.IsEmergency,
		RequiresApproval:   release.RequiresApproval,
		CreatedAt:          release.CreatedAt,
		UpdatedAt:          release.UpdatedAt,
	}
}

// ToReleaseResponseList converts a slice of ent.Release to ReleaseResponse slice
func ToReleaseResponseList(releases []*ent.Release) []ReleaseResponse {
	if releases == nil {
		return nil
	}
	responses := make([]ReleaseResponse, 0, len(releases))
	for _, r := range releases {
		if r != nil {
			resp := ToReleaseResponse(r)
			if resp != nil {
				responses = append(responses, *resp)
			}
		}
	}
	return responses
}

// ===================================
// Asset Mappers
// ===================================

// ToAssetResponse converts an ent.Asset to AssetResponse
func ToAssetResponse(asset *ent.Asset) *AssetResponse {
	if asset == nil {
		return nil
	}
	return &AssetResponse{
		ID:             asset.ID,
		AssetNumber:    asset.AssetNumber,
		Name:           asset.Name,
		Description:    asset.Description,
		Type:           asset.Type,
		Status:         asset.Status,
		Category:       asset.Category,
		Subcategory:    asset.Subcategory,
		TenantID:       asset.TenantID,
		CIID:           intToPtr(asset.CiID),
		AssignedTo:     intToPtr(asset.AssignedTo),
		LocationID:     intToPtr(asset.LocationID),
		SerialNumber:   asset.SerialNumber,
		Model:          asset.Model,
		Manufacturer:   asset.Manufacturer,
		Vendor:         asset.Vendor,
		PurchaseDate:   asset.PurchaseDate,
		PurchasePrice:  float64ToPtr(asset.PurchasePrice),
		WarrantyExpiry: asset.WarrantyExpiry,
		SupportExpiry:  asset.SupportExpiry,
		Location:       asset.Location,
		Department:     asset.Department,
		ParentAssetID:  intToPtr(asset.ParentAssetID),
		Specifications: asset.Specifications,
		CustomFields:   asset.CustomFields,
		Tags:           asset.Tags,
		CreatedAt:      asset.CreatedAt,
		UpdatedAt:      asset.UpdatedAt,
	}
}

// ToAssetResponseList converts a slice of ent.Asset to AssetResponse slice
func ToAssetResponseList(assets []*ent.Asset) []AssetResponse {
	if assets == nil {
		return nil
	}
	responses := make([]AssetResponse, 0, len(assets))
	for _, a := range assets {
		if a != nil {
			resp := ToAssetResponse(a)
			if resp != nil {
				responses = append(responses, *resp)
			}
		}
	}
	return responses
}

// ===================================
// AssetLicense Mappers
// ===================================

// ToLicenseResponse converts an ent.AssetLicense to LicenseResponse
func ToLicenseResponse(license *ent.AssetLicense) *LicenseResponse {
	if license == nil {
		return nil
	}
	return &LicenseResponse{
		ID:                license.ID,
		LicenseKey:        license.LicenseKey,
		Name:              license.Name,
		Description:       license.Description,
		Vendor:            license.Vendor,
		LicenseType:       license.LicenseType,
		TotalQuantity:     license.TotalQuantity,
		UsedQuantity:      license.UsedQuantity,
		AvailableQuantity: license.TotalQuantity - license.UsedQuantity,
		TenantID:          license.TenantID,
		AssetID:           intToPtr(license.AssetID),
		PurchaseDate:      license.PurchaseDate,
		PurchasePrice:     float64ToPtr(license.PurchasePrice),
		ExpiryDate:        license.ExpiryDate,
		SupportVendor:     license.SupportVendor,
		SupportContact:    license.SupportContact,
		RenewalCost:       license.RenewalCost,
		Status:            license.Status,
		Notes:             license.Notes,
		Users:             license.Users,
		Tags:              license.Tags,
		CreatedAt:         license.CreatedAt,
		UpdatedAt:         license.UpdatedAt,
	}
}

// ToLicenseResponseList converts a slice of ent.AssetLicense to LicenseResponse slice
func ToLicenseResponseList(licenses []*ent.AssetLicense) []LicenseResponse {
	if licenses == nil {
		return nil
	}
	responses := make([]LicenseResponse, 0, len(licenses))
	for _, l := range licenses {
		if l != nil {
			resp := ToLicenseResponse(l)
			if resp != nil {
				responses = append(responses, *resp)
			}
		}
	}
	return responses
}
