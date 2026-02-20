/**
 * 通用系统常量
 */

export const UserRoles = {
    SUPER_ADMIN: 'super_admin',
    ADMIN: 'admin',
    MANAGER: 'manager',
    AGENT: 'agent',
    TECHNICIAN: 'technician',
    SECURITY: 'security',
    END_USER: 'end_user',
};

export const UserRoleLabels = {
    [UserRoles.SUPER_ADMIN]: '超级管理员',
    [UserRoles.ADMIN]: '管理员',
    [UserRoles.MANAGER]: '经理',
    [UserRoles.AGENT]: '座席',
    [UserRoles.TECHNICIAN]: '技术人员',
    [UserRoles.SECURITY]: '安全人员',
    [UserRoles.END_USER]: '普通用户',
};

export const TeamStatuses = {
    ACTIVE: 'active',
    INACTIVE: 'inactive',
};

export const TeamStatusLabels = {
    [TeamStatuses.ACTIVE]: '活跃',
    [TeamStatuses.INACTIVE]: '禁用',
};
