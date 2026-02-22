---
name: "api-integration-testing"
description: "Tests API integration between frontend and backend, validates request/response formats, handles authentication, and ensures API contract compliance. Invoke when user needs to test API endpoints, fix integration issues, or validate API functionality."
---

# API Integration Testing Guide

This skill provides comprehensive guidance for testing API integration between frontend and backend, validating request/response formats, handling authentication, and ensuring API contract compliance.

## API Structure Overview

### Backend API Organization
- **Base URL**: `/api/v1/`
- **Authentication**: JWT Bearer token in Authorization header
- **Content Type**: `application/json`
- **Error Format**: Standardized error response with code and message

### Key API Modules
1. **Authentication** (`/api/v1/auth/*`)
2. **Tickets** (`/api/v1/tickets/*`)
3. **Users** (`/api/v1/users/*`)
4. **Incidents** (`/api/v1/incidents/*`)
5. **SLA** (`/api/v1/sla/*`)
6. **CMDB** (`/api/v1/cmdb/*`)
7. **Knowledge Base** (`/api/v1/knowledge/*`)
8. **Workflow** (`/api/v1/workflow/*`)

## API Testing Framework

### Test File Structure
```
tests/api/
├── __init__.py
├── test_auth_api.py          # Authentication endpoints
├── test_tickets_api.py       # Ticket management
├── test_incidents_api.py     # Incident management
├── test_sla_api.py          # SLA management
├── test_cmdb_api.py         # CMDB operations
├── test_users_api.py        # User management
└── test_complete_api.py     # End-to-end workflows
```

### Test Configuration (`tests/config.ini`)
```ini
[api]
base_url = http://localhost:8080/api/v1
timeout = 30

[auth]
test_user = testuser
test_password = password123
test_admin = admin
test_admin_password = admin123

[database]
connection_string = sqlite:///../itsm-backend/test.db
```

## Authentication Testing

### Login Test Pattern
```python
import requests
import pytest

class TestAuthAPI:
    def test_login_success(self):
        """Test successful login"""
        payload = {
            "username": "testuser",
            "password": "password123"
        }
        response = requests.post(f"{BASE_URL}/auth/login", json=payload)
        
        assert response.status_code == 200
        data = response.json()
        assert "token" in data
        assert "user" in data
        assert data["user"]["username"] == "testuser"
    
    def test_login_invalid_credentials(self):
        """Test login with invalid credentials"""
        payload = {
            "username": "testuser",
            "password": "wrongpassword"
        }
        response = requests.post(f"{BASE_URL}/auth/login", json=payload)
        
        assert response.status_code == 401
        data = response.json()
        assert "error" in data
```

### Token Validation Test
```python
def test_token_validation(self):
    """Test JWT token validation"""
    # Login to get token
    login_response = requests.post(f"{BASE_URL}/auth/login", json={
        "username": "testuser",
        "password": "password123"
    })
    token = login_response.json()["token"]
    
    # Test authenticated endpoint
    headers = {"Authorization": f"Bearer {token}"}
    response = requests.get(f"{BASE_URL}/users/profile", headers=headers)
    
    assert response.status_code == 200
    assert response.json()["username"] == "testuser"
```

## CRUD Operations Testing

