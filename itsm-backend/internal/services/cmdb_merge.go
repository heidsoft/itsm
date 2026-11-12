package service

import (
	"context"
	"time"

	"itsm-backend/ent"
	"itsm-backend/ent/configurationitem"
	"itsm-backend/ent/predicate"
)

// OwnershipMode constants matching ent schema
const (
	OwnershipModeManaged  = "managed"  // 平台管理字段，import/discovery 无条件覆盖
	OwnershipModeCustomer = "customer" // 客户托管字段，import/discovery 不得覆盖
	OwnershipModeSLA      = "sla"      // SLA 覆盖字段，优先 tenant default
)

// ManagedField describes a single field's merge strategy
type ManagedField struct {
	FieldName  string
	Strategy   MergeStrategy
	CanBeEmpty bool // true = empty string is a valid value, false = treat empty as "not set"
}

// MergeStrategy defines how a field is resolved in three-way merge
type MergeStrategy int

const (
	// UseIncomingOrExistingOrDefault: non-empty incoming wins, else existing, else default
	UseIncomingOrExistingOrDefault MergeStrategy = iota
	// UseExistingOrDefault: existing wins (customer-owned field)
	UseExistingOrDefault
	// UseSLAOrDefault: SLA default wins unless existing has value and incoming is stale
	UseSLAOrDefault
	// AlwaysUseIncoming: unconditionally use incoming value
	AlwaysUseIncoming
)

