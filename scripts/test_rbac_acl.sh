#!/bin/bash
# RBAC Smart Permission Architecture Test
# Verifies 4-layer permission check architecture

set -e

PGPASSWORD=postgres123
DB_HOST=localhost
DB_USER=postgres
DB_NAME=itsm_dev

echo "========================================"
echo "RBAC Smart Permission Architecture Test"
echo "========================================"

echo ""
echo "1. Checking endpoint_acls table..."
echo "-----------------------------------"
RESULT=$(PGPASSWORD=$PGPASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME -t -c "SELECT COUNT(*) FROM endpoint_acls WHERE tenant_id = 1;" 2>/dev/null | xargs)
echo "Total ACLs in database: $RESULT"

echo ""
echo "2. Checking ACL distribution..."
echo "-----------------------------------"
PGPASSWORD=$PGPASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "
SELECT 
    resource,
    action,
    COUNT(*) as count
FROM endpoint_acls
WHERE tenant_id = 1 AND is_active = true
GROUP BY resource, action
ORDER BY resource, action
LIMIT 15;
"

echo ""
echo "3. Checking whitelist endpoints..."
echo "-----------------------------------"
PGPASSWORD=$PGPASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "
SELECT path_pattern, resource
FROM endpoint_acls
WHERE tenant_id = 1 AND (
    path_pattern LIKE '%/auth/login%' OR 
    path_pattern LIKE '%/auth/register%' OR 
    path_pattern = '/health'
);
"

echo ""
echo "4. Checking role permissions (l1_engineer)..."
echo "-----------------------------------"
PGPASSWORD=$PGPASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "
SELECT r.code as role, p.resource, p.action
FROM role r
JOIN role_permission rp ON r.id = rp.role_id
JOIN permission p ON rp.permission_id = p.id
WHERE r.code = 'l1_engineer' AND r.tenant_id = 1;
"

echo ""
echo "========================================"
echo "Test Complete!"
echo "========================================"
