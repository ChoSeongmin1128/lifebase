"use client";

import { Suspense, useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  isAuthenticated,
  isTokenExpiringSoon,
  refreshAccessToken,
  clearTokens,
} from "@/features/auth/infrastructure/token-auth";
import { Sidebar } from "@/components/layout/Sidebar";
import { BottomTabBar } from "@/components/layout/BottomTabBar";
import { useSidebar } from "@/hooks/useSidebar";

export default function AuthenticatedLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const router = useRouter();
  const { expanded, toggle } = useSidebar();

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace("/");
      return;
    }

    // 앱 시작 시 토큰 만료 임박이면 선제 갱신
    if (isTokenExpiringSoon()) {
      refreshAccessToken().then((token) => {
        if (!token) {
          clearTokens();
          router.replace("/");
        }
      });
    }
  }, [router]);

  const handleLogout = () => {
    clearTokens();
    router.replace("/");
  };

  return (
    <div className="flex h-screen flex-col md:flex-row">
      <Suspense fallback={null}>
        <Sidebar expanded={expanded} onToggle={toggle} onLogout={handleLogout} />
      </Suspense>

      <main className="flex-1 overflow-auto pb-14 md:pb-0">
        {children}
      </main>

      <BottomTabBar />
    </div>
  );
}
