package cloud

// BaseResource 所有资源共有的字段
type BaseResource struct {
	ResourceID   string            `json:"resource_id"`
	ResourceName string            `json:"resource_name"`
	Region       string            `json:"region"`
	Zone         string            `json:"zone,omitempty"`
	Status       string            `json:"status"`
	Tags         map[string]string `json:"tags,omitempty"`
	CreatedTime  string            `json:"created_time,omitempty"`
}

// DiscoveredResource 发现层输出的统一资源类型
type DiscoveredResource struct {
	BaseResource
	CloudServiceCode string                 `json:"cloud_service_code"`
	CloudServiceName string                 `json:"cloud_service_name"`
	Extra            map[string]interface{} `json:"extra,omitempty"`
}

// DiscoveryWarning 发现过程中的非致命问题
type DiscoveryWarning struct {
	Region string `json:"region,omitempty"`
	Code   string `json:"code"`
	Msg    string `json:"message"`
}

// PageResult 单次分页/单 Region 的发现结果
type PageResult struct {
	Resources []DiscoveredResource `json:"resources"`
	Warnings  []DiscoveryWarning   `json:"warnings,omitempty"`
	NextToken string               `json:"next_token,omitempty"`
}

// ReconciliationResult 对账结果
type ReconciliationResult struct {
	Created   []int `json:"created"`
	Updated   []int `json:"updated"`
	Retired   []int `json:"retired"`
	Conflicts []int `json:"conflicts"`
}

// ReconcilePolicy 对账策略
type ReconcilePolicy string

const (
	ReconcileDiscoveredWins ReconcilePolicy = "discovered_wins"
	ReconcileCMDBWins       ReconcilePolicy = "cmdb_wins"
	ReconcileManual         ReconcilePolicy = "manual"
)
