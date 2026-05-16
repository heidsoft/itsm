#!/bin/bash
# Clean up test screenshots and temporary files from the project

cd /Users/heidsoft/Downloads/research/itsm

# Delete test screenshots (keeping docs/images for documentation)
find . -maxdepth 1 -name "*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "*_test_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "*_debug*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "*current*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "test_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "dashboard_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "login_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "incidents_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "problems_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "cmdb_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "role_test_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "multi_role_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "permission_test_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "comprehensive_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "l1_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "user1_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "end_user_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "employee_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "regression_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "ui_test_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "change_management_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "restart_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "workflow_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "reports_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "msp_*.png" -type f -delete 2>/dev/null
find . -maxdepth 1 -name "operations_role_*.png" -type f -delete 2>/dev/null

# Delete temporary files
rm -f .env.backup itsm_baseline.py test_workflow.py auto_test.py quick-start.sh
rm -rf logs memory .openclaw .mypy_cache
rm -f itsm-backend/server.log itsm-backend/coverage.out
rm -f *.log debug*.txt test_result*.log

echo "Cleanup completed"