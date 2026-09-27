import { redirect } from 'next/navigation';

/**
 * 用户管理页面
 * 重定向到 /admin/users
 * 保留 /system/users 路由以兼容旧链接
 */
export default function UsersPage() {
  redirect('/admin/users');
}
