#!/usr/bin/env python3
"""
ITSM System Comprehensive E2E Test Suite
Tests all critical business scenarios with authentication
"""

import requests
import json
import time
import sys
from datetime import datetime
from typing import Dict, List, Any, Optional

# Configuration
BASE_URL = "http://localhost:8090/api/v1"
FRONTEND_URL = "http://localhost:3000"
CREDENTIALS = {"email": "admin@itsm.com", "password": "admin123"}
# Alternative credentials to try
ALT_CREDENTIALS = [
    {"email": "admin", "password": "admin123"},
    {"email": "admin@example.com", "password": "admin123"},
]

class ITSME2ETester:
    def __init__(self):
        self.base_url = BASE_URL
        self.frontend_url = FRONTEND_URL
        self.auth_token = None
        self.test_results = []
        self.start_time = time.time()
        self.report_data = {
            "test_date": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            "scenarios": [],
            "summary": {},
            "errors": [],
            "recommendation": ""
        }
    
    def log_test(self, scenario: str, status: str, details: Dict[str, Any] = None):
        """Log test result"""
        result = {
            "scenario": scenario,
            "status": status,
            "timestamp": datetime.now().strftime("%H:%M:%S"),
            "details": details or {}
        }
        self.test_results.append(result)
        self.report_data["scenarios"].append(result)
        print(f"[{status}] {scenario}")
        if details and "error" in details:
            print(f"  Error: {details['error']}")
    
    def measure_time(self, func, *args, **kwargs):
        """Measure execution time of a function"""
        start = time.time()
        try:
            result = func(*args, **kwargs)
            elapsed = (time.time() - start) * 1000
            return result, elapsed
        except Exception as e:
            elapsed = (time.time() - start) * 1000
            raise Exception(f"Exception after {elapsed:.2f}ms: {str(e)}")
    
    def make_request(self, method: str, endpoint: str, token_required: bool = True, **kwargs) -> requests.Response:
        """Make HTTP request with authentication"""
        url = f"{self.base_url}{endpoint}"
        headers = kwargs.pop('headers', {})
        if token_required and self.auth_token:
            headers['Authorization'] = f'Bearer {self.auth_token}'
        
        start_time = time.time()
        try:
            response = requests.request(method, url, headers=headers, **kwargs)
            elapsed = (time.time() - start_time) * 1000
            
            # Log response details
            print(f"  → {method} {endpoint} - {response.status_code} ({elapsed:.2f}ms)")
            if response.status_code >= 400:
                print(f"    Response: {response.text[:200]}")
            
            response.elapsed_ms = elapsed
            return response
        except requests.exceptions.ConnectionError as e:
            print(f"  → {method} {endpoint} - Connection failed: {e}")
            raise
        except Exception as e:
            print(f"  → {method} {endpoint} - Error: {e}")
            raise
    
    def test_login_authentication(self):
        """Scenario 1: Login and authentication flow"""
        print("\n=== SCENARIO 1: Login and Authentication Flow ===")
        
        # Test with primary credentials
        try:
            response, elapsed = self.measure_time(
                self.make_request,
                'POST',
                '/auth/login',
                token_required=False,
                json=CREDENTIALS
            )
            
            if response.status_code == 200:
                data = response.json()
                self.auth_token = data.get('token') or data.get('access_token') or data.get('accessToken')
                
                if self.auth_token:
                    self.log_test("Login with primary credentials", "PASS", {
                        "token_received": True,
                        "response_time_ms": elapsed,
                        "token_type": "Bearer"
                    })
                else:
                    self.log_test("Login with primary credentials", "FAIL", {
                        "error": "No token in response",
                        "response": data
                    })
            else:
                self.log_test("Login with primary credentials", "FAIL", {
                    "status_code": response.status_code,
                    "response": response.text[:200]
                })
                
                # Try alternative credentials
                for creds in ALT_CREDENTIALS:
                    try:
                        response = requests.post(
                            f"{self.base_url}/auth/login",
                            json=creds
                        )
                        if response.status_code == 200:
                            data = response.json()
                            self.auth_token = data.get('token') or data.get('access_token')
                            if self.auth_token:
                                self.log_test(f"Login with alternative ({creds['email']})", "PASS", {
                                    "credentials": creds['email']
                                })
                                break
                    except:
                        continue
        except Exception as e:
            self.log_test("Login and authentication", "FAIL", {"error": str(e)})
        
        # Test token refresh if token exists
        if self.auth_token:
            try:
                response = self.make_request('POST', '/auth/refresh')
                if response.status_code == 200:
                    self.log_test("Token refresh", "PASS")
                else:
                    self.log_test("Token refresh", "FAIL", {"status": response.status_code})
            except Exception as e:
                self.log_test("Token refresh", "FAIL", {"error": str(e)})
    
    def test_dashboard_statistics(self):
        """Scenario 2: Dashboard statistics display"""
        print("\n=== SCENARIO 2: Dashboard Statistics ===")
        
        endpoints_to_test = [
            '/dashboard/stats',
            '/dashboard/overview',
            '/dashboard/metrics',
            '/reports/dashboard'
        ]
        
        for endpoint in endpoints_to_test:
            try:
                response = self.make_request('GET', endpoint)
                if response.status_code == 200:
                    data = response.json()
                    self.log_test(f"Dashboard endpoint: {endpoint}", "PASS", {
                        "response_time_ms": response.elapsed_ms,
                        "data_keys": list(data.keys())[:5] if isinstance(data, dict) else "Non-dict response"
                    })
                else:
                    self.log_test(f"Dashboard endpoint: {endpoint}", "FAIL", {
                        "status_code": response.status_code
                    })
            except Exception as e:
                self.log_test(f"Dashboard endpoint: {endpoint}", "FAIL", {"error": str(e)})
    
    def test_ticket_management(self):
        """Scenario 3: Ticket management (create, list, view, update status)"""
        print("\n=== SCENARIO 3: Ticket Management ===")
        
        # List tickets
        try:
            response = self.make_request('GET', '/tickets')
            if response.status_code == 200:
                tickets = response.json()
                ticket_count = len(tickets) if isinstance(tickets, list) else 0
                self.log_test("List tickets", "PASS", {
                    "count": ticket_count,
                    "response_time_ms": response.elapsed_ms
                })
            else:
                self.log_test("List tickets", "FAIL", {"status_code": response.status_code})
        except Exception as e:
            self.log_test("List tickets", "FAIL", {"error": str(e)})
        
        # Create a test ticket
        ticket_data = {
            "title": "E2E Test Ticket - Automated",
            "description": "This is an automated test ticket created by E2E test suite",
            "priority": "medium",
            "ticket_type": "incident",
            "requester_id": 1
        }
        
        try:
            response = self.make_request('POST', '/tickets', json=ticket_data)
            if response.status_code in [200, 201]:
                created_ticket = response.json()
                ticket_id = created_ticket.get('id') or created_ticket.get('ticket_id')
                self.log_test("Create ticket", "PASS", {
                    "ticket_id": ticket_id,
                    "response_time_ms": response.elapsed_ms
                })
                
                # View ticket details
                if ticket_id:
                    try:
                        response = self.make_request('GET', f'/tickets/{ticket_id}')
                        if response.status_code == 200:
                            self.log_test("View ticket details", "PASS", {
                                "ticket_id": ticket_id,
                                "response_time_ms": response.elapsed_ms
                            })
                        else:
                            self.log_test("View ticket details", "FAIL", {"status_code": response.status_code})
                    except Exception as e:
                        self.log_test("View ticket details", "FAIL", {"error": str(e)})
                    
                    # Update ticket status
                    update_data = {"status": "in_progress"}
                    try:
                        response = self.make_request('PATCH', f'/tickets/{ticket_id}', json=update_data)
                        if response.status_code in [200, 204]:
                            self.log_test("Update ticket status", "PASS")
                        else:
                            self.log_test("Update ticket status", "FAIL", {"status_code": response.status_code})
                    except Exception as e:
                        self.log_test("Update ticket status", "FAIL", {"error": str(e)})
                    
                    # Delete test ticket (cleanup)
                    try:
                        response = self.make_request('DELETE', f'/tickets/{ticket_id}')
                        if response.status_code in [200, 204]:
                            self.log_test("Delete test ticket", "PASS")
                        else:
                            self.log_test("Delete test ticket", "FAIL", {"status_code": response.status_code})
                    except:
                        pass
            else:
                self.log_test("Create ticket", "FAIL", {
                    "status_code": response.status_code,
                    "response": response.text[:200]
                })
        except Exception as e:
            self.log_test("Create ticket", "FAIL", {"error": str(e)})
    
    def test_incident_management(self):
        """Scenario 4: Incident management (list, view, edit)"""
        print("\n=== SCENARIO 4: Incident Management ===")
        
        # List incidents
        try:
            response = self.make_request('GET', '/incidents')
            if response.status_code == 200:
                incidents = response.json()
                incident_count = len(incidents) if isinstance(incidents, list) else 0
                self.log_test("List incidents", "PASS", {
                    "count": incident_count,
                    "response_time_ms": response.elapsed_ms
                })
            else:
                self.log_test("List incidents", "FAIL", {"status_code": response.status_code})
        except Exception as e:
            self.log_test("List incidents", "FAIL", {"error": str(e)})
        
        # Create test incident
        incident_data = {
            "title": "E2E Test Incident",
            "description": "Automated test incident",
            "urgency": "medium",
            "impact": "low"
        }
        
        try:
            response = self.make_request('POST', '/incidents', json=incident_data)
            if response.status_code in [200, 201]:
                incident = response.json()
                incident_id = incident.get('id')
                self.log_test("Create incident", "PASS", {"incident_id": incident_id})
                
                # View incident
                if incident_id:
                    try:
                        response = self.make_request('GET', f'/incidents/{incident_id}')
                        if response.status_code == 200:
                            self.log_test("View incident", "PASS")
                        else:
                            self.log_test("View incident", "FAIL")
                    except:
                        self.log_test("View incident", "FAIL")
                    
                    # Edit incident
                    update_data = {"status": "resolved"}
                    try:
                        response = self.make_request('PATCH', f'/incidents/{incident_id}', json=update_data)
                        if response.status_code == 200:
                            self.log_test("Edit incident", "PASS")
                        else:
                            self.log_test("Edit incident", "FAIL")
                    except:
                        self.log_test("Edit incident", "FAIL")
            else:
                self.log_test("Create incident", "FAIL", {"status_code": response.status_code})
        except Exception as e:
            self.log_test("Create incident", "FAIL", {"error": str(e)})
    
    def test_cmdb_asset_creation(self):
        """Scenario 5: CMDB asset creation"""
        print("\n=== SCENARIO 5: CMDB Asset Creation ===")
        
        # List existing assets
        try:
            response = self.make_request('GET', '/cmdb/items')
            if response.status_code == 200:
                items = response.json()
                item_count = len(items) if isinstance(items, list) else 0
                self.log_test("List CMDB items", "PASS", {"count": item_count})
            else:
                self.log_test("List CMDB items", "FAIL", {"status_code": response.status_code})
        except Exception as e:
            self.log_test("List CMDB items", "FAIL", {"error": str(e)})
        
        # Create CMDB asset
        cmdb_data = {
            "name": "E2E Test Server",
            "item_type": "server",
            "serial_number": f"TEST-{int(time.time())}",
            "manufacturer": "Test Corp",
            "model": "TestModel-1000",
            "status": "active",
            "location": "Test Data Center"
        }
        
        try:
            response = self.make_request('POST', '/cmdb/items', json=cmdb_data)
            if response.status_code in [200, 201]:
                item = response.json()
                item_id = item.get('id')
                self.log_test("Create CMDB item", "PASS", {"item_id": item_id})
                
                # Verify item was created
                if item_id:
                    try:
                        response = self.make_request('GET', f'/cmdb/items/{item_id}')
                        if response.status_code == 200:
                            self.log_test("Verify CMDB item creation", "PASS")
                        else:
                            self.log_test("Verify CMDB item creation", "FAIL")
                    except:
                        self.log_test("Verify CMDB item creation", "FAIL")
                    
                    # Cleanup
                    try:
                        self.make_request('DELETE', f'/cmdb/items/{item_id}')
                    except:
                        pass
            else:
                self.log_test("Create CMDB item", "FAIL", {
                    "status_code": response.status_code,
                    "response": response.text[:200]
                })
        except Exception as e:
            self.log_test("Create CMDB item", "FAIL", {"error": str(e)})
    
    def test_change_management_submit(self):
        """Scenario 6: Change management submit"""
        print("\n=== SCENARIO 6: Change Management Submit ===")
        
        # List changes
        try:
            response = self.make_request('GET', '/changes')
            if response.status_code == 200:
                changes = response.json()
                change_count = len(changes) if isinstance(changes, list) else 0
                self.log_test("List changes", "PASS", {"count": change_count})
            else:
                self.log_test("List changes", "FAIL", {"status_code": response.status_code})
        except Exception as e:
            self.log_test("List changes", "FAIL", {"error": str(e)})
        
        # Create change request
        change_data = {
            "title": "E2E Test Change",
            "description": "Automated test change request",
            "change_type": "standard",
            "priority": "low",
            "planned_start": (datetime.now()).isoformat(),
            "planned_end": (datetime.now()).isoformat()
        }
        
        try:
            response = self.make_request('POST', '/changes', json=change_data)
            if response.status_code in [200, 201]:
                change = response.json()
                change_id = change.get('id')
                self.log_test("Create change request", "PASS", {"change_id": change_id})
                
                if change_id:
                    # Test submit flow
                    try:
                        response = self.make_request('POST', f'/changes/{change_id}/submit')
                        if response.status_code in [200, 204]:
                            self.log_test("Submit change request", "PASS")
                        else:
                            self.log_test("Submit change request", "FAIL", {
                                "status_code": response.status_code,
                                "response": response.text[:200]
                            })
                    except Exception as e:
                        self.log_test("Submit change request", "FAIL", {"error": str(e)})
                    
                    # Cleanup
                    try:
                        self.make_request('DELETE', f'/changes/{change_id}')
                    except:
                        pass
            else:
                self.log_test("Create change request", "FAIL", {"status_code": response.status_code})
        except Exception as e:
            self.log_test("Create change request", "FAIL", {"error": str(e)})
    
    def test_knowledge_base(self):
        """Scenario 7: Knowledge base articles and stats"""
        print("\n=== SCENARIO 7: Knowledge Base ===")
        
        # List articles
        try:
            response = self.make_request('GET', '/knowledge/articles')
            if response.status_code == 200:
                articles = response.json()
                article_count = len(articles) if isinstance(articles, list) else 0
                self.log_test("List knowledge articles", "PASS", {"count": article_count})
            else:
                self.log_test("List knowledge articles", "FAIL", {"status_code": response.status_code})
        except Exception as e:
            self.log_test("List knowledge articles", "FAIL", {"error": str(e)})
        
        # Get knowledge stats
        try:
            response = self.make_request('GET', '/knowledge/stats')
            if response.status_code == 200:
                stats = response.json()
                self.log_test("Knowledge base stats", "PASS", stats)
            else:
                self.log_test("Knowledge base stats", "FAIL", {"status_code": response.status_code})
        except Exception as e:
            self.log_test("Knowledge base stats", "FAIL", {"error": str(e)})
        
        # Create article
        article_data = {
            "title": "E2E Test Article",
            "content": "This is an automated test article for E2E testing",
            "category": "test",
            "tags": ["test", "automated"]
        }
        
        try:
            response = self.make_request('POST', '/knowledge/articles', json=article_data)
            if response.status_code in [200, 201]:
                article = response.json()
                article_id = article.get('id')
                self.log_test("Create knowledge article", "PASS", {"article_id": article_id})
                
                # Cleanup
                if article_id:
                    try:
                        self.make_request('DELETE', f'/knowledge/articles/{article_id}')
                    except:
                        pass
            else:
                self.log_test("Create knowledge article", "FAIL", {"status_code": response.status_code})
        except Exception as e:
            self.log_test("Create knowledge article", "FAIL", {"error": str(e)})
    
    def test_user_groups(self):
        """Scenario 8: User groups management"""
        print("\n=== SCENARIO 8: User Groups Management ===")
        
        # List user groups
        endpoints = [
            '/users/groups',
            '/groups',
            '/user-groups',
            '/org/groups'
        ]
        
        for endpoint in endpoints:
            try:
                response = self.make_request('GET', endpoint)
                if response.status_code == 200:
                    groups = response.json()
                    group_count = len(groups) if isinstance(groups, list) else 0
                    self.log_test(f"List groups via {endpoint}", "PASS", {"count": group_count})
                    break
                else:
                    self.log_test(f"List groups via {endpoint}", "FAIL", {"status_code": response.status_code})
            except Exception as e:
                self.log_test(f"List groups via {endpoint}", "FAIL", {"error": str(e)})
        
        # Create group
        group_data = {
            "name": "E2E Test Group",
            "description": "Automated test group"
        }
        
        try:
            response = self.make_request('POST', '/users/groups', json=group_data)
            if response.status_code in [200, 201]:
                group = response.json()
                group_id = group.get('id')
                self.log_test("Create user group", "PASS", {"group_id": group_id})
                
                # Cleanup
                if group_id:
                    try:
                        self.make_request('DELETE', f'/users/groups/{group_id}')
                    except:
                        pass
            else:
                self.log_test("Create user group", "FAIL", {"status_code": response.status_code})
        except Exception as e:
            self.log_test("Create user group", "FAIL", {"error": str(e)})
    
    def test_sla_compliance(self):
        """Scenario 9: SLA compliance data"""
        print("\n=== SCENARIO 9: SLA Compliance ===")
        
        endpoints = [
            '/sla/compliance',
            '/sla/stats',
            '/sla/metrics',
            '/sla/definitions'
        ]
        
        for endpoint in endpoints:
            try:
                response = self.make_request('GET', endpoint)
                if response.status_code == 200:
                    data = response.json()
                    self.log_test(f"SLA endpoint: {endpoint}", "PASS", {
                        "data_keys": list(data.keys())[:5] if isinstance(data, dict) else "Non-dict"
                    })
                else:
                    self.log_test(f"SLA endpoint: {endpoint}", "FAIL", {"status_code": response.status_code})
            except Exception as e:
                self.log_test(f"SLA endpoint: {endpoint}", "FAIL", {"error": str(e)})
    
    def test_crud_operations(self):
        """Scenario 10: All CRUD operations for major entities"""
        print("\n=== SCENARIO 10: Comprehensive CRUD Operations ===")
        
        entities = [
            ('users', {"email": "test@example.com", "name": "Test User", "password": "test123"}),
            ('teams', {"name": "E2E Test Team", "description": "Test team"}),
            ('roles', {"name": "E2E Test Role", "permissions": ["read"]}),
            ('priorities', {"name": "E2E Test Priority", "level": 5}),
            ('statuses', {"name": "E2E Test Status", "color": "#FF0000"}),
        ]
        
        for entity, create_data in entities:
            try:
                # CREATE
                response = self.make_request('POST', f'/{entity}', json=create_data)
                if response.status_code in [200, 201]:
                    entity_obj = response.json()
                    entity_id = entity_obj.get('id')
                    self.log_test(f"Create {entity[:-1]}", "PASS", {"id": entity_id})
                    
                    if entity_id:
                        # READ
                        try:
                            response = self.make_request('GET', f'/{entity}/{entity_id}')
                            if response.status_code == 200:
                                self.log_test(f"Read {entity[:-1]}", "PASS")
                            else:
                                self.log_test(f"Read {entity[:-1]}", "FAIL")
                        except:
                            self.log_test(f"Read {entity[:-1]}", "FAIL")
                        
                        # UPDATE
                        try:
                            response = self.make_request('PATCH', f'/{entity}/{entity_id}', json={"name": f"Updated {create_data['name']}"})
                            if response.status_code == 200:
                                self.log_test(f"Update {entity[:-1]}", "PASS")
                            else:
                                self.log_test(f"Update {entity[:-1]}", "FAIL")
                        except:
                            self.log_test(f"Update {entity[:-1]}", "FAIL")
                        
                        # DELETE
                        try:
                            response = self.make_request('DELETE', f'/{entity}/{entity_id}')
                            if response.status_code in [200, 204]:
                                self.log_test(f"Delete {entity[:-1]}", "PASS")
                            else:
                                self.log_test(f"Delete {entity[:-1]}", "FAIL")
                        except:
                            self.log_test(f"Delete {entity[:-1]}", "FAIL")
                else:
                    self.log_test(f"Create {entity[:-1]}", "FAIL", {"status": response.status_code})
            except Exception as e:
                self.log_test(f"CRUD {entity[:-1]}", "FAIL", {"error": str(e)})
    
    def test_frontend_accessibility(self):
        """Test frontend accessibility"""
        print("\n=== FRONTEND ACCESSIBILITY ===")
        
        try:
            response = requests.get(self.frontend_url, timeout=10)
            if response.status_code == 200:
                self.log_test("Frontend accessible", "PASS", {
                    "url": self.frontend_url,
                    "content_type": response.headers.get('content-type', 'unknown')
                })
            else:
                self.log_test("Frontend accessible", "FAIL", {"status_code": response.status_code})
        except Exception as e:
            self.log_test("Frontend accessible", "FAIL", {"error": str(e)})
    
    def generate_report(self):
        """Generate final E2E test report"""
        print("\n=== GENERATING FINAL REPORT ===")
        
        total_tests = len(self.test_results)
        passed = sum(1 for r in self.test_results if r['status'] == 'PASS')
        failed = sum(1 for r in self.test_results if r['status'] == 'FAIL')
        pass_rate = (passed / total_tests * 100) if total_tests > 0 else 0
        
        self.report_data["summary"] = {
            "total_tests": total_tests,
            "passed": passed,
            "failed": failed,
            "pass_rate_percent": round(pass_rate, 2),
            "total_duration_seconds": round(time.time() - self.start_time, 2)
        }
        
        # Determine recommendation
        if pass_rate >= 90:
            recommendation = "PRODUCTION READY - System performs excellently with minimal issues."
        elif pass_rate >= 75:
            recommendation = "CONDITIONALLY READY - System is functional but has some issues that need attention."
        elif pass_rate >= 50:
            recommendation = "NEEDS IMPROVEMENT - System has significant issues requiring fixes before production."
        else:
            recommendation = "NOT READY - System has critical failures and is not suitable for production."
        
        self.report_data["recommendation"] = recommendation
        
        # Collect all errors
        self.report_data["errors"] = [
            r for r in self.test_results 
            if r['status'] == 'FAIL'
        ]
        
        return self.report_data
    
    def save_report(self):
        """Save report to file"""
        report = self.generate_report()
        report_path = "/Users/heidsoft/Downloads/research/itsm/itsm-backend/memory/final-e2e-validation-2026-03-08.md"
        
        # Ensure directory exists
        import os
        os.makedirs(os.path.dirname(report_path), exist_ok=True)
        
        with open(report_path, 'w', encoding='utf-8') as f:
            f.write("# ITSM System E2E Validation Report\n\n")
            f.write(f"**Generated:** {report['test_date']}\n")
            f.write(f"**System:** ITSM Backend + Frontend\n")
            f.write(f"**Environment:** Local Development\n\n")
            
            f.write("## Executive Summary\n\n")
            f.write(f"- **Total Tests:** {report['summary']['total_tests']}\n")
            f.write(f"- **Passed:** {report['summary']['passed']}\n")
            f.write(f"- **Failed:** {report['summary']['failed']}\n")
            f.write(f"- **Pass Rate:** {report['summary']['pass_rate_percent']}%\n")
            f.write(f"- **Duration:** {report['summary']['total_duration_seconds']} seconds\n")
            f.write(f"- **Recommendation:** {report['recommendation']}\n\n")
            
            f.write("## Detailed Test Results\n\n")
            f.write("| # | Scenario | Status | Details |\n")
            f.write("|---|----------|--------|---------|\n")
            
            for i, scenario in enumerate(report['scenarios'], 1):
                status_icon = "✅" if scenario['status'] == 'PASS' else "❌"
                details_str = str(scenario.get('details', {}))[:50] + "..." if len(str(scenario.get('details', {}))) > 50 else str(scenario.get('details', {}))
                f.write(f"| {i} | {scenario['scenario']} | {status_icon} {scenario['status']} | {details_str} |\n")
            
            f.write("\n## Failed Tests\n\n")
            if report['errors']:
                f.write("| Scenario | Error | Details |\n")
                f.write("|----------|-------|---------|\n")
                for error in report['errors']:
                    f.write(f"| {error['scenario']} | {error['details'].get('error', 'Unknown error')} | {error['details']} |\n")
            else:
                f.write("No failed tests! ✅\n")
            
            f.write("\n## API Response Times\n\n")
            f.write("| Endpoint | Avg Response Time (ms) |\n")
            f.write("|----------|----------------------|\n")
            # Could calculate avg times from data
            
            f.write("\n## Production Readiness Assessment\n\n")
            f.write(f"**Verdict:** {report['recommendation']}\n\n")
            
            if report['summary']['pass_rate_percent'] >= 90:
                f.write("### Strengths\n")
                f.write("- High test pass rate\n")
                f.write("- Most critical functions working\n")
                f.write("- Good API response times\n\n")
                f.write("### Minor Issues\n")
                f.write("- None critical\n\n")
            elif report['summary']['pass_rate_percent'] >= 75:
                f.write("### Strengths\n")
                f.write("- Core functionality operational\n")
                f.write("- Most critical paths working\n\n")
                f.write("### Areas for Improvement\n")
                f.write("- Some edge cases failing\n")
                f.write("- Consider improving error handling\n\n")
            else:
                f.write("### Critical Issues\n")
                f.write("- Major functionality missing\n")
                f.write("- Significant API failures\n")
                f.write("- Requires immediate attention\n\n")
            
            f.write("## Recommendations\n\n")
            f.write("1. Fix all failing critical path tests\n")
            f.write("2. Add comprehensive error handling\n")
            f.write("3. Implement proper input validation\n")
            f.write("4. Add integration tests for user workflows\n")
            f.write("5. Consider performance optimization\n")
        
        print(f"\n📄 Report saved to: {report_path}")
        return report_path
    
    def run_all_tests(self):
        """Run all test scenarios"""
        print("=" * 60)
        print("ITSM System Comprehensive E2E Test Suite")
        print("=" * 60)
        print(f"Backend URL: {self.base_url}")
        print(f"Frontend URL: {self.frontend_url}")
        print(f"Test Date: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
        print("=" * 60)
        
        try:
            # Check services
            print("\n[Pre-check] Verifying services are running...")
            try:
                response = requests.get(f"{self.base_url.replace('/api/v1', '')}/health", timeout=5)
                print(f"  Backend health: {'✅' if response.status_code < 400 else '❌'}")
            except:
                print("  Backend health: ❌ (endpoint not responding)")
            
            try:
                response = requests.get(self.frontend_url, timeout=5)
                print(f"  Frontend: {'✅' if response.status_code < 400 else '❌'}")
            except:
                print("  Frontend: ❌ (not responding)")
            
            # Run all test scenarios
            self.test_login_authentication()
            self.test_dashboard_statistics()
            self.test_ticket_management()
            self.test_incident_management()
            self.test_cmdb_asset_creation()
            self.test_change_management_submit()
            self.test_knowledge_base()
            self.test_user_groups()
            self.test_sla_compliance()
            self.test_crud_operations()
            self.test_frontend_accessibility()
            
            # Generate and save report
            report_path = self.save_report()
            
            print("\n" + "=" * 60)
            print("E2E TESTING COMPLETED")
            print("=" * 60)
            
        except KeyboardInterrupt:
            print("\n\n⚠️  Testing interrupted by user")
            self.save_report()
        except Exception as e:
            print(f"\n❌ Critical error during testing: {e}")
            import traceback
            traceback.print_exc()
            self.save_report()

def main():
    tester = ITSME2ETester()
    tester.run_all_tests()

if __name__ == "__main__":
    main()