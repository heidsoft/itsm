import React from "react";
import AppLayout from "../components/AppLayout";

export default function SLALayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="SLA管理" breadcrumb={[{ title: "SLA管理" }]}>
      {children}
    </AppLayout>
  );
}
