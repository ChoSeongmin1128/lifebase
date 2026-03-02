"use client";

import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getAdminAccessToken } from "@/lib/admin-auth";
import { adminApi } from "@/lib/admin-api";

type AdminUser = {
  ID: string;
  UserID: string;
  Email: string;
  Name: string;
  Role: "admin" | "super_admin";
  IsActive: boolean;
};

export default function AdminAdminsPage() {
  const [admins, setAdmins] = useState<AdminUser[]>([]);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<"admin" | "super_admin">("admin");
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const roleLabel: Record<AdminUser["Role"], string> = {
    admin: "관리자",
    super_admin: "최고 관리자",
  };

  const load = async () => {
    const token = getAdminAccessToken();
    if (!token) return;
    setLoading(true);
    setError(null);
    try {
      const data = await adminApi<{ admins: AdminUser[] }>("/admin/admins", { token });
      setAdmins(data.admins);
    } catch (e) {
      setError(e instanceof Error ? e.message : "조회 실패");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const addAdmin = async () => {
    const token = getAdminAccessToken();
    if (!token) return;
    setError(null);
    setMessage(null);
    try {
      await adminApi("/admin/admins", {
        method: "POST",
        token,
        body: { email: email.trim(), role },
      });
      setMessage("관리자를 추가했습니다.");
      setEmail("");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "추가 실패");
    }
  };

  const updateRole = async (adminID: string, nextRole: "admin" | "super_admin") => {
    const token = getAdminAccessToken();
    if (!token) return;
    setError(null);
    setMessage(null);
    try {
      await adminApi(`/admin/admins/${adminID}/role`, {
        method: "PATCH",
        token,
        body: { role: nextRole },
      });
      setMessage("역할을 변경했습니다.");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "역할 변경 실패");
    }
  };

  const deactivate = async (adminID: string) => {
    const token = getAdminAccessToken();
    if (!token) return;
    setError(null);
    setMessage(null);
    try {
      await adminApi(`/admin/admins/${adminID}/deactivate`, {
        method: "PATCH",
        token,
      });
      setMessage("관리자를 비활성화했습니다.");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "비활성화 실패");
    }
  };

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold text-text-strong">관리자 관리</h1>

      {error ? <p className="text-sm text-error">{error}</p> : null}
      {message ? <p className="text-sm text-success">{message}</p> : null}
      {loading ? <p className="text-sm text-text-muted">로딩 중...</p> : null}

      <div className="rounded-lg border border-border bg-surface p-4">
        <p className="text-sm font-medium text-text-strong">새 관리자 추가</p>
        <div className="mt-2 flex flex-wrap items-center gap-2">
          <Input
            className="max-w-sm"
            placeholder="사용자 이메일"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
          <select
            className="h-9 rounded-lg border border-border bg-surface px-2 text-sm"
            value={role}
            onChange={(e) => setRole(e.target.value as "admin" | "super_admin")}
          >
            <option value="admin">관리자</option>
            <option value="super_admin">최고 관리자</option>
          </select>
          <Button variant="secondary" onClick={addAdmin}>
            추가
          </Button>
        </div>
      </div>

      <div className="overflow-x-auto rounded-lg border border-border bg-surface">
        <table className="w-full min-w-[760px] text-sm">
          <thead className="bg-surface-accent text-left text-text-secondary">
            <tr>
              <th className="px-3 py-2">이메일</th>
              <th className="px-3 py-2">이름</th>
              <th className="px-3 py-2">역할</th>
              <th className="px-3 py-2">활성</th>
              <th className="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {admins.map((admin) => (
              <tr key={admin.ID} className="border-t border-border">
                <td className="px-3 py-2">{admin.Email}</td>
                <td className="px-3 py-2">{admin.Name || "-"}</td>
                <td className="px-3 py-2">{roleLabel[admin.Role]}</td>
                <td className="px-3 py-2">{admin.IsActive ? "활성" : "비활성"}</td>
                <td className="px-3 py-2">
                  <div className="flex flex-wrap gap-2">
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => updateRole(admin.ID, admin.Role === "admin" ? "super_admin" : "admin")}
                    >
                      권한 전환
                    </Button>
                    {admin.IsActive ? (
                      <Button variant="danger" size="sm" onClick={() => deactivate(admin.ID)}>
                        비활성화
                      </Button>
                    ) : null}
                  </div>
                </td>
              </tr>
            ))}
            {admins.length === 0 && !loading ? (
              <tr>
                <td className="px-3 py-4 text-text-muted" colSpan={5}>
                  관리자 정보가 없습니다.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  );
}
