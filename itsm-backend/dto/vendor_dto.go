package dto

import "time"

type VendorResponse struct {
    ID            int       `json:"id"`
    Name          string    `json:"name"`
    Code          string    `json:"code"`
    VendorType    string    `json:"vendorType"`
    ContactName   string    `json:"contactName"`
    ContactEmail  string    `json:"contactEmail"`
    ContactPhone  string    `json:"contactPhone"`
    Address       string    `json:"address"`
    Website       string    `json:"website"`
    Rating        float64   `json:"rating"`
    Status        string    `json:"status"`
    TenantID      int       `json:"tenantId"`
    CreatedAt     time.Time `json:"createdAt"`
    UpdatedAt     time.Time `json:"updatedAt"`
}

type CreateVendorRequest struct {
    Name         string `json:"name" binding:"required"`
    Code         string `json:"code" binding:"required"`
    VendorType   string `json:"vendorType"`
    ContactName  string `json:"contactName"`
    ContactEmail string `json:"contactEmail"`
    ContactPhone string `json:"contactPhone"`
    Address      string `json:"address"`
    Website      string `json:"website"`
}

type ContractResponse struct {
    ID              int       `json:"id"`
    ContractNumber  string    `json:"contractNumber"`
    Title           string    `json:"title"`
    ContractType    string    `json:"contractType"`
    Value           float64   `json:"value"`
    StartDate       time.Time `json:"startDate"`
    EndDate         time.Time `json:"endDate"`
    Status          string    `json:"status"`
    Description     string    `json:"description"`
    VendorID        int       `json:"vendorId"`
    TenantID        int       `json:"tenantId"`
    CreatedAt       time.Time `json:"createdAt"`
    UpdatedAt       time.Time `json:"updatedAt"`
}