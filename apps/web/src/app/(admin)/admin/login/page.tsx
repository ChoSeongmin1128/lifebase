"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { clearAdminTokens } from "@/features/admin/infrastructure/admin-auth";
import { useAdminActions } from "@/features/admin/ui/hooks/useAdminActions";

export default function AdminLoginPage() {
  const router = useRouter();
  const { requestAuthUrl } = useAdminActions();

  const handleLogin = async () => {
    try {
      clearAdminTokens();
      const data = await requestAuthUrl();
      sessionStorage.setItem("oauth_state_admin", data.state);
      window.location.href = data.url;
    } catch {
      alert("관리자 로그인 요청에 실패했습니다.");
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-b from-background to-surface-accent">
      <div className="flex w-full max-w-md flex-col gap-5 rounded-xl border border-border bg-surface p-6">
        <h1 className="text-xl font-semibold text-text-strong">LifeBase 관리자</h1>
        <p className="text-sm text-text-secondary">
          관리자 권한이 있는 Google 계정으로 로그인해야 접근할 수 있습니다.
        </p>
        <Button variant="primary" onClick={handleLogin}>
          Google로 관리자 로그인
        </Button>
        <Button variant="ghost" onClick={() => router.push("/")}>
          사용자 홈으로 이동
        </Button>
      </div>
    </div>
  );
}
