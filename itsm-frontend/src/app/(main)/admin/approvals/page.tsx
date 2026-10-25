import { redirect } from 'next/navigation';

// 旧「审批工作流」配置页：与 /admin/approval-chains（审批链）是同一配置面的
// 重复实现，且从未挂过菜单（0 入口 0 引用）。审批配置已收敛到审批链入口，
// 保留本路径仅作历史书签重定向。
export default function AdminApprovalsPage() {
  redirect('/admin/approval-chains');
}
