"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { formatBytes } from "@/lib/bytes";
import { useAdminActions } from "@/features/admin/ui/hooks/useAdminActions";
import type { UserItem } from "@/features/admin/domain/AdminEntities";

export default function AdminUsersPage() {
  const [q, setQ] = useState("");
  const [users, setUsers] = useState<UserItem[]>([]);
  const [nextCursor, setNextCursor] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const { listUsers } = useAdminActions();

  const load = useCallback(async (cursor?: string, query: string = "") => {
    setLoading(true);
    setError(null);
    try {
      const data = await listUsers(cursor, query);
      setUsers((prev) => (cursor ? [...prev, ...data.users] : data.users));
      setNextCursor(data.nextCursor || "");
    } catch (e) {
      setError(e instanceof Error ? e.message : "조회 실패");
    } finally {
      setLoading(false);
    }
  }, [listUsers]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-text-strong">사용자 관리</h1>
      </div>

      <div className="flex gap-2">
        <Input
          placeholder="이메일 또는 이름 검색"
          value={q}
            onChange={(e) => setQ(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") load(undefined, q.trim());
            }}
          />
        <Button variant="secondary" onClick={() => load(undefined, q.trim())}>
          검색
        </Button>
      </div>

      {error ? <p className="text-sm text-error">{error}</p> : null}
      {loading ? <p className="text-sm text-text-muted">로딩 중...</p> : null}

      <div className="overflow-x-auto rounded-lg border border-border bg-surface">
        <table className="w-full min-w-[720px] text-sm">
          <thead className="bg-surface-accent text-left text-text-secondary">
            <tr>
              <th className="px-3 py-2">이메일</th>
              <th className="px-3 py-2">이름</th>
              <th className="px-3 py-2">사용량</th>
              <th className="px-3 py-2">할당량</th>
              <th className="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {users.map((user) => (
              <tr key={user.ID} className="border-t border-border">
                <td className="px-3 py-2">{user.Email}</td>
                <td className="px-3 py-2">{user.Name || "-"}</td>
                <td className="px-3 py-2 tabular-nums">{formatBytes(user.StorageUsedBytes)}</td>
                <td className="px-3 py-2 tabular-nums">{formatBytes(user.StorageQuotaBytes)}</td>
                <td className="px-3 py-2 text-right">
                  <Link className="text-primary hover:underline" href={`/admin/users/${user.ID}`}>
                    상세
                  </Link>
                </td>
              </tr>
            ))}
            {users.length === 0 && !loading ? (
              <tr>
                <td className="px-3 py-4 text-text-muted" colSpan={5}>
                  조회 결과가 없습니다.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>

      {nextCursor ? (
        <Button variant="secondary" onClick={() => load(nextCursor, q.trim())}>
          더 보기
        </Button>
      ) : null}
    </div>
  );
}
