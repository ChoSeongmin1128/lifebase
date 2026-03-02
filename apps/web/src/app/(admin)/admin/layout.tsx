import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "LifeBase Admin",
};

export default function AdminSegmentLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return children;
}
