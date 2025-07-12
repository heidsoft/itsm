psql postgres
psql (14.18 (Homebrew))
Type "help" for help.

postgres=# CREATE ROLE dev WITH LOGIN PASSWORD '123456!@#$%^';
CREATE DATABASE itsm OWNER dev;
GRANT ALL PRIVILEGES ON DATABASE itsm TO dev;
CREATE ROLE
CREATE DATABASE
GRANT
postgres=# \itsm
invalid command \itsm
Try \? for help.
postgres=# \h
postgres=# \c itsm
You are now connected to database "itsm" as user "heidsoft".
itsm=# show tables;
ERROR:  unrecognized configuration parameter "tables"
itsm=# show t
tcp_keepalives_count       temp_tablespaces           track_activities           track_wal_io_timing
tcp_keepalives_idle        timezone                   track_activity_query_size  transaction_deferrable
tcp_keepalives_interval    timezone_abbreviations     track_commit_timestamp     transaction_isolation
tcp_user_timeout           trace_notify               track_counts               transaction_read_only
temp_buffers               trace_recovery_messages    track_functions            transform_null_equals
temp_file_limit            trace_sort                 track_io_timing
itsm=# \l
                             List of databases
   Name    |  Owner   | Encoding | Collate | Ctype |   Access privileges
-----------+----------+----------+---------+-------+-----------------------
 itsm      | dev      | UTF8     | C       | C     | =Tc/dev              +
           |          |          |         |       | dev=CTc/dev
 postgres  | heidsoft | UTF8     | C       | C     |
 template0 | heidsoft | UTF8     | C       | C     | =c/heidsoft          +
           |          |          |         |       | heidsoft=CTc/heidsoft
 template1 | heidsoft | UTF8     | C       | C     | =c/heidsoft          +
           |          |          |         |       | heidsoft=CTc/heidsoft
(4 rows)

itsm=# \dt
            List of relations
 Schema |      Name      | Type  | Owner
--------+----------------+-------+-------
 public | approval_logs  | table | dev
 public | flow_instances | table | dev
 public | tickets        | table | dev
 public | users          | table | dev
(4 rows)

itsm=# select * from approval_logs;
 id | comment | status | step_order | step_name | metadata | approved_at | created_at | ticket_id | approver_id
----+---------+--------+------------+-----------+----------+-------------+------------+-----------+-------------
(0 rows)

itsm=# INSERT INTO users (username, email, name, department, phone, password_hash, active, created_at, updated_at)
VALUES (
  'testuser',
  '<test@example.com>',
  '测试用户',
  'IT部门',
  '13800138000',
  '$2a$10$dummy.hash.for.testing',
  true,
  NOW(),
  NOW()
);
INSERT 0 1
itsm=# SELECT id, username, email, name FROM users WHERE username = 'testuser';
 id | username |      email       |   name
----+----------+------------------+----------
  1 | testuser | <test@example.com> | 测试用户
(1 row)

itsm=# select *from tickets;
itsm=# select* from tickets;
itsm=# \dt
             List of relations
 Schema |       Name       | Type  | Owner
--------+------------------+-------+-------
 public | approval_logs    | table | dev
 public | flow_instances   | table | dev
 public | service_catalogs | table | dev
 public | service_requests | table | dev
 public | status_logs      | table | dev
 public | tickets          | table | dev
 public | users            | table | dev
(7 rows)

itsm=# select * from service_catalogs;
 id | name | category | description | delivery_time | status | created_at | updated_at
----+------+----------+-------------+---------------+--------+------------+------------
(0 rows)
