import React from "react";
import AppLayout from "@/components/layout/AppLayout";

export default function ProblemsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="问题管理" description="分析根本原因和解决方案" showPageHeader={true}>
      {children}
    </AppLayout>
  );
}