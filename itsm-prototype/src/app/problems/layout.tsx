import React from "react";
import AppLayout from "../components/AppLayout";

export default function ProblemsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="问题管理" breadcrumb={[{ title: "问题管理" }]}>
      {children}
    </AppLayout>
  );
}
