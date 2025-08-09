import React from "react";
import AppLayout from "../components/AppLayout";

export default function KnowledgeBaseLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <AppLayout title="知识库" breadcrumb={[{ title: "知识库" }]}>
      {children}
    </AppLayout>
  );
}