### Ticket API Testing
```python
class TestTicketsAPI:
    @pytest.fixture
    def auth_headers(self):
        """Get authentication headers"""
        response = requests.post(f"{BASE_URL}/auth/login", json={
            "username": "testuser",
            "password": "password123"
        })
        token = response.json()["token"]
        return {"Authorization": f"Bearer {token}"}
    
    def test_create_ticket(self, auth_headers):
        """Test ticket creation"""
        payload = {
            "title": "Test Ticket",
            "description": "Test description",
            "priority": "high",
            "type": "incident",
            "category_id": 1
        }
        response = requests.post(f"{BASE_URL}/tickets", json=payload, headers=auth_headers)
        
        assert response.status_code == 201
        data = response.json()
        assert data["title"] == "Test Ticket"
        assert data["priority"] == "high"
        assert "id" in data
        return data["id"]
    
    def test_get_ticket(self, auth_headers):
        """Test getting ticket by ID"""
        # First create a ticket
        ticket_id = self.test_create_ticket(auth_headers)
        
        # Get ticket
        response = requests.get(f"{BASE_URL}/tickets/{ticket_id}", headers=auth_headers)
        
        assert response.status_code == 200
        data = response.json()
        assert data["id"] == ticket_id
        assert data["title"] == "Test Ticket"
    
    def test_update_ticket(self, auth_headers):
        """Test ticket update"""
        ticket_id = self.test_create_ticket(auth_headers)
        
        update_payload = {
            "title": "Updated Ticket Title",
            "priority": "medium",
            "status": "in_progress"
        }
        response = requests.put(f"{BASE_URL}/tickets/{ticket_id}", json=update_payload, headers=auth_headers)
        
        assert response.status_code == 200
        data = response.json()
        assert data["title"] == "Updated Ticket Title"
        assert data["priority"] == "medium"
        assert data["status"] == "in_progress"
    
    def test_delete_ticket(self, auth_headers):
        """Test ticket deletion"""
        ticket_id = self.test_create_ticket(auth_headers)
        
        response = requests.delete(f"{BASE_URL}/tickets/{ticket_id}", headers=auth_headers)
        assert response.status_code == 204
        
        # Verify deletion
        get_response = requests.get(f"{BASE_URL}/tickets/{ticket_id}", headers=auth_headers)
        assert get_response.status_code == 404
```

## API Contract Validation

### Request Validation Testing
```python
def test_ticket_validation(self, auth_headers):
    """Test request validation"""
    # Missing required field
    payload = {
        "description": "Test description",
        "priority": "high"
    }
    response = requests.post(f"{BASE_URL}/tickets", json=payload, headers=auth_headers)
    
    assert response.status_code == 400
    data = response.json()
    assert "error" in data
    assert "title" in data["error"]
    
    # Invalid enum value
    payload = {
        "title": "Test Ticket",
        "priority": "invalid_priority"
    }
    response = requests.post(f"{BASE_URL}/tickets", json=payload, headers=auth_headers)
    
    assert response.status_code == 400
    assert "priority" in response.json()["error"]
```

### Response Format Testing
```python
def test_api_response_format(self, auth_headers):
    """Test consistent API response format"""
    response = requests.get(f"{BASE_URL}/tickets", headers=auth_headers)
    
    assert response.status_code == 200
    data = response.json()
    
    # Check pagination format
    assert "data" in data
    assert "pagination" in data
    assert "total" in data["pagination"]
    assert "page" in data["pagination"]
    assert "limit" in data["pagination"]
    
    # Check data structure
    if data["data"]:
        ticket = data["data"][0]
        assert "id" in ticket
        assert "title" in ticket
        assert "status" in ticket
        assert "created_at" in ticket
```

## Error Handling Testing

### HTTP Status Code Testing
```python
def test_http_status_codes(self, auth_headers):
    """Test proper HTTP status codes"""
    
    # 401 - Unauthorized
    response = requests.get(f"{BASE_URL}/tickets")
    assert response.status_code == 401
    
    # 404 - Not Found
    response = requests.get(f"{BASE_URL}/tickets/99999", headers=auth_headers)
    assert response.status_code == 404
    
    # 403 - Forbidden (test user trying to access admin endpoint)
    response = requests.get(f"{BASE_URL}/admin/users", headers=auth_headers)
    assert response.status_code == 403
```

### Error Message Testing
```python
def test_error_messages(self, auth_headers):
    """Test helpful error messages"""
    
    # Test validation error
    payload = {"title": ""}  # Empty title
    response = requests.post(f"{BASE_URL}/tickets", json=payload, headers=auth_headers)
    
    assert response.status_code == 400
    error = response.json()
    assert "error" in error
    assert isinstance(error["error"], dict)
    assert "title" in error["error"]
    assert "Title cannot be empty" in error["error"]["title"]
```

## Performance Testing

### Response Time Testing
```python
def test_api_response_time(self, auth_headers):
    """Test API response times"""
    import time
    
    start_time = time.time()
    response = requests.get(f"{BASE_URL}/tickets", headers=auth_headers)
    end_time = time.time()
    
    response_time = end_time - start_time
    assert response.status_code == 200
    assert response_time < 2.0  # Should respond within 2 seconds
```

