import React from "react";
import AppLayout from "../components/AppLayout";

export default function ReportsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="报表分析" breadcrumb={[{ title: "报表分析" }]}>
      {children}
    </AppLayout>
  );
}
