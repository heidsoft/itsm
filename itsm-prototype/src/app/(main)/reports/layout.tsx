import React from "react";
import AppLayout from "@/components/layout/AppLayout";

export default function ReportsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="报表分析" description="数据分析和报表" showPageHeader={true}>
      {children}
    </AppLayout>
  );
}