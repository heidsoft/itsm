package servicerequest

// Status values are shared by the service-request domain and its provisioning consumer.
const (
	StatusSubmitted        = "submitted"
	StatusManagerApproved  = "manager_approved"
	StatusITApproved       = "it_approved"
	StatusSecurityApproved = "security_approved"
	StatusRejected         = "rejected"
	StatusProvisioning     = "provisioning"
	StatusDelivered        = "delivered"
	StatusFailed           = "failed"
	StatusCancelled        = "cancelled"
)
