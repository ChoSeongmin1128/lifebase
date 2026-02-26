"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { api } from "@/lib/api";
import { setTokens } from "@/lib/auth";
import { Button } from "@/components/ui/button";

function CallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [error, setError] = useState<string | null>(null);
  const code = searchParams.get("code");

  useEffect(() => {
    if (!code) return;

    const exchangeCode = async () => {
      try {
        const data = await api<{
          access_token: string;
          refresh_token: string;
          expires_in: number;
        }>("/auth/callback", {
          method: "POST",
          body: { code },
        });

        setTokens(data.access_token, data.refresh_token);
        router.replace("/cloud");
      } catch {
        setError("로그인에 실패했습니다. 다시 시도해주세요.");
      }
    };

    exchangeCode();
  }, [code, router]);

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
