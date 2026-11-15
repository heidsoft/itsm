import { redirect } from 'next/navigation';

/**
 * 顶级菜单「工作流」的落地页。
 *
 * 这里必须落到**运行概览**（/workflow/dashboard），不能落到流程定义管理。
 * 历史原因：本页曾 redirect 到 /admin/workflows（元数据 CRUD 页：新建/复制/启停/删除）。
 * 后果是用户点侧边栏的「工作流」（menubaseline 里它是 ParentPath 为空的顶级菜单，
 * SortOrder 100，与「SLA 管理」「AI 中心」同级），却被弹进「管理功能 → 工作流配置」这个
 * 系统管理子项，看到的是一张流程定义表格 —— 既不是"我的流程跑到哪了"，
 * 也不在预期位置，属于审计口径的父子语义错位。
 *
 * 落地页选型依据：dashboard 页消费 GET /bpmn/dashboard/metrics（实测 code=0，
 * 返回 totalProcesses / activeInstances / openTasks / processHealth / trendData），
 * 是唯一覆盖"运行态全貌"的页面；设计器、实例、版本管理都需要先选对象，
 * 不适合当模块首页。流程定义管理仍可从 /workflow 子菜单与 /admin/workflows 进入。
 */
export default function WorkflowPage() {
  redirect('/workflow/dashboard');
}
