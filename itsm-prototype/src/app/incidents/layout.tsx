import React from "react";
import AppLayout from "../components/AppLayout";

export default function IncidentsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="事件管理" breadcrumb={[{ title: "事件管理" }]}>
      {children}
    </AppLayout>
  );
}
