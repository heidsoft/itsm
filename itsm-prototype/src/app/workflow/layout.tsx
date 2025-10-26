import AppLayout from "@/components/layout/AppLayout";

export default function WorkflowLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <AppLayout>{children}</AppLayout>;
}
