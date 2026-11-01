package cloudsignal

import "time"

// AlertResponse 是云监控告警的对外 DTO。Annotations 里的 bridgeStatus/incidentId
// 供页面展示建单状态，不包含任何凭据。
type AlertResponse struct {
	ID             int64             `json:"id"`
	AlertID        string            `json:"alertId"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Severity       string            `json:"severity"`
	Status         string            `json:"status"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	SourceIP       string            `json:"sourceIp,omitempty"`
	Service        string            `json:"service,omitempty"`
	FiredAt        time.Time         `json:"firedAt"`
	AcknowledgedAt *time.Time        `json:"acknowledgedAt,omitempty"`
	ResolvedAt     *time.Time        `json:"resolvedAt,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

// FindingResponse 是安全漏洞的对外 DTO。RawPayload 是云厂商原始报文，
// 只留在库里供排障，不通过接口下发。
type FindingResponse struct {
	ID             int64      `json:"id"`
	CloudAccountID int64      `json:"cloudAccountId"`
	Provider       string     `json:"provider"`
	VulnType       string     `json:"vulnType"`
	FindingKey     string     `json:"findingKey"`
	Name           string     `json:"name"`
	AliasName      string     `json:"aliasName,omitempty"`
	Severity       string     `json:"severity"`
	Status         string     `json:"status"`
	Necessity      string     `json:"necessity,omitempty"`
	InstanceID     string     `json:"instanceId,omitempty"`
	InstanceName   string     `json:"instanceName,omitempty"`
	UUID           string     `json:"uuid,omitempty"`
	InternetIP     string     `json:"internetIp,omitempty"`
	IntranetIP     string     `json:"intranetIp,omitempty"`
	Region         string     `json:"region,omitempty"`
	CIID           *int64     `json:"ciId,omitempty"`
	CINumber       string     `json:"ciNumber,omitempty"`
	CIName         string     `json:"ciName,omitempty"`
	CVSSScore      *float64   `json:"cvssScore,omitempty"`
	FirstSeenAt    *time.Time `json:"firstSeenAt,omitempty"`
	LastSeenAt     *time.Time `json:"lastSeenAt,omitempty"`
	LastSyncedAt   time.Time  `json:"lastSyncedAt"`
	ResolvedAt     *time.Time `json:"resolvedAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// SyncStateResponse 说明「哪个账号的哪种信号现在是什么状态」，
// 页面据此区分未配置、不可用和就绪，而不是把空列表当成同步正常。
type SyncStateResponse struct {
	CloudAccountID      int64      `json:"cloudAccountId"`
	AccountName         string     `json:"accountName"`
	Provider            string     `json:"provider"`
	AccountActive       bool       `json:"accountActive"`
	SignalKind          string     `json:"signalKind"`
	State               string     `json:"state"`
	LastAttemptAt       *time.Time `json:"lastAttemptAt,omitempty"`
	LastSuccessAt       *time.Time `json:"lastSuccessAt,omitempty"`
	Watermark           *time.Time `json:"watermark,omitempty"`
	LastErrorCode       string     `json:"lastErrorCode,omitempty"`
	LastErrorMessage    string     `json:"lastErrorMessage,omitempty"`
	ItemsSynced         int64      `json:"itemsSynced"`
	ConsecutiveFailures int        `json:"consecutiveFailures"`
	UpdatedAt           time.Time  `json:"updatedAt"`
}

// AlertListQuery 是告警列表的过滤参数。status/severity 支持重复参数多选。
type AlertListQuery struct {
	Statuses       []string `form:"status"`
	Severities     []string `form:"severity"`
	CloudAccountID int64    `form:"cloudAccountId"`
	Search         string   `form:"search"`
}

// FindingListQuery 是漏洞列表的过滤参数。
type FindingListQuery struct {
	Statuses       []string `form:"status"`
	Severities     []string `form:"severity"`
	VulnType       string   `form:"vulnType"`
	InstanceID     string   `form:"instanceId"`
	CloudAccountID int64    `form:"cloudAccountId"`
	Search         string   `form:"search"`
}

func toAlertDTO(record *AlertRecord) *AlertResponse {
	if record == nil {
		return nil
	}
	return &AlertResponse{
		ID:             record.ID,
		AlertID:        record.AlertID,
		Name:           record.Name,
		Description:    record.Description,
		Severity:       record.Severity,
		Status:         record.Status,
		Labels:         record.Labels,
		Annotations:    record.Annotations,
		SourceIP:       record.SourceIP,
		Service:        record.Service,
		FiredAt:        record.FiredAt,
		AcknowledgedAt: record.AcknowledgedAt,
		ResolvedAt:     record.ResolvedAt,
		CreatedAt:      record.CreatedAt,
		UpdatedAt:      record.UpdatedAt,
	}
}

func toAlertDTOList(records []*AlertRecord) []*AlertResponse {
	items := make([]*AlertResponse, 0, len(records))
	for _, record := range records {
		items = append(items, toAlertDTO(record))
	}
	return items
}

func toFindingDTO(finding *SecurityFinding) *FindingResponse {
	if finding == nil {
		return nil
	}
	return &FindingResponse{
		ID:             finding.ID,
		CloudAccountID: finding.CloudAccountID,
		Provider:       finding.Provider,
		VulnType:       finding.VulnType,
		FindingKey:     finding.FindingKey,
		Name:           finding.Name,
		AliasName:      finding.AliasName,
		Severity:       finding.Severity,
		Status:         finding.Status,
		Necessity:      finding.Necessity,
		InstanceID:     finding.InstanceID,
		InstanceName:   finding.InstanceName,
		UUID:           finding.UUID,
		InternetIP:     finding.InternetIP,
		IntranetIP:     finding.IntranetIP,
		Region:         finding.Region,
		CIID:           finding.CIID,
		CINumber:       finding.CINumber,
		CIName:         finding.CIName,
		CVSSScore:      finding.CVSSScore,
		FirstSeenAt:    finding.FirstSeenAt,
		LastSeenAt:     finding.LastSeenAt,
		LastSyncedAt:   finding.LastSyncedAt,
		ResolvedAt:     finding.ResolvedAt,
		CreatedAt:      finding.CreatedAt,
		UpdatedAt:      finding.UpdatedAt,
	}
}

func toFindingDTOList(findings []*SecurityFinding) []*FindingResponse {
	items := make([]*FindingResponse, 0, len(findings))
	for _, finding := range findings {
		items = append(items, toFindingDTO(finding))
	}
	return items
}

func toSyncStateDTOList(states []*SyncState) []*SyncStateResponse {
	items := make([]*SyncStateResponse, 0, len(states))
	for _, state := range states {
		if state == nil {
			continue
		}
		items = append(items, &SyncStateResponse{
			CloudAccountID:      state.CloudAccountID,
			AccountName:         state.AccountName,
			Provider:            state.Provider,
			AccountActive:       state.AccountActive,
			SignalKind:          string(state.SignalKind),
			State:               state.State,
			LastAttemptAt:       state.LastAttemptAt,
			LastSuccessAt:       state.LastSuccessAt,
			Watermark:           state.Watermark,
			LastErrorCode:       state.LastErrorCode,
			LastErrorMessage:    state.LastErrorMessage,
			ItemsSynced:         state.ItemsSynced,
			ConsecutiveFailures: state.ConsecutiveFailures,
			UpdatedAt:           state.UpdatedAt,
		})
	}
	return items
}
