import AppLayout from "@/components/layout/AppLayout";

export default function WorkflowLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <AppLayout title="工作流管理" description="配置业务流程和自动化" showPageHeader={true}>{children}</AppLayout>;
}