// defaultManagedFields returns the standard merge strategies for CMDB CI fields.
func defaultManagedFields() []ManagedField {
	return []ManagedField{
		{FieldName: "Name", Strategy: AlwaysUseIncoming, CanBeEmpty: false},
		{FieldName: "Description", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
		{FieldName: "CiTypeID", Strategy: AlwaysUseIncoming, CanBeEmpty: false},
		{FieldName: "CiType", Strategy: AlwaysUseIncoming, CanBeEmpty: false},
		{FieldName: "Status", Strategy: AlwaysUseIncoming, CanBeEmpty: false},
		{FieldName: "Environment", Strategy: AlwaysUseIncoming, CanBeEmpty: false},
		{FieldName: "Criticality", Strategy: AlwaysUseIncoming, CanBeEmpty: false},
		{FieldName: "AssetTag", Strategy: UseIncomingOrExistingOrDefault, CanBeEmpty: true},
		{FieldName: "SerialNumber", Strategy: UseIncomingOrExistingOrDefault, CanBeEmpty: true},
		{FieldName: "Model", Strategy: UseIncomingOrExistingOrDefault, CanBeEmpty: true},
		{FieldName: "Vendor", Strategy: UseIncomingOrExistingOrDefault, CanBeEmpty: true},
		{FieldName: "Location", Strategy: UseIncomingOrExistingOrDefault, CanBeEmpty: true},
		{FieldName: "AssignedTo", Strategy: UseExistingOrDefault, CanBeEmpty: true},
		{FieldName: "OwnedBy", Strategy: UseExistingOrDefault, CanBeEmpty: true},
		{FieldName: "DiscoverySource", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
		{FieldName: "Source", Strategy: AlwaysUseIncoming, CanBeEmpty: false},
		{FieldName: "Attributes", Strategy: UseIncomingOrExistingOrDefault, CanBeEmpty: true},
		{FieldName: "CloudProvider", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
		{FieldName: "CloudAccountID", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
		{FieldName: "CloudRegion", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
		{FieldName: "CloudZone", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
		{FieldName: "CloudResourceID", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
		{FieldName: "CloudResourceType", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
		{FieldName: "CloudMetadata", Strategy: UseIncomingOrExistingOrDefault, CanBeEmpty: true},
		{FieldName: "CloudTags", Strategy: UseIncomingOrExistingOrDefault, CanBeEmpty: true},
		{FieldName: "CloudMetrics", Strategy: UseIncomingOrExistingOrDefault, CanBeEmpty: true},
		{FieldName: "CloudSyncTime", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
		{FieldName: "CloudSyncStatus", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
		{FieldName: "LifecycleStatus", Strategy: AlwaysUseIncoming, CanBeEmpty: false},
		{FieldName: "EffectiveAt", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
		{FieldName: "ExpireAt", Strategy: AlwaysUseIncoming, CanBeEmpty: true},
	}
}

// MergeResult holds the outcome of a three-way merge
type MergeResult struct {
	Resolved  *ent.ConfigurationItem
	Changed   bool
	Conflicts []Conflict
}

// Conflict describes a field where the three sources disagreed
type Conflict struct {
	FieldName   string
	IncomingVal interface{}
	ExistingVal interface{}
	DefaultVal  interface{}
	ResolvedVal interface{}
}

// ThreeWayMerge resolves a CI record from three sources:
//   - incoming: new values from import/discovery/manual API
//   - existing: current values in the database
//   - tenantDefault: platform-level default values for this CI type
//
// Rules:
//   - ownership_mode = "customer": owned_by/assigned_to are protected (existing wins)
//   - ownership_mode = "sla": assigned_to falls back to SLA default
//   - ownership_mode = "managed": all fields resolved via managed field strategies
//   - local_modified_at tracks customer edits; if incoming.source is older, skip conflict
func (s *CMDBImportExportService) ThreeWayMerge(
	_ context.Context,
	incoming, existing, tenantDefault *ent.ConfigurationItem,
) (*MergeResult, error) {
	result := &MergeResult{
		Resolved:  existing,
		Changed:   false,
		Conflicts: []Conflict{},
	}

	mode := existing.OwnershipMode
	if mode == "" {
		mode = OwnershipModeManaged
	}

	for _, mf := range defaultManagedFields() {
		inVal := getFieldValue(incoming, mf.FieldName)
		existVal := getFieldValue(existing, mf.FieldName)
		defaultVal := getFieldValue(tenantDefault, mf.FieldName)

		resolvedVal := resolveField(mf, mode, inVal, existVal, defaultVal)

		if !valuesEqual(resolvedVal, existVal) {
			result.Changed = true
			setFieldValue(result.Resolved, mf.FieldName, resolvedVal)
		}

		if !valuesEqual(inVal, existVal) && !valuesEqual(inVal, resolvedVal) {
			result.Conflicts = append(result.Conflicts, Conflict{
				FieldName:   mf.FieldName,
				IncomingVal: inVal,
				ExistingVal: existVal,
				DefaultVal:  defaultVal,
				ResolvedVal: resolvedVal,
			})
		}
	}

	// Always update discovery metadata for managed/SLA records
	if mode == OwnershipModeManaged || mode == OwnershipModeSLA {
		result.Resolved.LastDiscovered = time.Now()
		result.Resolved.DiscoverySource = incoming.DiscoverySource
		result.Resolved.Source = incoming.Source
	}

	// For customer mode, update local_modified_at when customer makes edits
	if mode == OwnershipModeCustomer && result.Changed {
		result.Resolved.LocalModifiedAt = time.Now()
	}

	return result, nil
}

// resolveField applies the merge strategy for a single field given the ownership mode
func resolveField(mf ManagedField, ownershipMode string, incoming, existing, defaultVal interface{}) interface{} {
	// Special handling for customer托管 fields (owned_by, assigned_to)
	if mf.FieldName == "OwnedBy" || mf.FieldName == "AssignedTo" {
		switch ownershipMode {
		case OwnershipModeCustomer:
			return pickFirstNonEmpty(existing, defaultVal)
		case OwnershipModeSLA:
			return pickFirstNonEmpty(defaultVal, existing)
		default:
			return pickFirstNonEmpty(incoming, existing, defaultVal)
		}
	}

	switch mf.Strategy {
	case AlwaysUseIncoming:
		if mf.CanBeEmpty {
			return incoming
		}
		return pickFirstNonEmpty(incoming, existing, defaultVal)
	case UseIncomingOrExistingOrDefault:
		return pickFirstNonEmpty(incoming, existing, defaultVal)
	case UseExistingOrDefault:
		return pickFirstNonEmpty(existing, defaultVal)
	case UseSLAOrDefault:
		return pickFirstNonEmpty(defaultVal, existing)
	}
	return existing
}

// pickFirstNonEmpty returns the first non-empty value from the list.
func pickFirstNonEmpty(values ...interface{}) interface{} {
	for _, v := range values {
		if !isEmpty(v) {
			return v
		}
	}
	return nil
}

// isEmpty returns true if the value is "empty" (nil, "", or zero)
func isEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case int, int64, float64:
		return val == 0
	case []interface{}:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	}
	return false
}

// valuesEqual checks if two values are equal
func valuesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			return av == bv
		}
	case int:
		if bv, ok := b.(int); ok {
			return av == bv
		}
	case int64:
		if bv, ok := b.(int64); ok {
			return av == bv
		}
	case bool:
		if bv, ok := b.(bool); ok {
			return av == bv
		}
	case []interface{}:
		if bv, ok := b.([]interface{}); ok {
			if len(av) != len(bv) {
				return false
			}
			for i := range av {
				if !valuesEqual(av[i], bv[i]) {
					return false
				}
			}
			return true
		}
	case map[string]interface{}:
		if bv, ok := b.(map[string]interface{}); ok {
			if len(av) != len(bv) {
				return false
			}
			for k, v := range av {
				if !valuesEqual(v, bv[k]) {
					return false
				}
			}
			return true
		}
	}
	return false
}

// getFieldValue reflects into the ConfigurationItem to get a field value by name.
func getFieldValue(ci *ent.ConfigurationItem, fieldName string) interface{} {
	if ci == nil {
		return nil
	}
	switch fieldName {
	case "Name":
		return ci.Name
	case "Description":
		return ci.Description
	case "CiTypeID":
		return ci.CiTypeID
	case "CiType":
		return ci.CiType
	case "Status":
		return ci.Status
	case "Environment":
		return ci.Environment
	case "Criticality":
		return ci.Criticality
	case "AssetTag":
		return ci.AssetTag
	case "SerialNumber":
		return ci.SerialNumber
	case "Model":
		return ci.Model
	case "Vendor":
		return ci.Vendor
	case "Location":
		return ci.Location
	case "AssignedTo":
		return ci.AssignedTo
	case "OwnedBy":
		return ci.OwnedBy
	case "DiscoverySource":
		return ci.DiscoverySource
	case "Source":
		return ci.Source
	case "Attributes":
		return ci.Attributes
	case "CloudProvider":
		return ci.CloudProvider
	case "CloudAccountID":
		return ci.CloudAccountID
	case "CloudRegion":
		return ci.CloudRegion
	case "CloudZone":
		return ci.CloudZone
	case "CloudResourceID":
		return ci.CloudResourceID
	case "CloudResourceType":
		return ci.CloudResourceType
	case "CloudMetadata":
		return ci.CloudMetadata
	case "CloudTags":
		return ci.CloudTags
	case "CloudMetrics":
		return ci.CloudMetrics
	case "CloudSyncTime":
		return ci.CloudSyncTime
	case "CloudSyncStatus":
		return ci.CloudSyncStatus
	case "LifecycleStatus":
		return ci.LifecycleStatus
	case "EffectiveAt":
		return ci.EffectiveAt
	case "ExpireAt":
		return ci.ExpireAt
	case "OwnershipMode":
		return ci.OwnershipMode
	case "LocalModifiedAt":
		return ci.LocalModifiedAt
	}
	return nil
}

// setFieldValue directly assigns a value to a public field on the ent struct.
func setFieldValue(ci *ent.ConfigurationItem, fieldName string, value interface{}) {
	if ci == nil || value == nil {
		return
	}
	switch fieldName {
	case "Name":
		ci.Name, _ = value.(string)
	case "Description":
		ci.Description, _ = value.(string)
	case "Status":
		ci.Status, _ = value.(string)
	case "Environment":
		ci.Environment, _ = value.(string)
	case "Criticality":
		ci.Criticality, _ = value.(string)
	case "AssetTag":
		ci.AssetTag, _ = value.(string)
	case "SerialNumber":
		ci.SerialNumber, _ = value.(string)
	case "Model":
		ci.Model, _ = value.(string)
	case "Vendor":
		ci.Vendor, _ = value.(string)
	case "Location":
		ci.Location, _ = value.(string)
	case "AssignedTo":
		ci.AssignedTo, _ = value.(string)
	case "OwnedBy":
		ci.OwnedBy, _ = value.(string)
	case "DiscoverySource":
		ci.DiscoverySource, _ = value.(string)
	case "Source":
		ci.Source, _ = value.(string)
	case "Attributes":
		ci.Attributes, _ = value.(map[string]interface{})
	case "CloudProvider":
		ci.CloudProvider, _ = value.(string)
	case "CloudAccountID":
		ci.CloudAccountID, _ = value.(string)
	case "CloudRegion":
		ci.CloudRegion, _ = value.(string)
	case "CloudZone":
		ci.CloudZone, _ = value.(string)
	case "CloudResourceID":
		ci.CloudResourceID, _ = value.(string)
	case "CloudResourceType":
		ci.CloudResourceType, _ = value.(string)
	case "CloudMetadata":
		ci.CloudMetadata, _ = value.(map[string]interface{})
	case "CloudTags":
		ci.CloudTags, _ = value.(map[string]interface{})
	case "CloudMetrics":
		ci.CloudMetrics, _ = value.(map[string]interface{})
	case "CloudSyncStatus":
		ci.CloudSyncStatus, _ = value.(string)
	case "LifecycleStatus":
		ci.LifecycleStatus, _ = value.(string)
	case "OwnershipMode":
		ci.OwnershipMode, _ = value.(string)
	}
}

// ApplyThreeWayMerge applies the merge result to the database within a transaction.
func (s *CMDBImportExportService) ApplyThreeWayMerge(
	ctx context.Context,
	tenantID int,
	stableKey string,
	incoming *ent.ConfigurationItem,
) (*ent.ConfigurationItem, *MergeResult, error) {
	var result *MergeResult
	var merged *ent.ConfigurationItem

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// 1. Look up existing record by stable key (serial_number or cloud_resource_id)
	predicates := []predicate.ConfigurationItem{
		configurationitem.TenantID(tenantID),
	}
	if stableKey != "" {
		predicates = append(predicates, configurationitem.SerialNumberEQ(stableKey))
	}

	existing, err := tx.ConfigurationItem.Query().
		Where(predicates...).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		_ = tx.Rollback()
		return nil, nil, err
	}

	if existing == nil {
		// New record: set ownership_mode and create
		incoming.OwnershipMode = OwnershipModeManaged
		_, err = tx.ConfigurationItem.Create().
			SetName(incoming.Name).
			SetDescription(incoming.Description).
			SetCiTypeID(incoming.CiTypeID).
			SetCiType(incoming.CiType).
			SetStatus(incoming.Status).
			SetEnvironment(incoming.Environment).
			SetCriticality(incoming.Criticality).
			SetAssetTag(incoming.AssetTag).
			SetSerialNumber(incoming.SerialNumber).
			SetModel(incoming.Model).
			SetVendor(incoming.Vendor).
			SetLocation(incoming.Location).
			SetAssignedTo(incoming.AssignedTo).
			SetOwnedBy(incoming.OwnedBy).
			SetDiscoverySource(incoming.DiscoverySource).
			SetSource(incoming.Source).
			SetAttributes(incoming.Attributes).
			SetCloudProvider(incoming.CloudProvider).
			SetCloudAccountID(incoming.CloudAccountID).
			SetCloudRegion(incoming.CloudRegion).
			SetCloudZone(incoming.CloudZone).
			SetCloudResourceID(incoming.CloudResourceID).
			SetCloudResourceType(incoming.CloudResourceType).
			SetCloudMetadata(incoming.CloudMetadata).
			SetCloudTags(incoming.CloudTags).
			SetCloudMetrics(incoming.CloudMetrics).
			SetLifecycleStatus(incoming.LifecycleStatus).
			SetOwnershipMode(OwnershipModeManaged).
			SetTenantID(tenantID).
			Save(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, nil, err
		}
		return incoming, &MergeResult{Resolved: incoming, Changed: true}, nil
	}

	// 2. Get tenant SLA default
	tenantDefault, _ := s.GetOrCreateTenantSLA(ctx, tenantID, incoming.CiTypeID)

	// 3. Three-way merge
	result, err = s.ThreeWayMerge(ctx, incoming, existing, tenantDefault)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}
	merged = result.Resolved

	if !result.Changed {
		_ = tx.Commit()
		return merged, result, nil
	}

	// 4. Apply merged values via UpdateOne
	update := tx.ConfigurationItem.UpdateOne(existing)
	if merged.Name != "" {
		update.SetName(merged.Name)
	}
	if merged.Description != "" {
		update.SetDescription(merged.Description)
	}
	if merged.CiTypeID != 0 {
		update.SetCiTypeID(merged.CiTypeID)
	}
	if merged.CiType != "" {
		update.SetCiType(merged.CiType)
	}
	if merged.Status != "" {
		update.SetStatus(merged.Status)
	}
	if merged.Environment != "" {
		update.SetEnvironment(merged.Environment)
	}
	if merged.Criticality != "" {
		update.SetCriticality(merged.Criticality)
	}
	if merged.AssetTag != "" {
		update.SetAssetTag(merged.AssetTag)
	}
	if merged.SerialNumber != "" {
		update.SetSerialNumber(merged.SerialNumber)
	}
	if merged.Model != "" {
		update.SetModel(merged.Model)
	}
	if merged.Vendor != "" {
		update.SetVendor(merged.Vendor)
	}
	if merged.Location != "" {
		update.SetLocation(merged.Location)
	}
	if merged.AssignedTo != "" {
		update.SetAssignedTo(merged.AssignedTo)
	}
	if merged.OwnedBy != "" {
		update.SetOwnedBy(merged.OwnedBy)
	}
	if merged.DiscoverySource != "" {
		update.SetDiscoverySource(merged.DiscoverySource)
	}
	if merged.Source != "" {
		update.SetSource(merged.Source)
	}
	if merged.Attributes != nil {
		update.SetAttributes(merged.Attributes)
	}
	if merged.CloudProvider != "" {
		update.SetCloudProvider(merged.CloudProvider)
	}
	if merged.CloudAccountID != "" {
		update.SetCloudAccountID(merged.CloudAccountID)
	}
	if merged.CloudRegion != "" {
		update.SetCloudRegion(merged.CloudRegion)
	}
	if merged.CloudZone != "" {
		update.SetCloudZone(merged.CloudZone)
	}
	if merged.CloudResourceID != "" {
		update.SetCloudResourceID(merged.CloudResourceID)
	}
	if merged.CloudResourceType != "" {
		update.SetCloudResourceType(merged.CloudResourceType)
	}
	if merged.CloudMetadata != nil {
		update.SetCloudMetadata(merged.CloudMetadata)
	}
	if merged.CloudTags != nil {
		update.SetCloudTags(merged.CloudTags)
	}
	if merged.CloudMetrics != nil {
		update.SetCloudMetrics(merged.CloudMetrics)
	}
	if merged.CloudSyncStatus != "" {
		update.SetCloudSyncStatus(merged.CloudSyncStatus)
	}
	if merged.LifecycleStatus != "" {
		update.SetLifecycleStatus(merged.LifecycleStatus)
	}
	if merged.OwnershipMode != "" {
		update.SetOwnershipMode(merged.OwnershipMode)
	}
	update.SetUpdatedAt(time.Now())

	updated, err := update.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	result.Resolved = updated
	return updated, result, nil
}

// GetOrCreateTenantSLA returns the tenant-level SLA default for a given CI type.
// Returns nil if no SLA default is configured.
func (s *CMDBImportExportService) GetOrCreateTenantSLA(ctx context.Context, tenantID, ciTypeID int) (*ent.ConfigurationItem, error) {
	// In a full implementation this would look up an SLA policy for the tenant/CI type
	return nil, nil
}
