import React from "react";
import AppLayout from "@/components/layout/AppLayout";

export default function ChangesLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="变更管理" description="管理IT变更和发布" showPageHeader={true}>
      {children}
    </AppLayout>
  );
}