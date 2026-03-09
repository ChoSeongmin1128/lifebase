"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { setAdminTokens } from "@/features/admin/infrastructure/admin-auth";
import { Button } from "@/components/ui/button";
import { useAdminActions } from "@/features/admin/ui/hooks/useAdminActions";

function CallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [error, setError] = useState<string | null>(null);
  const code = searchParams.get("code");
  const { exchangeCode } = useAdminActions();

  useEffect(() => {
    if (!code) return;
    const runExchange = async () => {
      try {
        const state = sessionStorage.getItem("oauth_state_admin") || undefined;
        const data = await exchangeCode(code, state);

        setAdminTokens(data.access_token, data.refresh_token);
        router.replace("/admin");
      } catch (e) {
        const message = e instanceof Error ? e.message : "관리자 로그인에 실패했습니다.";
        if (message === "admin access denied") {
          setError("관리자 계정이 아닙니다.");
          return;
        }
        if (message === "admin check failed") {
          setError("관리자 권한 확인에 실패했습니다.");
          return;
        }
        setError(message);
      } finally {
        sessionStorage.removeItem("oauth_state_admin");
      }
    };
    runExchange();
  }, [code, exchangeCode, router]);

  if (!code) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <p className="text-error">인증 코드가 없습니다.</p>
          <Button variant="secondary" onClick={() => router.replace("/admin/login")}>
            돌아가기
          </Button>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <p className="text-error">{error}</p>
          <Button variant="secondary" onClick={() => router.replace("/admin/login")}>
            다시 로그인
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <p className="text-text-muted">관리자 로그인 처리 중...</p>
    </div>
  );
}

export default function AdminAuthCallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center">
          <p className="text-text-muted">관리자 로그인 처리 중...</p>
        </div>
      }
    >
      <CallbackContent />
    </Suspense>
  );
}
