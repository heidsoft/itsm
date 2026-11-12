package service

import (
	"context"

	"itsm-backend/ent"
)

func createCMDBTestTenant(ctx context.Context, client *ent.Client, name, code, domain string) (*ent.Tenant, error) {
	return client.Tenant.Create().SetName(name).SetCode(code).SetDomain(domain).SetStatus("active").Save(ctx)
}

func createTestCIType(ctx context.Context, client *ent.Client, tenantID int, name string) (*ent.CIType, error) {
	return client.CIType.Create().SetName(name).SetTenantID(tenantID).Save(ctx)
}

func createTestCI(ctx context.Context, client *ent.Client, tenantID, ciTypeID int, name string) (*ent.ConfigurationItem, error) {
	return client.ConfigurationItem.Create().SetName(name).SetCiType("server").SetCiTypeID(ciTypeID).
		SetStatus("active").SetEnvironment("production").SetCriticality("high").SetTenantID(tenantID).Save(ctx)
}
