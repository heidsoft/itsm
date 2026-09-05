import { redirect } from 'next/navigation';

/**
 * 旧版“工单审批设计器”入口。
 *
 * 该页面与 /workflow/designer（标准 BPMN 设计器）功能重复，是历史并行实现，
 * 保留会造成两套设计器行为不一致。现统一收敛到 /workflow/designer，
 * 旧链接（菜单、收藏、外部跳转）通过本重定向继续可用。
 */
export default function TicketApprovalPage() {
  redirect('/workflow/designer');
}
