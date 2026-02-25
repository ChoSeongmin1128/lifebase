"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { api } from "@/lib/api";
import { setTokens } from "@/lib/auth";

function CallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const code = searchParams.get("code");
    if (!code) {
      setError("인증 코드가 없습니다.");
      return;
    }

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
  }, [searchParams, router]);

  if (error) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <p className="text-red-500">{error}</p>
          <button
            onClick={() => router.replace("/")}
            className="rounded-lg border border-foreground/10 px-4 py-2 transition-colors hover:bg-foreground/5"
          >
            돌아가기
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center">
      <p className="text-foreground/60">로그인 중...</p>
    </div>
  );
}

export default function AuthCallback() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center">
          <p className="text-foreground/60">로그인 중...</p>
        </div>
      }
    >
      <CallbackContent />
    </Suspense>
  );
}
