---
name: "backend-testing-guide"
description: "Comprehensive guide for backend testing including unit tests, integration tests, and test data management. Invoke when user needs to write tests, fix test failures, or improve test coverage."
---

# Backend Testing Guide

This skill provides comprehensive guidance for testing the ITSM backend services, including unit tests, integration tests, and test data management.

## Test Structure

### Unit Tests (`*_test.go`)
- Located alongside service files in `/itsm-backend/service/`
- Test individual service methods in isolation
- Use `enttest` for in-memory database testing
- Follow naming convention: `Test<ServiceName>_<MethodName>`

### Integration Tests (`tests/api/`)
- Located in `/tests/api/`
- Test complete API endpoints and workflows
- Use Python with pytest framework
- Cover end-to-end scenarios

### E2E Tests (`tests/e2e/`)
- Located in `/tests/e2e/`
- Test user workflows through the UI
- Use Playwright for browser automation
- Cover critical user journeys

## Test Data Management

### In-Memory Database Setup
```go
client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
defer client.Close()
```

### Creating Test Entities
```go
// Create tenant
testTenant, err := client.Tenant.Create().
    SetName("Test Tenant").
    SetCode("test").
    SetDomain("test.com").
    SetStatus("active").
    Save(ctx)

// Create user
testUser, err := client.User.Create().
    SetUsername("testuser").
    SetEmail("test@example.com").
    SetName("Test User").
    SetPasswordHash("hashedpassword").
    SetRole("end_user").
    SetActive(true).
    SetTenantID(testTenant.ID).
    Save(ctx)
```

### Service Initialization
```go
logger := zaptest.NewLogger(t).Sugar()
ticketService := NewTicketService(client, logger)
```

## Common Test Patterns

### Testing Create Operations
```go
func TestTicketService_CreateTicket(t *testing.T) {
    // Setup
    client := enttest.Open(t, "sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
    defer client.Close()
    
    // Create test data
    ctx := context.Background()
    testTenant := createTestTenant(t, client)
    testUser := createTestUser(t, client, testTenant.ID)
    
    // Test
    request := &dto.CreateTicketRequest{
        Title:       "Test Ticket",
        Description: "Test Description",
        Priority:    "medium",
        Type:        "incident",
        UserID:      testUser.ID,
        TenantID:    testTenant.ID,
    }
    
    response, err := ticketService.CreateTicket(ctx, request)
    
    // Assertions
    assert.NoError(t, err)
    assert.NotNil(t, response)
    assert.Equal(t, request.Title, response.Title)
}
```

### Testing Update Operations
```go
func TestTicketService_UpdateTicket(t *testing.T) {
    // Setup existing ticket
    existingTicket, err := client.Ticket.Create().
        SetTitle("Original Title").
        SetDescription("Original Description").
        SetPriority("low").
        SetTenantID(testTenant.ID).
        SetCreatorID(testUser.ID).
        Save(ctx)
    
    // Update
    request := &dto.UpdateTicketRequest{
        Title:    "Updated Title",
        Priority: "high",
        UserID:   testUser.ID,
    }
    
    updated, err := ticketService.UpdateTicket(ctx, existingTicket.ID, request)
    
    // Assertions
    assert.NoError(t, err)
    assert.Equal(t, "Updated Title", updated.Title)
    assert.Equal(t, "high", updated.Priority)
}
```

### Testing Error Conditions
```go
func TestTicketService_UpdateTicket_NotFound(t *testing.T) {
    request := &dto.UpdateTicketRequest{
        Title:  "Updated Title",
        UserID: testUser.ID,
    }
    
    // Try to update non-existent ticket
    _, err := ticketService.UpdateTicket(ctx, 99999, request)
    
    // Should return error
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not found")
}
```

## Test Utilities

### Helper Functions
```go
func createTestTenant(t *testing.T, client *ent.Client) *ent.Tenant {
    t.Helper()
    tenant, err := client.Tenant.Create().
        SetName("Test Tenant").
        SetCode("test").
        SetDomain("test.com").
        SetStatus("active").
        Save(context.Background())
    require.NoError(t, err)
    return tenant
}

func createTestUser(t *testing.T, client *ent.Client, tenantID int) *ent.User {
    t.Helper()
    user, err := client.User.Create().
        SetUsername(fmt.Sprintf("testuser_%d", time.Now().Unix())).
        SetEmail(fmt.Sprintf("test_%d@example.com", time.Now().Unix())).
        SetName("Test User").
        SetPasswordHash("hashedpassword").
        SetRole("end_user").
        SetActive(true).
        SetTenantID(tenantID).
        Save(context.Background())
    require.NoError(t, err)
    return user
}
```

## Running Tests

### Backend Unit Tests
```bash
cd itsm-backend
go test ./service -v                    # Run all service tests
go test ./service -run TestTicketService -v  # Run specific test
go test ./service -v -cover           # With coverage
```

### API Integration Tests
```bash
cd tests/api
python -m pytest test_api_cases.py -v
```

### E2E Tests
```bash
cd tests/e2e
python -m pytest test_navigation.py -v
```

## Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| FOREIGN KEY constraint failed | Create required related entities (users, tenants) before tests |
| nil pointer dereference | Use factory functions for service initialization |
| SetID undefined | Remove manual ID setting, let Ent auto-generate |
| undefined: ent.UserIDEQ | Use `user.ID()` from imported entity package |
| Category not in Ticket entity | Relations are handled via edges, not direct fields |

## Best Practices

1. **Isolation**: Each test should be independent and not rely on test execution order
2. **Cleanup**: Use `defer` to clean up resources (database connections, etc.)
3. **Assertions**: Use `testify/assert` for clear, readable assertions
4. **Error Handling**: Test both success and error cases
5. **Test Data**: Create unique test data to avoid conflicts between tests
6. **Coverage**: Aim for high test coverage but focus on critical business logic