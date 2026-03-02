"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";
import {
  clearAdminTokens,
  isAdminAuthenticated,
  isAdminTokenExpiringSoon,
  refreshAdminAccessToken,
} from "@/features/admin/infrastructure/admin-auth";
import { Button } from "@/components/ui/button";

const PUBLIC_PATHS = new Set(["/admin/login", "/admin/auth/callback"]);

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const isPublicPath = pathname ? PUBLIC_PATHS.has(pathname) : false;

  useEffect(() => {
    if (isPublicPath) return;
    if (!isAdminAuthenticated()) {
      router.replace("/admin/login");
      return;
    }
    if (isAdminTokenExpiringSoon()) {
      refreshAdminAccessToken().then((token) => {
        if (!token) {
          clearAdminTokens();
          router.replace("/admin/login");
        }
      });
    }
  }, [isPublicPath, router]);

  const handleLogout = () => {
    clearAdminTokens();
    router.replace("/admin/login");
  };

  if (isPublicPath) {
    return <>{children}</>;
  }

  return (
    <div className="min-h-screen bg-background text-text-primary">
      <header className="border-b border-border bg-surface">
        <div className="mx-auto flex h-14 w-full max-w-6xl items-center justify-between px-4">
          <div className="flex items-center gap-5">
            <Link href="/admin" className="text-sm font-semibold text-text-strong">
              LifeBase 관리자
            </Link>
            <nav className="flex items-center gap-3 text-sm text-text-secondary">
              <Link href="/admin/users" className="hover:text-text-strong">
                사용자
              </Link>
              <Link href="/admin/admins" className="hover:text-text-strong">
                관리자
              </Link>
            </nav>
          </div>
          <Button variant="secondary" size="sm" onClick={handleLogout}>
            로그아웃
          </Button>
        </div>
      </header>
      <main className="mx-auto w-full max-w-6xl px-4 py-6">{children}</main>
    </div>
  );
}
