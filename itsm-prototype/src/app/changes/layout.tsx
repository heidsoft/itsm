import React from "react";
import AppLayout from "../components/AppLayout";

export default function ChangesLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="变更管理" breadcrumb={[{ title: "变更管理" }]}>
      {children}
    </AppLayout>
  );
}
