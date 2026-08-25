package email_intake

import (
	"time"

	"itsm-backend/ent"
)

type customerResponse struct {
	ID                     int       `json:"id"`
	Name                   string    `json:"name"`
	NormalizedName         string    `json:"normalizedName"`
	ShortName              string    `json:"shortName"`
	Aliases                []string  `json:"aliases"`
	HistoricalNames        []string  `json:"historicalNames"`
	Status                 string    `json:"status"`
	LinkedCustomerTenantID *int      `json:"linkedCustomerTenantId,omitempty"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

func mapCustomer(v *ent.ServiceCustomer) customerResponse {
	return customerResponse{ID: v.ID, Name: v.Name, NormalizedName: v.NormalizedName, ShortName: v.ShortName, Aliases: v.Aliases, HistoricalNames: v.HistoricalNames, Status: v.Status, LinkedCustomerTenantID: v.LinkedCustomerTenantID, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

type branchResponse struct {
	ID             int       `json:"id"`
	CustomerID     int       `json:"customerId"`
	Name           string    `json:"name"`
	NormalizedName string    `json:"normalizedName"`
	Aliases        []string  `json:"aliases"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func mapBranch(v *ent.CustomerBranch) branchResponse {
	return branchResponse{ID: v.ID, CustomerID: v.CustomerID, Name: v.Name, NormalizedName: v.NormalizedName, Aliases: v.Aliases, Status: v.Status, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

type sourceOrganizationResponse struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	NormalizedName string    `json:"normalizedName"`
	EmailAddresses []string  `json:"emailAddresses"`
	EmailDomains   []string  `json:"emailDomains"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func mapSourceOrganization(v *ent.SourceOrganization) sourceOrganizationResponse {
	return sourceOrganizationResponse{ID: v.ID, Name: v.Name, NormalizedName: v.NormalizedName, EmailAddresses: v.EmailAddresses, EmailDomains: v.EmailDomains, Status: v.Status, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

type supportContractResponse struct {
	ID             int        `json:"id"`
	CustomerID     int        `json:"customerId"`
	BranchID       *int       `json:"branchId,omitempty"`
	ContractNumber string     `json:"contractNumber"`
	Status         string     `json:"status"`
	StartAt        *time.Time `json:"startAt,omitempty"`
	EndAt          *time.Time `json:"endAt,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

func mapSupportContract(v *ent.SupportContract) supportContractResponse {
	return supportContractResponse{ID: v.ID, CustomerID: v.CustomerID, BranchID: v.BranchID, ContractNumber: v.ContractNumber, Status: v.Status, StartAt: v.StartAt, EndAt: v.EndAt, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

type externalReferenceResponse struct {
	ID                     int       `json:"id"`
	SourceOrganizationID   int       `json:"sourceOrganizationId"`
	SupportContractID      int       `json:"supportContractId"`
	CustomerID             int       `json:"customerId"`
	BranchID               *int      `json:"branchId,omitempty"`
	ExternalContractNumber string    `json:"externalContractNumber"`
	CreatedAt              time.Time `json:"createdAt"`
}

func mapExternalReference(v *ent.ExternalContractReference) externalReferenceResponse {
	return externalReferenceResponse{ID: v.ID, SourceOrganizationID: v.SourceOrganizationID, SupportContractID: v.SupportContractID, CustomerID: v.CustomerID, BranchID: v.BranchID, ExternalContractNumber: v.ExternalContractNumber, CreatedAt: v.CreatedAt}
}

type scheduleResponse struct {
	ID        int       `json:"id"`
	GroupID   int       `json:"groupId"`
	Name      string    `json:"name"`
	Timezone  string    `json:"timezone"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func mapSchedule(v *ent.OnCallSchedule) scheduleResponse {
	return scheduleResponse{ID: v.ID, GroupID: v.GroupID, Name: v.Name, Timezone: v.Timezone, Status: v.Status, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

type shiftResponse struct {
	ID         int       `json:"id"`
	ScheduleID int       `json:"scheduleId"`
	UserID     int       `json:"userId"`
	StartAt    time.Time `json:"startAt"`
	EndAt      time.Time `json:"endAt"`
}

type conversationResponse struct {
	ID                int       `json:"id"`
	Token             string    `json:"conversationToken"`
	Status            string    `json:"status"`
	CustomerID        *int      `json:"customerId,omitempty"`
	CustomerName      string    `json:"customerName,omitempty"`
	BranchID          *int      `json:"branchId,omitempty"`
	BranchName        string    `json:"branchName,omitempty"`
	SupportContractID *int      `json:"supportContractId,omitempty"`
	ContractNumber    string    `json:"contractNumber,omitempty"`
	IncidentID        *int      `json:"incidentId,omitempty"`
	IncidentNumber    string    `json:"incidentNumber,omitempty"`
	Confidence        float64   `json:"confidence"`
	MissingFields     []string  `json:"missingFields"`
	Version           int       `json:"version"`
	LastMessageAt     time.Time `json:"lastMessageAt"`
	CreatedAt         time.Time `json:"createdAt"`
}

func mapConversation(v *ent.EmailConversation) conversationResponse {
	result := conversationResponse{ID: v.ID, Token: v.ConversationToken, Status: v.Status, CustomerID: v.CustomerID, BranchID: v.BranchID, SupportContractID: v.SupportContractID, Confidence: v.Confidence, MissingFields: v.MissingFields, Version: v.Version, LastMessageAt: v.LastMessageAt, CreatedAt: v.CreatedAt}
	if v.Edges.Customer != nil {
		result.CustomerName = v.Edges.Customer.Name
	}
	if v.Edges.Branch != nil {
		result.BranchName = v.Edges.Branch.Name
	}
	if v.Edges.SupportContract != nil {
		result.ContractNumber = v.Edges.SupportContract.ContractNumber
	}
	if len(v.Edges.Incidents) > 0 {
		result.IncidentID = &v.Edges.Incidents[0].ID
		result.IncidentNumber = v.Edges.Incidents[0].IncidentNumber
	}
	return result
}

type inboundMessageResponse struct {
	ID                int       `json:"id"`
	ExternalMessageID string    `json:"externalMessageId"`
	FromAddress       string    `json:"fromAddress"`
	ToAddresses       []string  `json:"toAddresses"`
	Subject           string    `json:"subject"`
	PlainText         string    `json:"plainText"`
	SanitizedHTML     string    `json:"sanitizedHtml"`
	ProcessingStatus  string    `json:"processingStatus"`
	LastError         string    `json:"lastError,omitempty"`
	ReceivedAt        time.Time `json:"receivedAt"`
}
type analysisResponse struct {
	ID              int                    `json:"id"`
	Provider        string                 `json:"provider"`
	Model           string                 `json:"model"`
	PromptVersion   string                 `json:"promptVersion"`
	Result          map[string]interface{} `json:"result"`
	Confidence      float64                `json:"confidence"`
	Status          string                 `json:"status"`
	ValidationError string                 `json:"validationError,omitempty"`
	Corrections     map[string]interface{} `json:"corrections"`
	CreatedAt       time.Time              `json:"createdAt"`
}
type outboundResponse struct {
	ID        int        `json:"id"`
	ReplyType string     `json:"replyType"`
	Revision  int        `json:"revision"`
	ToAddress string     `json:"toAddress"`
	Subject   string     `json:"subject"`
	Status    string     `json:"status"`
	Attempts  int        `json:"attempts"`
	LastError string     `json:"lastError,omitempty"`
	SentAt    *time.Time `json:"sentAt,omitempty"`
}
type conversationDetailResponse struct {
	conversationResponse
	CanonicalData    map[string]interface{}   `json:"canonicalData"`
	FieldSources     map[string]interface{}   `json:"fieldSources"`
	Messages         []inboundMessageResponse `json:"messages"`
	Analyses         []analysisResponse       `json:"analyses"`
	OutboundMessages []outboundResponse       `json:"outboundMessages"`
}

func mapConversationDetail(v *ent.EmailConversation) conversationDetailResponse {
	result := conversationDetailResponse{conversationResponse: mapConversation(v), CanonicalData: v.CanonicalData, FieldSources: v.FieldSources}
	for _, message := range v.Edges.Messages {
		result.Messages = append(result.Messages, inboundMessageResponse{ID: message.ID, ExternalMessageID: message.ExternalMessageID, FromAddress: message.FromAddress, ToAddresses: message.ToAddresses, Subject: message.Subject, PlainText: message.PlainText, SanitizedHTML: message.SanitizedHTML, ProcessingStatus: message.ProcessingStatus, LastError: message.LastError, ReceivedAt: message.ReceivedAt})
	}
	for _, analysis := range v.Edges.Analyses {
		result.Analyses = append(result.Analyses, analysisResponse{ID: analysis.ID, Provider: analysis.Provider, Model: analysis.Model, PromptVersion: analysis.PromptVersion, Result: analysis.Result, Confidence: analysis.Confidence, Status: analysis.Status, ValidationError: analysis.ValidationError, Corrections: analysis.Corrections, CreatedAt: analysis.CreatedAt})
	}
	for _, outbound := range v.Edges.OutboundMessages {
		result.OutboundMessages = append(result.OutboundMessages, outboundResponse{ID: outbound.ID, ReplyType: outbound.ReplyType, Revision: outbound.Revision, ToAddress: outbound.ToAddress, Subject: outbound.Subject, Status: outbound.Status, Attempts: outbound.Attempts, LastError: outbound.LastError, SentAt: outbound.SentAt})
	}
	return result
}
