package tenantmode

const (
	DeploymentModePrivate = "private"
	DeploymentModeSaaS    = "saas"
	DeploymentModeSaaSMSP = "saas_msp"
)

const (
	TenantTypeStandard     = "standard"
	TenantTypeInternal     = "internal"
	TenantTypeSaaSCustomer = "saas_customer"
	TenantTypeMSPProvider  = "msp_provider"
	TenantTypeMSPCustomer  = "msp_customer"

	// Legacy aliases kept for backward compatibility with existing data/tests.
	TenantTypeLegacyMSP      = "msp"
	TenantTypeLegacyCustomer = "customer"
)

func IsMSPProviderTenantType(kind string) bool {
	return kind == TenantTypeMSPProvider || kind == TenantTypeLegacyMSP
}

func IsCustomerTenantType(kind string) bool {
	return kind == TenantTypeMSPCustomer || kind == TenantTypeSaaSCustomer || kind == TenantTypeLegacyCustomer
}

func IsInternalTenantType(kind string) bool {
	return kind == TenantTypeInternal || kind == TenantTypeStandard
}
