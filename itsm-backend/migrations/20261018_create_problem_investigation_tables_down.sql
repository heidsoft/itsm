-- 20261018_create_problem_investigation_tables_down.sql
--
-- 回滚 20261018_create_problem_investigation_tables.sql。
-- 顺序与建表相反：先删依赖方（steps / solutions / root_cause_analyses），再删 investigations。

DROP TABLE IF EXISTS problem_investigation_steps;
DROP TABLE IF EXISTS problem_solutions;
DROP TABLE IF EXISTS problem_root_cause_analyses;
DROP TABLE IF EXISTS problem_investigations;
