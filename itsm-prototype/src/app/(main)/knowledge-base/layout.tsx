import React from "react";
import AppLayout from "@/components/layout/AppLayout";

export default function KnowledgeBaseLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="知识库" description="技术文档和解决方案" showPageHeader={true}>
      {children}
    </AppLayout>
  );
}