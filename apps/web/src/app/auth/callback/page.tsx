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
  const { exchangeCode, linkGoogleAccount } = useAuthFlow();

  useEffect(() => {
    if (!code) return;

    const runExchange = async () => {
      const state = sessionStorage.getItem("oauth_state") || undefined;
      const intent = sessionStorage.getItem("oauth_intent");
      const returnPath = sessionStorage.getItem("oauth_return_path") || "/settings";

      try {
        if (intent === "link_google_account") {
          await linkGoogleAccount({ code, state, app: "web" });
          router.replace(returnPath);
          return;
        }

        const data = await exchangeCode({ code, state, app: "web" });
        setTokens(data.access_token, data.refresh_token);
        router.replace("/home");
      } catch {
        if (intent === "link_google_account") {
          setError("Google 계정 추가 연결에 실패했습니다. 다시 시도해주세요.");
          return;
        }
        setError("로그인에 실패했습니다. 다시 시도해주세요.");
      } finally {
        sessionStorage.removeItem("oauth_state");
        sessionStorage.removeItem("oauth_intent");
        sessionStorage.removeItem("oauth_return_path");
      }
    };

    runExchange();
  }, [code, exchangeCode, linkGoogleAccount, router]);

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
