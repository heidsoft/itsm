import React from "react";
import AppLayout from "@/components/layout/AppLayout";

export default function IncidentsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="事件管理" description="处理IT事件和故障" showPageHeader={true}>
      {children}
    </AppLayout>
  );
}