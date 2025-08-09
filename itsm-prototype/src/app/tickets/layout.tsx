import React from "react";
import AppLayout from "../components/AppLayout";

export default function TicketsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="工单管理" breadcrumb={[{ title: "工单管理" }]}>
      {children}
    </AppLayout>
  );
}
