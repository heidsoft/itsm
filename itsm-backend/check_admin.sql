-- 检查管理员账号
SELECT id, username, email, role, active, created_at 
FROM users 
WHERE username = 'admin';

-- 检查用户总数
SELECT COUNT(*) as total_users FROM users;

-- 检查角色
SELECT DISTINCT role FROM users;
