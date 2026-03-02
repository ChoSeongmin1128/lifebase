"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useAdminActions } from "@/features/admin/ui/hooks/useAdminActions";

export default function AdminHomePage() {
  const [loading, setLoading] = useState(true);
  const [userCount, setUserCount] = useState<number | null>(null);
  const [adminCount, setAdminCount] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { listUsers, listAdmins } = useAdminActions();

  useEffect(() => {
    const run = async () => {
      try {
        const users = await listUsers();
        setUserCount(users.users.length);
        try {
          const admins = await listAdmins();
          setAdminCount(admins.length);
        } catch {
          setAdminCount(null);
        }
      } catch (e) {
        setError(e instanceof Error ? e.message : "로딩 실패");
      } finally {
        setLoading(false);
      }
    };
    run();
  }, [listAdmins, listUsers]);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-text-strong">운영 대시보드</h1>
        <p className="mt-1 text-sm text-text-secondary">사용자와 저장공간 운영 작업을 수행합니다.</p>
      </div>

      {loading ? <p className="text-sm text-text-muted">로딩 중...</p> : null}
      {error ? <p className="text-sm text-error">{error}</p> : null}

      <div className="grid gap-4 md:grid-cols-2">
        <div className="rounded-lg border border-border bg-surface p-4">
          <p className="text-xs text-text-muted">사용자 상태</p>
          <p className="mt-2 text-lg font-semibold text-text-strong">
            {userCount === null ? "-" : `조회 가능 사용자 ${userCount}명 (페이지 샘플)`}
          </p>
          <Link href="/admin/users" className="mt-3 inline-block text-sm text-primary hover:underline">
            사용자 관리로 이동
          </Link>
        </div>
        <div className="rounded-lg border border-border bg-surface p-4">
          <p className="text-xs text-text-muted">관리자 상태</p>
          <p className="mt-2 text-lg font-semibold text-text-strong">
            {adminCount === null ? "권한 확인 필요" : `관리자 ${adminCount}명`}
          </p>
          <Link href="/admin/admins" className="mt-3 inline-block text-sm text-primary hover:underline">
            관리자 관리로 이동
          </Link>
        </div>
      </div>
    </div>
  );
}
