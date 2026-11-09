package escalation_matrix

import "itsm-backend/dto"

// EscalationMatrixService 是本域对升级矩阵服务的窄契约，方法集等于本域所有
// 生产文件对 escalationMatrixService 字段的真实调用面。
//
// EscalationLevel / EscalationMatrix 已下沉到 dto 包，共享层以类型别名暴露，
// 因此 GetMatrix / SetMatrix 的签名在本文件内可完整表达而无需引用共享层。
type EscalationMatrixService interface {
	GetMatrix(tenantID int) dto.EscalationMatrix
	SetMatrix(tenantID int, matrix dto.EscalationMatrix) error
	InvalidateCache(tenantID int)
}