### Load Testing Pattern
```python
def test_api_concurrent_requests(self, auth_headers):
    """Test API under concurrent load"""
    import concurrent.futures
    
    def make_request(i):
        payload = {
            "title": f"Load Test Ticket {i}",
            "description": f"Description {i}",
            "priority": "medium"
        }
        return requests.post(f"{BASE_URL}/tickets", json=payload, headers=auth_headers)
    
    # Make 10 concurrent requests
    with concurrent.futures.ThreadPoolExecutor(max_workers=10) as executor:
        futures = [executor.submit(make_request, i) for i in range(10)]
        results = [future.result() for future in concurrent.futures.as_completed(futures)]
    
    # All requests should succeed
    success_count = sum(1 for r in results if r.status_code == 201)
    assert success_count == 10
```

## Integration Testing

### End-to-End Workflow Testing
```python
def test_complete_ticket_workflow(self):
    """Test complete ticket lifecycle"""
    
    # 1. Login
    auth_response = requests.post(f"{BASE_URL}/auth/login", json={
        "username": "testuser",
        "password": "password123"
    })
    token = auth_response.json()["token"]
    headers = {"Authorization": f"Bearer {token}"}
    
    # 2. Create ticket
    ticket_payload = {
        "title": "Integration Test Ticket",
        "description": "Testing complete workflow",
        "priority": "high",
        "type": "incident"
    }
    create_response = requests.post(f"{BASE_URL}/tickets", json=ticket_payload, headers=headers)
    ticket_id = create_response.json()["id"]
    
    # 3. Update ticket status
    update_payload = {"status": "in_progress", "assignee_id": 1}
    update_response = requests.put(f"{BASE_URL}/tickets/{ticket_id}", json=update_payload, headers=headers)
    assert update_response.status_code == 200
    
    # 4. Add comment
    comment_payload = {"content": "Working on this issue"}
    comment_response = requests.post(f"{BASE_URL}/tickets/{ticket_id}/comments", json=comment_payload, headers=headers)
    assert comment_response.status_code == 201
    
    # 5. Close ticket
    close_payload = {"status": "closed", "resolution": "Issue resolved"}
    close_response = requests.put(f"{BASE_URL}/tickets/{ticket_id}", json=close_payload, headers=headers)
    assert close_response.status_code == 200
```

## Running API Tests

### Setup Test Environment
```bash
# Start backend server
cd itsm-backend
go run cmd/simple/main.go

# Run API tests
cd tests/api
python -m pytest -v

# Run specific test file
python -m pytest test_tickets_api.py -v

# Run with coverage
python -m pytest --cov=. --cov-report=html

# Run with specific markers
python -m pytest -m "not slow" -v
```

### Test Data Setup
```python
# conftest.py
import pytest
import requests

@pytest.fixture(scope="session")
def base_url():
    return "http://localhost:8080/api/v1"

@pytest.fixture(scope="session")
def admin_token(base_url):
    response = requests.post(f"{base_url}/auth/login", json={
        "username": "admin",
        "password": "admin123"
    })
    return response.json()["token"]

@pytest.fixture
def auth_headers(admin_token):
    return {"Authorization": f"Bearer {admin_token}"}

@pytest.fixture(scope="session", autouse=True)
def setup_test_data(base_url, admin_token):
    """Setup test data before running tests"""
    headers = {"Authorization": f"Bearer {admin_token}"}
    
    # Create test categories
    categories = ["Hardware", "Software", "Network"]
    for category in categories:
        requests.post(f"{base_url}/categories", json={"name": category}, headers=headers)
    
    yield  # Run tests
    
    # Cleanup (if needed)
    # requests.delete(f"{base_url}/test-data", headers=headers)
```

## Common Issues & Solutions

| Issue | Solution |
|-------|----------|
| 401 Unauthorized | Check token expiration and refresh logic |
| 404 Not Found | Verify API endpoint paths and parameters |
| 400 Bad Request | Validate request payload against API schema |
| 500 Internal Error | Check server logs for detailed error messages |
| Slow Response Times | Monitor database queries and add indexes |
| CORS Issues | Configure proper CORS headers in backend |
| Rate Limiting | Implement rate limiting and retry logic |
| Database Locks | Use proper transaction isolation levels |

## Best Practices

1. **Test Isolation**: Each test should be independent and not rely on other tests
2. **Data Management**: Use fixtures and cleanup to manage test data
3. **Error Handling**: Test both success and error scenarios
4. **Performance**: Include performance and load testing
5. **Security**: Test authentication and authorization thoroughly
6. **Documentation**: Keep API documentation in sync with tests
7. **Automation**: Run tests automatically in CI/CD pipeline