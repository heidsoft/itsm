-- P0-C reverse migration: drop ai_analysis_result table and its RLS policy.
-- Run only when rolling back 20260831_ai_analysis_result_expand.sql.

BEGIN;

DROP POLICY IF EXISTS tenant_isolation_ai_analysis_result ON ai_analysis_result;
ALTER TABLE ai_analysis_result DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS ai_analysis_result;

COMMIT;
