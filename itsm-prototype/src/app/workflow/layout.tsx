import { AppLayout } from "../components/AppLayout";

export default function WorkflowLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <AppLayout>{children}</AppLayout>;
}
