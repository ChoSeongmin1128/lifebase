"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuthFlow } from "@/features/auth/ui/hooks/useAuthFlow";
import { setTokens } from "@/features/auth/infrastructure/token-auth";
import { Button } from "@/components/ui/button";

function CallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [error, setError] = useState<string | null>(null);
  const code = searchParams.get("code");
  const { exchangeCode } = useAuthFlow();

  useEffect(() => {
    if (!code) return;

    const runExchange = async () => {
      try {
        const state = sessionStorage.getItem("oauth_state") || undefined;
        const data = await exchangeCode({ code, state, app: "web" });

        setTokens(data.access_token, data.refresh_token);
        router.replace("/cloud");
      } catch {
        setError("로그인에 실패했습니다. 다시 시도해주세요.");
      } finally {
        sessionStorage.removeItem("oauth_state");
      }
    };

    runExchange();
  }, [code, exchangeCode, router]);

  if (!code) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <p className="text-error">인증 코드가 없습니다.</p>
          <Button variant="secondary" onClick={() => router.replace("/")}>
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
          <Button variant="secondary" onClick={() => router.replace("/")}>
            돌아가기
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <p className="text-text-muted">로그인 중...</p>
    </div>
  );
}

export default function AuthCallback() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center">
          <p className="text-text-muted">로그인 중...</p>
        </div>
      }
    >
      <CallbackContent />
    </Suspense>
  );
}
