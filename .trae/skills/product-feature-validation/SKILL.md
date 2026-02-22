---
name: "product-feature-validation"
description: "Validates product features against requirements, checks feature completeness, and identifies missing functionality. Invoke when user needs to verify if features are working correctly or check product readiness."
---

# Product Feature Validation Skill

This skill helps validate ITSM product features against requirements, check feature completeness, and identify missing functionality or issues that affect product quality.

## Feature Validation Framework

### Core Modules to Validate
1. **Ticket Management** - CRUD operations, lifecycle, assignments
2. **Incident Management** - Incident creation, escalation, resolution
3. **Problem Management** - Problem investigation, root cause analysis
4. **Change Management** - Change requests, approvals, implementation
5. **Service Catalog** - Service requests, catalog management
6. **CMDB** - Configuration items, relationships, discovery
7. **SLA Management** - SLA definitions, monitoring, violations
8. **Workflow Engine** - Business process automation, approvals
9. **Knowledge Base** - Article creation, search, management
10. **User Management** - Authentication, authorization, roles
11. **Dashboard & Analytics** - Reporting, metrics, visualization
12. **Notifications** - Email, SMS, in-app notifications

### Validation Checklist

#### Ticket Management
- [ ] Ticket creation with all required fields
- [ ] Ticket assignment (manual and automatic)
- [ ] Status transitions and workflow
- [ ] Priority and category management
- [ ] Ticket comments and collaboration
- [ ] File attachments and uploads
- [ ] Ticket linking and dependencies
- [ ] Search and filtering capabilities
- [ ] Bulk operations and batch processing
- [ ] Ticket templates and quick creation

#### Incident Management
- [ ] Incident detection and creation
- [ ] Escalation rules and automation
- [ ] Major incident handling
- [ ] Incident communication
- [ ] Post-incident review
- [ ] Incident metrics and KPIs

#### SLA Management
- [ ] SLA definition creation
- [ ] SLA monitoring and tracking
- [ ] Breach notifications
- [ ] SLA reporting and analytics
- [ ] Escalation based on SLA violations

#### User Experience
- [ ] Responsive design for mobile devices
- [ ] Accessibility compliance (WCAG)
- [ ] Multi-language support
- [ ] Performance and loading times
- [ ] Error handling and user feedback
- [ ] Intuitive navigation and workflow

### Feature Testing Scenarios

#### Scenario 1: End-to-End Ticket Lifecycle
1. Create ticket as end user
2. Verify ticket appears in list
3. Assign ticket to agent
4. Add comments and attachments
5. Change status through workflow
6. Resolve and close ticket
7. Verify notifications sent
8. Check audit trail

#### Scenario 2: SLA Breach Handling
1. Create high-priority ticket
2. Set aggressive SLA (e.g., 1 hour)
3. Wait for SLA breach
4. Verify breach notifications
5. Check escalation triggered
6. Verify SLA metrics updated

#### Scenario 3: Multi-User Collaboration
1. User A creates ticket
2. User B adds comment
3. User C assigns ticket
4. User D changes status
5. Verify all users see updates
6. Check notification delivery

### Common Feature Issues

| Issue | Symptoms | Impact | Solution |
|-------|----------|---------|----------|
| Missing field validation | Can submit invalid data | Data quality issues | Add frontend and backend validation |
| Broken workflow transitions | Status changes don't work | Process disruption | Fix workflow engine logic |
| Failed notifications | Users don't receive alerts | Communication gaps | Check notification service |
| Slow page loading | >3 second load times | Poor user experience | Optimize queries and caching |
| Missing permissions | Users can access unauthorized data | Security risk | Implement proper authorization |
| Data inconsistency | Different data in different views | User confusion | Fix data synchronization |

### Feature Readiness Assessment

#### Alpha Ready (Development)
- Core functionality working
- Basic CRUD operations complete
- No critical bugs
- Developer testing passed

#### Beta Ready (Testing)
- All features implemented
- Integration testing complete
- Performance acceptable
- Documentation updated

#### Production Ready
- Comprehensive testing passed
- Security audit complete
- Performance optimized
- User acceptance testing passed
- Monitoring and alerting configured

### Validation Tools

#### Automated Testing
```bash
# Run all tests
cd itsm-backend && go test ./...
cd itsm-frontend && npm test
cd tests && python -m pytest

# Run specific feature tests
go test ./service -run TestTicketService
npm test -- --testNamePattern="Ticket"
python -m pytest tests/e2e/test_tickets_full.py
```

#### Manual Testing Checklist
- [ ] Create test account for each user role
- [ ] Test all user journeys
- [ ] Verify data persistence
- [ ] Check cross-browser compatibility
- [ ] Test mobile responsiveness
- [ ] Verify accessibility features
- [ ] Test error handling
- [ ] Validate input constraints

#### Performance Testing
- [ ] Page load times < 3 seconds
- [ ] API response times < 500ms
- [ ] Database queries optimized
- [ ] Concurrent user testing
- [ ] Memory usage monitoring

### Feature Gap Analysis

#### High Priority Gaps
1. **Advanced Search** - Full-text search across all modules
2. **Bulk Operations** - Mass update and delete operations
3. **Advanced Reporting** - Custom report builder
4. **API Rate Limiting** - Protect against abuse
5. **Data Export** - Backup and migration tools

#### Medium Priority Gaps
1. **Advanced Workflow** - Complex business process automation
2. **AI Integration** - Intelligent ticket routing and suggestions
3. **Advanced Analytics** - Predictive analytics and insights
4. **Multi-tenancy** - Complete tenant isolation
5. **Advanced Notifications** - Push notifications, Slack integration

#### Low Priority Gaps
1. **Advanced UI Themes** - Custom branding and themes
2. **Advanced Integrations** - Third-party tool integrations
3. **Advanced Mobile App** - Native mobile applications
4. **Advanced Security** - Multi-factor authentication
5. **Advanced Compliance** - GDPR, SOX compliance tools

### Product Quality Metrics

#### Functional Quality
- Feature completeness: 85%
- Bug density: < 2 bugs per 1000 LOC
- Test coverage: > 80%
- User story completion: 95%

#### Performance Quality
- Average response time: < 500ms
- Page load time: < 3 seconds
- System availability: 99.9%
- Concurrent user support: 1000+

#### User Experience Quality
- User satisfaction score: > 4.0/5.0
- Task completion rate: > 90%
- Error recovery rate: > 95%
- Learning curve: < 2 hours for basic features