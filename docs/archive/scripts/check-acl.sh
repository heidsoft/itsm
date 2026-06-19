#!/bin/bash
PGPASSWORD=postgres123 psql -h localhost -U postgres -d itsm_dev -c "SELECT path_pattern, method, resource, action FROM endpoint_acls LIMIT 10;"
